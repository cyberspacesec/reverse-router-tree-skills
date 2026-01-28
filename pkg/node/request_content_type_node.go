package node

// RequestContentTypeNode 定义了内容类型节点，用于匹配请求的Content-Type头
type RequestContentTypeNode struct {
	*BaseNode[NodeContext]
}

// NewRequestContentTypeNode 创建一个新的内容类型节点
// 参数:
//   - contentType: 要匹配的内容类型，如 "application/json", "text/plain" 等
//
// 返回:
//   - *RequestContentTypeNode: 新创建的内容类型节点
func NewRequestContentTypeNode(contentType string) *RequestContentTypeNode {
	context := NewBaseNodeContext()
	baseNode := NewBaseNode[NodeContext]("request_content_type", contentType, "", context)

	return &RequestContentTypeNode{
		BaseNode: baseNode,
	}
}

// IsMatch 重写匹配方法，判断请求的Content-Type是否匹配
// 参数:
//   - contentType: 请求的Content-Type值
//
// 返回:
//   - bool: 如果Content-Type匹配则返回true，否则返回false
func (n *RequestContentTypeNode) IsMatch(contentType string) bool {
	// 简单匹配，对于更复杂的匹配可以扩展此方法
	// 例如，可以支持通配符或部分匹配（application/*）
	return n.GetKey() == contentType
}

// 确保 RequestContentTypeNode 实现了 Node 接口
var _ Node[NodeContext] = (*RequestContentTypeNode)(nil)
