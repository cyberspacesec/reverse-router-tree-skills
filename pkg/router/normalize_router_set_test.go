package router

import (
	"fmt"
	"sync"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
)

func mkReq(url string) *request.HttpRequest {
	return request.NewHttpRequest(url, nil, "GET", nil)
}

// NormalizeURL：整数变量、参数、方法、未命中、host 透传
func TestNormalizeURL(t *testing.T) {
	r := NewReverseRouter()
	r.ReverseHttpRequest(mkReq("/api/users/123"))
	r.ReverseHttpRequest(mkReq("/api/users/456"))
	r.ReverseHttpRequest(mkReq("/api/users/789"))
	r.ReverseHttpRequest(request.NewHttpRequest("/api/users?page=1&size=20", nil, "GET", nil))

	n, ok := r.NormalizeURL(mkReq("/api/users/42"))
	if !ok {
		t.Fatal("已知路由归一化应成功")
	}
	if n.Method != "GET" {
		t.Errorf("Method = %q", n.Method)
	}
	if n.Template != "/api/users/{users_id}" {
		t.Errorf("Template = %q, want /api/users/{users_id}", n.Template)
	}
	if len(n.PathParams) != 1 || n.PathParams[0] != "users_id" {
		t.Errorf("PathParams = %v", n.PathParams)
	}
	if n.AssetKey() != "GET /api/users/{users_id}" {
		t.Errorf("AssetKey = %q", n.AssetKey())
	}

	// 查询参数归一化
	n2, ok := r.NormalizeURL(request.NewHttpRequest("/api/users?page=3", nil, "GET", nil))
	if !ok {
		t.Fatal("查询参数归一化应成功")
	}
	if len(n2.QueryParams) != 2 {
		t.Errorf("QueryParams = %v, want [page size]", n2.QueryParams)
	}
}

func TestNormalizeURL_UnknownRoute(t *testing.T) {
	r := NewReverseRouter()
	r.ReverseHttpRequest(mkReq("/api/users/123"))
	if _, ok := r.NormalizeURL(mkReq("/api/unknown/42")); ok {
		t.Error("未命中路由应返回 false")
	}
}

func TestNormalizeURL_HostPassthrough(t *testing.T) {
	r := NewReverseRouter()
	for _, id := range []string{"123", "456", "789"} {
		r.ReverseHttpRequest(mkReq("/api/users/" + id))
	}
	req := mkReq("/api/users/123")
	req.SetHost("example.com")
	n, ok := r.NormalizeURL(mkReq("/api/users/1"))
	if !ok {
		t.Fatal("归一化应成功")
	}
	if n.Host != "" {
		t.Errorf("单目标 ReverseRouter 不应注入 host，实际 %q", n.Host)
	}
}

func TestNormalizeURLs_Bucket(t *testing.T) {
	r := NewReverseRouter()
	reqs := []*request.HttpRequest{
		mkReq("/api/users/1"), mkReq("/api/users/2"), mkReq("/api/users/3"),
		mkReq("/api/orders/10"),
	}
	for _, q := range reqs {
		r.ReverseHttpRequest(q)
	}
	buckets := r.NormalizeURLs(reqs)
	if len(buckets["GET /api/users/{users_id}"]) != 3 {
		t.Errorf("users 桶应有 2 条，实际 %v", buckets["GET /api/users/{users_id}"])
	}
}

// RouterSet：两 host 同路径不互相污染
func TestRouterSet_HostIsolation(t *testing.T) {
	s := NewRouterSet()
	for i := 1; i <= 3; i++ {
		s.ReverseHttpRequest(request.NewHttpRequest(fmt.Sprintf("http://a.example.com/api/users/%d", i), nil, "GET", nil))
	}
	for i := 1; i <= 3; i++ {
		s.ReverseHttpRequest(request.NewHttpRequest(fmt.Sprintf("http://b.example.com/api/users/%d", i), nil, "GET", nil))
	}
	hosts := s.Hosts()
	if len(hosts) != 2 {
		t.Fatalf("Hosts = %v, want 2", hosts)
	}

	// a 的树已合并 users 变量
	ra := s.RouterFor(mkReq("http://a.example.com/x"))
	if ra.Tree.Root.FindChildByKey("api").FindChildByKey("users").GetChildByType("request_path_variable") == nil {
		t.Error("host a 的 users 应合并为变量")
	}
	rb := s.RouterFor(mkReq("http://b.example.com/x"))
	if rb.Tree.Root.FindChildByKey("api").FindChildByKey("users").GetChildByType("request_path_variable") == nil {
		t.Error("host b 的 users 应合并为变量")
	}
}

func TestRouterSet_NormalizeURL(t *testing.T) {
	s := NewRouterSet()
	for i := 1; i <= 3; i++ {
		s.ReverseHttpRequest(request.NewHttpRequest(fmt.Sprintf("http://a.example.com/api/users/%d", i), nil, "GET", nil))
	}
	n, ok := s.NormalizeURL(request.NewHttpRequest("http://a.example.com/api/users/7", nil, "GET", nil))
	if !ok {
		t.Fatal("归一化应成功")
	}
	if n.Host != "a.example.com" {
		t.Errorf("Host = %q, want a.example.com", n.Host)
	}
	if n.AssetKey() != "GET /api/users/{users_id}" {
		t.Errorf("AssetKey = %q", n.AssetKey())
	}
}

func TestRouterSet_Concurrent(t *testing.T) {
	s := NewRouterSet()
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(seed int) {
			defer wg.Done()
			for j := 1; j <= 30; j++ {
				host := fmt.Sprintf("h%d.example.com", seed%3)
				s.ReverseHttpRequest(request.NewHttpRequest(
					fmt.Sprintf("http://%s/api/users/%d", host, j), nil, "GET", nil))
			}
		}(i)
	}
	wg.Wait()
	if len(s.Hosts()) != 3 {
		t.Errorf("应建 3 个 host 桶，实际 %d", len(s.Hosts()))
	}
	if len(s.Stats()) != 3 {
		t.Errorf("Stats 应有 3 项，实际 %d", len(s.Stats()))
	}
}

func TestRouterSet_ReverseCurls(t *testing.T) {
	s := NewRouterSet()
	curls := []string{
		`curl 'http://a.example.com/api/users/1'`,
		`curl 'http://a.example.com/api/users/2'`,
		`curl 'http://a.example.com/api/users/3'`,
		"",
	}
	res := s.ReverseCurls(curls)
	if res.Processed != 3 {
		t.Errorf("Processed = %d, want 3", res.Processed)
	}
	if res.Failed != 1 {
		t.Errorf("Failed = %d, want 1", res.Failed)
	}
}
