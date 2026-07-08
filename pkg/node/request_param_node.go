package node

import (
	"strings"
	"sync"
	"sync/atomic"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

// RequestParamNode 定义了查询参数节点，用于匹配URL查询参数（如?name=value）
//
// 该节点支持：
//   - 参数名精确匹配（大小写不敏感）
//   - 参数值观察和统计（ValueMetric）
//   - 多值参数支持（如 ?tag=go&tag=web）
//   - 类型推断（物理类型 + 逻辑类型）
//   - 必需/可选参数区分
//   - 必需性自动推断（基于参数在请求中的出现频率）
type RequestParamNode struct {
	*BaseNode[NodeContext]
	required      bool                // 参数是否必需
	valueMetric   *value.ValueMetric  // 值统计
	valueType     value.Type          // 推断的值类型
	logicalType   value.LogicalType   // 推断的逻辑类型
	multiValue    bool                // 是否为多值参数（同一参数名出现多次）
	presenceCount int64               // 参数在请求中出现的次数（用于必需性推断，atomic）
	// typeMu 保护 required/valueType/logicalType/multiValue 的并发读写。
	// 多个 goroutine 命中同一已存在参数节点时，findOrCreateParamNode 会并发
	// 推断并回填类型/必需性，需同步保护。presenceCount 已用 atomic，单独处理。
	typeMu sync.RWMutex
}

// NewRequestParamNode 创建一个新的查询参数节点
// 参数:
//   - name: 参数名称，如 "page", "size" 等（会被转为小写）
//   - defaultValue: 默认值，当参数不存在时使用
//   - required: 参数是否必需
//
// 返回:
//   - *RequestParamNode: 新创建的查询参数节点
func NewRequestParamNode(name string, defaultValue string, required bool) *RequestParamNode {
	context := NewBaseNodeContext()
	// 参数名统一小写存储，确保大小写不敏感匹配
	normalizedName := strings.ToLower(name)
	baseNode := NewBaseNode[NodeContext]("request_param", normalizedName, defaultValue, context)

	paramNode := &RequestParamNode{
		BaseNode:    baseNode,
		required:    required,
		valueMetric: value.NewValueMetric(),
		valueType:   value.Type(value.PhysicalTypeString),
		logicalType: value.LogicalTypeString,
		multiValue:  false,
	}

	// 如果有默认值，自动观察它
	if defaultValue != "" {
		paramNode.valueMetric.AddValue(defaultValue)
	}

	return paramNode
}

// IsMatch 重写匹配方法，判断请求的查询参数是否匹配
// 参数:
//   - queryString: 完整的查询字符串或参数名称
//
// 返回:
//   - bool: 如果查询参数匹配则返回true，否则返回false
func (n *RequestParamNode) IsMatch(queryString string) bool {
	paramName := n.GetKey()

	// 精确参数名匹配（大小写不敏感）
	if strings.EqualFold(queryString, paramName) {
		return true
	}

	// 对于完整的查询字符串，检查是否包含此参数
	// 使用更精确的匹配：参数名必须出现在查询字符串的开头，
	// 或者紧跟在 & 之后，然后紧跟 = 号
	if n.containsParam(queryString, paramName) {
		return true
	}

	// 非必需参数在没有出现时不应该匹配
	// 以前的逻辑：!required → return true，这太宽松了
	// 修复：非必需参数只在参数名实际出现时才匹配
	return false
}

// containsParam 检查查询字符串中是否包含指定的参数名
// 使用更精确的匹配逻辑，避免误匹配（如 "page_size" 不应该匹配 "page"）
// 大小写不敏感匹配
func (n *RequestParamNode) containsParam(queryString, paramName string) bool {
	// 将查询字符串转为小写进行匹配
	lowerQuery := strings.ToLower(queryString)
	lowerParam := strings.ToLower(paramName)

	// 查找 paramName= 的位置
	target := lowerParam + "="

	// 检查是否在开头
	if strings.HasPrefix(lowerQuery, target) {
		return true
	}

	// 检查是否在 & 之后
	target = "&" + target
	return strings.Contains(lowerQuery, target)
}

// GetParamName 获取参数名称
// 返回:
//   - string: 参数名称（小写）
func (n *RequestParamNode) GetParamName() string {
	return n.GetKey()
}

// GetDefaultValue 获取参数默认值
// 返回:
//   - string: 参数默认值
func (n *RequestParamNode) GetDefaultValue() string {
	return n.GetValue()
}

// IsRequired 检查参数是否必需
// 返回:
//   - bool: 如果参数是必需的则返回true，否则返回false
func (n *RequestParamNode) IsRequired() bool {
	n.typeMu.RLock()
	defer n.typeMu.RUnlock()
	return n.required
}

// SetRequired 设置参数是否必需
func (n *RequestParamNode) SetRequired(required bool) {
	n.typeMu.Lock()
	defer n.typeMu.Unlock()
	n.required = required
}

// IncrementPresenceCount 增加参数在请求中出现的次数
// 每次请求包含此参数时调用，用于必需性推断
func (n *RequestParamNode) IncrementPresenceCount() {
	atomic.AddInt64(&n.presenceCount, 1)
}

// GetPresenceCount 获取参数在请求中出现的次数
func (n *RequestParamNode) GetPresenceCount() int64 {
	return atomic.LoadInt64(&n.presenceCount)
}

// SetPresenceCount 设置参数在请求中出现的次数
// 主要用于从持久化数据（如JSON）恢复节点状态
func (n *RequestParamNode) SetPresenceCount(count int64) {
	atomic.StoreInt64(&n.presenceCount, count)
}

// InferRequired 基于参数出现频率推断是否为必需参数
//
// 必需参数的判定依据：参数在同一路由的请求中出现频率足够高。
// 出现率 = 参数出现次数 / 路由总请求次数。
//
// 参数:
//   - totalRequestCount: 该路由（方法节点）被请求的总次数
//   - threshold: 出现率阈值（0.0-1.0），出现率 >= threshold 判定为必需
//
// 返回:
//   - bool: 推断的必需性
//
// 注意：
//   - 当 totalRequestCount <= 1 时无法可靠推断，返回当前 required 值（保持默认）
//   - 该方法会更新 required 字段
func (n *RequestParamNode) InferRequired(totalRequestCount int64, threshold float64) bool {
	// 样本不足，无法可靠推断，保持现状
	if totalRequestCount <= 1 {
		n.typeMu.RLock()
		defer n.typeMu.RUnlock()
		return n.required
	}

	presenceCount := n.GetPresenceCount()
	presenceRatio := float64(presenceCount) / float64(totalRequestCount)

	// 读-改-写需在同一把锁内，避免并发 InferRequired 间的 TOCTOU
	n.typeMu.Lock()
	defer n.typeMu.Unlock()
	n.required = presenceRatio >= threshold
	return n.required
}

// IsMultiValue 检查是否为多值参数
func (n *RequestParamNode) IsMultiValue() bool {
	n.typeMu.RLock()
	defer n.typeMu.RUnlock()
	return n.multiValue
}

// SetMultiValue 设置多值参数标记
func (n *RequestParamNode) SetMultiValue(multiValue bool) {
	n.typeMu.Lock()
	defer n.typeMu.Unlock()
	n.multiValue = multiValue
}

// ObserveValue 记录观察到的参数值，用于类型推断
func (n *RequestParamNode) ObserveValue(val string) {
	n.valueMetric.AddValue(val)
}

// GetValueMetric 获取值统计信息
func (n *RequestParamNode) GetValueMetric() *value.ValueMetric {
	return n.valueMetric
}

// GetValueType 获取推断的值类型
func (n *RequestParamNode) GetValueType() value.Type {
	n.typeMu.RLock()
	defer n.typeMu.RUnlock()
	return n.valueType
}

// SetValueType 设置值类型
func (n *RequestParamNode) SetValueType(t value.Type) {
	n.typeMu.Lock()
	defer n.typeMu.Unlock()
	n.valueType = t
}

// GetLogicalType 获取推断的逻辑类型
func (n *RequestParamNode) GetLogicalType() value.LogicalType {
	n.typeMu.RLock()
	defer n.typeMu.RUnlock()
	return n.logicalType
}

// SetLogicalType 设置逻辑类型
func (n *RequestParamNode) SetLogicalType(lt value.LogicalType) {
	n.typeMu.Lock()
	defer n.typeMu.Unlock()
	n.logicalType = lt
}

// ExtractValue 从查询字符串中提取参数值并存储在上下文中
// 参数:
//   - queryString: 完整的查询字符串
//
// 返回:
//   - bool: 如果成功提取并存储参数值则返回true，否则返回false
func (n *RequestParamNode) ExtractValue(queryString string) bool {
	paramName := n.GetParamName()
	context := n.GetContext()

	// 提取参数值（支持多值）
	paramValues := n.extractParamValues(queryString, paramName)

	// 如果未找到值但有默认值
	if len(paramValues) == 0 && n.GetDefaultValue() != "" {
		paramValues = []string{n.GetDefaultValue()}
	}

	// 如果是必需参数但未提供值
	if n.required && len(paramValues) == 0 {
		return false
	}

	// 记录观察到的值
	for _, val := range paramValues {
		if val != "" {
			n.ObserveValue(val)
		}
	}

	// 多值参数用逗号连接存储
	if len(paramValues) > 1 {
		n.SetMultiValue(true)
		context.SetKey(paramName, strings.Join(paramValues, ","))
	} else if len(paramValues) == 1 {
		context.SetKey(paramName, paramValues[0])
	} else {
		context.SetKey(paramName, "")
	}
	return true
}

// extractParamValues 从查询字符串中提取特定参数的所有值（多值版本）
// 支持同一参数名出现多次的情况，如 ?tag=go&tag=web
func (n *RequestParamNode) extractParamValues(queryString string, paramName string) []string {
	var values []string
	params := strings.Split(queryString, "&")
	for _, param := range params {
		keyValue := strings.SplitN(param, "=", 2)
		if len(keyValue) >= 1 && strings.EqualFold(keyValue[0], paramName) {
			if len(keyValue) == 2 {
				values = append(values, keyValue[1])
			} else {
				values = append(values, "")
			}
		}
	}
	return values
}

// 确保 RequestParamNode 实现了 Node 接口
var _ Node[NodeContext] = (*RequestParamNode)(nil)
