package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/router"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/tree"
)

func TestNormalize_CurlAndURL(t *testing.T) {
	body, _ := json.Marshal(NormalizeRequest{Lines: []string{
		"  ",
		`curl 'http://api.example.com/api/users/1'`,
		"GET http://api.example.com/api/users/2",
		"http://api.example.com/api/users/3",
		"curl -X POST http://nope",
	}})
	req := httptest.NewRequest(http.MethodPost, "/api/normalize", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handleNormalize(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("应 200，got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp NormalizeResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Summary.Assets == 0 || resp.Tree == nil {
		t.Fatalf("应还原出资产与树：%+v", resp.Summary)
	}
}

func TestNormalize_BadRequests(t *testing.T) {
	rec := httptest.NewRecorder()
	handleNormalize(rec, httptest.NewRequest(http.MethodGet, "/api/normalize", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET 应 405，got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	handleNormalize(rec, httptest.NewRequest(http.MethodPost, "/api/normalize", strings.NewReader("{")))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("坏 JSON 应 400，got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	handleNormalize(rec, httptest.NewRequest(http.MethodPost, "/api/normalize", strings.NewReader(`{"lines":[]}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("空行应 400，got %d", rec.Code)
	}
}

func TestHealthAndMux(t *testing.T) {
	mux := newMux("")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "ok") {
		t.Fatalf("健康检查失败：%d %s", rec.Code, rec.Body.String())
	}

	prefixed := newMux("/demo")
	rec = httptest.NewRecorder()
	prefixed.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/demo/api/health", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("带前缀健康检查失败：%d", rec.Code)
	}
}

func TestHelpers(t *testing.T) {
	if !looksLikeCurl("  curl http://x") || looksLikeCurl("http://x") {
		t.Fatal("curl 识别错误")
	}
	if !isMethod("post") || isMethod("FETCH") {
		t.Fatal("方法识别错误")
	}
	if serveError(errors.New("boom")) == "" {
		t.Fatal("启动错误应有内容")
	}

	dup := []router.NormalizedRoute{
		{Method: "GET", Template: "/a"},
		{Method: "GET", Template: "/a"},
		{Method: "POST", Template: "/b"},
	}
	got := assetsFrom(dup)
	if len(got) != 2 {
		t.Fatalf("重复资产应去重，got %d", len(got))
	}

	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusTeapot, map[string]string{"k": "v"})
	if rec.Code != http.StatusTeapot || !strings.Contains(rec.Body.String(), "v") {
		t.Fatalf("writeJSON 不符：%d %s", rec.Code, rec.Body.String())
	}
}

// failWriter 写入即失败，覆盖 writeJSON 的编码错误分支。
type failWriter struct{ h http.Header }

func (f *failWriter) Header() http.Header {
	if f.h == nil {
		f.h = http.Header{}
	}
	return f.h
}
func (f *failWriter) WriteHeader(int) {}
func (f *failWriter) Write([]byte) (int, error) {
	return 0, errors.New("broken pipe")
}

func TestWriteJSON_EncodeError(t *testing.T) {
	writeJSON(&failWriter{}, http.StatusOK, map[string]string{"k": "v"})
}

func TestRun_BadAddr(t *testing.T) {
	if err := run("", "bad-addr"); err == nil {
		t.Fatal("非法监听地址应返回错误")
	}
}

// TestMain_BadAddrExits 覆盖 main 在监听失败时走到退出点。
func TestMain_BadAddrExits(t *testing.T) {
	oldAddr := *addr
	*addr = "bad-addr"
	t.Cleanup(func() { *addr = oldAddr })

	var got []any
	oldExit := exitOn
	exitOn = func(v ...any) { got = append([]any{}, v...) }
	t.Cleanup(func() { exitOn = oldExit })

	main()
	if len(got) != 1 || !strings.Contains(fmt.Sprint(got[0]), "bad-addr") {
		t.Fatalf("监听失败应走到退出点，got %v", got)
	}

	// 真实二进制第一次解析命令行：用全新 FlagSet 覆盖 flag.Parse 分支。
	fs := flag.NewFlagSet("demo", flag.ContinueOnError)
	fs.String("addr", oldAddr, "")
	fs.String("prefix", "/demo/", "")
	if msg := boot(fs, []string{"-prefix", "/demo/"}); !strings.Contains(msg, "bad-addr") {
		t.Fatalf("首次解析后仍应因监听失败退出，got %q", msg)
	}
	if boot(nil, nil) == "" {
		t.Fatal("空 FlagSet 也应因监听失败返回退出信息")
	}
}

func TestShouldExit(t *testing.T) {
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

func TestBoot_ListenSucceeds(t *testing.T) {
	old := listenAndServe
	listenAndServe = func(string, http.Handler) error { return nil }
	t.Cleanup(func() { listenAndServe = old })

	if msg := boot(flag.CommandLine, nil); msg != "" {
		t.Fatalf("监听成功应正常返回，got %q", msg)
	}
}

func TestShowcase(t *testing.T) {
	mux := newMux("")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/showcase", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("showcase 应 200，got %d %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Tree    *tree.RouteNodeJSON `json:"tree"`
		Summary Summary             `json:"summary"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Tree == nil || resp.Summary.Assets == 0 || resp.Summary.Parsed == 0 {
		t.Fatalf("showcase 应还原出真实资产：%+v", resp.Summary)
	}

	bad := httptest.NewRecorder()
	mux.ServeHTTP(bad, httptest.NewRequest(http.MethodPut, "/api/showcase", nil))
	if bad.Code != http.StatusMethodNotAllowed {
		t.Fatalf("PUT 应 405，got %d", bad.Code)
	}
}

// TestBatchErrorJSON 验证失败原因序列化成字符串而不是空对象，
// 同时覆盖没有错误对象的记录（只计数、原因留空）。
func TestBatchErrorJSON(t *testing.T) {
	body, _ := json.Marshal(NormalizeRequest{Lines: []string{
		"curl 'http://api.example.com/a",
		"curl 'http://api.example.com/api/users/1'",
	}})
	rec := httptest.NewRecorder()
	handleNormalize(rec, httptest.NewRequest(http.MethodPost, "/api/normalize", bytes.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("应 200，got %d", rec.Code)
	}
	var resp struct {
		Errors []batchErrorJSON `json:"errors"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Errors) != 1 || resp.Errors[0].Index != 0 || resp.Errors[0].Error == "" {
		t.Fatalf("失败原因应是可读字符串：%+v", resp.Errors)
	}
	if strings.Contains(rec.Body.String(), `"Err"`) {
		t.Fatalf("不应再输出空的 Err 对象：%s", rec.Body.String())
	}

	empty := toBatchErrors([]router.BatchError{{Index: 3, Raw: "x"}})
	if len(empty) != 1 || empty[0].Error != "" || empty[0].Index != 3 {
		t.Fatalf("无错误对象时应输出空原因：%+v", empty)
	}
}
