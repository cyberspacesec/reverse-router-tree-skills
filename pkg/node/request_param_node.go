package node

import "strings"

// RequestParamNode 定义了查询参数节点，用于匹配URL查询参数（如?name=value）
type RequestParamNode struct {
	*BaseNode[NodeContext]
	required bool // 参数是否必需
}

// NewRequestParamNode 创建一个新的查询参数节点
// 参数:
//   - name: 参数名称，如 "page", "size" 等
//   - defaultValue: 默认值，当参数不存在时使用
//   - required: 参数是否必需
//
// 返回:
//   - *RequestParamNode: 新创建的查询参数节点
func NewRequestParamNode(name string, defaultValue string, required bool) *RequestParamNode {
	context := NewBaseNodeContext()
	baseNode := NewBaseNode[NodeContext]("request_param", name, defaultValue, context)

	return &RequestParamNode{
		BaseNode: baseNode,
		required: required,
	}
}

// IsMatch 重写匹配方法，判断请求的查询参数是否匹配
// 参数:
//   - queryString: 完整的查询字符串或参数名称
//
// 返回:
//   - bool: 如果查询参数匹配则返回true，否则返回false
func (n *RequestParamNode) IsMatch(queryString string) bool {
	// 简单检查参数名是否存在
	paramName := n.GetKey()

	// 只针对参数名匹配
	if queryString == paramName {
		return true
	}

	// 对于完整的查询字符串，检查是否包含此参数
	if strings.Contains(queryString, paramName+"=") {
		return true
	}

	// 如果参数不是必需的，也视为匹配
	if !n.required {
		return true
	}

	return false
}

// GetParamName 获取参数名称
// 返回:
//   - string: 参数名称
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
	return n.required
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

	// 提取参数值
	paramValue := n.extractParam(queryString, paramName)

	// 如果未找到值但有默认值
	if paramValue == "" && n.GetDefaultValue() != "" {
		paramValue = n.GetDefaultValue()
	}

	// 如果是必需参数但未提供值
	if n.required && paramValue == "" {
		return false
	}

	// 在上下文中存储参数值
	context.SetKey(paramName, paramValue)
	return true
}

// extractParam 从查询字符串中提取特定参数的值
// 参数:
//   - queryString: 查询字符串，格式为 "param1=value1&param2=value2"
//   - paramName: 要提取的参数名
//
// 返回:
//   - string: 提取到的参数值，如果未找到则返回空字符串
func (n *RequestParamNode) extractParam(queryString string, paramName string) string {
	// 分割查询字符串为参数对
	params := strings.Split(queryString, "&")
	for _, param := range params {
		keyValue := strings.Split(param, "=")
		if len(keyValue) == 2 && keyValue[0] == paramName {
			return keyValue[1]
		}
	}
	return ""
}

// 确保 RequestParamNode 实现了 Node 接口
var _ Node[NodeContext] = (*RequestParamNode)(nil)
