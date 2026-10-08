// Package main 演示 reverse-router-tree 的端到端用法：从一组 HTTP 请求
// 还原路由树，并导出为 OpenAPI 3.0.3 规范。
//
// 运行：go run ./examples/quickstart
//
// 这是给上层集成项目的最小参考样板：喂数据 → 拿路由树 → 导出规范。
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/exporter"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/router"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/tree"
)

// 以下变量由 goreleaser 构建时通过 -ldflags -X 注入版本/提交/构建时间。
// 本地 go run 时不注入，保持默认占位。
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

// exitOn 是进程退出点。测试里换成记录函数，避免 log.Fatal 直接结束测试进程。
var osExit = os.Exit

var exitOn = func(v ...any) {
	if shouldExit(v) {
		log.Print(v...)
		osExit(1)
	}
}

// shouldExit 判断退出信息是否真的要结束进程。空信息表示正常结束。
func shouldExit(v []any) bool {
	return len(v) > 0 && v[0] != ""
}

// exportTree 导出路由树。测试可替换成返回错误的实现，覆盖导出失败退出。
var exportTree = func(exp *exporter.OpenAPIExporter, t *tree.Tree) ([]byte, error) {
	return exp.Export(t)
}

func main() {
	// 版本标志仅对真实二进制生效：--version 打印后退出。
	// quickstart 测试直接调用 main() 时，全局 flag.CommandLine 已被测试框架
	// 解析过，故只在未解析过（真实运行）时才注册并解析，避免与测试参数冲突、
	// 保持测试可重复运行。
	exitOn(boot(flag.CommandLine, flag.Args()))
}

// boot 处理 --version，否则跑样例。返回非空字符串表示应以该信息退出。
func boot(fs *flag.FlagSet, args []string) string {
	if handleFlags(fs, args) {
		return ""
	}
	return runQuickstart()
}

// runQuickstart 喂入样例、打印路由树与归一化结果。导出失败时返回退出信息。
func runQuickstart() string {
	r := router.NewReverseRouter()

	// 模拟测绘平台导出的一批 curl 命令：数字 ID 合并为 {users_id}，
	// query 参数还原，Authorization 头推断为 Bearer 安全方案。
	// 用 ReverseCurls 批量喂入：单条坏样本不中断整批（fail-soft）。
	curls := []string{
		`curl 'http://api.example.com/api/users?page=1&size=20' -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiJ9'`,
		`curl 'http://api.example.com/api/users?page=2&size=20' -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiJ9'`,
		`curl 'http://api.example.com/api/users/123' -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiJ9'`,
		`curl 'http://api.example.com/api/users/456' -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiJ9'`,
		`curl 'http://api.example.com/api/users/789' -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiJ9'`,
		`curl 'http://api.example.com/api/users' -X POST -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiJ9' -H 'Content-Type: application/json' -d '{"name":"alice","age":30}'`,
		`curl 'http://api.example.com/api/users/123' -X PUT -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiJ9' -H 'Content-Type: application/json' -d '{"name":"alice2"}'`,
		`curl 'http://api.example.com/api/users/123' -X DELETE -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiJ9'`,
	}
	result := r.ReverseCurls(curls)
	fmt.Print(batchSummary(result))

	// 打印还原出的路由树。
	fmt.Println("=== 还原路由树 ===")
	fmt.Println(r.Tree.String())

	// 导出为 OpenAPI 3.0.3。
	exp := exporter.NewOpenAPIExporter()
	exp.Title = "Users API (Reverse Engineered)"
	exp.Version = "1.0.0"
	exp.ServerURL = "https://api.example.com"
	doc, err := exportTree(exp, r.Tree)
	if msg := exportError(err); msg != "" {
		return msg
	}

	fmt.Println("=== OpenAPI 3.0.3 规范 ===")
	fmt.Println(string(doc))

	// 统计指标。
	st := r.GetStats()
	fmt.Printf("=== 统计 ===\n请求数: %d  路径变量: %d  参数: %d  类型推断: %d\n",
		st.RequestsProcessed, st.PathVariablesIdentified, st.ParamsCreated, st.TypeInferences)

	// === URL 资产归一化 ===
	// 散落的同接口 URL 归一到稳定的路由模板，支撑测绘资产去重/聚合/检索。
	fmt.Println("=== 已知资产清单（ListAssets） ===")
	for _, a := range r.ListAssets() {
		fmt.Printf("  %-6s %-24s params=%v required=%v\n",
			a.Method, a.Template, a.QueryParams, a.RequiredParams)
	}

	// 裸 URL 直达：新来的 /api/users/999 应归到已有模板，而非新开节点。
	fmt.Println("=== 归一化直达（NormalizeURLString） ===")
	samples := []string{
		"http://api.example.com/api/users/999",
		"http://api.example.com/api/users?page=3&size=20",
	}
	for _, u := range samples {
		route, ok := r.NormalizeURLString(u, "GET")
		fmt.Printf("  %-52s → %s\n", u, normalizeLine(route, ok))
	}
	return ""
}

// printVersion 在要求打印版本时输出一行并返回 true。
func printVersion(show bool) bool {
	if !show {
		return false
	}
	fmt.Printf("quickstart %s (commit %s, built %s)\n", version, commit, date)
	return true
}

// exportError 把导出失败整理成退出信息。
func exportError(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("导出 OpenAPI 失败: %v", err)
}

// normalizeLine 把归一化结果格式化成一行；未命中时给出固定说明。
func normalizeLine(route router.NormalizedRoute, ok bool) string {
	if !ok {
		return "未命中已知路由"
	}
	return route.Method + " " + route.Template
}

// handleFlags 解析 --version。命令行已经解析过，或解析失败时，什么都不做。
// 返回 true 表示只打印了版本，应直接退出。
func handleFlags(fs *flag.FlagSet, args []string) bool {
	if fs == nil || fs.Parsed() {
		return false
	}
	showVersion := fs.Bool("version", false, "打印版本信息并退出")
	if err := fs.Parse(args); err != nil {
		return false
	}
	return printVersion(*showVersion)
}

// batchSummary 汇总批量喂入结果，失败样本逐条列出。
func batchSummary(result router.BatchResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "=== 批量喂入结果 ===\n成功 %d 条，失败 %d 条\n", result.Processed, result.Failed)
	for _, e := range result.Errors {
		fmt.Fprintf(&b, "  失败[%d] %s: %v\n", e.Index, e.Raw, e.Err)
	}
	return b.String()
}
