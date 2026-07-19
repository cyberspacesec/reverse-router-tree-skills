package router

import (
	"fmt"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
)

// BatchResult 批量喂入的结果聚合。
//
// 单条样本解析或处理失败不中断整批；失败记录在 Errors 中，
// 调用方可据此决定是否告警或丢弃坏样本。Processed 为成功喂入数。
type BatchResult struct {
	// Processed 成功喂入路由树的样本数
	Processed int
	// Failed 解析或处理失败的样本数
	Failed int
	// Errors 失败样本的详细信息（索引 + 错误），长度上限 maxBatchErrors 防爆炸
	Errors []BatchError
}

// BatchError 单个失败样本的错误信息。
type BatchError struct {
	// Index 样本在输入切片中的下标（从 0 起）
	Index int
	// Raw 样本原始值（截断到 maxBatchErrorRawLen 字节防超长撑爆日志）
	Raw string
	// Err 失败原因
	Err error
}

const (
	// maxBatchErrors 记录的最大失败详情数，超出后只计数不记详情。
	maxBatchErrors = 100
	// maxBatchErrorRawLen 单条失败样本原始值的截断长度。
	maxBatchErrorRawLen = 128
)

// ReverseRequests 批量喂入 HTTP 请求样本。
//
// 逐条调用 ReverseHttpRequest，单条失败（处理错误）不中断整批，
// 失败样本记入 result.Errors。批次结束后自动调用 InferRequiredParams。
// 并发安全由 ReverseHttpRequest 内部保证（mergeMu + 节点 typeMu），本方法串行喂入；
// 如需并行由调用方自行分片调多个 ReverseRequests（共享同一 ReverseRouter）。
//
// 返回的 BatchResult.Errors 切片非 nil（可能为空）。
func (x *ReverseRouter) ReverseRequests(reqs []*request.HttpRequest) BatchResult {
	result := BatchResult{Errors: make([]BatchError, 0)}
	for i, req := range reqs {
		if req == nil {
			result.Failed++
			x.appendBatchError(&result, i, "<nil>", fmt.Errorf("请求为 nil"))
			continue
		}
		if err := x.ReverseHttpRequest(req); err != nil {
			result.Failed++
			x.appendBatchError(&result, i, req.GetUrl(), err)
			continue
		}
		result.Processed++
	}
	x.InferRequiredParams()
	return result
}

// ReverseCurls 批量解析 curl 命令并喂入路由树。
//
// 逐条 request.ParseCurl + ReverseHttpRequest，单条 curl 解析失败不中断整批。
// 适合上层从测绘平台一次导出上万条 curl 的场景：坏样本（语法错误/畸形 flag）
// 被跳过并记入 Errors，不影响其余样本的还原。批次结束自动 InferRequiredParams。
func (x *ReverseRouter) ReverseCurls(curls []string) BatchResult {
	result := BatchResult{Errors: make([]BatchError, 0)}
	for i, curl := range curls {
		req, err := request.ParseCurl(curl)
		if err != nil {
			result.Failed++
			x.appendBatchError(&result, i, curl, err)
			continue
		}
		if err := x.ReverseHttpRequest(req); err != nil {
			result.Failed++
			x.appendBatchError(&result, i, curl, err)
			continue
		}
		result.Processed++
	}
	x.InferRequiredParams()
	return result
}

// appendBatchError 追加一条失败记录，超出上限后只计数不记详情。
func (x *ReverseRouter) appendBatchError(result *BatchResult, index int, raw string, err error) {
	if len(result.Errors) >= maxBatchErrors {
		return
	}
	if len(raw) > maxBatchErrorRawLen {
		raw = raw[:maxBatchErrorRawLen] + "...(truncated)"
	}
	result.Errors = append(result.Errors, BatchError{
		Index: index,
		Raw:   raw,
		Err:   err,
	})
}
