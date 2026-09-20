package router

// coverage_gap_rn_test.go — 补 normalize/project_manager/router_set/batch/merge_rule 缺口。
// 前缀 TestGapRN_，只新增测试，不碰业务代码。

import (
	"bytes"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/inference"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
)

// --- batch：ReverseCurls 解析成功但 Reverse 失败分支 ---

func TestGapRN_BatchCurlsReverseError(t *testing.T) {
	// curl 解析成功但 URL 非法（%zz），ReverseHttpRequest 报错。
	r := newSilentRouter()
	res := r.ReverseCurls([]string{"curl 'http://h.test/%zz%'"})
	if res.Failed != 1 || res.Processed != 0 {
		t.Errorf("Processed=%d Failed=%d want 0/1", res.Processed, res.Failed)
	}
	if len(res.Errors) != 1 {
		t.Errorf("应记录 1 条失败详情，得 %d", len(res.Errors))
	}
}

// --- merge_rule：空交集直接返回 nil ---

func TestGapRN_IntersectEmptyOthers(t *testing.T) {
	// others 为空时直接返回 nil，不触碰 base。
	if got := intersectNodes(nil, nil); got != nil {
		t.Errorf("空输入应返回 nil，得 %v", got)
	}
	base := []node.Node[node.NodeContext]{node.NewRequestParamNode("a", "", false)}
	if got := intersectNodes(base, nil); got != nil {
		t.Errorf("空 others 应返回 nil，得 %v", got)
	}
	if got := intersectNodes(base, []node.Node[node.NodeContext]{}); got != nil {
		t.Errorf("空 others 应返回 nil，得 %v", got)
	}
}

// --- normalize：URL 解析失败分支 ---

func TestGapRN_NormalizeParseError(t *testing.T) {
	r := newSilentRouter()
	_, reason := r.NormalizeURLDetailed(request.NewHttpRequest("http://h.test/%zz%", nil, "GET", nil))
	if reason != NormalizeReasonInvalidRequest {
		t.Errorf("非法 URL 应返回 invalid_request，得 %q", reason)
	}
}

// --- normalize：参数遍历三分支（非参数跳过/类型断言失败跳过/必需参数） ---
// 同一次构造同时覆盖 NormalizeURLDetailed 与 buildAsset（ListAssets）两处循环。

func TestGapRN_NormalizeParamBranches(t *testing.T) {
	r := newSilentRouter()
	// POST + Content-Type：在方法节点下同时产生参数节点与非参数子节点。
	feed := request.NewHttpRequest("/api/norm?q=1", request.Headers{"Content-Type": "application/json"}, "POST", nil)
	if err := r.ReverseHttpRequest(feed); err != nil {
		t.Fatalf("喂入失败: %v", err)
	}
	// 拿到方法节点句柄，用于注入与标记。
	mNode, _, err := r.FindRouteNode(request.NewHttpRequest("/api/norm", nil, "POST", nil))
	if err != nil || mNode == nil {
		t.Fatalf("查找方法节点失败: %v %v", mNode, err)
	}
	// 将真实参数标为必需，覆盖 IsRequired 真分支。
	found := false
	for _, child := range mNode.GetChildren() {
		if pn, ok := child.(*node.RequestParamNode); ok && pn.GetParamName() == "q" {
			pn.SetRequired(true)
			found = true
		}
	}
	if !found {
		t.Fatal("未找到参数节点 q")
	}
	// 注入类型为 request_param 但 Go 类型不对的假节点，覆盖断言失败分支。
	fake := node.NewBaseNode[node.NodeContext]("request_param", "zzfake", "", node.NewBaseNodeContext())
	if err := mNode.AddChild(fake); err != nil {
		t.Fatalf("注入假节点失败: %v", err)
	}
	// 归一化：应成功，且必需参数被收录；非参数与假节点被跳过。
	route, reason := r.NormalizeURLDetailed(request.NewHttpRequest("/api/norm?q=1", nil, "POST", nil))
	if reason != NormalizeOK {
		t.Fatalf("归一化应成功，得 %q", reason)
	}
	hasQ := false
	for _, n := range route.QueryParams {
		if n == "q" {
			hasQ = true
		}
		if n == "zzfake" {
			t.Error("假节点不应出现在 QueryParams")
		}
	}
	if !hasQ {
		t.Error("QueryParams 应包含 q")
	}
	hasReq := false
	for _, n := range route.RequiredParams {
		if n == "q" {
			hasReq = true
		}
	}
	if !hasReq {
		t.Error("RequiredParams 应包含被标为必需的 q")
	}
	// 资产清单走 buildAsset 同构循环，同样覆盖三分支。
	assets := r.ListAssets()
	if len(assets) == 0 {
		t.Fatal("资产清单不应为空")
	}
	hit := false
	for _, a := range assets {
		if a.Template == "/api/norm" && a.Method == "POST" {
			hit = true
			hq, hr := false, false
			for _, n := range a.QueryParams {
				if n == "q" {
					hq = true
				}
				if n == "zzfake" {
					t.Error("资产 QueryParams 不应含假节点")
				}
			}
			for _, n := range a.RequiredParams {
				if n == "q" {
					hr = true
				}
			}
			if !hq || !hr {
				t.Errorf("资产应含 q/必需 q，得 %+v", a)
			}
		}
	}
	if !hit {
		t.Error("未找到 /api/norm POST 资产")
	}
}

