package router

// normalize_api_test.go — 归一化 API 可用性单测：失败原因、批量明细、
// 资产清单、curl/裸URL直达、只读无副作用。

import (
	"strings"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
)

// feedURLs 批量喂裸 URL（GET），错误直接 fatal。
func feedURLs(t *testing.T, r *ReverseRouter, urls ...string) {
	t.Helper()
	for _, u := range urls {
		if err := r.ReverseHttpRequest(request.NewHttpRequest(u, nil, "GET", nil)); err != nil {
			t.Fatalf("喂入 %s 失败: %v", u, err)
		}
	}
}

// --- 失败原因 ---

func TestNormalizeDetailed_Reasons(t *testing.T) {
	r := newSilentRouter()
	feedURLs(t, r, "/api/users/1", "/api/users/2", "/api/users/3")

	// 路径命中 + 方法命中 → ok
	if _, reason := r.NormalizeURLDetailed(request.NewHttpRequest("/api/users/99", nil, "GET", nil)); reason != NormalizeOK {
		t.Errorf("已知路由应 ok，实际 %v", reason)
	}
	// 路径从未见过 → unknown_path
	if _, reason := r.NormalizeURLDetailed(request.NewHttpRequest("/never/seen", nil, "GET", nil)); reason != NormalizeReasonUnknownPath {
		t.Errorf("未知路径应 unknown_path，实际 %v", reason)
	}
	// 路径命中但方法没采集过 → unknown_method
	if _, reason := r.NormalizeURLDetailed(request.NewHttpRequest("/api/users/9", nil, "POST", nil)); reason != NormalizeReasonUnknownMethod {
		t.Errorf("未知方法应 unknown_method，实际 %v", reason)
	}
	// nil 请求 → invalid_request
	if _, reason := r.NormalizeURLDetailed(nil); reason != NormalizeReasonInvalidRequest {
		t.Errorf("nil 应 invalid_request，实际 %v", reason)
	}
	// 旧签名保持一致：成功 true / 失败 false
	if _, ok := r.NormalizeURL(request.NewHttpRequest("/api/users/5", nil, "GET", nil)); !ok {
		t.Error("旧签名成功路径应 true")
	}
	if _, ok := r.NormalizeURL(request.NewHttpRequest("/nope", nil, "GET", nil)); ok {
		t.Error("旧签名失败路径应 false")
	}
}

func TestNormalizeDetailed_ReasonStrings(t *testing.T) {
	for _, reason := range []NormalizeReason{
		NormalizeOK, NormalizeReasonInvalidRequest, NormalizeReasonUnknownHost,
		NormalizeReasonUnknownProject, NormalizeReasonUnknownPath, NormalizeReasonUnknownMethod,
		NormalizeReason("weird"),
	} {
		if s := reason.String(); s == "" {
			t.Errorf("原因 %q 的可读串不应为空", reason)
		}
	}
}

// --- 批量明细 ---

func TestNormalizeURLsDetailed_Report(t *testing.T) {
	r := newSilentRouter()
	feedURLs(t, r, "/api/users/1", "/api/users/2", "/api/users/3")
	report := r.NormalizeURLsDetailed([]*request.HttpRequest{
		request.NewHttpRequest("/api/users/10", nil, "GET", nil),
		request.NewHttpRequest("/api/users/11", nil, "GET", nil),
		request.NewHttpRequest("/unknown/path", nil, "GET", nil),
		nil,
	})
	if report.MatchedCount() != 2 {
		t.Errorf("MatchedCount=%d want 2", report.MatchedCount())
	}
	if report.UnmatchedCount() != 2 {
		t.Fatalf("UnmatchedCount=%d want 2", report.UnmatchedCount())
	}
	if report.Unmatched[0].Reason != NormalizeReasonUnknownPath || report.Unmatched[0].Index != 2 {
		t.Errorf("第 3 条应 unknown_path，实际 %+v", report.Unmatched[0])
	}
	if report.Unmatched[1].Reason != NormalizeReasonInvalidRequest || report.Unmatched[1].URL != "<nil>" {
		t.Errorf("nil 样本应 invalid_request+<nil>，实际 %+v", report.Unmatched[1])
	}
	// 旧批量签名仍静默跳过、结果一致
	compat := r.NormalizeURLs([]*request.HttpRequest{
		request.NewHttpRequest("/api/users/10", nil, "GET", nil),
		request.NewHttpRequest("/unknown/path", nil, "GET", nil),
	})
	if len(compat) != 1 {
		t.Errorf("旧批量应只含 1 个资产键，实际 %v", compat)
	}
}

// --- 资产清单 ---

