package router

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync/atomic"
)

// LogLevel 日志级别类型
type LogLevel int

const (
	// LogLevelDebug 调试级别（记录所有决策细节，包括每次合并、类型推断）
	LogLevelDebug LogLevel = iota
	// LogLevelInfo 信息级别（记录关键事件：请求处理、变量识别、必需性推断）
	LogLevelInfo
	// LogLevelWarn 警告级别（记录异常但不影响运行的情况，如非法 JSON body）
	LogLevelWarn
	// LogLevelError 错误级别（仅记录错误）
	LogLevelError
	// LogLevelOff 关闭日志
	LogLevelOff
)

// RouterLogger 路由器日志器，封装 slog 提供结构化日志。
//
// 日志按级别记录逆向工程过程中的关键决策：
//   - Debug：每次路径变量合并、模式检测结果、类型推断结果
//   - Info：请求处理、变量识别、必需性推断完成
//   - Warn：异常数据兼容（非法 body、模式匹配失败等）
//   - Error：处理失败
type RouterLogger struct {
	logger *slog.Logger
	enabled bool
}

// NewRouterLogger 创建默认日志器（输出到 os.Stderr，Warn 级别，纯文本格式）。
//
// 默认 Warn 级别：仅输出警告和错误，保持低噪音。
// 调试逆向过程时用 SetLogLevel(LogLevelDebug) 或 NewRouterLoggerWithLevel(LogLevelDebug, w)。
func NewRouterLogger() *RouterLogger {
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})
	return &RouterLogger{
		logger:  slog.New(handler),
		enabled: true,
	}
}

// NewRouterLoggerWithWriter 创建指定输出流的日志器（Warn 级别，纯文本）。
func NewRouterLoggerWithWriter(w io.Writer) *RouterLogger {
	handler := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})
	return &RouterLogger{
		logger:  slog.New(handler),
		enabled: true,
	}
}

// NewRouterLoggerWithLevel 创建指定级别和输出流的日志器。
// level 控制输出哪些日志；w 为 nil 时输出到 os.Stderr。
func NewRouterLoggerWithLevel(level LogLevel, w io.Writer) *RouterLogger {
	if w == nil {
		w = os.Stderr
	}
	var slogLevel slog.Level
	switch level {
	case LogLevelDebug:
		slogLevel = slog.LevelDebug
	case LogLevelInfo:
		slogLevel = slog.LevelInfo
	case LogLevelWarn:
		slogLevel = slog.LevelWarn
	case LogLevelError:
		slogLevel = slog.LevelError
	case LogLevelOff:
		// 用一个丢弃所有输出的 handler
		return &RouterLogger{logger: slog.New(discardHandler{}), enabled: false}
	default:
		slogLevel = slog.LevelInfo
	}
	handler := slog.NewTextHandler(w, &slog.HandlerOptions{Level: slogLevel})
	return &RouterLogger{logger: slog.New(handler), enabled: level != LogLevelOff}
}

// SetLevel 调整日志级别。仅当 logger 已初始化时生效。
func (l *RouterLogger) SetLevel(level LogLevel) {
	if l == nil || l.logger == nil {
		return
	}
	// 重建 handler 以应用新级别
	// 注意：slog.Logger 内部 handler 不可变，需通过 SetLogger 替换
	var slogLevel slog.Level
	switch level {
	case LogLevelDebug:
		slogLevel = slog.LevelDebug
	case LogLevelInfo:
		slogLevel = slog.LevelInfo
	case LogLevelWarn:
		slogLevel = slog.LevelWarn
	case LogLevelError:
		slogLevel = slog.LevelError
	case LogLevelOff:
		l.enabled = false
		l.logger = slog.New(discardHandler{})
		return
	default:
		slogLevel = slog.LevelInfo
	}
	l.enabled = true
	l.logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slogLevel}))
}

// discardHandler 丢弃所有日志的 handler
type discardHandler struct{}

func (discardHandler) Enabled(_ context.Context, _ slog.Level) bool      { return false }
func (discardHandler) Handle(_ context.Context, _ slog.Record) error    { return nil }
func (h discardHandler) WithAttrs(_ []slog.Attr) slog.Handler           { return h }
func (h discardHandler) WithGroup(_ string) slog.Handler                { return h }

// 以下方法封装 slog，nil 安全
//
// 对超长字符串值（如恶意超长 URL）截断到 maxLogValueLen，防止撑爆日志。
// slog 的 args 是 key, value 交替传入，故按奇数下标取 value 处理。

const maxLogValueLen = 512

// truncateLogArgs 遍历 key/value 交替的 args，对超长 string 值截断。
// 非 string 值原样返回。args 长度为奇数时末尾的孤立 key 原样保留。
func truncateLogArgs(args []any) []any {
	if len(args) <= 1 {
		return args
	}
	// 仅在有长字符串时才复制，避免无谓分配
	hasLong := false
	for i := 1; i < len(args); i += 2 {
		if s, ok := args[i].(string); ok && len(s) > maxLogValueLen {
			hasLong = true
			break
		}
	}
	if !hasLong {
		return args
	}
	out := make([]any, len(args))
	copy(out, args)
	for i := 1; i < len(out); i += 2 {
		if s, ok := out[i].(string); ok && len(s) > maxLogValueLen {
			out[i] = s[:maxLogValueLen] + fmt.Sprintf("...(truncated %d bytes)", len(s)-maxLogValueLen)
		}
	}
	return out
}

