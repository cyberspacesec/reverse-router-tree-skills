package main

import (
	"bytes"
	"io"
	"os"
	"sort"
	"strings"
	"testing"
)

// captureQSOutput 把 fn 执行期间写入 os.Stdout 的内容捕获为字符串返回。
// 用独立 goroutine 边读边写，避免 main 输出量大时管道阻塞死锁。
func captureQSOutput(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("创建管道失败: %v", err)
	}
	os.Stdout = w

	var buf bytes.Buffer
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		_, _ = io.Copy(&buf, r)
	}()

	fn()

	_ = w.Close()
	os.Stdout = old
	<-readDone
	_ = r.Close()
	return buf.String()
}

// TestGapQS_OutputSections 校验 main 端到端输出包含全部四个阶段：
// 批量喂入结果、路由树、OpenAPI 规范、统计指标。
func TestGapQS_OutputSections(t *testing.T) {
	out := captureQSOutput(t, main)

	// 各阶段标题都应出现。
	for _, want := range []string{
		"=== 批量喂入结果 ===",
		"=== 还原路由树 ===",
		"=== OpenAPI 3.0.3 规范 ===",
		"=== 统计 ===",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("输出缺少阶段标题 %q，实际输出:\n%s", want, out)
		}
	}

	// 8 条 curl 全合法：成功 8 条、失败 0 条。
	if !strings.Contains(out, "成功 8 条，失败 0 条") {
		t.Errorf("输出未含'成功 8 条，失败 0 条'，实际输出:\n%s", out)
	}

	// 路由树应还原出 users 相关路径。
	if !strings.Contains(out, "users") {
		t.Errorf("路由树输出未含 users 路径，实际输出:\n%s", out)
	}

	// OpenAPI 文档应包含自定义标题与版本号。
	for _, want := range []string{"Users API (Reverse Engineered)", "3.0.3"} {
		if !strings.Contains(out, want) {
			t.Errorf("OpenAPI 输出未含 %q，实际输出:\n%s", want, out)
		}
	}

	// 统计行应报告处理了 8 个请求。
	if !strings.Contains(out, "请求数: 8") {
		t.Errorf("统计输出未含'请求数: 8'，实际输出:\n%s", out)
	}
}

// TestGapQS_RerunStable 多次调用 main 输出应语义一致（固定输入，确定性还原）。
// 注意：路由树打印中同级参数节点的先后顺序不保证稳定（如 page/size 互换），
// 故比较行集合而非原始字符串，避免顺序抖动导致偶发失败。
func TestGapQS_RerunStable(t *testing.T) {
	first := captureQSOutput(t, main)
	second := captureQSOutput(t, main)
	if normalizeGapQSLines(first) != normalizeGapQSLines(second) {
		t.Errorf("main 两次运行输出不一致:\n第一次:\n%s\n第二次:\n%s", first, second)
	}
	if first == "" {
		t.Error("main 输出为空")
	}
}

// normalizeGapQSLines 将多行输出按行切分排序后重组，用于消除同级节点顺序抖动。
func normalizeGapQSLines(s string) string {
	lines := strings.Split(s, "\n")
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}
