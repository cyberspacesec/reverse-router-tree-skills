package main

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/exporter"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/router"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/tree"
)

// 捕获 stdout 后执行 fn，返回捕获到的输出。
// main 输出约 6KB，远小于管道缓冲区，同步调用即可，无需 goroutine。
func captureQSStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("创建管道失败: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()

	fn()

	w.Close()
	os.Stdout = old
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("读取管道失败: %v", err)
	}
	return string(out)
}

// TestGapQS_MainHappyPath 调用 main() 走完成功主路径。
// 断言批量喂入、路由树、OpenAPI、统计四段输出都存在。
func TestGapQS_MainHappyPath(t *testing.T) {
	out := captureQSStdout(t, main)

	for _, want := range []string{"成功 8 条", "还原路由树", "OpenAPI", "统计"} {
		if !strings.Contains(out, want) {
			t.Errorf("main 输出缺少 %q", want)
		}
	}
	// 数字 ID 应合并为路径变量，query 参数应还原。
	if !strings.Contains(out, "{users_id}") && !strings.Contains(out, "users_id") {
		t.Errorf("输出未见路径变量 users_id")
	}
}

// TestGapQS_BadCurlFailSoft 覆盖 main.go 错误打印分支（result.Errors 循环）。
// main 内 8 条 curl 全合法，该循环在 main() 中不可达；
// 这里用坏样本复刻同一 fail-soft 语义：坏样本被跳过并记入 Errors。
func TestGapQS_BadCurlFailSoft(t *testing.T) {
	r := router.NewReverseRouter()
	result := r.ReverseCurls([]string{
		`curl 'http://api.example.com/api/users?page=1'`,
		`这不是 curl 命令`,                               // 解析失败样本
		`curl 'http://api.example.com/api/unclosed`, // 未闭合引号样本
	})
	if result.Processed != 1 {
		t.Errorf("成功数 = %d，期望 1", result.Processed)
	}
	if result.Failed != 2 {
		t.Errorf("失败数 = %d，期望 2", result.Failed)
	}
	if len(result.Errors) != 2 {
		t.Fatalf("Errors 长度 = %d，期望 2", len(result.Errors))
	}
	// 复刻 main 中的错误打印格式，确保分支逻辑可运行。
	for _, e := range result.Errors {
		if e.Err == nil {
			t.Errorf("失败[%d] 缺少错误原因", e.Index)
		}
	}
}

// TestGapQS_ExportNilTreeError 覆盖 main.go 导出失败分支（log.Fatalf）。
// main 内树恒合法且 Export 恒成功，该分支在 main() 中不可达；
// 这里直接对空树调用 Export，验证其返回错误（即 main 会 Fatal 的条件）。
func TestGapQS_ExportNilTreeError(t *testing.T) {
	exp := exporter.NewOpenAPIExporter()

	if _, err := exp.Export(nil); err == nil {
		t.Errorf("Export(nil) 期望返回错误")
	}
	if _, err := exp.Export(&tree.Tree{}); err == nil {
		t.Errorf("Export(空树) 期望返回错误")
	}

	// 合法树应导出成功，对应 main 成功路径。
	r := router.NewReverseRouter()
	r.ReverseCurls([]string{`curl 'http://api.example.com/api/users/123'`})
	doc, err := exp.Export(r.Tree)
	if err != nil {
		t.Fatalf("合法树导出失败: %v", err)
	}
	if !strings.Contains(string(doc), "openapi") {
		t.Errorf("导出文档缺少 openapi 标记")
	}
}

// TestGapQS_StatsConsistency 覆盖 main.go 统计打印分支。
// 用与 main 相同的样本喂入，断言统计计数自洽。
func TestGapQS_StatsConsistency(t *testing.T) {
	r := router.NewReverseRouter()
	result := r.ReverseCurls([]string{
		`curl 'http://api.example.com/api/users?page=1&size=20'`,
		`curl 'http://api.example.com/api/users/123'`,
	})
	if result.Processed != 2 {
		t.Fatalf("成功数 = %d，期望 2", result.Processed)
	}
	st := r.GetStats()
	if st.RequestsProcessed != 2 {
		t.Errorf("RequestsProcessed = %d，期望 2", st.RequestsProcessed)
	}
	if r.Tree.String() == "" {
		t.Errorf("路由树字符串为空")
	}
}
