package router

// observability_project_test.go — 可观测机制与项目级隔离单测：
// Health 报告、延迟埋点、细分护栏计数、ProjectManager 隔离/治理/聚合。

import (
	"strings"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
)

// --- ReverseRouter.Health ---

func TestHealth_BasicReport(t *testing.T) {
	r := newSilentRouter()
	for i := 0; i < 5; i++ {
		feedOne(t, r, request.NewHttpRequest("/api/items", nil, "GET", nil))
	}
	rep := r.Health()
	if rep.Stats.RequestsProcessed != 5 {
		t.Errorf("请求数=%d want 5", rep.Stats.RequestsProcessed)
	}
	if rep.Stats.TotalProcessingNanos <= 0 {
		t.Error("累计耗时应 >0（ReverseHttpRequest 成功路径埋点）")
	}
	if rep.Stats.MaxProcessingNanos <= 0 {
		t.Error("最大耗时应 >0")
	}
	if rep.Stats.AvgProcessingNanos() != rep.Stats.TotalProcessingNanos/5 {
		t.Error("平均耗时应=总量/请求数")
	}
	if rep.AvgLatencyHuman == "" || rep.MaxLatencyHuman == "" {
		t.Error("人类可读延迟串不应为空")
	}
	if rep.GuardRejectRate != 0 || rep.ErrorRate != 0 {
		t.Errorf("无护栏/错误时拒绝率与错误率应为 0，实际 %v/%v", rep.GuardRejectRate, rep.ErrorRate)
	}
	if rep.Tree.TotalNodes == 0 {
		t.Error("健康报告应携带 Tree 规模统计")
	}
}

func TestHealth_GuardAndErrorRates(t *testing.T) {
	r := newSilentRouter()
	r.SetResourceLimits(ResourceLimits{MaxChildrenPerNode: 1})
	feedOne(t, r, request.NewHttpRequest("/a", nil, "GET", nil))
	// 第 2 个新段被护栏拒绝（fail-soft 错误）
	if err := r.ReverseHttpRequest(request.NewHttpRequest("/b", nil, "GET", nil)); err == nil {
		t.Fatal("超限新建应返回错误")
	}
	rep := r.Health()
	if rep.Stats.RejectedChildren == 0 {
		t.Error("护栏拒绝应计 RejectedChildren")
	}
	if rep.GuardRejectRate <= 0 {
		t.Error("拒绝率应 >0")
	}
	if rep.ErrorRate <= 0 {
		t.Error("错误率应 >0（拒绝路径计 Errors）")
	}
}

func TestHealth_ZeroRequests(t *testing.T) {
	r := newSilentRouter()
	rep := r.Health()
	if rep.GuardRejectRate != 0 || rep.ErrorRate != 0 {
		t.Error("无请求时比率应为 0（避免除零）")
	}
	if rep.AvgLatencyHuman != "0ns" || rep.MaxLatencyHuman != "0ns" {
		t.Errorf("无请求时延迟串应为 0ns，实际 %q/%q", rep.AvgLatencyHuman, rep.MaxLatencyHuman)
	}
}

func TestFormatNanos_Units(t *testing.T) {
	cases := []struct {
		ns   int64
		want string
	}{
		{0, "0ns"},
		{-5, "0ns"},
		{725, "725ns"},
		{1500, "1.5us"},
		{2_500_000, "2.5ms"},
		{3_000_000_000, "3.00s"},
	}
	for _, c := range cases {
		if got := formatNanos(c.ns); got != c.want {
			t.Errorf("formatNanos(%d)=%q want %q", c.ns, got, c.want)
		}
	}
}

func TestStats_LatencyAndGuardCounters(t *testing.T) {
	r := newSilentRouter()
	r.SetResourceLimits(ResourceLimits{MaxChildrenPerNode: 10000, MaxSegmentLen: 4})
	feedOne(t, r, request.NewHttpRequest("/toolongsegment", nil, "GET", nil))      // 截断
	feedOne(t, r, request.NewHttpRequest("/api?password=secret", nil, "GET", nil)) // 脱敏
	snap := r.GetStats()
	if snap.TruncatedSegments == 0 {
		t.Error("超长段应计 TruncatedSegments")
	}
	if snap.RedactedValues == 0 {
		t.Error("脱敏值应计 RedactedValues")
	}
	if !strings.Contains(snap.String(), "truncated=") || !strings.Contains(snap.String(), "redacted=") {
		t.Errorf("String 应含细分字段，实际 %q", snap.String())
	}
}

// --- RouterSet.Health ---

