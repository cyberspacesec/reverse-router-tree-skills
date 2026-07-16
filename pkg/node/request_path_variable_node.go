package node

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

// RequestPathVariableNode 定义了路径变量节点
// 该节点通过黑盒流量分析动态识别出的路径变量
// 不同于白盒路由定义，这里的路径变量是根据实际请求流量推断出来的
type RequestPathVariableNode struct {
	*BaseNode[NodeContext]

	// 值统计，用于记录观察到的路径变量值
	valueMetric *value.ValueMetric
	// 推断出的值类型（物理类型）
	valueType value.Type
	// 推断出的逻辑类型（在物理类型之上的语义类型）
	logicalType value.LogicalType
	// typeMu 保护 valueType/logicalType/lastInferredUniqueCount 的并发读写。
	// 多个 goroutine 命中同一已存在变量节点时，findOrCreatePathNode 会并发
	// ObserveValue + InferPhysicalAndLogical 回填类型，需同步保护。
	typeMu sync.RWMutex
	// 可选的正则模式，用于匹配路径变量值（构造后只读，无需锁）
	pattern *regexp.Regexp
	// lastInferredUniqueCount 上次类型推断时的 unique 值数。
	// findOrCreatePathNode 用它判断是否需要重新推断：仅当 unique 值数自上次
	// 推断后变化时才重算，避免每个请求都对全部累积值做 O(N) 全量推断
	// （合并后的变量节点可能累积上千个值，消除 O(N²)）。用 typeMu 保护。
	// 初值 0 表示"从未推断"，首个值（uniqueCount=1）即触发首次推断。
	lastInferredUniqueCount int
}

// NewRequestPathVariableNode 创建一个新的路径变量节点
// 该节点初始没有预定义的名称或模式，而是随着流量分析逐步建立模式
//
// 参数:
//   - position: 在路径中的位置标识符（如 "id" 表示用户ID变量）
//   - patternStr: 可选的正则模式字符串，用于匹配路径变量值。空字符串表示匹配任何非空值
//
// 返回:
//   - *RequestPathVariableNode: 新创建的路径变量节点
func NewRequestPathVariableNode(position string, patternStr string) *RequestPathVariableNode {
	context := NewBaseNodeContext()
	// 初始没有明确的value，它们会根据观察到的流量推断
	baseNode := NewBaseNode[NodeContext]("request_path_variable", position, "", context)

	node := &RequestPathVariableNode{
		BaseNode:    baseNode,
		valueMetric: value.NewValueMetric(),
		valueType:   value.Type(value.PhysicalTypeString), // 默认为字符串类型
		logicalType: value.LogicalTypeString,              // 默认逻辑类型为 string
		pattern:     nil,
	}

	// 如果提供了正则模式，编译并设置
	if patternStr != "" {
		re, err := regexp.Compile(patternStr)
		if err == nil {
			node.pattern = re
		}
		// 如果编译失败，忽略模式，节点将匹配任何非空值
	}

	return node
}

// IsMatch 判断请求的路径段是否匹配
// 在黑盒分析模式下，如果有正则模式则按模式匹配，否则认为变量节点可以匹配任何非空路径段
// 但会排除一些明显不是变量的路径段：
//   - 包含文件扩展名的段（如 data.json, style.css）除非模式明确匹配
//   - 纯字母的常见固定路径词（如 api, users, admin）除非模式明确匹配
//
// 参数:
//   - pathSegment: URL路径的一个段
//
// 返回:
//   - bool: 如果路径段可能是变量则返回true，否则返回false
func (n *RequestPathVariableNode) IsMatch(pathSegment string) bool {
	// 变量路径段不应该为空，并且不应该包含路径分隔符
	if pathSegment == "" || pathSegment == "/" {
		return false
	}

	// 如果有正则模式，使用模式匹配（严格匹配）
	if n.pattern != nil {
		return n.pattern.MatchString(pathSegment)
	}

	// 没有模式时，使用启发式规则判断
	// 排除明显是固定路径的段

	// 排除包含常见文件扩展名的段（如 .json, .xml, .html, .css, .js, .png 等）
	// 这些通常是固定资源路径，不是变量
	if hasFileExtension(pathSegment) {
		return false
	}

	// 记录观察到的值，用于后续类型推断
	n.ObserveValue(pathSegment)

	return true
}

