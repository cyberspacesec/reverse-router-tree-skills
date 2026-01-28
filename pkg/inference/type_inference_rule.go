package inference

import (
	"github.com/cyberspacesec/go-reverse-router-tree/pkg/node"
	"github.com/cyberspacesec/go-reverse-router-tree/pkg/value"
)

// TypeInferenceRule 类型推断规则接口
type TypeInferenceRule interface {
	// Infer 根据节点上下文推断类型
	Infer(node node.Node[node.NodeContext]) (value.Type, error)
}
