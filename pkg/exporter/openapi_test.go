package exporter

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/router"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/tree"
)

// parseDoc 解析导出的 JSON 为 map，便于断言
func parseDoc(t *testing.T, data []byte) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("导出的JSON非法: %v\n%s", err, string(data))
	}
	return m
}

func getPaths(t *testing.T, doc map[string]interface{}) map[string]interface{} {
	t.Helper()
	paths, ok := doc["paths"].(map[string]interface{})
	if !ok {
		t.Fatalf("缺少 paths 字段或类型错误")
	}
	return paths
}

func getPathItem(t *testing.T, paths map[string]interface{}, path string) map[string]interface{} {
	t.Helper()
	item, ok := paths[path].(map[string]interface{})
	if !ok {
		t.Fatalf("路径 %s 不存在", path)
	}
	return item
}

func getOperation(t *testing.T, item map[string]interface{}, method string) map[string]interface{} {
	t.Helper()
	op, ok := item[method].(map[string]interface{})
	if !ok {
		t.Fatalf("路径下缺少方法 %s", method)
	}
	return op
}

// === 基础结构测试 ===

func TestOpenAPIExport_BasicStructure(t *testing.T) {
	r := router.NewReverseRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/users", nil, "GET", nil))

	exp := NewOpenAPIExporter()
	data, err := exp.Export(r.Tree)
	if err != nil {
		t.Fatal(err)
	}

	doc := parseDoc(t, data)
	if doc["openapi"] != "3.0.3" {
		t.Errorf("openapi 版本应为 3.0.3，实际 %v", doc["openapi"])
	}
	info, _ := doc["info"].(map[string]interface{})
	if info["title"] != "Reverse Engineered API" {
		t.Errorf("标题不符: %v", info["title"])
	}
	paths := getPaths(t, doc)
	if _, ok := paths["/api/users"]; !ok {
		t.Errorf("应包含路径 /api/users，实际 %+v", paths)
	}
}

func TestOpenAPIExport_ServerURL(t *testing.T) {
	r := router.NewReverseRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/x", nil, "GET", nil))

	exp := NewOpenAPIExporter()
	exp.ServerURL = "https://api.example.com"
	data, _ := exp.Export(r.Tree)
	doc := parseDoc(t, data)

	servers, ok := doc["servers"].([]interface{})
	if !ok || len(servers) != 1 {
		t.Fatalf("应输出1个server，实际 %v", doc["servers"])
	}
	srv := servers[0].(map[string]interface{})
	if srv["url"] != "https://api.example.com" {
		t.Errorf("server URL 不符: %v", srv["url"])
	}
}

func TestOpenAPIExport_NoServerByDefault(t *testing.T) {
	r := router.NewReverseRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/x", nil, "GET", nil))

	exp := NewOpenAPIExporter()
	data, _ := exp.Export(r.Tree)
	doc := parseDoc(t, data)

	if _, ok := doc["servers"]; ok {
		t.Error("未设置 ServerURL 时不应输出 servers 字段")
	}
}

// === 路径变量测试 ===

func TestOpenAPIExport_PathVariable(t *testing.T) {
	r := router.NewReverseRouter()
	for _, id := range []string{"101", "102", "103"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/users/"+id, nil, "GET", nil))
	}

	exp := NewOpenAPIExporter()
	data, _ := exp.Export(r.Tree)
	doc := parseDoc(t, data)
	paths := getPaths(t, doc)

	item := getPathItem(t, paths, "/api/users/{users_id}")
	op := getOperation(t, item, "get")

	params, _ := op["parameters"].([]interface{})
	// 应有1个 path 参数
	var pathParam map[string]interface{}
	for _, p := range params {
		pm := p.(map[string]interface{})
		if pm["in"] == "path" {
			pathParam = pm
		}
	}
	if pathParam == nil {
		t.Fatal("应有 path 类型的参数")
	}
	if pathParam["name"] != "users_id" {
		t.Errorf("path 参数名应为 users_id，实际 %v", pathParam["name"])
	}
	if pathParam["required"] != true {
		t.Errorf("path 参数应为 required=true")
	}
	schema := pathParam["schema"].(map[string]interface{})
	if schema["type"] != "integer" {
		t.Errorf("path 参数类型应为 integer，实际 %v", schema["type"])
	}
	// 路径变量应输出合并时推断出的正则模式（[0-9]+）
	if schema["pattern"] != "[0-9]+" {
		t.Errorf("path 参数 pattern 应为 [0-9]+，实际 %v", schema["pattern"])
	}
}

