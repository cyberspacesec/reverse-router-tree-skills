package router

import (
	"strings"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
)

// RequestParamRouter 根据 HTTP 请求参数名进行路由
type RequestParamRouter struct {
}

// 确保类型实现了接口
var _ Router[string] = (*RequestParamRouter)(nil)

// FindNode 从起点节点 n 查找名为 paramName 的查询参数子节点。
//
// 起点不再限定为 *RequestParamNode，可从方法节点等任意节点出发，
// 便于 ReverseRouter 查询侧复用。参数名会被转为小写以匹配参数节点的
// 存储 key（与构建侧 findOrCreateParamNode / NewRequestParamNode 一致）。
//
// 参数:
//   - n: 起始节点
//   - paramName: 参数名（大小写不敏感）
//
// 返回:
//   - 命中的参数子节点；未命中或入参非法时返回 nil
func (x *RequestParamRouter) FindNode(n node.Node[node.NodeContext], paramName string) (node.Node[node.NodeContext], error) {
	if n == nil || paramName == "" {
		return nil, nil
	}
	// 参数名统一小写（与 request_param_node.go NewRequestParamNode 对齐）
	// 与查询侧 IsNeedRequest 现状一致：不加 type 校验
	if child := n.FindChildByKey(strings.ToLower(paramName)); child != nil {
		return child, nil
	}
	return nil, nil
}
