package node

// coverage_gap_main2_test.go — node 包第二轮补齐：
// VisitLevelOrder 队列扩容的两个子分支（按需扩容 / 翻倍即够），
// 以及 AddChild / RemoveChild / MergeWith 中 SetParent 与 AddChild
// 失败的防御分支（用内嵌 BaseNode 的桩节点触发，不改业务代码）。
// 只新增测试，不改业务代码。

import (
	"errors"
	"fmt"
	"testing"
)

// gapStubNode 内嵌 BaseNode，按需覆写个别方法以触发错误分支。
type gapStubNode struct {
	*BaseNode[NodeContext]
	setParentErr error
	cloneRet     Node[NodeContext]
	cloneRetSet  bool
	childrenStub []Node[NodeContext]
	childrenSet  bool
}

func (s *gapStubNode) SetParent(p Node[NodeContext]) error {
	if s.setParentErr != nil {
		return s.setParentErr
	}
	return s.BaseNode.SetParent(p)
}

func (s *gapStubNode) DeepClone() Node[NodeContext] {
	if s.cloneRetSet {
		return s.cloneRet
	}
	return s.BaseNode.DeepClone()
}

func (s *gapStubNode) GetChildren() []Node[NodeContext] {
	if s.childrenSet {
		return s.childrenStub
	}
	return s.BaseNode.GetChildren()
}

// AddChild 的 SetParent 失败分支：桩子节点的 SetParent 返回错误。
func TestGapNode2_AddChildSetParentFail(t *testing.T) {
	parent := NewBaseNode[NodeContext]("root", "root", "", NewBaseNodeContext())
	stub := &gapStubNode{
		BaseNode:     NewBaseNode[NodeContext]("c", "k", "", NewBaseNodeContext()),
		setParentErr: errors.New("桩错误"),
	}
	if err := parent.AddChild(stub); err == nil {
		t.Error("SetParent 失败时 AddChild 应返回错误")
	}
}

// RemoveChild 的解绑失败分支：先正常挂载，再让 SetParent(nil) 报错。
func TestGapNode2_RemoveChildUnbindFail(t *testing.T) {
	parent := NewBaseNode[NodeContext]("root", "root", "", NewBaseNodeContext())
	stub := &gapStubNode{
		BaseNode: NewBaseNode[NodeContext]("c", "k", "", NewBaseNodeContext()),
	}
	if err := parent.AddChild(stub); err != nil {
		t.Fatal(err)
	}
	stub.setParentErr = errors.New("桩错误")
	if err := parent.RemoveChild(stub); err == nil {
		t.Error("解绑失败时 RemoveChild 应返回错误")
	}
}

// MergeWith 的合入失败分支：源节点的子节点 DeepClone 返回 nil，
// 导致 AddChild(nil) 失败，走“合并子节点失败”分支。
func TestGapNode2_MergeWithAddFail(t *testing.T) {
	dst := NewBaseNode[NodeContext]("root", "dst", "", NewBaseNodeContext())
	if err := dst.AddChild(NewBaseNode[NodeContext]("c", "k", "old", NewBaseNodeContext())); err != nil {
		t.Fatal(err)
	}
	badChild := &gapStubNode{
		BaseNode:    NewBaseNode[NodeContext]("c", "fresh", "", NewBaseNodeContext()),
		cloneRet:    nil,
		cloneRetSet: true,
	}
	src := &gapStubNode{
		BaseNode:     NewBaseNode[NodeContext]("root", "src", "", NewBaseNodeContext()),
		childrenStub: []Node[NodeContext]{badChild},
		childrenSet:  true,
	}
	if err := dst.MergeWith(src); err == nil {
		t.Error("子节点合入失败时 MergeWith 应返回错误")
	}
}

// VisitLevelOrder 按需扩容分支：内层 newCapacity 不足，按实际需要扩容。
func TestGapNode2_LevelOrderGrowOnDemand(t *testing.T) {
	root := NewBaseNode[NodeContext]("root", "root", "", NewBaseNodeContext())
	mid := NewBaseNode[NodeContext]("mid", "mid", "", NewBaseNodeContext())
	if err := root.AddChild(mid); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 200; i++ {
		c := NewBaseNode[NodeContext]("c", fmt.Sprintf("k%03d", i), "", NewBaseNodeContext())
		if err := mid.AddChild(c); err != nil {
			t.Fatal(err)
		}
	}
	count := 0
	root.VisitLevelOrder(func(Node[NodeContext]) bool {
		count++
		return true
	})
	if count != 202 {
		t.Errorf("层序访问数=%d want 202", count)
	}
}

// VisitLevelOrder 翻倍即够分支：容量翻倍后无需再按需追加。
func TestGapNode2_LevelOrderGrowDoubleEnough(t *testing.T) {
	root := NewBaseNode[NodeContext]("root", "root", "", NewBaseNodeContext())
	for i := 0; i < 16; i++ {
		p := NewBaseNode[NodeContext]("p", fmt.Sprintf("p%02d", i), "", NewBaseNodeContext())
		if err := root.AddChild(p); err != nil {
			t.Fatal(err)
		}
		for j := 0; j < 10; j++ {
			c := NewBaseNode[NodeContext]("c", fmt.Sprintf("c%02d%02d", i, j), "", NewBaseNodeContext())
			if err := p.AddChild(c); err != nil {
				t.Fatal(err)
			}
		}
	}
	count := 0
	root.VisitLevelOrder(func(Node[NodeContext]) bool {
		count++
		return true
	})
	if count != 177 {
		t.Errorf("层序访问数=%d want 177", count)
	}
}
