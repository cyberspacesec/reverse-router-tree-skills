package request

import (
	"testing"
)

// === Headers 测试 ===

func TestHeaders_Get(t *testing.T) {
	h := Headers{
		"Content-Type": "application/json",
		"Accept":       "text/html",
	}

	// 精确匹配
	if h.Get("Content-Type") != "application/json" {
		t.Error("Get 精确匹配失败")
	}

	// 大小写不敏感匹配
	if h.Get("content-type") != "application/json" {
		t.Error("Get 大小写不敏感匹配失败")
	}

	if h.Get("CONTENT-TYPE") != "application/json" {
		t.Error("Get 全大写匹配失败")
	}

	// 不存在的 key
	if h.Get("X-Custom") != "" {
		t.Error("不存在的 key 应该返回空字符串")
	}
}

func TestHeaders_Set(t *testing.T) {
	h := Headers{}
	h.Set("Content-Type", "application/json")

	if h.Get("Content-Type") != "application/json" {
		t.Error("Set 后 Get 失败")
	}
}

func TestHeaders_Has(t *testing.T) {
	h := Headers{"Accept": "text/html"}

	if !h.Has("Accept") {
		t.Error("Has 应该返回 true")
	}
	if !h.Has("accept") {
		t.Error("Has 大小写不敏感失败")
	}
	if h.Has("X-Custom") {
		t.Error("不存在的 key 应该返回 false")
	}
}

func TestHeaders_GetAll(t *testing.T) {
	h := Headers{"Accept": "text/html", "Content-Type": "application/json"}
	all := h.GetAll()

	if len(all) != 2 {
		t.Errorf("GetAll 应该返回2个元素，实际: %d", len(all))
	}
}

func TestHeaders_ConvenienceMethods(t *testing.T) {
	h := Headers{
		"Accept":          "application/json",
		"Authorization":   "Bearer token123",
		"X-Requested-With": "XMLHttpRequest",
		"X-Forwarded-For": "192.168.1.1",
		"X-Api-Version":   "v2",
		"Accept-Language":  "zh-CN",
	}

	if h.GetAccept() != "application/json" {
		t.Errorf("GetAccept 失败: %s", h.GetAccept())
	}
	if h.GetAuthorization() != "Bearer token123" {
		t.Errorf("GetAuthorization 失败: %s", h.GetAuthorization())
	}
	if h.GetAuthScheme() != "Bearer" {
		t.Errorf("GetAuthScheme 失败: %s", h.GetAuthScheme())
	}
	if h.GetXRequestedWith() != "XMLHttpRequest" {
		t.Errorf("GetXRequestedWith 失败: %s", h.GetXRequestedWith())
	}
	if h.GetXForwardedFor() != "192.168.1.1" {
		t.Errorf("GetXForwardedFor 失败: %s", h.GetXForwardedFor())
	}
	if h.GetXApiVersion() != "v2" {
		t.Errorf("GetXApiVersion 失败: %s", h.GetXApiVersion())
	}
	if h.GetAcceptLanguage() != "zh-CN" {
		t.Errorf("GetAcceptLanguage 失败: %s", h.GetAcceptLanguage())
	}
	if !h.IsAjax() {
		t.Error("IsAjax 应该返回 true")
	}
}

func TestHeaders_IsAjax_False(t *testing.T) {
	h := Headers{"Accept": "text/html"}
	if h.IsAjax() {
		t.Error("非 AJAX 请求 IsAjax 应该返回 false")
	}
}

func TestHeaders_String(t *testing.T) {
	h := Headers{}
	if h.String() != "{}" {
		t.Errorf("空 Headers String 应该是 '{}'，实际: '%s'", h.String())
	}

	h = Headers{"Accept": "text/html"}
	s := h.String()
	if s == "{}" || s == "" {
		t.Error("非空 Headers String 不应该是空的")
	}
}

func TestHeaders_ToHttpHeader(t *testing.T) {
	h := Headers{"Content-Type": "application/json", "Accept": "text/html"}
	httpHeader := h.ToHttpHeader()

	if httpHeader.Get("Content-Type") != "application/json" {
		t.Error("ToHttpHeader 转换失败")
	}
}

