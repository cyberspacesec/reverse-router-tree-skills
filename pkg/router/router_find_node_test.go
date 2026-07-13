package router

import (
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
)

// === RequestPathRouter.FindNode ===

// TestRequestPathRouter_FixedPathHit 沿固定路径逐段命中末端节点。
func TestRequestPathRouter_FixedPathHit(t *testing.T) {
	r := NewReverseRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/users", nil, "GET", nil))

	paths := []*request.HttpRequestPath{
		request.NewHttpRequestPath("api"),
		request.NewHttpRequestPath("users"),
	}
	pr := &RequestPathRouter{}
	got, err := pr.FindNode(node.Node[node.NodeContext](r.Tree.Root), paths)
	if err != nil {
		t.Fatalf("FindNode 出错: %v", err)
	}
	if got == nil {
		t.Fatal("应命中 /api/users 末端节点")
	}
	if got.GetKey() != "users" {
		t.Errorf("末端节点 key 应为 users，得 %s", got.GetKey())
	}
}

// TestRequestPathRouter_PathVariableFallback 未命中的固定路径应回退到路径变量节点。
// 喂 3 个数字 ID 触发合并出 {users_id} 变量后，查未出现过的 999 应命中变量节点。
func TestRequestPathRouter_PathVariableFallback(t *testing.T) {
	r := NewReverseRouter()
	for _, u := range []string{"/api/users/1", "/api/users/2", "/api/users/3"} {
		r.ReverseHttpRequest(request.NewHttpRequest(u, nil, "GET", nil))
	}

	// 999 从未出现过，应通过变量回退命中
	paths := []*request.HttpRequestPath{
		request.NewHttpRequestPath("api"),
		request.NewHttpRequestPath("users"),
		request.NewHttpRequestPath("999"),
	}
	pr := &RequestPathRouter{}
	got, err := pr.FindNode(node.Node[node.NodeContext](r.Tree.Root), paths)
	if err != nil {
		t.Fatalf("FindNode 出错: %v", err)
	}
	if got == nil {
		t.Fatal("999 应通过路径变量回退命中 {users_id} 节点")
	}
	if got.GetType() != "request_path_variable" {
		t.Errorf("回退应命中 request_path_variable 类型节点，得 %s", got.GetType())
	}
}

// TestRequestPathRouter_PathMissReturnsNil 完全不存在的路径返回 nil。
func TestRequestPathRouter_PathMissReturnsNil(t *testing.T) {
	r := NewReverseRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/users", nil, "GET", nil))

	paths := []*request.HttpRequestPath{
		request.NewHttpRequestPath("orders"),
		request.NewHttpRequestPath("1"),
	}
	pr := &RequestPathRouter{}
	got, _ := pr.FindNode(node.Node[node.NodeContext](r.Tree.Root), paths)
	if got != nil {
		t.Errorf("不存在的路径应返回 nil，得 %v", got)
	}
}

// TestRequestPathRouter_EmptyPathsReturnsRoot 空路径返回起点节点。
func TestRequestPathRouter_EmptyPathsReturnsRoot(t *testing.T) {
	r := NewReverseRouter()
	root := node.Node[node.NodeContext](r.Tree.Root)
	pr := &RequestPathRouter{}
	got, _ := pr.FindNode(root, nil)
	if got == nil {
		t.Fatal("空路径应返回起点节点")
	}
	if got != root {
		t.Error("空路径应原样返回起点节点")
	}
}

// TestRequestPathRouter_NilStartReturnsNil nil 起点返回 nil。
func TestRequestPathRouter_NilStartReturnsNil(t *testing.T) {
	pr := &RequestPathRouter{}
	paths := []*request.HttpRequestPath{request.NewHttpRequestPath("api")}
	got, _ := pr.FindNode(nil, paths)
	if got != nil {
		t.Errorf("nil 起点应返回 nil，得 %v", got)
	}
}

// === RequestParamRouter.FindNode ===

// TestRequestParamRouter_HitByName 按参数名命中参数子节点。
func TestRequestParamRouter_HitByName(t *testing.T) {
	r := NewReverseRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/list?page=1&size=10", nil, "GET", nil))

	// 定位到方法节点
	methodNode := r.locateMethodNode([]*request.HttpRequestPath{
		request.NewHttpRequestPath("api"),
		request.NewHttpRequestPath("list"),
	}, "GET")
	if methodNode == nil {
		t.Fatal("前置：应命中 GET 方法节点")
	}

	pr := &RequestParamRouter{}
	got, err := pr.FindNode(methodNode, "page")
	if err != nil {
		t.Fatalf("FindNode 出错: %v", err)
	}
	if got == nil {
		t.Fatal("应命中 page 参数节点")
	}
	if got.GetKey() != "page" {
		t.Errorf("参数节点 key 应为 page，得 %s", got.GetKey())
	}
}

