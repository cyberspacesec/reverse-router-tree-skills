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