func TestOpenAPIExport_FixedPathNotVariable(t *testing.T) {
	r := router.NewReverseRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/users/list", nil, "GET", nil))

	exp := NewOpenAPIExporter()
	data, _ := exp.Export(r.Tree)
	doc := parseDoc(t, data)
	paths := getPaths(t, doc)

	if _, ok := paths["/api/users/list"]; !ok {
		t.Error("固定路径应保留为 /api/users/list")
	}
	// 不应出现变量路径
	for p := range paths {
		if strings.Contains(p, "{") {
			t.Errorf("不应有变量路径，但出现 %s", p)
		}
	}
}

// === 参数测试 ===

func TestOpenAPIExport_QueryParams(t *testing.T) {
	r := router.NewReverseRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/users?page=1&size=10", nil, "GET", nil))

	exp := NewOpenAPIExporter()
	data, _ := exp.Export(r.Tree)
	doc := parseDoc(t, data)
	op := getOperation(t, getPathItem(t, getPaths(t, doc), "/api/users"), "get")

	params, _ := op["parameters"].([]interface{})
	names := make(map[string]string)
	for _, p := range params {
		pm := p.(map[string]interface{})
		names[pm["name"].(string)] = pm["in"].(string)
	}
	if names["page"] != "query" {
		t.Errorf("page 应为 query 参数，实际 %s", names["page"])
	}
	if names["size"] != "query" {
		t.Errorf("size 应为 query 参数，实际 %s", names["size"])
	}
}

func TestOpenAPIExport_RequiredQueryParams(t *testing.T) {
	r := router.NewReverseRouter()
	// page 每次都有，size 只出现1次
	for i := 0; i < 10; i++ {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/list?page=1", nil, "GET", nil))
	}
	r.ReverseHttpRequest(request.NewHttpRequest("/api/list?page=1&size=10", nil, "GET", nil))
	r.InferRequiredParams()

	exp := NewOpenAPIExporter()
	data, _ := exp.Export(r.Tree)
	doc := parseDoc(t, data)
	op := getOperation(t, getPathItem(t, getPaths(t, doc), "/api/list"), "get")

	params, _ := op["parameters"].([]interface{})
	for _, p := range params {
		pm := p.(map[string]interface{})
		if pm["name"] == "page" && pm["required"] != true {
			t.Error("page 出现率高，应标记为 required")
		}
	}
}

func TestOpenAPIExport_ExcludeOptionalParameters(t *testing.T) {
	r := router.NewReverseRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/users?page=1&size=10", nil, "GET", nil))

	exp := NewOpenAPIExporter()
	exp.IncludeOptionalParameters = false
	data, _ := exp.Export(r.Tree)
	doc := parseDoc(t, data)
	op := getOperation(t, getPathItem(t, getPaths(t, doc), "/api/users"), "get")

	// 没调用 InferRequiredParams，page/size 都是可选，应被排除
	if params, ok := op["parameters"].([]interface{}); ok && len(params) > 0 {
		t.Errorf("排除可选参数后不应有参数，实际 %d 个", len(params))
	}
}

// === 请求体测试 ===

