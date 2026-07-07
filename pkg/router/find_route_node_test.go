package router

import (
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
)

// TestFindRouteNode_HitsMethodNode 验证已构建树中查找请求命中方法节点。
func TestFindRouteNode_HitsMethodNode(t *testing.T) {
	r := NewReverseRouter()
	// 构建树：/api/users/{id} GET
	for _, u := range []string{"/api/users/1", "/api/users/2", "/api/users/3"} {
		r.ReverseHttpRequest(request.NewHttpRequest(u, nil, "GET", nil))
	}

	// 查 /api/users/999（999 未出现过但应命中变量节点的方法节点）
	req := request.NewHttpRequest("/api/users/999", nil, "GET", nil)
	methodNode, ctNode, err := r.FindRouteNode(req)
	if err != nil {
		t.Fatalf("FindRouteNode 出错: %v", err)
	}
	if methodNode == nil {
		t.Fatal("应命中 users 下的 GET 方法节点")
	}
	if methodNode.GetKey() != "GET" {
		t.Errorf("方法节点 key 应为 GET，得 %s", methodNode.GetKey())
	}
	if ctNode != nil {
		t.Errorf("GET 请求不应有 Content-Type 子节点，得 %v", ctNode)
	}
}

// TestFindRouteNode_ReturnsContentTypeNode POST+JSON 应返回 ContentType 子节点。
func TestFindRouteNode_ReturnsContentTypeNode(t *testing.T) {
	r := NewReverseRouter()
	h := request.Headers{}
	h.Set("Content-Type", "application/json")
	r.ReverseHttpRequest(request.NewHttpRequest("/api/users", h, "POST", []byte(`{"a":1}`)))

	req := request.NewHttpRequest("/api/users", h, "POST", []byte(`{"b":2}`))
	methodNode, ctNode, err := r.FindRouteNode(req)
	if err != nil {
		t.Fatalf("出错: %v", err)
	}
	if methodNode == nil || methodNode.GetKey() != "POST" {
		t.Fatalf("应命中 POST 方法节点，得 %v", methodNode)
	}
	if ctNode == nil {
		t.Fatal("POST+JSON 应命中 Content-Type 子节点")
	}
	if ctNode.GetKey() != "application/json" {
		t.Errorf("Content-Type 节点 key 应为 application/json，得 %s", ctNode.GetKey())
	}
}

// TestFindRouteNode_PathMissReturnsNil 路径未命中返回 nil 无错。
func TestFindRouteNode_PathMissReturnsNil(t *testing.T) {
	r := NewReverseRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/users", nil, "GET", nil))

	req := request.NewHttpRequest("/api/orders/1", nil, "GET", nil)
	methodNode, _, err := r.FindRouteNode(req)
	if err != nil {
		t.Fatalf("不应出错: %v", err)
	}
	if methodNode != nil {
		t.Errorf("路径未命中应返回 nil，得 %v", methodNode)
	}
}

// TestFindRouteNode_MethodMissReturnsNil 路径命中但方法未命中。
func TestFindRouteNode_MethodMissReturnsNil(t *testing.T) {
	r := NewReverseRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/users", nil, "GET", nil))

	req := request.NewHttpRequest("/api/users", nil, "DELETE", nil)
	methodNode, _, err := r.FindRouteNode(req)
	if err != nil {
		t.Fatalf("不应出错: %v", err)
	}
	if methodNode != nil {
		t.Errorf("方法未命中应返回 nil，得 %v", methodNode)
	}
}

// TestFindRouteNode_NilRequestReturnsError nil 请求返回错误。
func TestFindRouteNode_NilRequestReturnsError(t *testing.T) {
	r := NewReverseRouter()
	_, _, err := r.FindRouteNode(nil)
	if err == nil {
		t.Fatal("nil 请求应返回错误")
	}
}

// TestFindRouteNode_BadURLReturnsError 非法 URL 返回错误。
func TestFindRouteNode_BadURLReturnsError(t *testing.T) {
	r := NewReverseRouter()
	_, _, err := r.FindRouteNode(request.NewHttpRequest("/api/%ZZ", nil, "GET", nil))
	if err == nil {
		t.Fatal("非法 URL 应返回错误")
	}
}

// TestFindRouteNode_HitsPathVariable 数字命中已合并的变量节点路径。
func TestFindRouteNode_HitsPathVariable(t *testing.T) {
	r := NewReverseRouter()
	for _, u := range []string{"/api/items/1", "/api/items/2", "/api/items/3"} {
		r.ReverseHttpRequest(request.NewHttpRequest(u, nil, "GET", nil))
	}
	// 变量节点已建立，查一个新数字应命中
	req := request.NewHttpRequest("/api/items/999", nil, "GET", nil)
	methodNode, _, err := r.FindRouteNode(req)
	if err != nil {
		t.Fatalf("出错: %v", err)
	}
	if methodNode == nil {
		t.Fatal("应命中变量节点下的 GET 方法节点")
	}
}
