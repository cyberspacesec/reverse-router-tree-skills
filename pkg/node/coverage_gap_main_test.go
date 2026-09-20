package node

// coverage_gap_main_test.go — 主线程补齐 pkg/node 未覆盖分支：
// BaseNode 缓存命中（GetRoot/GetAllAncestors）、AddChild 重复添加、
// RemoveChild nil/未找到/类型索引单多、RemoveChildByType 混合、
// 兄弟查找未命中、Deserialize 空与非法输入、VisitLevelOrder 提前终止与
// 队列扩容及非 BaseNode 分支、FindNode 自命中、cookie 值节点命中复用、
// param/pathvariable 推断计数器与 ExtractValue 各分支。
// 只新增测试，不改业务代码。

import (
	"strings"
	"testing"
)

// --- BaseNode 缓存与增删分支 ---

func TestGapNode_RootAndAncestorsCache(t *testing.T) {
	ctx := NewBaseNodeContext()
	root := NewBaseNode[NodeContext]("root", "root", "", ctx)
	mid := NewBaseNode[NodeContext]("mid", "mid", "", ctx)
	leaf := NewBaseNode[NodeContext]("leaf", "leaf", "", ctx)
	if err := root.AddChild(mid); err != nil {
		t.Fatal(err)
	}
	if err := mid.AddChild(leaf); err != nil {
		t.Fatal(err)
	}
	// 两次调用：第二次命中 cachedRoot / cachedAncestors 分支
	if got := leaf.GetRoot(); got != Node[NodeContext](root) {
		t.Error("GetRoot 应返回根节点")
	}
	if got := leaf.GetRoot(); got != Node[NodeContext](root) {
		t.Error("GetRoot 缓存命中应返回根节点")
	}
	a1 := leaf.GetAllAncestors()
	a2 := leaf.GetAllAncestors() // 缓存副本分支
	if len(a1) != 2 || len(a2) != 2 {
		t.Errorf("祖先数=%d/%d want 2/2", len(a1), len(a2))
	}
	// 根节点自身两次调用同样覆盖缓存分支
	root.GetRoot()
	if got := root.GetRoot(); got != Node[NodeContext](root) {
		t.Error("根 GetRoot 应返回自身")
	}
}

func TestGapNode_AddChildDuplicate(t *testing.T) {
	ctx := NewBaseNodeContext()
	root := NewBaseNode[NodeContext]("root", "root", "", ctx)
	child := NewBaseNode[NodeContext]("c", "k", "", ctx)
	if err := root.AddChild(child); err != nil {
		t.Fatal(err)
	}
	// 同一指针重复添加走“已存在直接返回”分支
	if err := root.AddChild(child); err != nil {
		t.Errorf("重复添加同一子节点应返回 nil，实际 %v", err)
	}
	if got := root.GetChildCount(); got != 1 {
		t.Errorf("子节点数=%d want 1", got)
	}
}

func TestGapNode_RemoveChildBranches(t *testing.T) {
	ctx := NewBaseNodeContext()
	root := NewBaseNode[NodeContext]("root", "root", "", ctx)
	// nil 子节点分支
	if err := root.RemoveChild(nil); err == nil {
		t.Error("移除 nil 应报错")
	}
	// 不在列表分支
	orphan := NewBaseNode[NodeContext]("c", "orphan", "", ctx)
	if err := root.RemoveChild(orphan); err == nil {
		t.Error("移除陌生节点应报错")
	}
	// 类型索引多节点分支：两个同类型子节点，移除其中一个走“切片移除”分支
	a := NewBaseNode[NodeContext]("same", "a", "", ctx)
	b := NewBaseNode[NodeContext]("same", "b", "", ctx)
	if err := root.AddChild(a); err != nil {
		t.Fatal(err)
	}
	if err := root.AddChild(b); err != nil {
		t.Fatal(err)
	}
	if err := root.RemoveChild(a); err != nil {
		t.Fatal(err)
	}
	if got := root.GetChildCount(); got != 1 {
		t.Errorf("移除后子节点数=%d want 1", got)
	}
	// 类型索引单节点分支：移除最后一个该类型走“删除整个索引项”分支
	if err := root.RemoveChild(b); err != nil {
		t.Fatal(err)
	}
	if root.GetChildByType("same") != nil {
		t.Error("类型索引应已整体删除")
	}
}