// --- normalize RouterSet：批量未命中分支 ---

func TestGapRN_RouterSetBatchUnmatched(t *testing.T) {
	s := NewRouterSet()
	valid := request.NewHttpRequest("http://gap.test/hit", nil, "GET", nil)
	if err := s.ReverseHttpRequest(valid); err != nil {
		t.Fatalf("喂入失败: %v", err)
	}
	unknown := request.NewHttpRequest("http://unknown-host.test/nope", nil, "GET", nil)
	report := s.NormalizeURLsDetailed([]*request.HttpRequest{valid, nil, unknown})
	if report.MatchedCount() != 1 {
		t.Errorf("Matched=%d want 1", report.MatchedCount())
	}
	if report.UnmatchedCount() != 2 {
		t.Errorf("Unmatched=%d want 2", report.UnmatchedCount())
	}
}

// --- normalize RouterSet：curl 解析失败分支 ---

func TestGapRN_RouterSetCurlParseFail(t *testing.T) {
	s := NewRouterSet()
	if _, reason := s.NormalizeCurlDetailed("not-a-curl {{"); reason != NormalizeReasonInvalidRequest {
		t.Errorf("非法 curl 应返回 invalid_request，得 %q", reason)
	}
}

// --- normalize ProjectManager：批量未命中分支 ---

func TestGapRN_ProjectAssetsUnmatched(t *testing.T) {
	m := NewProjectManager()
	valid := request.NewHttpRequest("http://gap.test/hit", nil, "GET", nil)
	if err := m.ReverseHttpRequest("p", valid); err != nil {
		t.Fatalf("喂入失败: %v", err)
	}
	report := m.NormalizeAssetsDetailed("p", []*request.HttpRequest{valid, nil})
	if report.MatchedCount() != 1 {
		t.Errorf("Matched=%d want 1", report.MatchedCount())
	}
	if report.UnmatchedCount() != 1 {
		t.Errorf("Unmatched=%d want 1", report.UnmatchedCount())
	}
}

// --- normalize ProjectManager：curl 解析失败分支 ---

func TestGapRN_ProjectCurlParseFail(t *testing.T) {
	m := NewProjectManager()
	if _, reason := m.NormalizeCurlDetailed("p", "not-a-curl {{"); reason != NormalizeReasonInvalidRequest {
		t.Errorf("非法 curl 应返回 invalid_request，得 %q", reason)
	}
}