func TestOpenAPIExport_RequestBody(t *testing.T) {
	r := router.NewReverseRouter()
	h := request.Headers{"Content-Type": "application/json"}
	r.ReverseHttpRequest(request.NewHttpRequest("/api/users", h, "POST", []byte(`{"name":"bob","age":25}`)))

	exp := NewOpenAPIExporter()
	data, _ := exp.Export(r.Tree)
	doc := parseDoc(t, data)
	op := getOperation(t, getPathItem(t, getPaths(t, doc), "/api/users"), "post")

	body, ok := op["requestBody"].(map[string]interface{})
	if !ok {
		t.Fatal("POST 应有 requestBody")
	}
	content, _ := body["content"].(map[string]interface{})
	if _, ok := content["application/json"]; !ok {
		t.Errorf("requestBody 应包含 application/json，实际 %+v", content)
	}
}

func TestOpenAPIExport_ContentTypeCharsetStripped(t *testing.T) {
	r := router.NewReverseRouter()
	h := request.Headers{"Content-Type": "application/json; charset=utf-8"}
	r.ReverseHttpRequest(request.NewHttpRequest("/api/users", h, "POST", []byte(`{"name":"bob"}`)))

	exp := NewOpenAPIExporter()
	data, _ := exp.Export(r.Tree)
	doc := parseDoc(t, data)
	op := getOperation(t, getPathItem(t, getPaths(t, doc), "/api/users"), "post")

	body, _ := op["requestBody"].(map[string]interface{})
	content, _ := body["content"].(map[string]interface{})
	if _, ok := content["application/json"]; !ok {
		t.Errorf("Content-Type 带 charset 应规范化为 application/json，实际 %+v", content)
	}
	if _, ok := content["application/json; charset=utf-8"]; ok {
		t.Error("不应保留原始带 charset 的 key")
	}
}

// === Header/Cookie 测试 ===

func TestOpenAPIExport_HeaderParams(t *testing.T) {
	r := router.NewReverseRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/data",
		request.Headers{"Accept": "application/json"}, "GET", nil))
	r.ReverseHttpRequest(request.NewHttpRequest("/api/data",
		request.Headers{"Accept": "text/html"}, "GET", nil))

	exp := NewOpenAPIExporter()
	data, _ := exp.Export(r.Tree)
	doc := parseDoc(t, data)
	op := getOperation(t, getPathItem(t, getPaths(t, doc), "/api/data"), "get")

	params, _ := op["parameters"].([]interface{})
	// Accept 应只出现1次（去重）
	acceptCount := 0
	for _, p := range params {
		pm := p.(map[string]interface{})
		if pm["name"] == "Accept" && pm["in"] == "header" {
			acceptCount++
		}
	}
	if acceptCount != 1 {
		t.Errorf("Accept header 应去重为1个，实际 %d 个", acceptCount)
	}
}

func TestOpenAPIExport_CookieParams(t *testing.T) {
	r := router.NewReverseRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/home",
		request.Headers{"Cookie": "lang=zh-CN"}, "GET", nil))

	exp := NewOpenAPIExporter()
	data, _ := exp.Export(r.Tree)
	doc := parseDoc(t, data)
	op := getOperation(t, getPathItem(t, getPaths(t, doc), "/api/home"), "get")

	params, _ := op["parameters"].([]interface{})
	var cookieParam map[string]interface{}
	for _, p := range params {
		pm := p.(map[string]interface{})
		if pm["in"] == "cookie" {
			cookieParam = pm
		}
	}
	if cookieParam == nil {
		t.Fatal("应有 cookie 类型参数")
	}
	if cookieParam["name"] != "lang" {
		t.Errorf("cookie 参数名应为 lang，实际 %v", cookieParam["name"])
	}
}

// === 多方法测试 ===

func TestOpenAPIExport_MultipleMethods(t *testing.T) {
	r := router.NewReverseRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/users", nil, "GET", nil))
	r.ReverseHttpRequest(request.NewHttpRequest("/api/users", nil, "POST", nil))

	exp := NewOpenAPIExporter()
	data, _ := exp.Export(r.Tree)
	doc := parseDoc(t, data)
	item := getPathItem(t, getPaths(t, doc), "/api/users")

	if _, ok := item["get"]; !ok {
		t.Error("应有 GET 方法")
	}
	if _, ok := item["post"]; !ok {
		t.Error("应有 POST 方法")
	}
}

