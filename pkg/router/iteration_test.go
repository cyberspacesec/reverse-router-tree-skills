package router

import (
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
)

func TestHostAssetKeyAndNormalizeAssets(t *testing.T) {
	s := NewRouterSet()
	var reqs []*request.HttpRequest
	for _, host := range []string{"a.example.com", "b.example.com"} {
		for _, id := range []string{"1", "2", "3"} {
			r := request.NewHttpRequest("http://"+host+"/api/users/"+id, nil, "GET", nil)
			reqs = append(reqs, r)
			if err := s.ReverseHttpRequest(r); err != nil {
				t.Fatal(err)
			}
		}
	}
	assets := s.NormalizeAssets(reqs)
	if len(assets) != 2 {
		t.Fatalf("资产应按 Host 隔离，得到 %v", assets)
	}
	if len(assets["a.example.com GET /api/users/{users_id}"]) != 3 {
		t.Errorf("a host 资产错误: %v", assets)
	}
	if len(assets["b.example.com GET /api/users/{users_id}"]) != 3 {
		t.Errorf("b host 资产错误: %v", assets)
	}
}

func TestNormalizeURLDerivesHost(t *testing.T) {
	r := NewReverseRouter()
	for _, id := range []string{"1", "2", "3"} {
		if err := r.ReverseHttpRequest(request.NewHttpRequest("http://Example.COM/api/users/"+id, nil, "GET", nil)); err != nil {
			t.Fatal(err)
		}
	}
	n, ok := r.NormalizeURL(request.NewHttpRequest("http://Example.COM/api/users/4", nil, "GET", nil))
	if !ok || n.Host != "Example.COM" {
		t.Fatalf("应从 URL 推导 Host，结果=%+v, ok=%v", n, ok)
	}
}
