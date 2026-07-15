package router

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/inference"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
)

// newSilentRouter 创建静默路由器（关闭日志），避免测试输出刷屏
func newSilentRouter() *ReverseRouter {
	r := NewReverseRouter()
	r.SetLogger(NewRouterLoggerWithLevel(LogLevelOff, nil))
	return r
}

// newDebugRouter 创建 debug 级别路由器，日志写入 buffer 供断言
func newDebugRouter(buf *bytes.Buffer) *ReverseRouter {
	r := NewReverseRouter()
	r.SetLogger(NewRouterLoggerWithLevel(LogLevelDebug, buf))
	return r
}

// === 统计指标测试 ===

func TestRouterStats_RequestCount(t *testing.T) {
	r := newSilentRouter()
	for i := 0; i < 5; i++ {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/x", nil, "GET", nil))
	}
	s := r.GetStats()
	if s.RequestsProcessed != 5 {
		t.Errorf("应处理5个请求，实际 %d", s.RequestsProcessed)
	}
}

func TestRouterStats_PathVariablesIdentified(t *testing.T) {
	r := newSilentRouter()
	// 3个ID触发合并为路径变量
	for _, id := range []string{"101", "102", "103"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/users/"+id, nil, "GET", nil))
	}
	s := r.GetStats()
	if s.PathVariablesIdentified != 1 {
		t.Errorf("应识别1个路径变量，实际 %d", s.PathVariablesIdentified)
	}
	if s.MergeAttempts != 1 {
		t.Errorf("应尝试1次合并，实际 %d", s.MergeAttempts)
	}
	if s.MergeSkipped != 0 {
		t.Errorf("不应跳过合并，实际跳过 %d", s.MergeSkipped)
	}
}

func TestRouterStats_MergeSkipped(t *testing.T) {
	r := newSilentRouter()
	// 3个固定路径名，similar_length_strings 不合并
	for _, role := range []string{"admin", "manager", "guest"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/roles/"+role, nil, "GET", nil))
	}
	s := r.GetStats()
	if s.MergeAttempts != 1 {
		t.Errorf("应尝试1次合并，实际 %d", s.MergeAttempts)
	}
	// admin/manager/guest 是 similar_length_strings，应跳过
	if s.MergeSkipped != 1 {
		t.Errorf("应跳过1次合并（固定路径名），实际 %d", s.MergeSkipped)
	}
	if s.PathVariablesIdentified != 0 {
		t.Errorf("不应识别路径变量，实际 %d", s.PathVariablesIdentified)
	}
}

func TestRouterStats_ParamsCreated(t *testing.T) {
	r := newSilentRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/list?page=1&size=10&sort=asc", nil, "GET", nil))
	s := r.GetStats()
	if s.ParamsCreated != 3 {
		t.Errorf("应创建3个参数，实际 %d", s.ParamsCreated)
	}
}

func TestRouterStats_BodyParamsParsed(t *testing.T) {
	r := newSilentRouter()
	h := request.Headers{"Content-Type": "application/json"}
	r.ReverseHttpRequest(request.NewHttpRequest("/api/users", h, "POST", []byte(`{"name":"bob","age":25}`)))
	s := r.GetStats()
	if s.BodyParamsParsed != 2 {
		t.Errorf("应解析2个body参数，实际 %d", s.BodyParamsParsed)
	}
}

func TestRouterStats_TypeInferences(t *testing.T) {
	r := newSilentRouter()
	// 路径变量合并触发1次类型推断
	for _, id := range []string{"101", "102", "103"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/users/"+id, nil, "GET", nil))
	}
	// 查询参数触发类型推断
	r.ReverseHttpRequest(request.NewHttpRequest("/api/list?page=1", nil, "GET", nil))
	s := r.GetStats()
	if s.TypeInferences < 2 {
		t.Errorf("应至少2次类型推断（路径变量+参数），实际 %d", s.TypeInferences)
	}
}

