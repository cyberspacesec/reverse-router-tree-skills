package router

// coverage_gap_main2_test.go — 主线程补 router 治理层缺口：
// batch 批量分支、limits holder 未设置分支、logger 级别/recordLatency、
// RouterSet/ProjectManager 的 Set 传播与 nil 守卫。

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/inference"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
)

// --- batch ---

func TestGapMain2_ReverseRequestsBranches(t *testing.T) {
	r := newSilentRouter()
	// nil 样本 + 非法 URL 样本（处理失败）+ 正常样本
	res := r.ReverseRequests([]*request.HttpRequest{
		nil,
		request.NewHttpRequest("http://h.test/%zz%", nil, "GET", nil),
		request.NewHttpRequest("/ok", nil, "GET", nil),
	})
	if res.Processed != 1 || res.Failed != 2 {
		t.Errorf("Processed=%d Failed=%d want 1/2", res.Processed, res.Failed)
	}
	if res.Errors == nil {
		t.Error("Errors 切片应非 nil")
	}
	// 超长 raw 截断分支
	longRaw := strings.Repeat("x", maxBatchErrorRawLen+10)
	rr := BatchResult{Errors: make([]BatchError, 0)}
	r.appendBatchError(&rr, 0, longRaw, fmt.Errorf("测试错误"))
	if !strings.HasSuffix(rr.Errors[0].Raw, "...(truncated)") {
		t.Error("超长 raw 应截断")
	}
	// 上限分支：填满后只计数不记详情
	full := BatchResult{Errors: make([]BatchError, maxBatchErrors)}
	r.appendBatchError(&full, 0, "y", fmt.Errorf("测试错误"))
	if len(full.Errors) != maxBatchErrors {
		t.Error("超上限后不应再追加")
	}
}

func TestGapMain2_ReverseCurlsBranches(t *testing.T) {
	r := newSilentRouter()
	// 非法 curl（解析失败）+ 正常 curl
	res := r.ReverseCurls([]string{
		"not-a-curl {{",
		"curl 'http://h.test/curl-ok'",
	})
	if res.Processed != 1 || res.Failed != 1 {
		t.Errorf("Processed=%d Failed=%d want 1/1", res.Processed, res.Failed)
	}
	// RouterSet 侧的批量错误分支（解析失败 + 超长截断）
	s := NewRouterSet()
	res2 := s.ReverseCurls([]string{"not-a-curl {{", strings.Repeat("z", 200)})
	if res2.Failed != 2 {
		t.Errorf("RouterSet 批量 Failed=%d want 2", res2.Failed)
	}
	// 上限分支
	rr := BatchResult{Errors: make([]BatchError, maxBatchErrors)}
	appendBatchError(&rr, 0, "y", fmt.Errorf("测试错误"))
	if len(rr.Errors) != maxBatchErrors {
		t.Error("RouterSet 侧超上限后不应追加")
	}
	long := BatchResult{}
	appendBatchError(&long, 0, strings.Repeat("q", 300), fmt.Errorf("测试错误"))
	if !strings.HasSuffix(long.Errors[0].Raw, "...(truncated)") {
		t.Error("超长 raw 应截断")
	}
}

// --- limits holder ---

func TestGapMain2_LimitsHolderDefaults(t *testing.T) {
	// 从未 Store 时返回默认值（LoadLimits/loadSets 的 nil 分支）
	var lh ResourceLimitsHolder
	if got := lh.LoadLimits(); got != DefaultResourceLimits {
		t.Errorf("未设置时应返回默认值，实际 %+v", got)
	}
	var rh RedactHolder
	if rh.IsParamRedacted("password") {
		t.Error("空名单不应命中")
	}
	if rh.IsCookieRedacted("sessionid") {
		t.Error("空名单不应命中")
	}
}

// --- logger ---

func TestGapMain2_LoggerLevels(t *testing.T) {
	var buf bytes.Buffer
	for _, lv := range []LogLevel{LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError, LogLevelOff, LogLevel(99)} {
		l := NewRouterLoggerWithLevel(lv, &buf)
		if lv == LogLevelOff {
			if l.enabled {
				t.Error("Off 应禁用")
			}
		} else if !l.enabled {
			t.Errorf("级别 %d 应启用", lv)
		}
	}
	// nil writer 回退 Stderr
	l := NewRouterLoggerWithLevel(LogLevelWarn, nil)
	if l == nil || !l.enabled {
		t.Error("nil writer 应回退 Stderr 并启用")
	}
	// recordLatency：nil 接收者、负值、CAS 更新
	var nilStats *RouterStats
	nilStats.recordLatency(10) // 不应 panic
	s := NewRouterStats()
	s.recordLatency(-1)
	if s.TotalProcessingNanos.Load() != 0 {
		t.Error("负值不应累计")
	}
	s.recordLatency(100)
	s.recordLatency(50) // 小于当前 max，走 nanos<=cur 分支
	if got := s.MaxProcessingNanos.Load(); got != 100 {
		t.Errorf("max=%d want 100", got)
	}
}