// TestRequestParamRouter_CaseInsensitive 参数名大小写不敏感（存储小写，查大写命中）。
func TestRequestParamRouter_CaseInsensitive(t *testing.T) {
	r := NewReverseRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/list?page=1", nil, "GET", nil))

	methodNode := r.locateMethodNode([]*request.HttpRequestPath{
		request.NewHttpRequestPath("api"),
		request.NewHttpRequestPath("list"),
	}, "GET")

	pr := &RequestParamRouter{}
	// 存储为 page，查 PAGE 应命中（内部 ToLower）
	got, _ := pr.FindNode(methodNode, "PAGE")
	if got == nil {
		t.Error("大小写不敏感：查 PAGE 应命中 page 参数节点")
	}
}

// TestRequestParamRouter_MissReturnsNil 不存在的参数名返回 nil。
func TestRequestParamRouter_MissReturnsNil(t *testing.T) {
	r := NewReverseRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/list?page=1", nil, "GET", nil))

	methodNode := r.locateMethodNode([]*request.HttpRequestPath{
		request.NewHttpRequestPath("api"),
		request.NewHttpRequestPath("list"),
	}, "GET")

	pr := &RequestParamRouter{}
	got, _ := pr.FindNode(methodNode, "missing")
	if got != nil {
		t.Errorf("不存在的参数应返回 nil，得 %v", got)
	}
}

// TestRequestParamRouter_EmptyNameReturnsNil 空参数名返回 nil。
func TestRequestParamRouter_EmptyNameReturnsNil(t *testing.T) {
	r := NewReverseRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/list?page=1", nil, "GET", nil))

	methodNode := r.locateMethodNode([]*request.HttpRequestPath{
		request.NewHttpRequestPath("api"),
		request.NewHttpRequestPath("list"),
	}, "GET")

	pr := &RequestParamRouter{}
	got, _ := pr.FindNode(methodNode, "")
	if got != nil {
		t.Errorf("空参数名应返回 nil，得 %v", got)
	}
}

// TestRequestParamRouter_NilStartReturnsNil nil 起点返回 nil。
func TestRequestParamRouter_NilStartReturnsNil(t *testing.T) {
	pr := &RequestParamRouter{}
	got, _ := pr.FindNode(nil, "page")
	if got != nil {
		t.Errorf("nil 起点应返回 nil，得 %v", got)
	}
}

// === RequestContentTypeRouter.FindNode ===

// TestRequestContentTypeRouter_Hit 命中已存在的 Content-Type 子节点。
func TestRequestContentTypeRouter_Hit(t *testing.T) {
	r := NewReverseRouter()
	h := request.Headers{}
	h.Set("Content-Type", "application/json")
	r.ReverseHttpRequest(request.NewHttpRequest("/api/users", h, "POST", []byte(`{"a":1}`)))

	methodNode := r.locateMethodNode([]*request.HttpRequestPath{
		request.NewHttpRequestPath("api"),
		request.NewHttpRequestPath("users"),
	}, "POST")
	if methodNode == nil {
		t.Fatal("前置：应命中 POST 方法节点")
	}

	cr := &RequestContentTypeRouter{}
	got, err := cr.FindNode(methodNode, "application/json")
	if err != nil {
		t.Fatalf("FindNode 出错: %v", err)
	}
	if got == nil {
		t.Fatal("应命中 application/json CT 节点")
	}
	if got.GetKey() != "application/json" {
		t.Errorf("CT 节点 key 应为 application/json，得 %s", got.GetKey())
	}
}

// TestRequestContentTypeRouter_MissReturnsNil 不存在的 CT 返回 nil。
func TestRequestContentTypeRouter_MissReturnsNil(t *testing.T) {
	r := NewReverseRouter()
	h := request.Headers{}
	h.Set("Content-Type", "application/json")
	r.ReverseHttpRequest(request.NewHttpRequest("/api/users", h, "POST", []byte(`{"a":1}`)))

	methodNode := r.locateMethodNode([]*request.HttpRequestPath{
		request.NewHttpRequestPath("api"),
		request.NewHttpRequestPath("users"),
	}, "POST")

	cr := &RequestContentTypeRouter{}
	got, _ := cr.FindNode(methodNode, "text/xml")
	if got != nil {
		t.Errorf("不存在的 CT 应返回 nil，得 %v", got)
	}
}

// TestRequestContentTypeRouter_EmptyCTReturnsNil 空 Content-Type 返回 nil。
func TestRequestContentTypeRouter_EmptyCTReturnsNil(t *testing.T) {
	r := NewReverseRouter()
	h := request.Headers{}
	h.Set("Content-Type", "application/json")
	r.ReverseHttpRequest(request.NewHttpRequest("/api/users", h, "POST", []byte(`{"a":1}`)))

	methodNode := r.locateMethodNode([]*request.HttpRequestPath{
		request.NewHttpRequestPath("api"),
		request.NewHttpRequestPath("users"),
	}, "POST")

	cr := &RequestContentTypeRouter{}
	got, _ := cr.FindNode(methodNode, "")
	if got != nil {
		t.Errorf("空 CT 应返回 nil，得 %v", got)
	}
}

// TestRequestContentTypeRouter_NilStartReturnsNil nil 起点返回 nil。
func TestRequestContentTypeRouter_NilStartReturnsNil(t *testing.T) {
	cr := &RequestContentTypeRouter{}
	got, _ := cr.FindNode(nil, "application/json")
	if got != nil {
		t.Errorf("nil 起点应返回 nil，得 %v", got)
	}
}