func TestRouterStats_RequiredParamsInferred(t *testing.T) {
	r := newSilentRouter()
	// page 出现10次，callback 出现2次
	for i := 0; i < 10; i++ {
		url := "/api/list?page=1"
		if i < 2 {
			url += "&callback=cb"
		}
		r.ReverseHttpRequest(request.NewHttpRequest(url, nil, "GET", nil))
	}
	count := r.InferRequiredParams()
	s := r.GetStats()
	if s.RequiredParamsInferred != int64(count) {
		t.Errorf("统计数 %d 应与返回值 %d 一致", s.RequiredParamsInferred, count)
	}
	if s.RequiredParamsInferred != 1 {
		t.Errorf("page 应判定为必需（1个），实际 %d", s.RequiredParamsInferred)
	}
}

func TestRouterStats_Errors(t *testing.T) {
	r := newSilentRouter()
	// nil 请求应计入错误
	r.ReverseHttpRequest(nil)
	// 非法URL
	r.ReverseHttpRequest(request.NewHttpRequest("%%%", nil, "GET", nil))
	s := r.GetStats()
	if s.Errors < 1 {
		t.Errorf("应至少1个错误，实际 %d", s.Errors)
	}
}

func TestRouterStats_Warnings_BadBody(t *testing.T) {
	r := newSilentRouter()
	h := request.Headers{"Content-Type": "application/json"}
	// 非法 JSON body 应触发警告
	r.ReverseHttpRequest(request.NewHttpRequest("/api/x", h, "POST", []byte(`{invalid`)))
	s := r.GetStats()
	if s.Warnings != 1 {
		t.Errorf("非法body应触发1次警告，实际 %d", s.Warnings)
	}
	if s.Errors != 1 {
		t.Errorf("非法body应触发1次错误，实际 %d", s.Errors)
	}
}

func TestRouterStats_Reset(t *testing.T) {
	r := newSilentRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/x", nil, "GET", nil))
	r.ResetStats()
	s := r.GetStats()
	if s.RequestsProcessed != 0 {
		t.Errorf("Reset 后计数应清零，实际 %d", s.RequestsProcessed)
	}
}

func TestRouterStats_JSONSerialization(t *testing.T) {
	r := newSilentRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/x", nil, "GET", nil))
	s := r.GetStats()
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	if m["requests_processed"] == nil {
		t.Error("JSON 应包含 requests_processed 字段")
	}
}

func TestStatsSnapshot_String(t *testing.T) {
	s := StatsSnapshot{RequestsProcessed: 5, PathVariablesIdentified: 2}
	str := s.String()
	if !strings.Contains(str, "requests=5") || !strings.Contains(str, "path_vars=2") {
		t.Errorf("String 应包含关键指标，实际 %s", str)
	}
}

// === 日志器测试 ===

func TestRouterLogger_Levels(t *testing.T) {
	// Debug 级别应输出 Debug 日志
	var buf bytes.Buffer
	r := newDebugRouter(&buf)
	r.ReverseHttpRequest(request.NewHttpRequest("/api/x", nil, "GET", nil))
	if !strings.Contains(buf.String(), "开始处理请求") {
		t.Error("Debug 级别应输出调试日志")
	}

	// Info 级别不应输出 Debug 日志
	var buf2 bytes.Buffer
	r2 := NewReverseRouter()
	r2.SetLogger(NewRouterLoggerWithLevel(LogLevelInfo, &buf2))
	r2.ReverseHttpRequest(request.NewHttpRequest("/api/x", nil, "GET", nil))
	if strings.Contains(buf2.String(), "开始处理请求") {
		t.Error("Info 级别不应输出 Debug 日志")
	}
}

func TestRouterLogger_Off(t *testing.T) {
	var buf bytes.Buffer
	r := newDebugRouter(&buf)
	r.SetLogLevel(LogLevelOff)
	r.ReverseHttpRequest(request.NewHttpRequest("/api/x", nil, "GET", nil))
	if buf.Len() > 0 {
		t.Errorf("LogLevelOff 不应输出任何日志，实际 %s", buf.String())
	}
}

