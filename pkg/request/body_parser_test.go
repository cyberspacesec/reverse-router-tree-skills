package request

import (
	"sort"
	"testing"
)

// 按名称排序后比较参数列表，忽略顺序差异
func paramsEqual(a, b []*HttpParam) bool {
	if len(a) != len(b) {
		return false
	}
	ac := make([]*HttpParam, len(a))
	copy(ac, a)
	bc := make([]*HttpParam, len(b))
	copy(bc, b)
	sort.Slice(ac, func(i, j int) bool { return ac[i].Name < ac[j].Name })
	sort.Slice(bc, func(i, j int) bool { return bc[i].Name < bc[j].Name })
	for i := range ac {
		if ac[i].Name != bc[i].Name || ac[i].Value != bc[i].Value {
			return false
		}
	}
	return true
}

func TestBodyParser_FormUrlencoded(t *testing.T) {
	p := NewBodyParser()
	params, err := p.Parse("application/x-www-form-urlencoded",
		[]byte("name=alice&age=30&city=%E5%8C%97%E4%BA%AC"))
	if err != nil {
		t.Fatal(err)
	}

	expected := []*HttpParam{
		{Name: "name", Value: "alice"},
		{Name: "age", Value: "30"},
		{Name: "city", Value: "北京"}, // URL编码应被解码
	}
	if !paramsEqual(params, expected) {
		t.Errorf("表单解析不符，期望 %+v，实际 %+v", expected, params)
	}
}

func TestBodyParser_FormUrlencoded_MultiValue(t *testing.T) {
	p := NewBodyParser()
	params, err := p.Parse("application/x-www-form-urlencoded",
		[]byte("tag=go&tag=web"))
	if err != nil {
		t.Fatal(err)
	}

	if len(params) != 2 {
		t.Fatalf("应解析出2个多值参数，实际 %d", len(params))
	}
	values := []string{params[0].Value, params[1].Value}
	sort.Strings(values)
	if values[0] != "go" || values[1] != "web" {
		t.Errorf("多值参数应为 go/web，实际 %+v", params)
	}
}

func TestBodyParser_JSON_Flat(t *testing.T) {
	p := NewBodyParser()
	params, err := p.Parse("application/json",
		[]byte(`{"name":"bob","age":25,"active":true}`))
	if err != nil {
		t.Fatal(err)
	}

	expected := []*HttpParam{
		{Name: "name", Value: "bob"},
		{Name: "age", Value: "25"},
		{Name: "active", Value: "true"},
	}
	if !paramsEqual(params, expected) {
		t.Errorf("JSON扁平解析不符，期望 %+v，实际 %+v", expected, params)
	}
}

func TestBodyParser_JSON_Nested(t *testing.T) {
	p := NewBodyParser()
	params, err := p.Parse("application/json",
		[]byte(`{"user":{"name":"bob","address":{"city":"上海"}},"tags":["vip","new"]}`))
	if err != nil {
		t.Fatal(err)
	}

	expected := []*HttpParam{
		{Name: "user.name", Value: "bob"},
		{Name: "user.address.city", Value: "上海"},
		{Name: "tags.0", Value: "vip"},
		{Name: "tags.1", Value: "new"},
	}
	if !paramsEqual(params, expected) {
		t.Errorf("JSON嵌套解析不符，期望 %+v，实际 %+v", expected, params)
	}
}

func TestBodyParser_JSON_Float(t *testing.T) {
	p := NewBodyParser()
	params, err := p.Parse("application/json",
		[]byte(`{"price":99.5,"count":100}`))
	if err != nil {
		t.Fatal(err)
	}

	m := make(map[string]string)
	for _, param := range params {
		m[param.Name] = param.Value
	}
	if m["price"] != "99.5" {
		t.Errorf("浮点数应为 99.5，实际 %s", m["price"])
	}
	if m["count"] != "100" {
		t.Errorf("整数应格式化为 100（不带 .0），实际 %s", m["count"])
	}
}

func TestBodyParser_JSON_Null(t *testing.T) {
	p := NewBodyParser()
	params, err := p.Parse("application/json",
		[]byte(`{"name":"bob","avatar":null}`))
	if err != nil {
		t.Fatal(err)
	}

	m := make(map[string]string)
	for _, param := range params {
		m[param.Name] = param.Value
	}
	if m["name"] != "bob" {
		t.Errorf("name 应为 bob，实际 %s", m["name"])
	}
	if m["avatar"] != "" {
		t.Errorf("null 值应为空字符串，实际 %s", m["avatar"])
	}
}

