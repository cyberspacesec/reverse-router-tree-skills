package tree

import "github.com/cyberspacesec/go-reverse-router-tree/pkg/node"

type Tree struct {
	Root node.Node[node.NodeContext]
}

func NewTree() *Tree {
	// Create a new BaseNode with a BaseNodeContext as the root
	rootContext := node.NewBaseNodeContext()
	root := node.NewBaseNode[node.NodeContext]("root", rootContext)

	return &Tree{
		Root: root,
	}
}

func (x *Tree) AddNode(path string, node node.Node[node.NodeContext]) error {
	return nil
}