// === schema 类型映射测试 ===

func TestOpenAPIExport_SchemaTypeMapping(t *testing.T) {
	r := router.NewReverseRouter()
	// 手机号路径变量
	for _, p := range []string{"13812345678", "15912345678", "18612345678"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/users/"+p, nil, "GET", nil))
	}

	exp := NewOpenAPIExporter()
	data, _ := exp.Export(r.Tree)
	doc := parseDoc(t, data)
	paths := getPaths(t, doc)

	// 应有手机号变量路径
	var phonePath string
	for p := range paths {
		if strings.Contains(p, "{") {
			phonePath = p
		}
	}
	if phonePath == "" {
		t.Fatal("应有变量路径")
	}
	op := getOperation(t, getPathItem(t, paths, phonePath), "get")
	params, _ := op["parameters"].([]interface{})
	for _, p := range params {
		pm := p.(map[string]interface{})
		if pm["in"] == "path" {
			schema := pm["schema"].(map[string]interface{})
			// 手机号逻辑类型 → string type
			if schema["type"] != "string" {
				t.Errorf("手机号 schema type 应为 string，实际 %v", schema["type"])
			}
		}
	}
}

// === 边界测试 ===

func TestOpenAPIExport_EmptyTree(t *testing.T) {
	exp := NewOpenAPIExporter()
	_, err := exp.Export(tree.NewTree())
	if err != nil {
		t.Errorf("空树不应报错，实际 %v", err)
	}
}

func TestOpenAPIExport_NilTree(t *testing.T) {
	exp := NewOpenAPIExporter()
	_, err := exp.Export(nil)
	if err == nil {
		t.Error("nil 树应返回错误")
	}
}

func TestOpenAPIExport_OperationID(t *testing.T) {
	r := router.NewReverseRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/users/list", nil, "GET", nil))

	exp := NewOpenAPIExporter()
	data, _ := exp.Export(r.Tree)
	doc := parseDoc(t, data)
	op := getOperation(t, getPathItem(t, getPaths(t, doc), "/api/users/list"), "get")

	if op["operationId"] != "get_api_users_list" {
		t.Errorf("operationId 应为 get_api_users_list，实际 %v", op["operationId"])
	}
}

func TestOpenAPIExport_StableOrdering(t *testing.T) {
	r := router.NewReverseRouter()
	// 故意乱序添加
	r.ReverseHttpRequest(request.NewHttpRequest("/api/zebra", nil, "GET", nil))
	r.ReverseHttpRequest(request.NewHttpRequest("/api/apple", nil, "GET", nil))
	r.ReverseHttpRequest(request.NewHttpRequest("/api/mango", nil, "GET", nil))

	exp := NewOpenAPIExporter()
	data, _ := exp.Export(r.Tree)

	// 多次导出结果应一致（稳定排序）
	exp2 := NewOpenAPIExporter()
	data2, _ := exp2.Export(r.Tree)
	if string(data) != string(data2) {
		t.Error("相同输入多次导出结果应一致")
	}

	// 验证路径按字母序（map 遍历无序，排序后比较）
	doc := parseDoc(t, data)
	paths := getPaths(t, doc)
	keys := make([]string, 0)
	for k := range paths {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) != 3 || keys[0] != "/api/apple" || keys[2] != "/api/zebra" {
		t.Errorf("路径应按字母序输出，实际 %+v", keys)
	}
}

// === 安全方案测试 ===

