package router

import (
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/tree"
)

// 用于对请求进行逆向工程，反推出web应用的路由树
type ReverseRouter struct {
	Tree *tree.Tree
}

func (x *ReverseRouter) FindNode(node node.Node[node.NodeContext], routerContext node.NodeContext) (node.Node[node.NodeContext], error) {
	return nil, nil
}

// 核心方法之一：逆向一个请求
func (x *ReverseRouter) ReverseHttpRequest(request *request.HttpRequest) error {
	// 对于路径：
	// 判断此请求是否有对应的节点
	// 没有的话则添加对应的节点
	// 检测兄弟节点的数量，如果较多的话，则说明可能是路径变量，则需要合并识别出来，对于路径变量，可能是在中间，所以对于路径变量，并不是合并叶子结点，而是合并子树，这是一个需要明确的问题
	//
	// 对于http方法：
	// 判断路径节点下面的方法节点，不存在的话就新创建一个
	//
	// 对于参数：
	//  判断是否已经存在过请求参数
	// 如果存在，则记录一下这个请求也能命中
	// 如果不存在，则添加对应的参数节点
	//
	// TODO 2025-03-26 03:48:30 对于参数，需要能够识别出来路径参数那种，比如action=xxx
	//
	return nil, nil
}

// 核心方法：判断某个链接是否还需要请求
func (x *ReverseRouter) IsNeedRequest(request *request.HttpRequest) bool {
	// 同时满足以下两个条件，则请求将不再发送：
	// 1. 这个请求能够在路由树上找到对应的节点
	// 2. 这个节点已经被请求过若干次了，则不再需要一直重复请求了
}