func TestGapNode_RemoveChildByTypeMixed(t *testing.T) {
	ctx := NewBaseNodeContext()
	root := NewBaseNode[NodeContext]("root", "root", "", ctx)
	a := NewBaseNode[NodeContext]("t1", "a", "", ctx)
	b := NewBaseNode[NodeContext]("t2", "b", "", ctx)
	c := NewBaseNode[NodeContext]("t1", "c", "", ctx)
	for _, ch := range []Node[NodeContext]{a, b, c} {
		if err := root.AddChild(ch); err != nil {
			t.Fatal(err)
		}
	}
	// 混合类型：同时走保留分支与移除分支
	root.RemoveChildByType("t1")
	if got := root.GetChildCount(); got != 1 {
		t.Errorf("按类型移除后=%d want 1", got)
	}
	if got := root.FindChildByKey("b"); got == nil {
		t.Error("t2 子节点应保留")
	}
	// 不存在的类型：静默无操作
	root.RemoveChildByType("不存在")
	if got := root.GetChildCount(); got != 1 {
		t.Errorf("移除不存在类型后=%d want 1", got)
	}
}

func TestGapNode_SiblingMiss(t *testing.T) {
	ctx := NewBaseNodeContext()
	root := NewBaseNode[NodeContext]("root", "root", "", ctx)
	a := NewBaseNode[NodeContext]("ta", "a", "", ctx)
	b := NewBaseNode[NodeContext]("tb", "b", "", ctx)
	if err := root.AddChild(a); err != nil {
		t.Fatal(err)
	}
	if err := root.AddChild(b); err != nil {
		t.Fatal(err)
	}
	if got := a.GetSiblingByType("不存在"); got != nil {
		t.Error("无匹配类型兄弟应返回 nil")
	}
	if got := a.GetSiblingByKey("不存在"); got != nil {
		t.Error("无匹配键名兄弟应返回 nil")
	}
	// 命中分支对照
	if got := a.GetSiblingByType("tb"); got != Node[NodeContext](b) {
		t.Error("应找到 tb 类型兄弟")
	}
	if got := a.GetSiblingByKey("b"); got != Node[NodeContext](b) {
		t.Error("应找到键名为 b 的兄弟")
	}
}

// --- Deserialize ---

func TestGapNode_DeserializeBranches(t *testing.T) {
	ctx := NewBaseNodeContext()
	n := NewBaseNode[NodeContext]("t", "k", "v", ctx)
	if err := n.Deserialize(nil); err == nil {
		t.Error("空输入应报错")
	}
	if err := n.Deserialize([]byte("{非法")); err == nil {
		t.Error("非法 JSON 应报错")
	}
	if err := n.Deserialize([]byte(`{"type":"nt","key":"nk","value":"nv"}`)); err != nil {
		t.Fatalf("合法输入应成功：%v", err)
	}
	if n.GetType() != "nt" || n.GetKey() != "nk" || n.GetValue() != "nv" {
		t.Error("反序列化后属性不符合预期")
	}
}

// --- 遍历分支 ---

func TestGapNode_VisitLevelOrderBranches(t *testing.T) {
	ctx := NewBaseNodeContext()
	root := NewBaseNode[NodeContext]("root", "root", "", ctx)
	kid := NewBaseNode[NodeContext]("c", "k", "", ctx)
	if err := root.AddChild(kid); err != nil {
		t.Fatal(err)
	}
	// 提前终止分支
	visits := 0
	root.VisitLevelOrder(func(Node[NodeContext]) bool {
		visits++
		return false
	})
	if visits != 1 {
		t.Errorf("提前终止访问数=%d want 1", visits)
	}
	// 非 BaseNode 分支：挂一个 RequestParamNode（断言 *BaseNode 失败走接口分支）
	p := NewRequestParamNode("q", "", false)
	if err := root.AddChild(p); err != nil {
		t.Fatal(err)
	}
	total := 0
	root.VisitLevelOrder(func(Node[NodeContext]) bool {
		total++
		return true
	})
	if total != 3 {
		t.Errorf("层序访问数=%d want 3", total)
	}
	// 队列扩容分支：200 子节点的宽树触发 cap 翻倍与按需扩容
	wide := NewBaseNode[NodeContext]("root", "wide", "", NewBaseNodeContext())
	for i := 0; i < 200; i++ {
		c := NewBaseNode[NodeContext]("c", strings.Repeat("k", 3)+string(rune('a'+i%26))+string(rune('0'+i/26)), "", NewBaseNodeContext())
		if err := wide.AddChild(c); err != nil {
			t.Fatal(err)
		}
	}
	count := 0
	wide.VisitLevelOrder(func(Node[NodeContext]) bool {
		count++
		return true
	})
	if count != 201 {
		t.Errorf("宽树访问数=%d want 201", count)
	}
}

