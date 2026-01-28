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
