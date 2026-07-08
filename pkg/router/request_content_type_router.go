package router

import (
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
)

// RequestContentTypeRouter 根据HTTP请求的Content-Type进行路由
type RequestContentTypeRouter struct {
}

var _ Router[string] = (*RequestContentTypeRouter)(nil)

// FindNode 根据Content-Type查找匹配的节点
// 参数:
//   - n: 起始节点
//   - contentType: 请求的Content-Type值，如 "application/json"
func (x *RequestContentTypeRouter) FindNode(n node.Node[node.NodeContext], contentType string) (node.Node[node.NodeContext], error) {
	// 检查节点是否是 RequestContentTypeNode 类型（指针类型断言）
	ctNode, ok := n.(*node.RequestContentTypeNode)
	if !ok {
		return nil, nil
	}

	// 如果没有Content-Type，则看节点是否也是空的
	if contentType == "" {
		if ctNode.GetKey() == "" {
			return ctNode, nil
		}
		return nil, nil
	}

	// 匹配Content-Type
	if ctNode.IsMatch(contentType) {
		return ctNode, nil
	}

	return nil, nil
}
