package main

import (
	"errors"
	"flag"
	"strings"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/exporter"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/router"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/tree"
)

func TestRest_VersionAndNormalize(t *testing.T) {
	if printVersion(false) {
		t.Fatal("未要求打印版本时应返回 false")
	}
	if !printVersion(true) {
		t.Fatal("要求打印版本时应返回 true")
	}
	if exportError(errors.New("x")) == "" {
		t.Fatal("导出错误应有内容")
	}
	miss := normalizeLine(router.NormalizedRoute{}, false)
	if miss != "未命中已知路由" {
		t.Fatalf("未命中说明不符：%s", miss)
	}
	hit := normalizeLine(router.NormalizedRoute{Method: "GET", Template: "/api"}, true)
	if hit != "GET /api" {
		t.Fatalf("命中格式不符：%s", hit)
	}
}

func TestRest_NilExportAndBatch(t *testing.T) {
	if exportError(nil) != "" {
		t.Fatal("无错误时应为空")
	}
	got := batchSummary(router.BatchResult{Processed: 1, Failed: 1, Errors: []router.BatchError{{Index: 0, Raw: "bad", Err: errors.New("x")}}})
	if !strings.Contains(got, "失败 1") || !strings.Contains(got, "bad") {
		t.Fatalf("失败样本应列出：%s", got)
	}
	if handleFlags(flag.CommandLine, nil) {
		t.Fatal("测试框架已解析命令行，不应退出")
	}
	if handleFlags(nil, nil) {
		t.Fatal("空 FlagSet 不应退出")
	}

	bad := flag.NewFlagSet("bad", flag.ContinueOnError)
	if handleFlags(bad, []string{"-unknown"}) {
		t.Fatal("无法识别的参数应忽略，而不是当版本退出")
	}

	ver := flag.NewFlagSet("ver", flag.ContinueOnError)
	if boot(ver, []string{"-version"}) != "" {
		t.Fatal("-version 应只打印版本，正常返回")
	}
	plain := flag.NewFlagSet("plain", flag.ContinueOnError)
	if handleFlags(plain, nil) {
		t.Fatal("未带 -version 时应继续跑样例")
	}
}

// TestRest_MainExitPaths 覆盖 main 的正常结束与导出失败退出。
func TestRest_MainExitPaths(t *testing.T) {
	var got []any
	oldExit := exitOn
	exitOn = func(v ...any) {
		if len(v) > 0 && v[0] != "" {
			got = append([]any{}, v...)
		}
	}
	t.Cleanup(func() { exitOn = oldExit })

	main()
	if len(got) != 0 {
		t.Fatalf("样例应正常结束，got %v", got)
	}

	oldExport := exportTree
	exportTree = func(*exporter.OpenAPIExporter, *tree.Tree) ([]byte, error) {
		return nil, errors.New("disk")
	}
	t.Cleanup(func() { exportTree = oldExport })

	msg := runQuickstart()
	if !strings.Contains(msg, "disk") {
		t.Fatalf("导出失败应返回退出信息，got %q", msg)
	}
	exitOn(msg)
	if len(got) != 1 || got[0] != msg {
		t.Fatalf("退出点应收到导出失败信息，got %v", got)
	}
}

func TestRest_ShouldExit(t *testing.T) {
	if shouldExit(nil) || shouldExit([]any{""}) {
		t.Fatal("空退出信息不应结束进程")
	}
	if !shouldExit([]any{"boom"}) {
		t.Fatal("非空退出信息应结束进程")
	}

	var code int
	old := osExit
	osExit = func(c int) { code = c }
	t.Cleanup(func() { osExit = old })

	exitOn()
	exitOn("")
	if code != 0 {
		t.Fatalf("空信息不应退出，code=%d", code)
	}
	exitOn("boom")
	if code != 1 {
		t.Fatalf("非空信息应以状态 1 退出，code=%d", code)
	}
}
