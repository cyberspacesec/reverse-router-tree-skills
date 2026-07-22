package router

import (
	"strings"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
)

// TestReverseRequests_FailSoft 验证批量喂入时单条坏样本不中断整批，
// 且成功样本正常还原路由树。
func TestReverseRequests_FailSoft(t *testing.T) {
	r := NewReverseRouter()
	reqs := []*request.HttpRequest{
		request.NewHttpRequest("/api/users/123", nil, "GET", nil),
		nil, // 坏样本 1：nil 请求
		request.NewHttpRequest("/api/users/456", nil, "GET", nil),
		request.NewHttpRequest("/api/users/789", nil, "GET", nil),
	}
	result := r.ReverseRequests(reqs)

	if result.Processed != 3 {
		t.Errorf("Processed = %d, want 3", result.Processed)
	}
	if result.Failed != 1 {
		t.Errorf("Failed = %d, want 1", result.Failed)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("Errors len = %d, want 1", len(result.Errors))
	}
	if result.Errors[0].Index != 1 {
		t.Errorf("first error Index = %d, want 1", result.Errors[0].Index)
	}

	// 3 个数字 ID 应合并出路径变量节点
	users := r.Tree.Root.FindChildByKey("api").FindChildByKey("users")
	if users == nil {
		t.Fatal("api/users 丢失")
	}
	if users.GetChildByType("request_path_variable") == nil {
		t.Fatal("未合并出路径变量节点")
	}
}

// TestReverseCurls_FailSoft 验证批量 curl 喂入时解析失败的样本被跳过，
// 带值 flag（Task1 修复）的样本正常解析。
func TestReverseCurls_FailSoft(t *testing.T) {
	r := NewReverseRouter()
	curls := []string{
		`curl 'http://api.example.com/users/123'`,
		`not-a-curl-command`,                                    // 坏样本：非 curl
		`curl --max-time 30 'http://api.example.com/users/456'`, // 带值 flag（Task1 修复）
		``, // 坏样本：空
		`curl 'http://api.example.com/users/789'`,
	}
	result := r.ReverseCurls(curls)

	if result.Processed != 3 {
		t.Errorf("Processed = %d, want 3", result.Processed)
	}
	if result.Failed != 2 {
		t.Errorf("Failed = %d, want 2", result.Failed)
	}

	// 3 个数字 ID 应合并出路径变量节点（host api.example.com 被跳过，path=/users/123）
	users := r.Tree.Root.FindChildByKey("users")
	if users == nil {
		t.Fatal("users 丢失")
	}
	if users.GetChildByType("request_path_variable") == nil {
		t.Fatal("未合并出路径变量节点")
	}
}

// TestReverseRequests_ErrorsTruncated 验证失败超 100 条后只计数不记详情。
func TestReverseRequests_ErrorsTruncated(t *testing.T) {
	r := NewReverseRouter()
	reqs := make([]*request.HttpRequest, 150)
	for i := range reqs {
		reqs[i] = nil // 全部坏样本
	}
	result := r.ReverseRequests(reqs)

	if result.Failed != 150 {
		t.Errorf("Failed = %d, want 150", result.Failed)
	}
	if len(result.Errors) != 100 {
		t.Errorf("Errors len = %d, want 100（上限）", len(result.Errors))
	}
	if result.Processed != 0 {
		t.Errorf("Processed = %d, want 0", result.Processed)
	}
}

// TestReverseRequests_RawTruncated 验证超长 Raw 被截断到 128 字节。
func TestReverseRequests_RawTruncated(t *testing.T) {
	r := NewReverseRouter()
	longURL := "/api/x?" + strings.Repeat("a", 500)
	reqs := []*request.HttpRequest{
		request.NewHttpRequest(longURL, nil, "GET", nil),
	}
	result := r.ReverseRequests(reqs)

	// 超长 URL 正常处理不报错，Processed=1
	if result.Processed != 1 {
		t.Errorf("Processed = %d, want 1", result.Processed)
	}
}

// TestReverseCurls_AppendBatchErrorTruncated 验证超长 Raw 被截断并追加 "...(truncated)" 后缀。
func TestReverseCurls_AppendBatchErrorTruncated(t *testing.T) {
	r := NewReverseRouter()
	// 构造一条解析失败的超长 curl
	curls := []string{
		"not-a-curl-command" + strings.Repeat("x", 200),
	}
	result := r.ReverseCurls(curls)
	if result.Failed != 1 {
		t.Fatalf("Failed = %d, want 1", result.Failed)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("Errors len = %d, want 1", len(result.Errors))
	}
	raw := result.Errors[0].Raw
	if len(raw) > 128+len("...(truncated)") {
		t.Errorf("Raw 截断后过长: %d 字节", len(raw))
	}
	if !strings.HasSuffix(raw, "...(truncated)") {
		t.Errorf("Raw 尾部应为 ...(truncated)，实际 %q", raw)
	}
}

// TestReverseRequests_Empty 验证空输入不 panic、返回空结果。
func TestReverseRequests_Empty(t *testing.T) {
	r := NewReverseRouter()
	result := r.ReverseRequests(nil)
	if result.Processed != 0 || result.Failed != 0 {
		t.Errorf("空输入 Processed=%d Failed=%d, want 0/0", result.Processed, result.Failed)
	}
	if result.Errors == nil {
		t.Error("Errors 应为非 nil 空切片")
	}
}
