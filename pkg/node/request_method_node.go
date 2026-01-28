package node

// RequestMethodNode 定义了HTTP方法节点，用于匹配请求的HTTP方法（GET, POST, PUT等）
type RequestMethodNode struct {
	*BaseNode[NodeContext]
}

// HTTP方法常量
const (
	MethodGET     = "GET"
	MethodPOST    = "POST"
	MethodPUT     = "PUT"
	MethodDELETE  = "DELETE"
	MethodPATCH   = "PATCH"
	MethodHEAD    = "HEAD"
	MethodOPTIONS = "OPTIONS"
)

// NewRequestMethodNode 创建一个新的HTTP方法节点
// 参数:
//   - method: HTTP方法，如 "GET", "POST" 等
//
// 返回:
//   - *RequestMethodNode: 新创建的HTTP方法节点
func NewRequestMethodNode(method string) *RequestMethodNode {
	context := NewBaseNodeContext()
	baseNode := NewBaseNode[NodeContext]("request_method", method, "", context)

	return &RequestMethodNode{
		BaseNode: baseNode,
	}
}

// IsMatch 重写匹配方法，判断请求的HTTP方法是否匹配
// 参数:
//   - method: 请求的HTTP方法
//
// 返回:
//   - bool: 如果HTTP方法匹配则返回true，否则返回false
func (n *RequestMethodNode) IsMatch(method string) bool {
	return n.GetKey() == method
}

// 确保 RequestMethodNode 实现了 Node 接口
var _ Node[NodeContext] = (*RequestMethodNode)(nil)
