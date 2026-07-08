package node

import (
	"fmt"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

// RequestHeaderNode 定义了HTTP请求头路由节点
// 某些Web服务会根据特定Header做路由决策，例如：
//   - Accept: application/json vs text/html → 返回不同格式
//   - Authorization: Bearer vs Basic → 不同认证方式
//   - X-Api-Version: v1 vs v2 → API 版本路由
//   - Accept-Language: zh-CN vs en-US → 多语言路由
//
// 树结构设计（两层）：
//   - 第一层：Header名称节点（key=headerName，如 "Accept"）
//   - 第二层：Header值节点（key=normalizedValue，如 "application/json"）
//
// 这样同一个Header的不同值会作为兄弟节点挂在Header名称节点下，
// 便于后续做变量合并（如多个不同的Accept值可能合并为变量）。
//
// 此节点与 Content-Type 节点不同：
//   - Content-Type 是请求体的格式，仅用于 POST/PUT/PATCH
//   - Header 节点可以是任意 header，用于请求路由
type RequestHeaderNode struct {
	*BaseNode[NodeContext]
	headerName string // header 名称（如 "Accept", "X-Api-Version"）
}

// NewRequestHeaderNode 创建一个新的Header路由分组节点
// key 使用 headerName（如 "Accept"），用于在方法节点下查找
func NewRequestHeaderNode(headerName string) *RequestHeaderNode {
	context := NewBaseNodeContext()
	baseNode := NewBaseNode[NodeContext]("request_header", headerName, headerName, context)

	return &RequestHeaderNode{
		BaseNode:    baseNode,
		headerName:  headerName,
	}
}

// GetHeaderName 获取Header名称
func (n *RequestHeaderNode) GetHeaderName() string {
	return n.headerName
}

// FindOrCreateValueNode 查找或创建Header值子节点
func (n *RequestHeaderNode) FindOrCreateValueNode(headerValue string) *RequestHeaderValueNode {
	// 查找已有的值节点
	child := n.FindChildByKey(headerValue)
	if child != nil && child.GetType() == "request_header_value" {
		valueNode := child.(*RequestHeaderValueNode)
		valueNode.ObserveValue(headerValue)
		return valueNode
	}

	// 创建新的值节点
	newValueNode := NewRequestHeaderValueNode(n.headerName, headerValue)
	if err := n.AddChild(newValueNode); err == nil {
		return newValueNode
	}
	return nil
}

// String 返回节点的字符串表示
func (n *RequestHeaderNode) String() string {
	return fmt.Sprintf("%s [Header]", n.headerName)
}

// 确保 RequestHeaderNode 实现了 Node 接口
var _ Node[NodeContext] = (*RequestHeaderNode)(nil)

// RequestHeaderValueNode Header值节点
// 作为 RequestHeaderNode 的子节点，存储具体的Header值
type RequestHeaderValueNode struct {
	*BaseNode[NodeContext]
	headerName  string             // 所属的header名称
	headerValue string             // 规范化后的header值
	valueMetric *value.ValueMetric // 观察到的 header 值统计
}

// NewRequestHeaderValueNode 创建一个新的Header值节点
func NewRequestHeaderValueNode(headerName, headerValue string) *RequestHeaderValueNode {
	context := NewBaseNodeContext()
	baseNode := NewBaseNode[NodeContext]("request_header_value", headerValue, headerName, context)

	return &RequestHeaderValueNode{
		BaseNode:    baseNode,
		headerName:  headerName,
		headerValue: headerValue,
		valueMetric: value.NewValueMetric(),
	}
}

// GetHeaderName 获取所属Header名称
func (n *RequestHeaderValueNode) GetHeaderName() string {
	return n.headerName
}

// GetHeaderValue 获取Header值
func (n *RequestHeaderValueNode) GetHeaderValue() string {
	return n.headerValue
}

// IsMatch 判断给定的header值是否匹配
func (n *RequestHeaderValueNode) IsMatch(headerValue string) bool {
	return n.headerValue == headerValue
}

// ObserveValue 记录观察到的header值
func (n *RequestHeaderValueNode) ObserveValue(val string) {
	n.valueMetric.AddValue(val)
}

// GetValueMetric 获取值统计信息
func (n *RequestHeaderValueNode) GetValueMetric() *value.ValueMetric {
	return n.valueMetric
}

// String 返回节点的字符串表示
func (n *RequestHeaderValueNode) String() string {
	return fmt.Sprintf("%s: %s [HeaderValue]", n.headerName, n.headerValue)
}

// 确保 RequestHeaderValueNode 实现了 Node 接口
var _ Node[NodeContext] = (*RequestHeaderValueNode)(nil)
