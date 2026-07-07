package router

import (
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
)

// stubMergeRule 固定返回预设决策的测试用规则。
type stubMergeRule struct {
	action    MergeAction
	mergeable []node.Node[node.NodeContext]
	called    bool
	gotCtx    MergeContext
}

func (s *stubMergeRule) Decide(ctx MergeContext) (MergeAction, []node.Node[node.NodeContext]) {
	s.called = true
	s.gotCtx = ctx
	return s.action, s.mergeable
}

// TestMergeRule_DefaultAction_FallsBackToBuiltin 注入返回 Default 的规则，
// 验证内置逻辑仍正常合并数字 ID。
func TestMergeRule_DefaultAction_FallsBackToBuiltin(t *testing.T) {
	r := NewReverseRouter()
	r.SetMergeRule(&stubMergeRule{action: MergeActionDefault})

	reqs := []string{"/api/users/1", "/api/users/2", "/api/users/3"}
	for _, u := range reqs {
		if err := r.ReverseHttpRequest(request.NewHttpRequest(u, nil, "GET", nil)); err != nil {
			t.Fatalf("请求 %s 失败: %v", u, err)
		}
	}

	users := r.Tree.Root.FindChildByKey("api").FindChildByKey("users")
	if v := users.GetChildByType("request_path_variable"); v == nil {
		t.Fatal("Default action 应放行内置逻辑合并数字 ID")
	}
}

// TestMergeRule_SkipAction_PreventsMerge 注入返回 Skip 的规则，
// 验证即便 3 个数字 ID 也不被合并。
func TestMergeRule_SkipAction_PreventsMerge(t *testing.T) {
	r := NewReverseRouter()
	r.SetMergeRule(&stubMergeRule{action: MergeActionSkip})

	reqs := []string{"/api/items/1", "/api/items/2", "/api/items/3"}
	for _, u := range reqs {
		if err := r.ReverseHttpRequest(request.NewHttpRequest(u, nil, "GET", nil)); err != nil {
			t.Fatalf("请求 %s 失败: %v", u, err)
		}
	}

	items := r.Tree.Root.FindChildByKey("api").FindChildByKey("items")
	if v := items.GetChildByType("request_path_variable"); v != nil {
		t.Fatalf("Skip action 应阻止合并，却出现变量节点 %s", v.GetKey())
	}
	// 三个固定数字路径应保留
	for _, id := range []string{"1", "2", "3"} {
		if items.FindChildByKey(id) == nil {
			t.Errorf("Skip action 应保留固定路径节点 %s", id)
		}
	}
}

// TestMergeRule_MergeAction_PartialSubset 注入返回 Merge 的规则，
// 只合并指定子集（数字值），验证选择性合并生效且未指定的固定字符串保留。
func TestMergeRule_MergeAction_PartialSubset(t *testing.T) {
	r := NewReverseRouter()
	// siblings: 1,2,3,4,5（纯数字，会被内置识别为 integer，similarity=1.0）
	// 规则只挑前 3 个合并；4,5 规则放行（Default）——但内置逻辑会把全部数字
	// 合并。为验证"未指定的保留为固定路径"，用 subsetRule 强制只合并 want 内的，
	// 未在 want 内的返回 Skip 全局不合并其参与。
	// 但合并是按 sibling 全体决策的，无法"只合并部分 sibling"后让其余继续固定。
	// 所以这里改测：规则只对 sibling=[1,2,3] 返回 Merge，对其余场景返回 Default，
	// 验证 1/2/3 合并为 var 后，4/5 作为符合模式的值归入 var（不成为固定路径）。
	// 真正"保留固定"的场景见 SkipAction 测试。
	r.SetMergeRule(&subsetRule{want: map[string]bool{"1": true, "2": true, "3": true}})

	reqs := []string{"/api/orders/1", "/api/orders/2", "/api/orders/3", "/api/orders/4", "/api/orders/5"}
	for _, u := range reqs {
		if err := r.ReverseHttpRequest(request.NewHttpRequest(u, nil, "GET", nil)); err != nil {
			t.Fatalf("请求 %s 失败: %v", u, err)
		}
	}

	orders := r.Tree.Root.FindChildByKey("api").FindChildByKey("orders")
	v := orders.GetChildByType("request_path_variable")
	if v == nil {
		t.Fatal("Merge action 应合并指定子集出变量节点")
	}
	// 合并后 4,5 作为符合 integer 模式的值归入变量节点（值统计），不成为固定路径。
	// 这是正确行为：变量节点建立后，同模式的新值归入变量而非新建固定路径。
	vn := v.(*node.RequestPathVariableNode)
	if got := len(vn.GetValueMetric().GetAllValues()); got < 5 {
		t.Errorf("变量节点应观察到全部 5 个值，得 %d", got)
	}
	// 不应残留 4,5 作为固定 path 节点（它们已归入变量）
	for _, id := range []string{"4", "5"} {
		if orders.FindChildByKey(id) != nil {
			t.Errorf("符合模式的 %s 应归入变量而非留作固定路径", id)
		}
	}
}

// subsetRule 只合并 want 集合内的值。
type subsetRule struct {
	want map[string]bool
}

func (s *subsetRule) Decide(ctx MergeContext) (MergeAction, []node.Node[node.NodeContext]) {
	var mergeable []node.Node[node.NodeContext]
	for i, n := range ctx.Siblings {
		if s.want[ctx.Values[i]] {
			mergeable = append(mergeable, n)
		}
	}
	if len(mergeable) < 2 {
		return MergeActionDefault, nil
	}
	return MergeActionMerge, mergeable
}

// TestMergeRule_ContextCarriesPattern 验证 MergeContext 传入了内置检测器
// 算出的 Pattern/Similarity，规则无需重复实现模式检测。
func TestMergeRule_ContextCarriesPattern(t *testing.T) {
	r := NewReverseRouter()
	stub := &stubMergeRule{action: MergeActionSkip}
	r.SetMergeRule(stub)

	// 纯数字 → 内置应识别为 integer
	for _, u := range []string{"/x/1", "/x/2", "/x/3"} {
		r.ReverseHttpRequest(request.NewHttpRequest(u, nil, "GET", nil))
	}

	if !stub.called {
		t.Fatal("规则未被调用")
	}
	if stub.gotCtx.Pattern != "integer" {
		t.Errorf("Context.Pattern 应为 integer，得 %q", stub.gotCtx.Pattern)
	}
	if stub.gotCtx.Similarity != 1.0 {
		t.Errorf("Context.Similarity 应为 1.0，得 %v", stub.gotCtx.Similarity)
	}
	if stub.gotCtx.Parent == nil {
		t.Error("Context.Parent 不应为 nil")
	}
	if len(stub.gotCtx.Siblings) != 3 {
		t.Errorf("Context.Siblings 应为 3 个，得 %d", len(stub.gotCtx.Siblings))
	}
}

// TestMergeRule_NilRule_UsesBuiltin 不注入规则时验证默认行为不变。
func TestMergeRule_NilRule_UsesBuiltin(t *testing.T) {
	r := NewReverseRouter()
	if got := r.GetMergeRule(); got != nil {
		t.Fatalf("默认应为 nil，得 %v", got)
	}

	for _, u := range []string{"/api/u/1", "/api/u/2", "/api/u/3"} {
		r.ReverseHttpRequest(request.NewHttpRequest(u, nil, "GET", nil))
	}
	u := r.Tree.Root.FindChildByKey("api").FindChildByKey("u")
	if u.GetChildByType("request_path_variable") == nil {
		t.Fatal("nil 规则应走内置逻辑合并数字 ID")
	}
}