func (l *RouterLogger) Debug(msg string, args ...any) {
	if l == nil || !l.enabled {
		return
	}
	l.logger.Debug(msg, truncateLogArgs(args)...)
}

func (l *RouterLogger) Info(msg string, args ...any) {
	if l == nil || !l.enabled {
		return
	}
	l.logger.Info(msg, truncateLogArgs(args)...)
}

func (l *RouterLogger) Warn(msg string, args ...any) {
	if l == nil || !l.enabled {
		return
	}
	l.logger.Warn(msg, truncateLogArgs(args)...)
}

func (l *RouterLogger) Error(msg string, args ...any) {
	if l == nil || !l.enabled {
		return
	}
	l.logger.Error(msg, truncateLogArgs(args)...)
}

// === 可观测性统计 ===

// RouterStats 路由器运行时统计指标，用于量化逆向工程效果。
//
// 所有计数器使用 atomic 操作，线程安全。
// 通过 ReverseRouter.GetStats() 获取快照，可用于监控和调试。
type RouterStats struct {
	// RequestsProcessed 已处理的 HTTP 请求总数
	RequestsProcessed atomic.Int64
	// PathVariablesIdentified 识别出的路径变量节点数（合并次数）
	PathVariablesIdentified atomic.Int64
	// PatternDetections 模式检测调用次数
	PatternDetections atomic.Int64
	// ParamsCreated 创建的参数节点数
	ParamsCreated atomic.Int64
	// TypeInferences 类型推断调用次数
	TypeInferences atomic.Int64
	// BodyParamsParsed 从请求体解析出的参数总数
	BodyParamsParsed atomic.Int64
	// RequiredParamsInferred 推断为必需的参数数
	RequiredParamsInferred atomic.Int64
	// MergeAttempts 尝试合并兄弟节点的次数
	MergeAttempts atomic.Int64
	// MergeSkipped 因模式不匹配跳过的合并次数
	MergeSkipped atomic.Int64
	// Warnings 警告事件数（异常数据兼容等）
	Warnings atomic.Int64
	// Errors 处理错误数
	Errors atomic.Int64
}

// NewRouterStats 创建空的统计指标。
func NewRouterStats() *RouterStats {
	return &RouterStats{}
}

// StatsSnapshot 统计指标的只读快照（值类型，便于序列化和展示）。
type StatsSnapshot struct {
	RequestsProcessed     int64 `json:"requests_processed"`
	PathVariablesIdentified int64 `json:"path_variables_identified"`
	PatternDetections     int64 `json:"pattern_detections"`
	ParamsCreated         int64 `json:"params_created"`
	TypeInferences        int64 `json:"type_inferences"`
	BodyParamsParsed      int64 `json:"body_params_parsed"`
	RequiredParamsInferred int64 `json:"required_params_inferred"`
	MergeAttempts         int64 `json:"merge_attempts"`
	MergeSkipped          int64 `json:"merge_skipped"`
	Warnings              int64 `json:"warnings"`
	Errors                int64 `json:"errors"`
}

// Snapshot 返回当前统计的快照。
func (s *RouterStats) Snapshot() StatsSnapshot {
	if s == nil {
		return StatsSnapshot{}
	}
	return StatsSnapshot{
		RequestsProcessed:       s.RequestsProcessed.Load(),
		PathVariablesIdentified: s.PathVariablesIdentified.Load(),
		PatternDetections:       s.PatternDetections.Load(),
		ParamsCreated:           s.ParamsCreated.Load(),
		TypeInferences:          s.TypeInferences.Load(),
		BodyParamsParsed:        s.BodyParamsParsed.Load(),
		RequiredParamsInferred:  s.RequiredParamsInferred.Load(),
		MergeAttempts:           s.MergeAttempts.Load(),
		MergeSkipped:            s.MergeSkipped.Load(),
		Warnings:                s.Warnings.Load(),
		Errors:                  s.Errors.Load(),
	}
}

// Reset 清零所有计数器。
func (s *RouterStats) Reset() {
	if s == nil {
		return
	}
	s.RequestsProcessed.Store(0)
	s.PathVariablesIdentified.Store(0)
	s.PatternDetections.Store(0)
	s.ParamsCreated.Store(0)
	s.TypeInferences.Store(0)
	s.BodyParamsParsed.Store(0)
	s.RequiredParamsInferred.Store(0)
	s.MergeAttempts.Store(0)
	s.MergeSkipped.Store(0)
	s.Warnings.Store(0)
	s.Errors.Store(0)
}

// String 返回人类可读的统计摘要。
func (snap StatsSnapshot) String() string {
	return fmt.Sprintf("requests=%d, path_vars=%d, params=%d, body_params=%d, type_inferences=%d, required=%d, merges=%d(skipped=%d), warnings=%d, errors=%d",
		snap.RequestsProcessed,
		snap.PathVariablesIdentified,
		snap.ParamsCreated,
		snap.BodyParamsParsed,
		snap.TypeInferences,
		snap.RequiredParamsInferred,
		snap.MergeAttempts,
		snap.MergeSkipped,
		snap.Warnings,
		snap.Errors,
	)
}
