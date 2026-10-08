package node

// 这个文件定义了请求路径节点的实现
type RequestPathNode struct {
	*BaseNode[NodeContext]
}

// NewRequestPathNode 创建一个新的请求路径节点
func NewRequestPathNode(path string) *RequestPathNode {
	context := NewBaseNodeContext()
	baseNode := NewBaseNode[NodeContext]("request_path", path, "", context)

	return &RequestPathNode{
		BaseNode: baseNode,
	}
}

// 确保 RequestPathNode 实现了 Node 接口
var _ Node[NodeContext] = (*RequestPathNode)(nil)

// Clone 保留 request_path 类型。BaseNode.Clone 只会得到 BaseNode，
// 合并子树时按具体类型断言会失败。
func (n *RequestPathNode) Clone() Node[NodeContext] {
	c := NewRequestPathNode(n.GetKey())
	c.SetValue(n.GetValue())
	return c
}
func (n *RequestPathNode) DeepClone() Node[NodeContext] {
	return n.deepCloneInto(n.Clone())
}