// --- RouterSet 治理 ---

func TestGapMain2_RouterSetGovernance(t *testing.T) {
	var nilSet *RouterSet
	nilSet.SetMaxHosts(1)
	nilSet.SetResourceLimits(DefaultResourceLimits)
	nilSet.SetRedactConfig(RedactConfig{})
	nilSet.SetInferenceRule(nil) // nil set
	nilSet.SetMergeRule(nil)
	nilSet.SetLogger(nil)
	nilSet.SetLogLevel(LogLevelOff)
	if nilSet.RouterFor(nil) != nil {
		t.Error("nil set RouterFor 应 nil")
	}
	if routerHost(nil) != "" {
		t.Error("nil 请求 host 应空")
	}
	// rule==nil 守卫
	s := NewRouterSet()
	s.SetInferenceRule(nil)
	// 正常传播：先建桶再 Set，各传播走快照分支
	if err := s.ReverseHttpRequest(request.NewHttpRequest("http://a.test/x", nil, "GET", nil)); err != nil {
		t.Fatal(err)
	}
	s.SetMaxHosts(5)
	s.SetResourceLimits(ResourceLimits{MaxChildrenPerNode: 3})
	s.SetRedactConfig(RedactConfig{Params: []string{"p"}})
	s.SetInferenceRule(inference.NewChainTypeInferenceRule())
	s.SetMergeRule(nil)
	s.SetLogger(NewRouterLoggerWithWriter(&bytes.Buffer{}))
	s.SetLogLevel(LogLevelError)
	// nil logger 关闭日志
	s.SetLogger(nil)
	// snapshotRouters 空集分支
	empty := NewRouterSet()
	empty.SetMergeConfig(DefaultMergeConfig)
	empty.SetInferenceRule(inference.NewChainTypeInferenceRule())
	empty.SetMergeRule(nil)
	empty.SetLogger(NewRouterLogger())
	empty.SetLogLevel(LogLevelWarn)
	empty.SetResourceLimits(DefaultResourceLimits)
	empty.SetRedactConfig(RedactConfig{})
	if got := empty.snapshotRouters(); len(got) != 0 {
		t.Error("空集快照应为空")
	}
	// Delete 大小写/空白归一分支
	s2 := NewRouterSet()
	if err := s2.ReverseHttpRequest(request.NewHttpRequest("http://Mix.Test/x", nil, "GET", nil)); err != nil {
		t.Fatal(err)
	}
	if !s2.Delete("  MIX.test  ") {
		t.Error("Delete 应大小写/空白归一")
	}
}

// --- ProjectManager 治理 ---

func TestGapMain2_ProjectManagerGovernance(t *testing.T) {
	var nilMgr *ProjectManager
	nilMgr.SetMaxProjects(1)
	nilMgr.SetMaxHosts(1)
	nilMgr.SetMergeConfig(DefaultMergeConfig)
	nilMgr.SetResourceLimits(DefaultResourceLimits)
	nilMgr.SetRedactConfig(RedactConfig{})
	nilMgr.SetInferenceRule(nil) // nil rule 守卫
	nilMgr.SetMergeRule(nil)
	nilMgr.SetLogger(nil)
	nilMgr.SetLogLevel(LogLevelOff)
	// normalizeProjectID 空白/空分支
	if normalizeProjectID("") != "" {
		t.Error("空 ID 应空")
	}
	if normalizeProjectID("   ") != "" {
		t.Error("纯空白 ID 应空")
	}
	if got := normalizeProjectID("\t proj \n"); got != "proj" {
		t.Errorf("应去首尾空白，实际 %q", got)
	}
	// Set 传播：已有项目走快照分支
	m := NewProjectManager()
	if _, err := m.Project("p"); err != nil {
		t.Fatal(err)
	}
	m.SetInferenceRule(inference.NewChainTypeInferenceRule())
	m.SetMergeRule(nil)
	m.SetLogger(NewRouterLoggerWithWriter(&bytes.Buffer{}))
	m.SetLogLevel(LogLevelError)
	m.SetMaxHosts(9)
	// ReverseCurls 错误分支：非法 curl
	res := m.ReverseCurls("p", []string{"not-a-curl {{", "curl 'http://h.test/ok'"})
	if res.Processed != 1 || res.Failed != 1 {
		t.Errorf("Processed=%d Failed=%d want 1/1", res.Processed, res.Failed)
	}
	// lookupRouter nil 守卫
	var nilSet *RouterSet
	if nilSet.lookupRouter("h") != nil {
		t.Error("nil set lookup 应 nil")
	}
}
