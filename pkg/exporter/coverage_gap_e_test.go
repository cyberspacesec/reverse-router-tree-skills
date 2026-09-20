package exporter

// coverage_gap_e_test.go — 补齐 pkg/exporter 剩余 6 个零覆盖分支：
// collectEndpoints nil 守卫、方法节点直接返回、param 在 contentType 之后
// 遍历的 body 重分类、超长 operationID 截断、cookie 同名去重、
// 有 contentType 时的 requestBody 挂载。
// 只新增测试，不改业务代码。

import (
	"strings"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/router"
)

// Nil 守卫分支：collectEndpoints(nil) 返回 nil，不 panic。
func TestGapE_CollectEndpointsNil(t *testing.T) {
	exp := NewOpenAPIExporter()
	if got := exp.collectEndpoints(nil, nil); got != nil {
		t.Errorf("nil 输入应返回 nil，实际 %v", got)
	}
}

// 方法节点直接返回分支：collectEndpoints 在方法节点处直接构造端点。
func TestGapE_CollectEndpointsMethodDirect(t *testing.T) {
	r := router.NewReverseRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/direct", nil, "POST", nil))
	exp := NewOpenAPIExporter()
	got := exp.collectEndpoints(r.Tree.Root, nil)
	if len(got) == 0 {
		t.Fatal("应收集到端点")
	}
	found := false
	for _, ep := range got {
		if ep.path == "/api/direct" && ep.method == "post" {
			found = true
		}
	}
	if !found {
		t.Errorf("应含 POST /api/direct，实际 %+v", got)
	}
}

// body 重分类分支：param 子节点排在 contentType 之前被遍历时，
// 先归入 query，之后因有 contentType 整体重分类为 body。
func TestGapE_BodyReclassify(t *testing.T) {
	r := router.NewReverseRouter()
	h := request.Headers{}
	h.Set("Content-Type", "application/json")
	// 先喂一个无 body 的请求创建 param，再喂带 contentType 的同路径请求
	r.ReverseHttpRequest(request.NewHttpRequest("/api/items?tag=go", nil, "GET", nil))
	r.ReverseHttpRequest(request.NewHttpRequest("/api/items?tag=web", h, "POST", []byte(`{"tag":"web"}`)))

	exp := NewOpenAPIExporter()
	data, err := exp.Export(r.Tree)
	if err != nil {
		t.Fatal(err)
	}
	doc := parseDoc(t, data)
	paths := getPaths(t, doc)
	item := getPathItem(t, paths, "/api/items")
	op := getOperation(t, item, "post")
	// POST 有 contentType：应有 requestBody，而 query 参数被搬空
	if _, ok := op["requestBody"]; !ok {
		t.Error("POST 应有 requestBody")
	}
}

// 超长 operationID 截断分支：超长路径的方法端点 ID 被截到 100 字符。
func TestGapE_OperationIDTruncated(t *testing.T) {
	r := router.NewReverseRouter()
	longSeg := strings.Repeat("a", 120)
	r.ReverseHttpRequest(request.NewHttpRequest("/api/"+longSeg, nil, "GET", nil))

	exp := NewOpenAPIExporter()
	data, err := exp.Export(r.Tree)
	if err != nil {
		t.Fatal(err)
	}
	doc := parseDoc(t, data)
	paths := getPaths(t, doc)
	for _, itemRaw := range paths {
		item := itemRaw.(map[string]interface{})
		for method, opRaw := range item {
			if !strings.HasPrefix(method, "get") {
				continue
			}
			op := opRaw.(map[string]interface{})
			id, _ := op["operationId"].(string)
			if len(id) > 100 {
				t.Errorf("operationId 长度=%d，应截断到 100", len(id))
			}
		}
	}
}

// cookie 同名去重分支：同一 cookie 名出现多次只输出一个参数。
func TestGapE_CookieDedup(t *testing.T) {
	r := router.NewReverseRouter()
	for _, v := range []string{"zh", "en", "fr"} {
		h := request.Headers{}
		h.Set("Cookie", "lang="+v)
		r.ReverseHttpRequest(request.NewHttpRequest("/api/i18n", h, "GET", nil))
	}
	exp := NewOpenAPIExporter()
	data, err := exp.Export(r.Tree)
	if err != nil {
		t.Fatal(err)
	}
	doc := parseDoc(t, data)
	paths := getPaths(t, doc)
	item := getPathItem(t, paths, "/api/i18n")
	op := getOperation(t, item, "get")
	params, _ := op["parameters"].([]interface{})
	count := 0
	for _, p := range params {
		pm := p.(map[string]interface{})
		if pm["in"] == "cookie" && pm["name"] == "lang" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("同名 cookie 应去重为 1 个，实际 %d", count)
	}
}

// requestBody 挂载分支：POST JSON 时 operation 应带 requestBody。
// 同时覆盖 default 分支（路径栈中的未知类型不影响收集）。
func TestGapE_RequestBodyAttached(t *testing.T) {
	r := router.NewReverseRouter()
	h := request.Headers{}
	h.Set("Content-Type", "application/json")
	r.ReverseHttpRequest(request.NewHttpRequest("/api/users", h, "POST", []byte(`{"name":"a"}`)))
	exp := NewOpenAPIExporter()
	// 直接调 collectEndpoints 覆盖 default 空分支（param 类型出现在路径栈时被忽略）
	if got := exp.collectEndpoints(r.Tree.Root, []pathSegment{}); len(got) == 0 {
		t.Error("应收集到端点")
	}
	data, err := exp.Export(r.Tree)
	if err != nil {
		t.Fatal(err)
	}
	doc := parseDoc(t, data)
	paths := getPaths(t, doc)
	item := getPathItem(t, paths, "/api/users")
	op := getOperation(t, item, "post")
	if _, ok := op["requestBody"]; !ok {
		t.Error("POST JSON 应挂载 requestBody")
	}
}