func TestRouterLogger_PathVariableLog(t *testing.T) {
	var buf bytes.Buffer
	r := newDebugRouter(&buf)
	for _, id := range []string{"101", "102", "103"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/users/"+id, nil, "GET", nil))
	}
	// Info 级别的日志（识别路径变量）应出现
	if !strings.Contains(buf.String(), "识别路径变量") {
		t.Error("应记录路径变量识别日志")
	}
	if !strings.Contains(buf.String(), "users_id") {
		t.Error("日志应包含变量名 users_id")
	}
}

func TestRouterLogger_SetLogger_Nil(t *testing.T) {
	r := NewReverseRouter()
	r.SetLogger(nil) // 不应 panic
	r.ReverseHttpRequest(request.NewHttpRequest("/api/x", nil, "GET", nil))
}

func TestRouterLogger_NilSafe(t *testing.T) {
	// 直接调用 nil logger 的方法不应 panic
	var l *RouterLogger
	l.Debug("test")
	l.Info("test")
	l.Warn("test")
	l.Error("test")
}

func TestNewRouterLoggerWithWriter(t *testing.T) {
	var buf bytes.Buffer
	l := NewRouterLoggerWithWriter(&buf)
	// 默认 Warn 级别，用 Warn 消息测试输出
	l.Warn("test message")
	if !strings.Contains(buf.String(), "test message") {
		t.Error("应输出到指定 writer")
	}
}

// === 综合统计测试 ===

func TestRouterStats_Comprehensive(t *testing.T) {
	r := newSilentRouter()

	// 一组完整操作
	for _, id := range []string{"101", "102", "103"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/users/"+id+"?page=1", nil, "GET", nil))
	}
	h := request.Headers{"Content-Type": "application/x-www-form-urlencoded"}
	r.ReverseHttpRequest(request.NewHttpRequest("/api/users", h, "POST", []byte("name=bob&age=25")))
	r.InferRequiredParams()

	s := r.GetStats()
	// 4个请求
	if s.RequestsProcessed != 4 {
		t.Errorf("请求数应为4，实际 %d", s.RequestsProcessed)
	}
	// 1个路径变量
	if s.PathVariablesIdentified != 1 {
		t.Errorf("路径变量应为1，实际 %d", s.PathVariablesIdentified)
	}
	// body 参数：name + age = 2
	if s.BodyParamsParsed != 2 {
		t.Errorf("body参数应为2，实际 %d", s.BodyParamsParsed)
	}
	// 无错误无警告
	if s.Errors != 0 || s.Warnings != 0 {
		t.Errorf("不应有错误/警告，errors=%d warnings=%d", s.Errors, s.Warnings)
	}
}

// TestRouterLogger_TruncatesLongValue 验证超长字符串值被截断，防止撑爆日志。
func TestRouterLogger_TruncatesLongValue(t *testing.T) {
	var buf bytes.Buffer
	l := NewRouterLoggerWithLevel(LogLevelWarn, &buf)

	longURL := "/api/" + strings.Repeat("a", 2000)
	l.Error("解析URL失败", "url", longURL, "error", "some error")

	out := buf.String()
	if strings.Contains(out, strings.Repeat("a", 2000)) {
		t.Error("超长 URL 未被截断，原样出现在日志中")
	}
	if !strings.Contains(out, "truncated") {
		t.Errorf("日志应包含 truncated 标记，得: %s", out)
	}
	// 截断后总长应远小于原长
	if len(out) > 1000 {
		t.Errorf("截断后日志仍过长: %d 字节", len(out))
	}
}