func TestOpenAPIExport_SecurityFromAuthorization(t *testing.T) {
	r := router.NewReverseRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/orders",
		request.Headers{"Authorization": "Bearer eyJhbGciOiJIUzI1NiJ9.payload.sig"}, "GET", nil))
	r.ReverseHttpRequest(request.NewHttpRequest("/api/admin",
		request.Headers{"Authorization": "Basic dXNlcjpwYXNz"}, "GET", nil))

	data, _ := NewOpenAPIExporter().Export(r.Tree)
	doc := parseDoc(t, data)
	paths := getPaths(t, doc)

	// /api/orders 应声明 bearerAuth
	opOrder := getOperation(t, getPathItem(t, paths, "/api/orders"), "get")
	secOrder, _ := opOrder["security"].([]interface{})
	if len(secOrder) != 1 {
		t.Fatalf("/api/orders 应有1条 security 声明，实际 %d", len(secOrder))
	}
	if _, ok := secOrder[0].(map[string]interface{})["bearerAuth"]; !ok {
		t.Errorf("/api/orders security 应引用 bearerAuth，实际 %v", secOrder[0])
	}

	// /api/admin 应声明 basicAuth
	opAdmin := getOperation(t, getPathItem(t, paths, "/api/admin"), "get")
	secAdmin, _ := opAdmin["security"].([]interface{})
	if len(secAdmin) != 1 {
		t.Fatalf("/api/admin 应有1条 security 声明，实际 %d", len(secAdmin))
	}
	if _, ok := secAdmin[0].(map[string]interface{})["basicAuth"]; !ok {
		t.Errorf("/api/admin security 应引用 basicAuth，实际 %v", secAdmin[0])
	}

	// components.securitySchemes 应注册两个方案
	comps, _ := doc["components"].(map[string]interface{})
	schemes, _ := comps["securitySchemes"].(map[string]interface{})
	bearer, _ := schemes["bearerAuth"].(map[string]interface{})
	if bearer["type"] != "http" || bearer["scheme"] != "bearer" {
		t.Errorf("bearerAuth 应为 http/bearer，实际 %v", bearer)
	}
	basic, _ := schemes["basicAuth"].(map[string]interface{})
	if basic["type"] != "http" || basic["scheme"] != "basic" {
		t.Errorf("basicAuth 应为 http/basic，实际 %v", basic)
	}

	// Authorization 不应再作为普通 header 参数重复输出
	paramsOrder, _ := opOrder["parameters"].([]interface{})
	for _, p := range paramsOrder {
		pm := p.(map[string]interface{})
		if pm["name"] == "Authorization" {
			t.Error("Authorization 不应作为普通 header 参数输出（已由 security 表达）")
		}
	}
}

// TestOpenAPIExport_UnknownAuthFallsBackToHeader 无法识别的 Authorization
// 方案值（如自定义 Token）应回退为普通 header 参数，不生成 security。
func TestOpenAPIExport_UnknownAuthFallsBackToHeader(t *testing.T) {
	r := router.NewReverseRouter()
	// "Token" 不在 Bearer/Basic/Digest 之列 → 当普通 header
	r.ReverseHttpRequest(request.NewHttpRequest("/api/x",
		request.Headers{"Authorization": "Token abc123"}, "GET", nil))

	data, _ := NewOpenAPIExporter().Export(r.Tree)
	doc := parseDoc(t, data)
	op := getOperation(t, getPathItem(t, getPaths(t, doc), "/api/x"), "get")

	// 应有 Authorization 作为 header 参数
	paramsX, _ := op["parameters"].([]interface{})
	found := false
	for _, p := range paramsX {
		pm := p.(map[string]interface{})
		if pm["name"] == "Authorization" && pm["in"] == "header" {
			found = true
		}
	}
	if !found {
		t.Error("未识别方案应回退为 Authorization header 参数")
	}
	// 不应有 security 声明
	if _, ok := op["security"]; ok {
		t.Error("未识别方案不应生成 security 声明")
	}
}

