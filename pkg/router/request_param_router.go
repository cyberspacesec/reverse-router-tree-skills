package router

import (
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
)

// RequestParamRouter 根据HTTP请求参数进行路由
type RequestParamRouter struct {
}

// 确保类型实现了接口
var _ Router[[]*request.HttpRequestParam] = (*RequestParamRouter)(nil)

// FindNode 根据请求参数查找匹配的节点
func (x *RequestParamRouter) FindNode(n node.Node[node.NodeContext], requestParams []*request.HttpRequestParam) (node.Node[node.NodeContext], error) {
	// 检查节点是否是 RequestParamNode 类型（指针类型断言）
	requestParamNode, ok := n.(*node.RequestParamNode)
	if !ok {
		return nil, nil
	}

	// 如果没有参数，则看是否能匹配没有参数的节点
	if len(requestParams) == 0 {
		if requestParamNode.GetChildCount() == 0 {
			return requestParamNode, nil
		}
		return nil, nil
	}

	// 遍历所有请求参数，尝试找到匹配的子节点
	currentNode := node.Node[node.NodeContext](requestParamNode)
	for _, requestParam := range requestParams {
		// 首先尝试精确匹配参数名和值
		paramKey := requestParam.Name + "=" + requestParam.Value
		childNode := currentNode.FindChildByKey(paramKey)

		// 如果没找到，尝试匹配参数名的通配符
		if childNode == nil {
			paramKey = requestParam.Name + "=*"
			childNode = currentNode.FindChildByKey(paramKey)
		}

		// 如果仍未找到，则无法继续匹配
		if childNode == nil {
			return nil, nil
		}

		currentNode = childNode
	}

	return currentNode, nil
}
