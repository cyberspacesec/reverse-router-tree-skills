package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

// captureGapQS2Output 捕获 fn 执行期间写入 os.Stdout 的内容。
// 使用独立 goroutine 边读边写，避免 main 输出量稍大时管道阻塞。
// 函数名后加 2 是为了避免与同一目录下并行任务的 captureQSOutput 冲突。
func captureGapQS2Output(t *testing.T, fn func()) string {
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

// TestGapQS_QS2_BatchAndTree 校验批量喂入计数与路由树还原结果。
// 对应 main.go 中 ReverseCurls 成功路径与 Tree.String 打印分支。
func TestGapQS_QS2_BatchAndTree(t *testing.T) {
	out := captureGapQS2Output(t, main)

	// 8 条内置 curl 全部合法，应全部成功且失败数为 0。
	// 该断言覆盖“错误循环体不执行”的正常路径（result.Errors 为空）。
	if !strings.Contains(out, "成功 8 条，失败 0 条") {
		t.Errorf("批量喂入计数不符合预期，实际输出:\n%s", out)
	}

	// 路由树应把数字 ID 合并为路径变量，并保留 users 前缀。
	if !strings.Contains(out, "users") {
		t.Errorf("路由树输出未含 users，实际输出:\n%s", out)
	}
}

// TestGapQS_QS2_OpenAPIValidJSON 校验导出的 OpenAPI 文档是合法 JSON，
// 且包含 3.0.3 版本标识与 users 相关路径。
// 对应 main.go 中 Export 成功路径与打印文档分支。
func TestGapQS_QS2_OpenAPIValidJSON(t *testing.T) {
	out := captureGapQS2Output(t, main)

	// OpenAPI 文档是缩进格式 JSON：取规范标题之后、统计标题之前的内容。
	header := "=== OpenAPI 3.0.3 规范 ==="
	hi := strings.Index(out, header)
	if hi < 0 {
		t.Fatalf("未找到 OpenAPI 规范标题，实际输出:\n%s", out)
	}
	rest := out[hi+len(header):]
	start := strings.Index(rest, "{")
	if start < 0 {
		t.Fatalf("未找到 OpenAPI JSON 起点，实际输出:\n%s", out)
	}
	end := strings.Index(rest[start:], "=== 统计 ===")
	if end < 0 {
		t.Fatalf("未找到统计标题，无法截取 JSON，实际输出:\n%s", out)
	}
	raw := strings.TrimSpace(rest[start : start+end])

	var doc map[string]any
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatalf("OpenAPI 输出不是合法 JSON: %v\n原文:\n%s", err, raw)
	}
	if doc["openapi"] != "3.0.3" {
		t.Errorf("openapi 版本不是 3.0.3，实际为 %v", doc["openapi"])
	}
	paths, ok := doc["paths"].(map[string]any)
	if !ok || len(paths) == 0 {
		t.Errorf("OpenAPI paths 为空，实际文档:\n%s", raw)
	}
	found := false
	for p := range paths {
		if strings.Contains(p, "users") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("OpenAPI paths 未含 users 相关路径，实际 paths: %v", paths)
	}
}

// TestGapQS_QS2_StatsMetrics 校验统计行包含四项指标且请求数为 8。
// 对应 main.go 末尾 GetStats 与打印统计分支。
func TestGapQS_QS2_StatsMetrics(t *testing.T) {
	out := captureGapQS2Output(t, main)

	// 统计行应同时出现四个指标关键字。
	for _, want := range []string{"请求数:", "路径变量:", "参数:", "类型推断:"} {
		if !strings.Contains(out, want) {
			t.Errorf("统计输出缺少 %q，实际输出:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "请求数: 8") {
		t.Errorf("统计输出未含'请求数: 8'，实际输出:\n%s", out)
	}
}
