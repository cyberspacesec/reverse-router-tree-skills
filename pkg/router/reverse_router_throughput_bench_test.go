package router

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
)

// BenchmarkThroughput_PurePath 纯路径不合并场景的单协程吞吐（最理想情况）。
// 每 100 个不同 ID 循环，前 3 个触发合并后其余命中已存在变量节点走增量缓存。
func BenchmarkThroughput_PurePath(b *testing.B) {
	r := NewReverseRouter()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r.ReverseHttpRequest(request.NewHttpRequest(fmt.Sprintf("/api/res/%d", i%100), nil, "GET", nil))
	}
}

// BenchmarkThroughput_LargeMerge 大量 ID 合并场景（验证增量缓存消除 O(N²)）。
// 10000 个不同 ID，每次命中已存在变量节点——优化前每命中全量重算 1000+ 值，
// 优化后仅 uniqueCount 变化（每新值）才重算。
func BenchmarkThroughput_LargeMerge(b *testing.B) {
	r := NewReverseRouter()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		id := strconv.Itoa(1000 + i%10000)
		r.ReverseHttpRequest(request.NewHttpRequest("/api/resources/"+id, nil, "GET", nil))
	}
}

// BenchmarkThroughput_Concurrent 多协程并发吞吐（验证并行扩展性）。
// 受 mergeMu 串行合并临界区限制，纯路径命中阶段可并行，合并阶段串行。
func BenchmarkThroughput_Concurrent(b *testing.B) {
	r := NewReverseRouter()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			id := strconv.Itoa(1000 + i%10000)
			r.ReverseHttpRequest(request.NewHttpRequest("/api/resources/"+id, nil, "GET", nil))
			i++
		}
	})
}

// BenchmarkThroughput_WithBody 带 JSON body 的 POST 吞吐。
// 验证 body 参数解析 + 类型推断的整体开销。
func BenchmarkThroughput_WithBody(b *testing.B) {
	r := NewReverseRouter()
	h := request.Headers{"Content-Type": "application/json"}
	body := []byte(`{"name":"bob","age":25,"page":1}`)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r.ReverseHttpRequest(request.NewHttpRequest(fmt.Sprintf("/api/users/%d", i%100), h, "POST", body))
	}
}

// BenchmarkCurlParse 验证 curl 命令解析吞吐（网络测绘场景核心输入路径）。
func BenchmarkCurlParse(b *testing.B) {
	curl := `curl 'http://api.example.com/users/123' -X POST -H 'Authorization: Bearer token123' -H 'Content-Type: application/json' -d '{"name":"bob","age":25}'`
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		request.ParseCurl(curl)
	}
}

// BenchmarkCurlE2E curl 解析 + 路由还原全链路吞吐（真实产品路径）。
func BenchmarkCurlE2E(b *testing.B) {
	r := NewReverseRouter()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		curl := fmt.Sprintf(`curl 'http://api.example.com/users/%d' -H 'Authorization: Bearer token%d'`, i%100, i%100)
		req, _ := request.ParseCurl(curl)
		r.ReverseHttpRequest(req)
	}
}
