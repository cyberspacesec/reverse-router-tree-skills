package router

import (
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
)

// RequestPathRouter 根据 http 路径来路由，寻找到最末端的那个节点
type RequestPathRouter struct {
}

var _ Router[[]*request.HttpRequestPath] = (*RequestPathRouter)(nil)

// FindNode 从起点节点 n 沿 requestPaths 逐段下钻，定位路径末端节点。
//
// 起点不再限定为 *RequestPathNode，可从根节点等任意节点出发，便于
// ReverseRouter 查询侧复用。命中规则与 locateMethodNode 一致：
//   - 路径段优先按 key 精确匹配固定路径子节点；
//   - 未命中时无条件回退到 request_path_variable 子节点（不调 IsMatch，
//     与查询侧简化语义一致，使 /api/users/456 能命中已合并的 {users_id}）；
//   - 二者皆无则返回 nil。
//
// 参数:
//   - n: 起始节点
//   - requestPaths: 路径段列表
//
// 返回:
//   - 路径末端节点；任意段未命中或入参非法时返回 nil
func (x *RequestPathRouter) FindNode(n node.Node[node.NodeContext], requestPaths []*request.HttpRequestPath) (node.Node[node.NodeContext], error) {
	if n == nil {
		return nil, nil
	}
	// 空路径返回起点，与 locateMethodNode 空路径后 FindChildByKey(method) 一致
	if len(requestPaths) == 0 {
		return n, nil
	}

	currentNode := n
	for _, exceptRequestPath := range requestPaths {
		childNode := currentNode.FindChildByKey(exceptRequestPath.Path)
		if childNode == nil {
			// 无条件回退到路径变量节点（不调 IsMatch），与 locateMethodNode 一致
			pathVarChild := currentNode.GetChildByType("request_path_variable")
			if pathVarChild != nil {
				currentNode = pathVarChild
				continue
			}
			return nil, nil
		}
		currentNode = childNode
	}

	return currentNode, nil
}
