package router

import (
	"strconv"
	"sync"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
)

// TestReverseHttpRequest_ConcurrentSafe 验证 ReverseHttpRequest 可被多个 goroutine
// 并发调用而不触发数据竞争。
//
// 复现路径：多个 goroutine 命中同一已存在路径变量节点 + 同一 query/body 参数节点，
// 触发 findOrCreatePathNode / findOrCreateParamNode 的并发类型回填。
// 修复前 RequestPathVariableNode.SetType/SetLogicalType 与 RequestParamNode 的
// 类型字段无锁保护，-race 会报数据竞争。
func TestReverseHttpRequest_ConcurrentSafe(t *testing.T) {
	r := NewReverseRouter()
	var wg sync.WaitGroup
	const workers = 8
	const perWorker = 300

	h := request.Headers{}
	h.Set("Content-Type", "application/json")

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				// 轮换 ID 命中同一变量节点；带 query + body 命中同一参数节点
				id := strconv.Itoa((worker*perWorker + i) % 1000)
				req := request.NewHttpRequest(
					"/api/items/"+id+"?page=1&size=10",
					h, "PUT", []byte(`{"name":"x"}`))
				if err := r.ReverseHttpRequest(req); err != nil {
					t.Errorf("ReverseHttpRequest 失败: %v", err)
					return
				}
			}
		}(w)
	}
	wg.Wait()
	r.InferRequiredParams()

	// 并发喂入后树应一致：合并出变量节点
	items := r.Tree.Root.FindChildByKey("api").FindChildByKey("items")
	if items == nil {
		t.Fatal("并发后 api/items 丢失")
	}
	if items.GetChildByType("request_path_variable") == nil {
		t.Fatal("并发后未合并出路径变量节点")
	}
}

// TestConcurrentSafety_DistinctPaths 验证并发处理完全不同的路径时
// HttpRequestPath sync.Pool 与 paths slice 容器池无串扰、无数据竞争。
//
// 重点：paths 容器与 *HttpRequestPath 从池复用，多 goroutine 同时 Acquire/Release，
// 若归还时未清零字段或容器被并发写，会串读其他 goroutine 的数据。-race 检测。
func TestConcurrentSafety_DistinctPaths(t *testing.T) {
	r := NewReverseRouter()
	var wg sync.WaitGroup
	const workers = 8
	const perWorker = 100
	// 每个 worker 用独立资源前缀，确保路径不重叠，强制每次都新建路径节点
	prefixes := []string{"users", "orders", "items", "products", "customers", "invoices", "tickets", "sessions"}

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				prefix := prefixes[worker%len(prefixes)]
				id := strconv.Itoa(1000 + (worker*perWorker+i)%5000)
				req := request.NewHttpRequest("/api/"+prefix+"/"+id, nil, "GET", nil)
				if err := r.ReverseHttpRequest(req); err != nil {
					t.Errorf("ReverseHttpRequest 失败: %v", err)
					return
				}
			}
		}(w)
	}
	wg.Wait()

	// 并发喂入后树应一致：api 节点存在。
	// 注意：不同资源前缀（users/orders/...）在 api 下作为同类型兄弟 path 节点，
	// 达到合并阈值会被合并为单个路径变量节点（资源名变量化），这是正常的逆向结果，
	// 故不假设各 prefix 节点保留。重点验证并发无 race（-race 已覆盖）无错误。
	api := r.Tree.Root.FindChildByKey("api")
	if api == nil {
		t.Fatal("并发后 api 节点丢失")
	}
	// api 下应存在某种路径结构（固定 path 或合并出的 path_variable）
	if len(api.GetChildren()) == 0 {
		t.Fatal("并发后 api 下无子节点")
	}
}

// BenchmarkThroughput_Scalability 不同 GOMAXPROCS 下的并发吞吐曲线。
// 手动运行：go test -run=NONE -bench=Scalability -cpu=1,4,8,16,32 ./pkg/router/
// 验证 mergeMu 串行临界区与 Pool 争用对扩展性的影响。
func BenchmarkThroughput_Scalability(b *testing.B) {
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