// TestBuildSchema_LogicalTypeBranches 覆盖逻辑类型→schema 的所有分支。
func TestBuildSchema_LogicalTypeBranches(t *testing.T) {
	cases := []struct {
		name        string
		physical    string
		logical     string
		wantType    string
		wantFormat  string
	}{
		{"integer逻辑", "", "integer", "integer", ""},
		{"int逻辑", "", "int", "integer", ""},
		{"float逻辑", "", "float", "number", ""},
		{"decimal逻辑", "", "decimal", "number", ""},
		{"currency逻辑", "", "currency", "number", ""},
		{"percentage逻辑", "", "percentage", "number", ""},
		{"boolean逻辑", "", "boolean", "boolean", ""},
		{"date", "", "date", "string", "date"},
		{"datetime", "", "datetime", "string", "date-time"},
		{"time", "", "time", "string", "time"},
		{"email", "", "email", "string", "email"},
		{"url", "", "url", "string", "uri"},
		{"uuid", "", "uuid", "string", "uuid"},
		{"ipaddress", "", "ipaddress", "string", "ipv4"},
		{"phone", "", "phone", "string", ""},
		{"idcard", "", "idcard", "string", ""},
		{"bankcard", "", "bankcard", "string", ""},
		{"plate", "", "plate", "string", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := buildSchema(c.physical, c.logical, "", "")
			if s.Type != c.wantType {
				t.Errorf("Type = %q, want %q", s.Type, c.wantType)
			}
			if s.Format != c.wantFormat {
				t.Errorf("Format = %q, want %q", s.Format, c.wantFormat)
			}
		})
	}
}

// TestBuildSchema_PhysicalFallback 逻辑类型未命中时回退物理类型。
func TestBuildSchema_PhysicalFallback(t *testing.T) {
	cases := []struct {
		physical string
		wantType string
	}{
		{"integer", "integer"},
		{"float", "number"},
		{"boolean", "boolean"},
		{"array", "array"},
		{"object", "object"},
		{"string", "string"},
		{"", "string"}, // 未知物理类型默认 string
	}
	for _, c := range cases {
		t.Run(c.physical, func(t *testing.T) {
			s := buildSchema(c.physical, "", "", "")
			if s.Type != c.wantType {
				t.Errorf("physical=%q Type = %q, want %q", c.physical, s.Type, c.wantType)
			}
		})
	}
}

// TestBuildSchema_PatternAndDefault 验证 pattern/default 透传。
func TestBuildSchema_PatternAndDefault(t *testing.T) {
	s := buildSchema("", "integer", "1", `^[0-9]+$`)
	if s.Pattern != `^[0-9]+$` {
		t.Errorf("Pattern = %q, want ^[0-9]+$", s.Pattern)
	}
	if s.Default != "1" {
		t.Errorf("Default = %q, want 1", s.Default)
	}
}

// TestOpenAPIExport_AllHttpMethods 验证全部 HTTP 方法都能正确导出到对应 pathItem 字段。
func TestOpenAPIExport_AllHttpMethods(t *testing.T) {
	r := router.NewReverseRouter()
	// 禁用合并：本测试只验证方法→pathItem 字段映射，不关心合并行为
	r.SetMergeConfig(router.MergeConfig{
		SiblingMergeThreshold:       1000,
		PatternSimilarityThreshold:  0.6,
		SimilarLengthBreakThreshold: 0,
		RequiredParamThreshold:      0.9,
	})
	// 每方法用完全独立的顶层路径前缀
	cases := []struct {
		method string
		path   string
	}{
		{"GET", "/g1"}, {"POST", "/p2"}, {"PUT", "/u3"}, {"PATCH", "/pa4"},
		{"DELETE", "/d5"}, {"HEAD", "/h6"}, {"OPTIONS", "/o7"},
	}
	for _, c := range cases {
		req := request.NewHttpRequest(c.path, nil, c.method, nil)
		r.ReverseHttpRequest(req)
	}

	data, err := NewOpenAPIExporter().Export(r.Tree)
	if err != nil {
		t.Fatalf("导出失败: %v", err)
	}
	doc := parseDoc(t, data)
	paths := getPaths(t, doc)

	for _, c := range cases {
		item := getPathItem(t, paths, c.path)
		lower := strings.ToLower(c.method)
		if _, ok := item[lower]; !ok {
			t.Errorf("路径 %s 应包含方法字段 %q，实际 keys: %v", c.path, lower, keysOf(item))
		}
	}
}

// keysOf 返回 map 的键（测试断言失败时辅助显示）
func keysOf(m map[string]interface{}) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
