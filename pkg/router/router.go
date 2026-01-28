package router

import "github.com/cyberspacesec/go-reverse-router-tree/pkg/node"

// Router 路由器接口
// 这是一个泛型接口，RouterContext 表示路由上下文类型
// 路由器负责根据给定的上下文找到对应的节点
type Router[RouterContext any] interface {
	// FindNode 根据路由上下文查找节点
	// 参数:
	//   - node: 起始节点，通常是根节点
	//   - routerContext: 路由上下文，包含查找节点所需的信息
	// 返回:
	//   - 找到的节点
	//   - 错误信息，如果查找过程中出现问题
	FindNode(node node.Node[node.NodeContext], routerContext RouterContext) (node.Node[node.NodeContext], error)
}