// --- router_set：建桶时传播 mergeRule ---

func TestGapRN_RouterSetMergeRulePropagate(t *testing.T) {
	s := NewRouterSet()
	stub := &stubMergeRule{action: MergeActionSkip}
	s.SetMergeRule(stub)
	r := s.RouterFor(request.NewHttpRequest("http://merge.test/x", nil, "GET", nil))
	if r == nil {
		t.Fatal("建桶不应返回 nil")
	}
	if r.GetMergeRule() == nil {
		t.Error("新建桶应继承 mergeRule")
	}
}

// --- router_set：nil 删除守卫 + 非空快照 ---

func TestGapRN_RouterSetNilDeleteAndSnapshot(t *testing.T) {
	var nilSet *RouterSet
	if nilSet.Delete("x") {
		t.Error("nil 集合 Delete 应返回 false")
	}
	s := NewRouterSet()
	if err := s.ReverseHttpRequest(request.NewHttpRequest("http://snap.test/x", nil, "GET", nil)); err != nil {
		t.Fatalf("喂入失败: %v", err)
	}
	if got := s.snapshotRouters(); len(got) != 1 {
		t.Errorf("非空快照长度=%d want 1", len(got))
	}
}

// --- project_manager：新建项目继承四配置 ---

func TestGapRN_ProjectCreateInherit(t *testing.T) {
	m := NewProjectManager()
	m.SetInferenceRule(inference.NewChainTypeInferenceRule())
	m.SetMergeRule(&stubMergeRule{action: MergeActionSkip})
	m.SetLogger(NewRouterLoggerWithWriter(&bytes.Buffer{}))
	m.SetLogLevel(LogLevelError)
	rs, err := m.Project("inherit")
	if err != nil || rs == nil {
		t.Fatalf("新建项目失败: %v %v", rs, err)
	}
	if len(m.Projects()) != 1 {
		t.Errorf("项目数=%d want 1", len(m.Projects()))
	}
}

// --- project_manager：并发建同名项目命中写锁内二次检查 ---

func TestGapRN_ProjectConcurrentCreate(t *testing.T) {
	// 单线程第二次调用走读锁 fast-path，写锁内二次检查需并发触发；
	// 多轮并发建同名项目，极大概率命中该分支。
	for round := 0; round < 20; round++ {
		m := NewProjectManager()
		const n = 64
		var wg sync.WaitGroup
		var ready atomic.Int64
		var gate atomic.Bool // 自旋释放：避免 channel 调度偏差，让各 goroutine 真并行
		gate.Store(false)
		errs := make([]error, n)
		sets := make([]*RouterSet, n)
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				ready.Add(1)
				for !gate.Load() {
					runtime.Gosched() // 让出 P，避免 -race 下自旋饿死调度
				}
				rs, err := m.Project("concurrent")
				sets[idx], errs[idx] = rs, err
			}(i)
		}
		for ready.Load() < n { // 等全部就绪再开闸，最大化同时撞入读锁的数量
			runtime.Gosched()
		}
		gate.Store(true)
		wg.Wait()
		for i := 0; i < n; i++ {
			if errs[i] != nil || sets[i] == nil {
				t.Fatalf("轮次 %d 并发建项目失败: %v", round, errs[i])
			}
			if sets[i] != sets[0] {
				t.Fatalf("轮次 %d 同名项目应返回同一实例", round)
			}
		}
	}
}

// --- project_manager：ReverseCurls 解析成功但 Reverse 失败分支 ---

func TestGapRN_ProjectReverseCurlsReverseError(t *testing.T) {
	m := NewProjectManager()
	res := m.ReverseCurls("p", []string{"curl 'http://h.test/%zz%'"})
	if res.Failed != 1 || res.Processed != 0 {
		t.Errorf("Processed=%d Failed=%d want 0/1", res.Processed, res.Failed)
	}
}