func TestBodyParser_Multipart(t *testing.T) {
	p := NewBodyParser()
	contentType := "multipart/form-data; boundary=----Boundary"
	body := []byte("------Boundary\r\n" +
		"Content-Disposition: form-data; name=\"username\"\r\n\r\n" +
		"carl\r\n" +
		"------Boundary\r\n" +
		"Content-Disposition: form-data; name=\"avatar\"; filename=\"photo.jpg\"\r\n\r\n" +
		"<binary>\r\n" +
		"------Boundary--\r\n")

	params, err := p.Parse(contentType, body)
	if err != nil {
		t.Fatal(err)
	}

	m := make(map[string]string)
	for _, param := range params {
		m[param.Name] = param.Value
	}
	if m["username"] != "carl" {
		t.Errorf("username 应为 carl，实际 %s", m["username"])
	}
	// 文件字段应以文件名作为值（不读取文件内容）
	if m["avatar"] != "photo.jpg" {
		t.Errorf("avatar 文件字段值应为文件名 photo.jpg，实际 %s", m["avatar"])
	}
}

func TestBodyParser_ContentTypeWithCharset(t *testing.T) {
	p := NewBodyParser()
	// Content-Type 带 charset，应正确识别为 JSON
	params, err := p.Parse("application/json; charset=utf-8",
		[]byte(`{"key":"value"}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(params) != 1 || params[0].Name != "key" || params[0].Value != "value" {
		t.Errorf("带charset的JSON应正确解析，实际 %+v", params)
	}
}

func TestBodyParser_UnsupportedType(t *testing.T) {
	p := NewBodyParser()
	// text/plain 不支持，应返回空列表无错误
	params, err := p.Parse("text/plain", []byte("hello world"))
	if err != nil {
		t.Errorf("不支持的类型不应返回错误，实际 %v", err)
	}
	if len(params) != 0 {
		t.Errorf("不支持的类型应返回空列表，实际 %+v", params)
	}
}

func TestBodyParser_EmptyBody(t *testing.T) {
	p := NewBodyParser()
	params, err := p.Parse("application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(params) != 0 {
		t.Errorf("空body应返回空列表，实际 %+v", params)
	}
}

func TestBodyParser_InvalidJSON(t *testing.T) {
	p := NewBodyParser()
	_, err := p.Parse("application/json", []byte(`{invalid json`))
	if err == nil {
		t.Error("非法JSON应返回错误")
	}
}

func TestBodyParser_ParamNameLowercased(t *testing.T) {
	p := NewBodyParser()
	// 参数名应统一小写（与查询参数一致）
	params, err := p.Parse("application/x-www-form-urlencoded",
		[]byte("Name=Alice&AGE=30"))
	if err != nil {
		t.Fatal(err)
	}
	for _, param := range params {
		if param.Name != "name" && param.Name != "age" {
			t.Errorf("参数名应被小写化，实际 %s", param.Name)
		}
	}
}

func TestBodyParser_MaxParamsLimit(t *testing.T) {
	p := &BodyParser{MaxParams: 2}
	// 超过上限应截断
	params, err := p.Parse("application/x-www-form-urlencoded",
		[]byte("a=1&b=2&c=3&d=4"))
	if err != nil {
		t.Fatal(err)
	}
	if len(params) != 2 {
		t.Errorf("MaxParams=2 应截断为2个参数，实际 %d", len(params))
	}
}

func TestNormalizeContentType(t *testing.T) {
	cases := map[string]string{
		"application/json":                "application/json",
		"application/json; charset=utf-8": "application/json",
		"  Application/JSON  ":            "application/json",
		// normalizeContentType 只取主类型，boundary 参数被去掉
		//（boundary 在 Parse 内部用 extractBoundary 从原始 contentType 取）
		"multipart/form-data; boundary=X": "multipart/form-data",
		"":                                "",
	}
	for input, expected := range cases {
		got := normalizeContentType(input)
		if got != expected {
			t.Errorf("normalizeContentType(%q) = %q, 期望 %q", input, got, expected)
		}
	}
}

func TestExtractBoundary(t *testing.T) {
	cases := map[string]string{
		"multipart/form-data; boundary=----WebKit":     "----WebKit",
		"multipart/form-data; boundary=\"quotedBound\"": "quotedBound",
		"multipart/form-data":                          "",
	}
	for input, expected := range cases {
		got := extractBoundary(input)
		if got != expected {
			t.Errorf("extractBoundary(%q) = %q, 期望 %q", input, got, expected)
		}
	}
}