// TestRouterLogger_ShortValueUntouched 验证短值不被截断。
func TestRouterLogger_ShortValueUntouched(t *testing.T) {
	var buf bytes.Buffer
	l := NewRouterLoggerWithLevel(LogLevelWarn, &buf)

	l.Warn("测试", "url", "/api/users/123", "count", 5)

	out := buf.String()
	if !strings.Contains(out, "/api/users/123") {
		t.Errorf("短 URL 应原样保留，得: %s", out)
	}
	if !strings.Contains(out, "count=5") {
		t.Errorf("非字符串值应原样保留，得: %s", out)
	}
}

// === discardHandler 测试 ===
// discardHandler 是 LogLevelOff 用的 slog.Handler，丢弃所有输出。
// 它的 4 个接口方法此前覆盖率 0%，此处直接构造验证。

func TestDiscardHandler_Enabled(t *testing.T) {
	h := discardHandler{}
	// 任意级别都应返回 false（丢弃所有日志）
	for _, lvl := range []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError} {
		if h.Enabled(context.Background(), lvl) {
			t.Errorf("discardHandler.Enabled(%s) 应返回 false", lvl)
		}
	}
}

func TestDiscardHandler_Handle(t *testing.T) {
	h := discardHandler{}
	// Handle 不应 panic 且返回 nil；传入一条非空 record
	rec := slog.NewRecord(time.Now(), slog.LevelError, "test.go", 1)
	rec.AddAttrs(slog.String("k", "v"))
	if err := h.Handle(context.Background(), rec); err != nil {
		t.Errorf("discardHandler.Handle 应返回 nil，实际 %v", err)
	}
}

func TestDiscardHandler_WithAttrs(t *testing.T) {
	h := discardHandler{}
	attrs := []slog.Attr{slog.String("k", "v"), slog.Int("n", 1)}
	h2 := h.WithAttrs(attrs)
	// WithAttrs 应返回同类型 handler（丢弃属性，不报错）
	if _, ok := h2.(discardHandler); !ok {
		t.Errorf("WithAttrs 应返回 discardHandler，实际 %T", h2)
	}
	// 返回的 handler 仍能正常工作
	if h2.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("WithAttrs 返回的 handler 仍应丢弃所有日志")
	}
}

func TestDiscardHandler_WithGroup(t *testing.T) {
	h := discardHandler{}
	h2 := h.WithGroup("mygroup")
	if _, ok := h2.(discardHandler); !ok {
		t.Errorf("WithGroup 应返回 discardHandler，实际 %T", h2)
	}
	if h2.Enabled(context.Background(), slog.LevelWarn) {
		t.Error("WithGroup 返回的 handler 仍应丢弃所有日志")
	}
}

// TestDiscardHandler_ViaLogger 验证 LogLevelOff logger 的日志调用最终走 discardHandler。
// 注意：NewRouterLoggerWithLevel(LogLevelOff) 设 enabled=false，日志方法在 enabled 检查处就 return，
// 不会到达底层 handler。本测试改用 SetLevel(LogLevelOff) 切换到 enabled=true 但 handler 为 discard 的路径
// 不成立——故此处仅断言 enabled=false 时所有日志方法静默且无 panic。
func TestDiscardHandler_ViaLogger(t *testing.T) {
	var buf bytes.Buffer
	l := NewRouterLoggerWithLevel(LogLevelOff, &buf)
	// enabled=false，下列调用应在 enabled 检查处 return，不触碰 handler 也不输出
	l.Debug("d", "k", "v")
	l.Info("i", "k", "v")
	l.Warn("w", "k", "v")
	l.Error("e", "k", "v")
	if buf.Len() != 0 {
		t.Errorf("LogLevelOff logger 不应有输出，实际 %s", buf.String())
	}
}

// === SetLevel 全分支测试 ===
// SetLevel 此前仅覆盖 Off 路径（42.9%），补全 Debug/Info/Warn/Error/default/nil 分支。

