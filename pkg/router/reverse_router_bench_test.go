package router

import (
	"strconv"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
)

// BenchmarkReverseHttpRequest_Merge 单请求逆向+合并吞吐
func BenchmarkReverseHttpRequest_Merge(b *testing.B) {
	r := NewReverseRouter()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		id := strconv.Itoa(1000 + i%1000)
		r.ReverseHttpRequest(request.NewHttpRequest("/api/resources/"+id, nil, "GET", nil))
	}
}

// BenchmarkReverseHttpRequest_VariedResources 多资源多 ID 场景
func BenchmarkReverseHttpRequest_VariedResources(b *testing.B) {
	resources := []string{"users", "orders", "items", "products", "customers"}
	r := NewReverseRouter()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		res := resources[i%len(resources)]
		id := strconv.Itoa(1000 + i%1000)
		r.ReverseHttpRequest(request.NewHttpRequest("/api/"+res+"/"+id, nil, "GET", nil))
	}
}