// hasFileExtension 检查路径段是否包含文件扩展名
// 常见的静态资源扩展名
func hasFileExtension(segment string) bool {
	// 常见的静态文件扩展名
	staticExtensions := []string{
		".html", ".htm", ".css", ".js", ".json", ".xml",
		".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico",
		".pdf", ".doc", ".docx", ".xls", ".xlsx",
		".txt", ".csv", ".zip", ".tar", ".gz",
		".mp3", ".mp4", ".avi", ".mov",
		".woff", ".woff2", ".ttf", ".eot",
	}

	lower := strings.ToLower(segment)
	for _, ext := range staticExtensions {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

// ObserveValue 观察一个路径段值，累积到值统计中用于后续类型推断。
//
// 注意：本方法只收集值，不触发类型推断。类型推断由 router 在调用点用
// chainRule.InferPhysicalAndLogical 统一回填物理+逻辑类型（与 RequestParamNode
// 的处理模式一致），避免单一 Infer 把逻辑类型串（如 "uuid"）污染到物理类型字段。
//
// 参数:
//   - val: 观察到的路径段值
func (n *RequestPathVariableNode) ObserveValue(val string) {
	n.valueMetric.AddValue(val)
}

// GetValueType 获取推断出的值类型
//
// 返回:
//   - value.Type: 推断出的值类型
func (n *RequestPathVariableNode) GetValueType() value.Type {
	n.typeMu.RLock()
	defer n.typeMu.RUnlock()
	return n.valueType
}

// SetType 设置值类型
func (n *RequestPathVariableNode) SetType(t value.Type) {
	n.typeMu.Lock()
	defer n.typeMu.Unlock()
	n.valueType = t
}

// GetLogicalType 获取推断出的逻辑类型
//
// 返回:
//   - value.LogicalType: 推断出的逻辑类型
func (n *RequestPathVariableNode) GetLogicalType() value.LogicalType {
	n.typeMu.RLock()
	defer n.typeMu.RUnlock()
	return n.logicalType
}

// SetLogicalType 设置逻辑类型
func (n *RequestPathVariableNode) SetLogicalType(lt value.LogicalType) {
	n.typeMu.Lock()
	defer n.typeMu.Unlock()
	n.logicalType = lt
}

// GetPositionIdentifier 获取变量在路径中的位置标识符
//
// 返回:
//   - string: 变量的位置标识符
func (n *RequestPathVariableNode) GetPositionIdentifier() string {
	return n.GetKey()
}

// GetValueMetric 获取值统计信息
//
// 返回:
//   - *value.ValueMetric: 值统计信息
func (n *RequestPathVariableNode) GetValueMetric() *value.ValueMetric {
	return n.valueMetric
}

// GetLastInferredUniqueCount 返回上次类型推断时的 unique 值数（typeMu 读锁）。
// 增量推断缓存：调用方比较它与当前 uniqueCount，仅在变化时才重算推断。
func (n *RequestPathVariableNode) GetLastInferredUniqueCount() int {
	n.typeMu.RLock()
	defer n.typeMu.RUnlock()
	return n.lastInferredUniqueCount
}

// SetLastInferredUniqueCount 设置上次类型推断时的 unique 值数（typeMu 写锁）。
// 推断完成后调用，记录本次推断对应的 uniqueCount，供下次命中判断是否需重算。
func (n *RequestPathVariableNode) SetLastInferredUniqueCount(c int) {
	n.typeMu.Lock()
	defer n.typeMu.Unlock()
	n.lastInferredUniqueCount = c
}

// GetPattern 获取正则模式
//
// 返回:
//   - *regexp.Regexp: 正则模式，如果没有设置则返回nil
func (n *RequestPathVariableNode) GetPattern() *regexp.Regexp {
	return n.pattern
}

// ExtractValue 从路径段中提取变量值并存储在上下文中
//
// 参数:
//   - pathSegment: URL路径的一个段
//
// 返回:
//   - bool: 如果成功提取并存储变量值则返回true，否则返回false
func (n *RequestPathVariableNode) ExtractValue(pathSegment string) bool {
	if n.IsMatch(pathSegment) {
		// 在上下文中存储变量值，键为位置标识符
		varKey := n.GetPositionIdentifier()
		context := n.GetContext()
		context.SetKey(varKey, pathSegment)
		return true
	}
	return false
}

// String 返回变量节点的字符串表示
//
// 返回:
//   - string: 变量节点的字符串表示，格式为 "{POSITION:PATTERN}" 或 "{POSITION:TYPE}"
func (n *RequestPathVariableNode) String() string {
	position := n.GetPositionIdentifier()

	if n.pattern != nil {
		return fmt.Sprintf("{%s:%s}", position, n.pattern.String())
	}

	typeStr := string(n.GetValueType())
	return fmt.Sprintf("{%s:%s}", position, typeStr)
}

// IsDynamic 判断当前节点是否为动态节点
// 路径变量节点始终是动态的
func (n *RequestPathVariableNode) IsDynamic() bool {
	return true
}

// 确保 RequestPathVariableNode 实现了 Node 接口
var _ Node[NodeContext] = (*RequestPathVariableNode)(nil)