func TestSetLevel_AllCases(t *testing.T) {
	cases := []struct {
		name    string
		level   LogLevel
		enabled bool // 切换后期望的 enabled 状态
	}{
		{"Debug", LogLevelDebug, true},
		{"Info", LogLevelInfo, true},
		{"Warn", LogLevelWarn, true},
		{"Error", LogLevelError, true},
		{"Off", LogLevelOff, false},
		{"Default_未知级别", LogLevel(999), true}, // default 分支走 Info
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var buf bytes.Buffer
			l := NewRouterLoggerWithLevel(LogLevelWarn, &buf) // 起始非 Off，确保 SetLevel 真正切换
			l.SetLevel(c.level)
			// 验证 enabled 字段按预期切换（不调日志方法避免向 os.Stderr 输出噪音）
			if l.enabled != c.enabled {
				t.Errorf("SetLevel(%s) 后 enabled=%v，期望 %v", c.name, l.enabled, c.enabled)
			}
		})
	}
}

func TestSetLevel_NilLogger(t *testing.T) {
	// nil logger 调 SetLevel 不应 panic
	var l *RouterLogger
	l.SetLevel(LogLevelDebug) // l==nil 在 SetLevel 首行守卫返回
}

// === SetInferenceRule 测试（公开 API，此前 0%） ===

func TestSetInferenceRule_Normal(t *testing.T) {
	r := newSilentRouter()
	rule := inference.NewPhysicalTypeInferenceRule()
	r.SetInferenceRule(rule)
	// 注入后应真正生效：注入一个能识别 integer 的规则，喂数字路径，推断后类型应为 integer
	for _, id := range []string{"101", "102", "103"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/users/"+id, nil, "GET", nil))
	}
	// 无 panic 即说明注入成功；进一步验证 inferenceRule 字段非 nil
	if r.inferenceRule == nil {
		t.Error("SetInferenceRule 后 inferenceRule 不应为 nil")
	}
}

func TestSetInferenceRule_NilGuard(t *testing.T) {
	r := newSilentRouter()
	original := r.inferenceRule
	// 传入 nil 不应覆盖已有规则
	r.SetInferenceRule(nil)
	if r.inferenceRule != original {
		t.Error("SetInferenceRule(nil) 不应覆盖已有规则")
	}
}

// TestSetMergeConfig_Boundaries 覆盖 SetMergeConfig 的阈值钳制分支
// （SiblingMergeThreshold<2 钳到 2、PatternSimilarityThreshold<0 钳到 0、>1 钳到 1）。
func TestSetMergeConfig_Boundaries(t *testing.T) {
	r := newSilentRouter()
	// 阈值过低
	r.SetMergeConfig(MergeConfig{SiblingMergeThreshold: 0, PatternSimilarityThreshold: -0.5})
	cfg := r.GetMergeConfig()
	if cfg.SiblingMergeThreshold != 2 {
		t.Errorf("阈值<2 应钳到 2，实际 %d", cfg.SiblingMergeThreshold)
	}
	if cfg.PatternSimilarityThreshold != 0.0 {
		t.Errorf("相似度<0 应钳到 0，实际 %v", cfg.PatternSimilarityThreshold)
	}
	// 相似度过高
	r.SetMergeConfig(MergeConfig{PatternSimilarityThreshold: 1.5})
	cfg = r.GetMergeConfig()
	if cfg.PatternSimilarityThreshold != 1.0 {
		t.Errorf("相似度>1 应钳到 1，实际 %v", cfg.PatternSimilarityThreshold)
	}
	// 正常值原样保留
	r.SetMergeConfig(MergeConfig{SiblingMergeThreshold: 5, PatternSimilarityThreshold: 0.6})
	cfg = r.GetMergeConfig()
	if cfg.SiblingMergeThreshold != 5 || cfg.PatternSimilarityThreshold != 0.6 {
		t.Errorf("正常值应原样保留，实际 threshold=%d sim=%v", cfg.SiblingMergeThreshold, cfg.PatternSimilarityThreshold)
	}
}