func TestListAssets_Basic(t *testing.T) {
	r := newSilentRouter()
	feedURLs(t, r,
		"/api/users/1", "/api/users/2", "/api/users/3",
		"/api/orders/7", "/api/orders/8", "/api/orders/9",
		"/health",
	)
	r.InferRequiredParams()
	assets := r.ListAssets()
	if len(assets) != 3 {
		t.Fatalf("资产数=%d want 3，实际 %+v", len(assets), assets)
	}
	// 稳定排序：按 AssetKey 升序
	for i := 1; i < len(assets); i++ {
		if assets[i-1].AssetKey() >= assets[i].AssetKey() {
			t.Fatalf("资产清单应按 AssetKey 排序，实际 %v", assets)
		}
	}
	byKey := make(map[string]NormalizedRoute)
	for _, a := range assets {
		byKey[a.AssetKey()] = a
	}
	var userAsset NormalizedRoute
	found := false
	for k, a := range byKey {
		if strings.HasPrefix(k, "GET /api/users/{") {
			userAsset, found = a, true
		}
	}
	if !found {
		t.Fatalf("应含 users 变量资产，实际 %v", byKey)
	}
	if len(userAsset.PathParams) != 1 {
		t.Errorf("users 资产 PathParams=%v want 1 个", userAsset.PathParams)
	}
	// 空树返回空切片
	empty := newSilentRouter()
	if got := empty.ListAssets(); len(got) != 0 {
		t.Errorf("空树清单应为空，实际 %v", got)
	}
	var nilRouter *ReverseRouter
	if got := nilRouter.ListAssets(); len(got) != 0 {
		t.Error("nil 路由器清单应为空")
	}
}

func TestListAssets_RootRoute(t *testing.T) {
	r := newSilentRouter()
	feedURLs(t, r, "/")
	assets := r.ListAssets()
	if len(assets) != 1 || assets[0].Template != "/" {
		t.Errorf("根路由资产应为 GET /，实际 %+v", assets)
	}
}

// --- 便捷入口 ---

func TestNormalizeCurl_Basic(t *testing.T) {
	r := newSilentRouter()
	feedURLs(t, r, "/api/users/1", "/api/users/2", "/api/users/3")
	route, ok := r.NormalizeCurl("curl 'http://h.test/api/users/42'")
	if !ok {
		t.Fatal("已知路由的 curl 应归一化成功")
	}
	if !strings.HasPrefix(route.Template, "/api/users/{") {
		t.Errorf("模板=%q want users 变量模板", route.Template)
	}
	if _, ok := r.NormalizeCurl("not-a-curl {{"); ok {
		t.Error("非法 curl 应归一化失败")
	}
	if _, reason := r.NormalizeCurlDetailed("not-a-curl {{"); reason != NormalizeReasonInvalidRequest {
		t.Errorf("非法 curl 应 invalid_request，实际 %v", reason)
	}
}

func TestNormalizeURLString_Basic(t *testing.T) {
	r := newSilentRouter()
	feedURLs(t, r, "/api/users/1", "/api/users/2", "/api/users/3")
	route, ok := r.NormalizeURLString("http://any.test/api/users/42", "")
	if !ok {
		t.Fatal("空方法应视为 GET 归一化成功")
	}
	if route.Method != "GET" {
		t.Errorf("方法=%q want GET", route.Method)
	}
	if _, ok := r.NormalizeURLString("http://any.test/nope", "GET"); ok {
		t.Error("未知路径应归一化失败")
	}
}

// --- RouterSet 只读 + 聚合 ---

func TestRouterSet_NormalizeReadOnly(t *testing.T) {
	s := NewRouterSet()
	if err := s.ReverseHttpRequest(request.NewHttpRequest("http://a.test/x", nil, "GET", nil)); err != nil {
		t.Fatal(err)
	}
	before := len(s.Hosts())
	// 归一化未知 host：旧实现会懒建空桶，新实现必须只读
	if _, ok := s.NormalizeURL(request.NewHttpRequest("http://ghost.test/x", nil, "GET", nil)); ok {
		t.Error("未知 host 应归一化失败")
	}
	if _, reason := s.NormalizeURLDetailed(request.NewHttpRequest("http://ghost.test/x", nil, "GET", nil)); reason != NormalizeReasonUnknownHost {
		t.Errorf("未知 host 应 unknown_host，实际 %v", reason)
	}
	if got := len(s.Hosts()); got != before {
		t.Errorf("归一化不应建桶，桶数 %d→%d", before, got)
	}
	// 已知 host 正常归一化且 Host 回填
	n, ok := s.NormalizeURL(request.NewHttpRequest("http://a.test/x", nil, "GET", nil))
	if !ok || n.Host != "a.test" {
		t.Errorf("已知 host 应成功且 Host=a.test，实际 %+v ok=%v", n, ok)
	}
	// 批量明细：已知/未知混合
	report := s.NormalizeAssetsDetailed([]*request.HttpRequest{
		request.NewHttpRequest("http://a.test/x", nil, "GET", nil),
		request.NewHttpRequest("http://ghost.test/x", nil, "GET", nil),
	})
	if report.MatchedCount() != 1 || report.UnmatchedCount() != 1 {
		t.Errorf("报告应 1 命中 1 未命中，实际 %+v", report)
	}
	for k := range report.Matched {
		if !strings.HasPrefix(k, "a.test ") {
			t.Errorf("资产键应带 host 前缀，实际 %q", k)
		}
	}
	// curl / 裸 URL 直达
	if _, ok := s.NormalizeCurl("curl 'http://a.test/x'"); !ok {
		t.Error("RouterSet.NormalizeCurl 应成功")
	}
	if _, ok := s.NormalizeURLString("http://a.test/x", "GET"); !ok {
		t.Error("RouterSet.NormalizeURLString 应成功")
	}
	// 资产清单按 host 分组
	assets := s.ListAssets()
	if len(assets["a.test"]) != 1 {
		t.Errorf("a.test 资产数=%d want 1", len(assets["a.test"]))
	}
	if assets["a.test"][0].Host != "a.test" {
		t.Error("清单资产 Host 应回填")
	}
	var nilSet *RouterSet
	if len(nilSet.ListAssets()) != 0 {
		t.Error("nil RouterSet 清单应为空")
	}
}