func TestHeaders_GetContentType(t *testing.T) {
	h := Headers{"Content-Type": "application/json"}
	if h.GetContentType() != "application/json" {
		t.Errorf("GetContentType 失败: %s", h.GetContentType())
	}
}

func TestHeaders_EmptyAuthScheme(t *testing.T) {
	h := Headers{}
	if h.GetAuthScheme() != "" {
		t.Error("空 Authorization 应该返回空 scheme")
	}
}

// === Cookies 测试 ===

func TestParseCookies(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name:     "empty",
			input:    "",
			expected: map[string]string{},
		},
		{
			name:     "single cookie",
			input:    "lang=zh-CN",
			expected: map[string]string{"lang": "zh-CN"},
		},
		{
			name:     "multiple cookies",
			input:    "lang=zh-CN; theme=dark; session=abc123",
			expected: map[string]string{"lang": "zh-CN", "theme": "dark", "session": "abc123"},
		},
		{
			name:     "cookie without value",
			input:    "flag",
			expected: map[string]string{"flag": ""},
		},
		{
			name:     "cookie with empty value",
			input:    "flag=",
			expected: map[string]string{"flag": ""},
		},
		{
			name:     "cookies with spaces",
			input:    "  lang = zh-CN ; theme = dark  ",
			expected: map[string]string{"lang": "zh-CN", "theme": "dark"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cookies := ParseCookies(tt.input)
			for key, expectedVal := range tt.expected {
				actualVal := cookies.Get(key)
				if actualVal != expectedVal {
					t.Errorf("cookie[%s] = %q, want %q", key, actualVal, expectedVal)
				}
			}
			// 确保没有多余的 cookie
			if len(cookies) != len(tt.expected) {
				t.Errorf("cookie count = %d, want %d", len(cookies), len(tt.expected))
			}
		})
	}
}

func TestCookies_Has(t *testing.T) {
	cookies := ParseCookies("lang=zh-CN; theme=dark")

	if !cookies.Has("lang") {
		t.Error("Has 应该返回 true")
	}
	if cookies.Has("nonexistent") {
		t.Error("不存在的 cookie 应该返回 false")
	}
}

func TestCookies_GetAll(t *testing.T) {
	cookies := ParseCookies("lang=zh-CN; theme=dark")
	all := cookies.GetAll()

	if len(all) != 2 {
		t.Errorf("GetAll 应该返回2个元素，实际: %d", len(all))
	}
}

func TestCookies_String(t *testing.T) {
	cookies := Cookies{}
	if cookies.String() != "{}" {
		t.Errorf("空 Cookies String 应该是 '{}'，实际: '%s'", cookies.String())
	}

	cookies = ParseCookies("lang=zh-CN")
	s := cookies.String()
	if s == "{}" || s == "" {
		t.Error("非空 Cookies String 不应该是空的")
	}
}

// === HttpRequest 测试 ===

func TestHttpRequest_New(t *testing.T) {
	req := NewHttpRequest("/api/users", Headers{"Accept": "application/json"}, "GET", nil)

	if req.GetUrl() != "/api/users" {
		t.Errorf("URL 不匹配: %s", req.GetUrl())
	}
	if req.GetMethod() != "GET" {
		t.Errorf("Method 不匹配: %s", req.GetMethod())
	}
	if req.GetBody() != nil {
		t.Error("Body 应该是 nil")
	}
}

func TestHttpRequest_Setters(t *testing.T) {
	req := NewHttpRequest("", nil, "", nil)

	req.SetUrl("/api/data")
	if req.GetUrl() != "/api/data" {
		t.Errorf("SetUrl 失败: %s", req.GetUrl())
	}

	req.SetMethod("POST")
	if req.GetMethod() != "POST" {
		t.Errorf("SetMethod 失败: %s", req.GetMethod())
	}

	req.SetBody([]byte("test"))
	if string(req.GetBody()) != "test" {
		t.Errorf("SetBody 失败: %s", string(req.GetBody()))
	}

	headers := Headers{"Content-Type": "application/json"}
	req.SetHeaders(headers)
	if req.GetHeaders().GetContentType() != "application/json" {
		t.Error("SetHeaders 失败")
	}
}

