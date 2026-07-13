package router

import (
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
)

// RequestContentTypeRouter 根据 HTTP 请求的 Content-Type 进行路由
type RequestContentTypeRouter struct {
}

var _ Router[string] = (*RequestContentTypeRouter)(nil)

// FindNode 从起点节点 n 查找匹配 contentType 的 Content-Type 子节点。
//
// 起点不再限定为 *RequestContentTypeNode，可从方法节点等任意节点出发，
// 便于 ReverseRouter 查询侧复用。匹配语义为按子节点 key 精确查找
// （等价于原 RequestContentTypeNode.IsMatch 的 GetKey()==contentType）。
//
// 参数:
//   - n: 起始节点
//   - contentType: 请求的 Content-Type 值，如 "application/json"
//
// 返回:
//   - 命中的子节点；未命中或入参非法时返回 nil
func (x *RequestContentTypeRouter) FindNode(n node.Node[node.NodeContext], contentType string) (node.Node[node.NodeContext], error) {
	if n == nil || contentType == "" {
		return nil, nil
	}
	// 与 IsNeedRequest / FindRouteNode 的 FindChildByKey(contentType) 等价
	if child := n.FindChildByKey(contentType); child != nil {
		return child, nil
	}
	return nil, nil
}
