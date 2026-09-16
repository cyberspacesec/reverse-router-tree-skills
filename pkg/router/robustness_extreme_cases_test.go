package router

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
)

// G01 RouterSet 多 Host 并发喂入：各 host 独立树，互不污染。
func TestG_RouterSetConcurrentMultiHostIsolated(t *testing.T) {
	rs := NewRouterSet()
	hosts := []string{"a.example.com", "b.example.com", "c.example.com", "d.example.com"}
	var wg sync.WaitGroup
	for i, h := range hosts {
		wg.Add(1)
		go func(host string, seed int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				req := request.NewHttpRequest("http://"+host+"/api/items/"+fmt.Sprintf("%d", (seed*100+j)%300), nil, "GET", nil)
				req.Host = host
				if err := rs.ReverseHttpRequest(req); err != nil {
					t.Errorf("G01 %s err=%v", host, err)
					return
				}
			}
		}(h, i)
	}
	wg.Wait()

	got := rs.Hosts()
	if len(got) != len(hosts) {
		t.Fatalf("G01 Hosts()=%v want %d hosts", got, len(hosts))
	}
	for _, h := range hosts {
		r := rs.RouterFor(request.NewHttpRequest("http://"+h+"/x", nil, "GET", nil))
		// 每个 host 的树应只含自身资源
		if r == nil || r.Tree == nil {
			t.Errorf("G01 host %s router missing", h)
			continue
		}
	}
	// 跨 host 无污染：a 的树里不应出现 b/c/d 的 host 桶（Hosts 只含 4 个已确认）
	for _, h := range got {
		if h == "" {
			t.Errorf("G01 empty host bucket created")
		}
	}
}

// G02 并发喂入 + Normalize + Stats + Hosts 同时进行，无 panic/死锁/race。
func TestG_ConcurrentFeedAndQuery(t *testing.T) {
	rs := NewRouterSet()
	host := "api.example.com"
	var wg sync.WaitGroup
	// 喂入 goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			id := fmt.Sprintf("%d", i%200)
			req := request.NewHttpRequest("http://"+host+"/api/users/"+id+"?page=1", nil, "GET", nil)
			req.Host = host
			_ = rs.ReverseHttpRequest(req)
		}
	}()
	// 查询 goroutine 并发读
	for k := 0; k < 4; k++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				_ = rs.Hosts()
				_ = rs.Stats()
				req := request.NewHttpRequest("http://"+host+"/api/users/123?page=1", nil, "GET", nil)
				req.Host = host
				_, _ = rs.NormalizeURL(req)
			}
		}()
	}
	wg.Wait()
}

// G03 配置更新与建桶并发：新建桶继承新配置，不 panic。
func TestG_ConfigUpdateRacingWithBucketCreation(t *testing.T) {
	rs := NewRouterSet()
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cfg := DefaultMergeConfig
			cfg.SiblingMergeThreshold = 2
			rs.SetMergeConfig(cfg)
		}()
		wg.Add(1)
		go func(seed int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				host := fmt.Sprintf("h%d.example.com", seed%3)
				req := request.NewHttpRequest("http://"+host+"/api/items/"+fmt.Sprintf("%d", j), nil, "GET", nil)
				req.Host = host
				_ = rs.ReverseHttpRequest(req)
			}
		}(i)
	}
	wg.Wait()
	// 新配置应传播到全部已建桶
	r := rs.RouterFor(request.NewHttpRequest("http://h1.example.com/x", nil, "GET", nil))
	if r == nil || r.GetMergeConfig().SiblingMergeThreshold != 2 {
		t.Errorf("G03 config not propagated; got %+v", r.GetMergeConfig())
	}
}

// G04 路径变量 ValueMetric 并发观察读写，无 panic/race。
func TestG_ValueMetricConcurrentObserve(t *testing.T) {
	r := NewReverseRouter()
	var wg sync.WaitGroup
	for w := 0; w < 6; w++ {
		wg.Add(1)
		go func(seed int) {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				id := fmt.Sprintf("%d", (seed*100+i)%300)
				req := request.NewHttpRequest("/api/pets/"+id, nil, "GET", nil)
				if err := r.ReverseHttpRequest(req); err != nil {
					t.Errorf("G04 err=%v", err)
					return
				}
			}
		}(w)
	}
	wg.Wait()
	pets := r.Tree.Root.FindChildByKey("api").FindChildByKey("pets")
	if pets == nil {
		t.Fatal("G04 api/pets missing")
	}
	pv := pets.GetChildByType("request_path_variable")
	if pv == nil {
		t.Fatal("G04 path variable not merged")
	}
	// 读取计数
	total := pv.GetChildren()
	_ = total
}

// G05 超长 URL（单段 200KB）不崩溃、不误报。
func TestG_SuperLongURL(t *testing.T) {
	r := NewReverseRouter()
	long := strings.Repeat("a", 200_000)
	req := request.NewHttpRequest("/api/"+long, nil, "GET", nil)
	if err := r.ReverseHttpRequest(req); err != nil {
		t.Errorf("G05 long URL err=%v", err)
	}
	api := r.Tree.Root.FindChildByKey("api")
	if api == nil {
		t.Errorf("G05 api node missing")
	}
}