// 路径变量节点断言失败回退：非 *RequestPathVariableNode 但类型名为
// request_path_variable 时，varNode 为 nil 仍能拼接路径。
func TestGapE_PathVarTypeAssertFallback(t *testing.T) {
	exp := NewOpenAPIExporter()
	// 用普通 BaseNode 冒充路径变量类型名，触发 pv,_ 失败回退分支
	fake := node.NewBaseNode[node.NodeContext]("request_path_variable", "vid", "", node.NewBaseNodeContext())
	root := node.NewBaseNode[node.NodeContext]("root", "root", "", node.NewBaseNodeContext())
	if err := root.AddChild(fake); err != nil {
		t.Fatal(err)
	}
	got := exp.collectEndpoints(root, nil)
	// 冒充节点下无方法节点，应返回空（但分支已走过）
	if len(got) != 0 {
		t.Errorf("无方法节点应返回空，实际 %d", len(got))
	}
}

// param 在 contentType 已知时直接归 body：contentType 子节点排在 param
// 之前被遍历，走 267 分支而非重分类兜底。
func TestGapE_ParamAfterContentType(t *testing.T) {
	r := router.NewReverseRouter()
	h := request.Headers{}
	h.Set("Content-Type", "application/json")
	// 单个 POST JSON 请求：方法节点下 contentType 先遍历、param 后遍历
	r.ReverseHttpRequest(request.NewHttpRequest("/api/direct-body", h, "POST", []byte(`{"a":"1","b":"2"}`)))
	exp := NewOpenAPIExporter()
	data, err := exp.Export(r.Tree)
	if err != nil {
		t.Fatal(err)
	}
	doc := parseDoc(t, data)
	paths := getPaths(t, doc)
	item := getPathItem(t, paths, "/api/direct-body")
	op := getOperation(t, item, "post")
	if _, ok := op["requestBody"]; !ok {
		t.Error("POST JSON 应挂载 requestBody")
	}
	body, _ := op["requestBody"].(map[string]interface{})
	content, _ := body["content"].(map[string]interface{})
	ct, _ := content["application/json"].(map[string]interface{})
	schema, _ := ct["schema"].(map[string]interface{})
	props, _ := schema["properties"].(map[string]interface{})
	if len(props) < 2 {
		t.Errorf("body 参数应含 a/b，实际 %v", props)
	}
	if params, ok := op["parameters"].([]interface{}); ok {
		for _, p := range params {
			pm := p.(map[string]interface{})
			if pm["in"] == "query" {
				t.Errorf("body 参数不应出现在 query：%v", pm["name"])
			}
		}
	}
}

// 排序比较器 In 不同分支：同时有 query/header/cookie 参数时触发按 In 排序。
func TestGapE_SortByIn(t *testing.T) {
	r := router.NewReverseRouter()
	h := request.Headers{}
	h.Set("X-Trace", "1")
	h.Set("Cookie", "lang=zh")
	r.ReverseHttpRequest(request.NewHttpRequest("/api/multi-in?z=1&a=2", h, "GET", nil))
	exp := NewOpenAPIExporter()
	data, err := exp.Export(r.Tree)
	if err != nil {
		t.Fatal(err)
	}
	doc := parseDoc(t, data)
	paths := getPaths(t, doc)
	item := getPathItem(t, paths, "/api/multi-in")
	op := getOperation(t, item, "get")
	params, _ := op["parameters"].([]interface{})
	if len(params) < 3 {
		t.Fatalf("应同时有 query/header/cookie 参数，实际 %d", len(params))
	}
	// 验证已按 In 排序（cookie < header < query）
	last := ""
	for _, p := range params {
		pm := p.(map[string]interface{})
		cur, _ := pm["in"].(string)
		if cur < last {
			t.Errorf("参数未按 In 排序：%s 在 %s 之后", cur, last)
		}
		last = cur
	}
}

// 267 精准触发：手工构造方法节点，先挂 contentType 再挂 param，
// 使遍历 param 时 ep.contentType 已知，走直接归 body 分支。
func TestGapE_ParamDirectBody(t *testing.T) {
	exp := NewOpenAPIExporter()
	ctx := node.NewBaseNodeContext()
	method := node.NewBaseNode[node.NodeContext]("request_method", "post", "", ctx)
	ct := node.NewRequestContentTypeNode("application/json")
	p := node.NewRequestParamNode("a", "", false)
	// 先挂 contentType，再挂 param
	if err := method.AddChild(ct); err != nil {
		t.Fatal(err)
	}
	if err := method.AddChild(p); err != nil {
		t.Fatal(err)
	}
	got := exp.collectFromMethodNode(method, nil)
	if len(got) != 1 {
		t.Fatalf("应构造 1 个端点，实际 %d", len(got))
	}
	if len(got[0].bodyParams) != 1 || len(got[0].queryParams) != 0 {
		t.Errorf("param 应直接归 body：body=%d query=%d",
			len(got[0].bodyParams), len(got[0].queryParams))
	}
	// 267 走过后重分类兜底不应再搬移
	if got[0].contentType != "application/json" {
		t.Errorf("contentType=%q", got[0].contentType)
	}
}