func TestRouterSet_HealthAggregatesHosts(t *testing.T) {
	s := NewRouterSet()
	mustFeed := func(url string) {
		t.Helper()
		req := request.NewHttpRequest(url, nil, "GET", nil)
		if err := s.ReverseHttpRequest(req); err != nil {
			t.Fatalf("喂入 %s 失败: %v", url, err)
		}
	}
	mustFeed("http://a.test/api")
	mustFeed("http://b.test/api")
	h := s.Health()
	if len(h) != 2 {
		t.Fatalf("应聚合 2 个 host，实际 %d", len(h))
	}
	for host, rep := range h {
		if rep.Stats.RequestsProcessed != 1 {
			t.Errorf("host %s 请求数=%d want 1", host, rep.Stats.RequestsProcessed)
		}
	}
	var nilSet *RouterSet
	if got := nilSet.Health(); len(got) != 0 {
		t.Error("nil RouterSet.Health 应返回空 map")
	}
}

// --- ProjectManager ---

func TestProjectManager_Isolation(t *testing.T) {
	m := NewProjectManager()
	// 同 host 流量喂入两个项目，应互不污染
	for _, pid := range []string{"projA", "projB"} {
		req := request.NewHttpRequest("http://target.test/api/items", nil, "GET", nil)
		if err := m.ReverseHttpRequest(pid, req); err != nil {
			t.Fatalf("项目 %s 喂入失败: %v", pid, err)
		}
	}
	// projA 再喂 2 条，projB 不应被影响
	for i := 0; i < 2; i++ {
		req := request.NewHttpRequest("http://target.test/api/items", nil, "GET", nil)
		if err := m.ReverseHttpRequest("projA", req); err != nil {
			t.Fatalf("projA 追加喂入失败: %v", err)
		}
	}
	stats := m.Stats()
	if stats["projA"]["target.test"].RequestsProcessed != 3 {
		t.Errorf("projA 请求数=%d want 3", stats["projA"]["target.test"].RequestsProcessed)
	}
	if stats["projB"]["target.test"].RequestsProcessed != 1 {
		t.Errorf("projB 请求数=%d want 1（隔离，不应被 projA 污染）", stats["projB"]["target.test"].RequestsProcessed)
	}
	// NormalizeURL 也按项目隔离
	req := request.NewHttpRequest("http://target.test/api/items", nil, "GET", nil)
	if _, ok := m.NormalizeURL("projA", req); !ok {
		t.Error("projA 应能归一化已收录路由")
	}
	if _, ok := m.NormalizeURL("projNope", req); ok {
		t.Error("不存在的项目不应归一化成功")
	}
}

func TestProjectManager_MaxProjectsEnforced(t *testing.T) {
	m := NewProjectManager()
	m.SetMaxProjects(2)
	if _, err := m.Project("p1"); err != nil {
		t.Fatalf("p1 应创建成功: %v", err)
	}
	if _, err := m.Project("p2"); err != nil {
		t.Fatalf("p2 应创建成功: %v", err)
	}
	if _, err := m.Project("p3"); err == nil {
		t.Fatal("超限新建项目应返回错误")
	}
	// 已存在项目不受上限影响
	if _, err := m.Project("p1"); err != nil {
		t.Errorf("已存在项目应直接返回: %v", err)
	}
	// 删除后可新建
	if !m.Delete("p1") {
		t.Error("删除 p1 应返回 true")
	}
	if _, err := m.Project("p3"); err != nil {
		t.Errorf("删除后建 p3 应成功: %v", err)
	}
	// 0 表示不限制
	m.SetMaxProjects(0)
	for _, id := range []string{"q1", "q2", "q3", "q4"} {
		if _, err := m.Project(id); err != nil {
			t.Fatalf("0=不限制，建 %s 应成功: %v", id, err)
		}
	}
	if m.Delete("不存在") {
		t.Error("删除不存在的项目应返回 false")
	}
	// 空 ID 拒绝
	if _, err := m.Project("   "); err == nil {
		t.Error("空项目 ID 应返回错误")
	}
	if err := m.ReverseHttpRequest("", request.NewHttpRequest("/a", nil, "GET", nil)); err == nil {
		t.Error("空项目 ID 喂入应返回错误")
	}
}