func TestHttpRequest_String(t *testing.T) {
	req := NewHttpRequest("/api/users", nil, "GET", nil)
	s := req.String()
	if s == "" {
		t.Error("String 不应该返回空字符串")
	}
}

// === HttpParam 测试 ===

func TestHttpParam_New(t *testing.T) {
	p := NewHttpParam("page", "1")

	if p.GetName() != "page" {
		t.Errorf("Name 不匹配: %s", p.GetName())
	}
	if p.GetValue() != "1" {
		t.Errorf("Value 不匹配: %s", p.GetValue())
	}
}

func TestHttpParam_Setters(t *testing.T) {
	p := NewHttpParam("", "")

	p.SetName("size")
	if p.GetName() != "size" {
		t.Errorf("SetName 失败: %s", p.GetName())
	}

	p.SetValue("10")
	if p.GetValue() != "10" {
		t.Errorf("SetValue 失败: %s", p.GetValue())
	}
}

func TestHttpParam_String(t *testing.T) {
	p := NewHttpParam("page", "1")
	if p.String() != "page=1" {
		t.Errorf("String 不匹配: %s", p.String())
	}
}

// === HttpRequestPath 测试 ===

func TestHttpRequestPath_Basic(t *testing.T) {
	p := NewHttpRequestPath("users")

	if p.GetPath() != "users" {
		t.Errorf("Path 不匹配: %s", p.GetPath())
	}
	if p.String() != "users" {
		t.Errorf("String 不匹配: %s", p.String())
	}
}

func TestHttpRequestPath_SetPath(t *testing.T) {
	p := NewHttpRequestPath("users")
	p.SetPath("admin")

	if p.GetPath() != "admin" {
		t.Errorf("SetPath 失败: %s", p.GetPath())
	}

	// 设置为 key=value 格式应该重新检测路径参数
	p.SetPath("action=delete")
	if !p.IsPathParam() {
		t.Error("SetPath 后应该重新检测路径参数")
	}
}

// === UrlParser 边界条件测试 ===

func TestUrlParser_URLDecodedPath(t *testing.T) {
	parser := NewUrlParser("/api/%E7%94%A8%E6%88%B7/list")
	paths, _, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse 失败: %v", err)
	}

	// URL编码的路径应该被解码
	if len(paths) != 3 {
		t.Fatalf("应该有3个路径段，实际: %d", len(paths))
	}

	// 第二个段应该是解码后的中文
	if paths[1].GetPath() != "用户" {
		t.Errorf("URL解码失败: %s", paths[1].GetPath())
	}
}

func TestUrlParser_ParamCaseNormalization(t *testing.T) {
	parser := NewUrlParser("/api/data?Page=1&SIZE=10")
	_, params, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse 失败: %v", err)
	}

	// 参数名应该被统一为小写
	paramMap := make(map[string]string)
	for _, p := range params {
		paramMap[p.GetName()] = p.GetValue()
	}

	if _, ok := paramMap["page"]; !ok {
		t.Error("Page 应该被转换为 page")
	}
	if _, ok := paramMap["size"]; !ok {
		t.Error("SIZE 应该被转换为 size")
	}
}

func TestUrlParser_MultiValueParam(t *testing.T) {
	parser := NewUrlParser("/api/search?tag=go&tag=web&tag=api")
	_, params, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse 失败: %v", err)
	}

	// 应该有3个 tag 参数
	tagCount := 0
	for _, p := range params {
		if p.GetName() == "tag" {
			tagCount++
		}
	}
	if tagCount != 3 {
		t.Errorf("应该有3个 tag 参数，实际: %d", tagCount)
	}
}

func TestUrlParser_DotSegment(t *testing.T) {
	parser := NewUrlParser("/api/./users/../admin")
	paths, _, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse 失败: %v", err)
	}

	// . 和 .. 段应该被过滤
	for _, p := range paths {
		if p.GetPath() == "." || p.GetPath() == ".." {
			t.Errorf("不应该有 '%s' 路径段", p.GetPath())
		}
	}
}