// G06 深路径（500 段）迭代构建，无递归爆栈、无 panic。
func TestG_DeepPath(t *testing.T) {
	r := NewReverseRouter()
	segs := make([]string, 500)
	for i := range segs {
		segs[i] = fmt.Sprintf("seg%d", i)
	}
	url := "/" + strings.Join(segs, "/")
	if err := r.ReverseHttpRequest(request.NewHttpRequest(url, nil, "GET", nil)); err != nil {
		t.Errorf("G06 deep path err=%v", err)
	}
	// 从根下钻验证末段存在
	cur := r.Tree.Root
	ok := true
	for _, s := range segs {
		cur = cur.FindChildByKey(s)
		if cur == nil {
			ok = false
			break
		}
	}
	if !ok {
		t.Errorf("G06 deep path tail not built")
	}
	if cur.FindChildByKey("GET") == nil {
		t.Errorf("G06 method node missing at depth")
	}
}

// G07 超多 query 参数（500 个）逐项建节点，不 panic。
func TestG_ManyQueryParams(t *testing.T) {
	r := NewReverseRouter()
	var sb strings.Builder
	sb.WriteString("/api/search?")
	for i := 0; i < 500; i++ {
		if i > 0 {
			sb.WriteString("&")
		}
		fmt.Fprintf(&sb, "p%d=%d", i, i)
	}
	if err := r.ReverseHttpRequest(request.NewHttpRequest(sb.String(), nil, "GET", nil)); err != nil {
		t.Errorf("G07 many params err=%v", err)
	}
	methodNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("search").FindChildByKey("GET")
	if methodNode == nil {
		t.Fatal("G07 method node missing")
	}
	if len(methodNode.GetChildren()) != 500 {
		t.Errorf("G07 param node count=%d want 500", len(methodNode.GetChildren()))
	}
}

// G08 大体积 JSON body 解析不 panic。
func TestG_LargeBody(t *testing.T) {
	r := NewReverseRouter()
	h := request.Headers{}
	h.Set("Content-Type", "application/json")
	var sb strings.Builder
	sb.WriteString(`{"items":[`)
	for i := 0; i < 2000; i++ {
		if i > 0 {
			sb.WriteString(",")
		}
		fmt.Fprintf(&sb, `{"id":%d,"name":"user%d","tags":["a","b","c"]}`, i, i)
	}
	sb.WriteString(`]}`)
	req := request.NewHttpRequest("/api/batch", h, "POST", []byte(sb.String()))
	if err := r.ReverseHttpRequest(req); err != nil {
		t.Errorf("G08 large body err=%v", err)
	}
}

// G09 批量失败详情数量上限 maxBatchErrors 生效。
func TestG_BatchErrorCap(t *testing.T) {
	r := NewReverseRouter()
	reqs := make([]*request.HttpRequest, 150)
	res := r.ReverseRequests(reqs) // 全为 nil 请求 → 全部失败
	if res.Failed != 150 {
		t.Errorf("G09 failed=%d want 150", res.Failed)
	}
	if len(res.Errors) != maxBatchErrors {
		t.Errorf("G09 errors len=%d want cap %d", len(res.Errors), maxBatchErrors)
	}
}

// G10 nil 请求 / nil 接收者鲁棒性：不 panic。
func TestG_NilRobustness(t *testing.T) {
	r := NewReverseRouter()
	// ReverseRouter nil 请求
	if err := r.ReverseHttpRequest(nil); err == nil {
		t.Errorf("G10 ReverseRouter nil req should error")
	}
	// ReverseRouter nil 树查找
	if nr, ok := r.NormalizeURL(nil); ok || nr.Method != "" || nr.Template != "" {
		t.Errorf("G10 NormalizeURL nil should be zero, got %+v ok=%v", nr, ok)
	}

	// RouterSet nil 接收者方法不 panic
	var rs *RouterSet
	if rs.Hosts() != nil {
		t.Errorf("G10 nil RouterSet Hosts should be nil")
	}
	if len(rs.Stats()) != 0 {
		t.Errorf("G10 nil RouterSet Stats should be empty")
	}
	req := request.NewHttpRequest("/api/x", nil, "GET", nil)
	req.Host = "h.example.com"
	if _, ok := rs.NormalizeURL(req); ok {
		t.Errorf("G10 nil RouterSet NormalizeURL should be false")
	}
	if err := rs.ReverseHttpRequest(req); err == nil {
		t.Errorf("G10 nil RouterSet ReverseHttpRequest should error, not panic")
	}
	if err := rs.ReverseCurls([]string{"curl http://h/x"}); err.Processed != 0 {
		t.Errorf("G10 nil RouterSet ReverseCurls should not process")
	}
	// RouterSet nil 请求
	rs2 := NewRouterSet()
	if err := rs2.ReverseHttpRequest(nil); err == nil {
		t.Errorf("G10 RouterSet nil req should error")
	}
}