// --- ProjectManager 聚合 ---

func TestProjectManager_NormalizeAPIs(t *testing.T) {
	m := NewProjectManager()
	if err := m.ReverseHttpRequest("web", request.NewHttpRequest("http://h.test/api/1", nil, "GET", nil)); err != nil {
		t.Fatal(err)
	}
	if err := m.ReverseHttpRequest("web", request.NewHttpRequest("http://h.test/api/2", nil, "GET", nil)); err != nil {
		t.Fatal(err)
	}
	if err := m.ReverseHttpRequest("web", request.NewHttpRequest("http://h.test/api/3", nil, "GET", nil)); err != nil {
		t.Fatal(err)
	}
	// 项目不存在 → unknown_project，且不建项目
	if _, reason := m.NormalizeURLDetailed("ghost", request.NewHttpRequest("http://h.test/api/1", nil, "GET", nil)); reason != NormalizeReasonUnknownProject {
		t.Errorf("未知项目应 unknown_project，实际 %v", reason)
	}
	if got := m.Projects(); len(got) != 1 {
		t.Errorf("读操作不应建项目，实际 %v", got)
	}
	// 批量明细
	report := m.NormalizeURLsDetailed("web", []*request.HttpRequest{
		request.NewHttpRequest("http://h.test/api/9", nil, "GET", nil),
		request.NewHttpRequest("http://h.test/nope", nil, "GET", nil),
	})
	if report.MatchedCount() != 1 || report.UnmatchedCount() != 1 {
		t.Errorf("报告应 1 命中 1 未命中，实际 %+v", report)
	}
	if report.Unmatched[0].Reason != NormalizeReasonUnknownPath {
		t.Errorf("未命中原因应 unknown_path，实际 %v", report.Unmatched[0].Reason)
	}
	// 兼容批量 + Host 键批量
	if len(m.NormalizeURLs("web", []*request.HttpRequest{request.NewHttpRequest("http://h.test/api/9", nil, "GET", nil)})) != 1 {
		t.Error("兼容批量应返回 1 个资产键")
	}
	hostReport := m.NormalizeAssetsDetailed("web", []*request.HttpRequest{request.NewHttpRequest("http://h.test/api/9", nil, "GET", nil)})
	for k := range hostReport.Matched {
		if !strings.HasPrefix(k, "h.test ") {
			t.Errorf("Host 键应带 host 前缀，实际 %q", k)
		}
	}
	// curl 直达
	if _, ok := m.NormalizeCurl("web", "curl 'http://h.test/api/5'"); !ok {
		t.Error("项目内 curl 应归一化成功")
	}
	if _, reason := m.NormalizeCurlDetailed("ghost", "curl 'http://h.test/api/5'"); reason != NormalizeReasonUnknownProject {
		t.Errorf("未知项目 curl 应 unknown_project，实际 %v", reason)
	}
	// 资产清单：单项目 + 全项目
	if pa := m.ProjectAssets("web"); len(pa["h.test"]) == 0 {
		t.Errorf("单项目清单应非空，实际 %v", pa)
	}
	if len(m.ProjectAssets("ghost")) != 0 {
		t.Error("未知项目清单应为空")
	}
	all := m.ListAssets()
	if len(all["web"]["h.test"]) == 0 {
		t.Errorf("全项目清单应含 web/h.test，实际 %v", all)
	}
	var nilMgr *ProjectManager
	if len(nilMgr.ListAssets()) != 0 || len(nilMgr.ProjectAssets("x")) != 0 {
		t.Error("nil 管理器清单应为空")
	}
	if len(nilMgr.NormalizeURLs("x", nil)) != 0 {
		t.Error("nil 管理器批量应返回空")
	}
}