func TestGapNode_FindNodeSelf(t *testing.T) {
	ctx := NewBaseNodeContext()
	root := NewBaseNode[NodeContext]("root", "root", "", ctx)
	kid := NewBaseNode[NodeContext]("c", "k", "", ctx)
	if err := root.AddChild(kid); err != nil {
		t.Fatal(err)
	}
	// 谓词对自身为真，走直接返回分支
	if got := root.FindNode(func(Node[NodeContext]) bool { return true }); got != Node[NodeContext](root) {
		t.Error("谓词恒真应返回自身")
	}
}

// --- MergeWith 跳过分支 ---

func TestGapNode_MergeWithSkipExisting(t *testing.T) {
	ctx := NewBaseNodeContext()
	dst := NewBaseNode[NodeContext]("root", "dst", "", ctx)
	src := NewBaseNode[NodeContext]("root", "src", "", ctx)
	keep := NewBaseNode[NodeContext]("c", "k", "old", ctx)
	if err := dst.AddChild(keep); err != nil {
		t.Fatal(err)
	}
	dup := NewBaseNode[NodeContext]("c", "k", "new", ctx)
	fresh := NewBaseNode[NodeContext]("c", "fresh", "", ctx)
	if err := src.AddChild(dup); err != nil {
		t.Fatal(err)
	}
	if err := src.AddChild(fresh); err != nil {
		t.Fatal(err)
	}
	if err := dst.MergeWith(src); err != nil {
		t.Fatal(err)
	}
	// 同键跳过：保留旧值；新键合入
	if got := dst.FindChildByKey("k").GetValue(); got != "old" {
		t.Errorf("同键应保留旧值，实际 %q", got)
	}
	if got := dst.FindChildByKey("fresh"); got == nil {
		t.Error("新键应合入")
	}
}

// --- cookie / header 值节点复用 ---

func TestGapNode_CookieValueNodeReuse(t *testing.T) {
	n := NewRequestCookieNode("sess")
	first := n.FindOrCreateValueNode("abc")
	if first == nil {
		t.Fatal("首次应创建值节点")
	}
	// 第二次同值命中已存在分支并复用
	second := n.FindOrCreateValueNode("abc")
	if second == nil {
		t.Fatal("复用应非 nil")
	}
	if second != first {
		t.Error("同值应复用同一值节点")
	}
	if got := n.GetChildCount(); got != 1 {
		t.Errorf("值节点数=%d want 1", got)
	}
}

func TestGapNode_HeaderValueNodeReuse(t *testing.T) {
	n := NewRequestHeaderNode("Accept")
	first := n.FindOrCreateValueNode("application/json")
	if first == nil {
		t.Fatal("首次应创建值节点")
	}
	second := n.FindOrCreateValueNode("application/json")
	if second == nil {
		t.Fatal("复用应非 nil")
	}
	if second != first {
		t.Error("同值应复用同一值节点")
	}
}

// --- param / pathvariable 计数器与提取分支 ---

func TestGapNode_ParamInferredCount(t *testing.T) {
	p := NewRequestParamNode("q", "", false)
	p.SetLastInferredUniqueCount(7)
	if got := p.GetLastInferredUniqueCount(); got != 7 {
		t.Errorf("计数=%d want 7", got)
	}
}

func TestGapNode_ParamExtractBranches(t *testing.T) {
	// 必需参数缺失：走 required 返回 false 分支
	req := NewRequestParamNode("q", "", true)
	if req.ExtractValue("other=1") {
		t.Error("必需参数缺失应返回 false")
	}
	// 非必需且无值：走空值 SetKey("") 分支
	opt := NewRequestParamNode("q", "", false)
	if !opt.ExtractValue("other=1") {
		t.Error("非必需参数缺失应返回 true")
	}
	// 无等号参数：走 extractParamValues 无值分支
	flag := NewRequestParamNode("flag", "", false)
	if !flag.ExtractValue("flag") {
		t.Error("无等号参数应返回 true")
	}
	v, ok := flag.GetContext().GetKey("flag")
	if !ok || v != "" {
		t.Errorf("无等号参数应存空串，实际 %v/%v", v, ok)
	}
}

func TestGapNode_PathVarBranches(t *testing.T) {
	v := NewRequestPathVariableNode("id", "")
	// 空与斜杠不匹配分支
	if v.IsMatch("") {
		t.Error("空段不应匹配")
	}
	if v.IsMatch("/") {
		t.Error("'/' 不应匹配")
	}
	// ExtractValue 失败分支
	if v.ExtractValue("") {
		t.Error("空段提取应返回 false")
	}
	// 计数器分支
	v.SetLastInferredUniqueCount(3)
	if got := v.GetLastInferredUniqueCount(); got != 3 {
		t.Errorf("计数=%d want 3", got)
	}
	// 成功提取对照
	if !v.ExtractValue("123") {
		t.Error("数字段应匹配成功")
	}
}
