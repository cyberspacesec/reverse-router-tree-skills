package router

import (
	"strings"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
)

func TestHostCanonicalBucket(t *testing.T) {
	rs := NewRouterSet()
	urls := []string{
		"http://Example.COM/api/users/1",
		"http://example.com:80/api/users/2",
		"http://user:pass@example.com/api/users/3",
	}
	for _, u := range urls {
		if err := rs.ReverseHttpRequest(request.NewHttpRequest(u, nil, "GET", nil)); err != nil {
			t.Fatal(err)
		}
	}
	hosts := rs.Hosts()
	if len(hosts) != 1 || hosts[0] != "example.com" {
		t.Fatalf("应落在同一个桶，Hosts=%v", hosts)
	}
	route, ok := rs.NormalizeURLString("https://example.com:443/api/users/9", "GET")
	if !ok {
		t.Fatal("默认端口的 https 应命中同一资产")
	}
	if route.Template != "/api/users/{users_id}" {
		t.Fatalf("模板=%s", route.Template)
	}
	if route.HostAssetKey() != "example.com GET /api/users/{users_id}" {
		t.Fatalf("资产键=%s", route.HostAssetKey())
	}
}

func TestQuerySignatureAndNoise(t *testing.T) {
	r := NewReverseRouter()
	samples := []string{
		"/api/list?page=1&size=10&utm_source=ad&ids[]=1&filter[status]=open",
		"/api/list?page=2&size=20&fbclid=xyz&ids[]=2&filter[status]=closed",
	}
	for _, u := range samples {
		if err := r.ReverseHttpRequest(request.NewHttpRequest(u, nil, "GET", nil)); err != nil {
			t.Fatal(err)
		}
	}
	assets := r.ListAssets()
	if len(assets) != 1 {
		t.Fatalf("应只有一条资产，got %d", len(assets))
	}
	a := assets[0]
	if a.AssetKey() != "GET /api/list" {
		t.Fatalf("AssetKey=%s", a.AssetKey())
	}
	sig := a.SignatureKey()
	for _, name := range []string{"page", "size", "ids", "filter.status"} {
		if !strings.Contains(sig, name) {
			t.Errorf("签名 %s 缺少 %s", sig, name)
		}
	}
	if strings.Contains(sig, "utm") || strings.Contains(sig, "fbclid") {
		t.Errorf("签名不该含噪声参数: %s", sig)
	}
	if a.Hits != 2 {
		t.Errorf("Hits=%d want 2", a.Hits)
	}
	if a.SampleURL == "" {
		t.Error("应保留一条样例 URL")
	}
}

func TestVersionSegmentsStayFixedUnlessEnabled(t *testing.T) {
	feed := func(r *ReverseRouter) {
		for _, v := range []string{"v1", "v2", "v3"} {
			if err := r.ReverseHttpRequest(request.NewHttpRequest("/api/"+v+"/users", nil, "GET", nil)); err != nil {
				t.Fatal(err)
			}
		}
	}
	off := NewReverseRouter()
	feed(off)
	assets := off.ListAssets()
	if len(assets) != 3 {
		t.Fatalf("默认应是 3 条版本资产，got %d", len(assets))
	}

	on := NewReverseRouter()
	cfg := DefaultMergeConfig
	cfg.MergeVersionSegments = true
	on.SetMergeConfig(cfg)
	feed(on)
	merged := on.ListAssets()
	if len(merged) != 1 || merged[0].Template != "/api/{api_version}/users" {
		t.Fatalf("打开开关后应合并为版本变量，got %+v", merged)
	}
}

func TestRouterSetSnapshotRoundTrip(t *testing.T) {
	rs := NewRouterSet()
	for _, u := range []string{
		"http://a.com/api/users/1",
		"http://a.com/api/users/2",
		"http://a.com/api/users/3",
		"http://b.com/api/orders/10",
	} {
		if err := rs.ReverseHttpRequest(request.NewHttpRequest(u, nil, "GET", nil)); err != nil {
			t.Fatal(err)
		}
	}
	data, err := rs.ExportJSON()
	if err != nil {
		t.Fatal(err)
	}
	restored := NewRouterSet()
	if err := restored.ImportJSON(data); err != nil {
		t.Fatal(err)
	}
	route, ok := restored.NormalizeURLString("http://a.com/api/users/99", "GET")
	if !ok || route.Template != "/api/users/{users_id}" {
		t.Fatalf("导入后应能归一化，ok=%v route=%+v", ok, route)
	}
	assets := restored.ListAssets()["a.com"]
	if len(assets) != 1 || assets[0].Hits != 3 {
		t.Fatalf("命中数应随快照恢复，got %+v", assets)
	}
	if _, ok := restored.NormalizeURLString("http://b.com/api/orders/10", "GET"); !ok {
		t.Fatal("b.com 的树也应恢复")
	}

	bad := []byte(`{"version":99,"hosts":{}}`)
	if err := restored.ImportJSON(bad); err == nil {
		t.Fatal("更高版本应报错")
	}
	if _, ok := restored.NormalizeURLString("http://a.com/api/users/99", "GET"); !ok {
		t.Fatal("版本不兼容时不应改动已导入的数据")
	}
}

func TestProjectSnapshotRoundTrip(t *testing.T) {
	m := NewProjectManager()
	req := request.NewHttpRequest("http://a.com/api/x", nil, "GET", nil)
	if err := m.ReverseHttpRequest("proj-a", req); err != nil {
		t.Fatal(err)
	}
	data, err := m.ExportJSON()
	if err != nil {
		t.Fatal(err)
	}
	restored := NewProjectManager()
	if err := restored.ImportJSON(data); err != nil {
		t.Fatal(err)
	}
	if _, ok := restored.NormalizeURLString("proj-a", "http://a.com/api/x", "GET"); !ok {
		t.Fatal("项目快照导入后应能归一化")
	}
	if _, ok := restored.NormalizeURLString("proj-b", "http://a.com/api/x", "GET"); ok {
		t.Fatal("未导入的项目不应命中")
	}
}
