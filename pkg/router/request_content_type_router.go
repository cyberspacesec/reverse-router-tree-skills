package router

import (
	"github.com/cyberspacesec/go-reverse-router-tree/pkg/node"
	"github.com/cyberspacesec/go-reverse-router-tree/pkg/request"
)

type RequestContentTypeRouter struct {
}

var _ Router[[]*request.HttpRequestPath] = (*RequestContentTypeRouter)(nil)

func (x *RequestContentTypeRouter) FindNode(node node.Node[node.NodeContext], requestPaths []*request.HttpRequestPath) (node.Node[node.NodeContext], error) {
	// 检查节点是否是 RequestPathNode 类型
	requestPathNode, ok := node.(node.RequestPathNode)
	if !ok {
		return nil, nil
	}

	// 如果没有路径，则表示是一个根路径，则看下给定的节点是否也是根节点
	if len(requestPaths) == 0 {
		return requestPathNode.GetKey() == "", nil
	}

	for _, exceptRequestPath := range requestPaths {
		childNode := requestPathNode.FindChildByKey(exceptRequestPath.Path)
		if childNode == nil {
			return nil, nil
		}
		requestPathNode = childNode
	}

	// TODO: 实现路径匹配逻辑
	return requestPathNode, nil
}