func TestProjectManager_ConfigPropagation(t *testing.T) {
	m := NewProjectManager()
	// 先建项目，再改配置：已有项目应同步
	rsOld, err := m.Project("old")
	if err != nil {
		t.Fatalf("建 old 失败: %v", err)
	}
	m.SetMaxHosts(7)
	m.SetResourceLimits(ResourceLimits{MaxChildrenPerNode: 11})
	m.SetRedactConfig(RedactConfig{Params: []string{"sekret"}})
	cfg := DefaultMergeConfig
	cfg.SiblingMergeThreshold = 9
	m.SetMergeConfig(cfg)
	// 已有项目通过其 host 路由器的实际生效值验证同步
	probeOld := request.NewHttpRequest("http://old.test/probe", nil, "GET", nil)
	if err := rsOld.ReverseHttpRequest(probeOld); err != nil {
		t.Fatalf("old 项目喂入失败: %v", err)
	}
	rrOld := rsOld.RouterFor(probeOld)
	if got := rrOld.GetResourceLimits(); got.MaxChildrenPerNode != 11 {
		t.Errorf("已有项目上限未同步，实际 %+v", got)
	}
	if got := rrOld.GetMergeConfig().SiblingMergeThreshold; got != 9 {
		t.Errorf("已有项目合并配置未同步，实际 %d", got)
	}
	// 新建项目应继承管理器默认
	rsNew, err := m.Project("new")
	if err != nil {
		t.Fatalf("建 new 失败: %v", err)
	}
	probeNew := request.NewHttpRequest("http://h.test/probe", nil, "GET", nil)
	if err := rsNew.ReverseHttpRequest(probeNew); err != nil {
		t.Fatalf("new 项目喂入失败: %v", err)
	}
	if got := rsNew.RouterFor(probeNew).GetResourceLimits(); got.MaxChildrenPerNode != 11 {
		t.Errorf("新建项目应继承上限，实际 %+v", got)
	}
	// 脱敏名单通过行为验证（继承与否都影响原值存储）
	if err := m.ReverseHttpRequest("new", request.NewHttpRequest("http://h.test/api?sekret=abc", nil, "GET", nil)); err != nil {
		t.Fatalf("new 项目脱敏喂入失败: %v", err)
	}
	mn := rsNew.RouterFor(request.NewHttpRequest("http://h.test/api?sekret=abc", nil, "GET", nil))
	if mn == nil {
		t.Fatal("RouterFor 不应返回 nil")
	}
	// 通过 Health 的 redacted 计数验证脱敏生效
	if snap := mn.GetStats(); snap.RedactedValues == 0 {
		t.Error("继承的脱敏名单应对新建项目生效")
	}
	// Projects 稳定排序
	got := m.Projects()
	if len(got) != 2 || got[0] != "new" || got[1] != "old" {
		t.Errorf("Projects 应排序返回 [new old]，实际 %v", got)
	}
}

func TestProjectManager_ReverseCurlsAndHealth(t *testing.T) {
	m := NewProjectManager()
	res := m.ReverseCurls("web", []string{
		"curl 'http://web.test/api/users/1'",
		"curl 'http://web.test/api/users/2'",
		"not-a-curl-at-all",
	})
	if res.Processed != 2 || res.Failed != 1 {
		t.Errorf("Processed=%d Failed=%d want 2/1", res.Processed, res.Failed)
	}
	h := m.Health()
	if h["web"]["web.test"].Stats.RequestsProcessed != 2 {
		t.Errorf("web 项目请求数=%d want 2", h["web"]["web.test"].Stats.RequestsProcessed)
	}
	if ps := m.ProjectStats("web"); ps["web.test"].RequestsProcessed != 2 {
		t.Error("ProjectStats 应返回单项目快照")
	}
	if ps := m.ProjectStats("ghost"); len(ps) != 0 {
		t.Error("不存在的项目 ProjectStats 应返回空 map")
	}
	// 超限项目的 ReverseCurls 整体失败（fail-soft）
	m2 := NewProjectManager()
	m2.SetMaxProjects(1)
	if _, err := m2.Project("only"); err != nil {
		t.Fatalf("建 only 失败: %v", err)
	}
	res2 := m2.ReverseCurls("blocked", []string{"curl 'http://x.test/a'"})
	if res2.Failed != 1 || res2.Processed != 0 {
		t.Errorf("超限项目批量应全失败，实际 %+v", res2)
	}
}

func TestProjectManager_NilSafety(t *testing.T) {
	var m *ProjectManager
	if _, err := m.Project("x"); err == nil {
		t.Error("nil 管理器 Project 应返回错误")
	}
	if m.Delete("x") {
		t.Error("nil 管理器 Delete 应返回 false")
	}
	if m.Projects() != nil {
		t.Error("nil 管理器 Projects 应返回 nil")
	}
	if err := m.ReverseHttpRequest("x", request.NewHttpRequest("/a", nil, "GET", nil)); err == nil {
		t.Error("nil 管理器喂入应返回错误")
	}
	if err := m.ReverseHttpRequest("x", nil); err == nil {
		t.Error("nil 请求喂入应返回错误")
	}
	if _, ok := m.NormalizeURL("x", request.NewHttpRequest("/a", nil, "GET", nil)); ok {
		t.Error("nil 管理器归一化应返回 false")
	}
	if len(m.Stats()) != 0 || len(m.Health()) != 0 || len(m.ProjectStats("x")) != 0 {
		t.Error("nil 管理器统计聚合应返回空 map")
	}
	// nil 接收者的 Set 系方法不应 panic
	m.SetMaxProjects(1)
	m.SetMaxHosts(1)
	m.SetMergeConfig(DefaultMergeConfig)
	m.SetResourceLimits(DefaultResourceLimits)
	m.SetRedactConfig(RedactConfig{})
	m.SetInferenceRule(nil)
	m.SetMergeRule(nil)
	m.SetLogger(nil)
	m.SetLogLevel(LogLevelOff)
}
