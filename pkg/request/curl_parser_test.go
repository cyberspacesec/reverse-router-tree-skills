package request

import (
	"strings"
	"testing"
)

// TestParseCurl_SimpleURL 验证无引号 URL 的基础解析，默认 GET 方法。
func TestParseCurl_SimpleURL(t *testing.T) {
	req, err := ParseCurl("curl https://example.com/api/v1/users")
	if err != nil {
		t.Fatalf("解析失败：%v", err)
	}
	if req.Url != "https://example.com/api/v1/users" {
		t.Errorf("Url 不匹配，期望 %q，实际 %q", "https://example.com/api/v1/users", req.Url)
	}
	if req.Method != "GET" {
		t.Errorf("Method 不匹配，期望 GET，实际 %q", req.Method)
	}
	if len(req.Body) != 0 {
		t.Errorf("Body 应为空，实际 %q", string(req.Body))
	}
	if len(req.Headers) != 0 {
		t.Errorf("Headers 应为空，实际 %v", req.Headers)
	}
}

// TestParseCurl_SingleQuotes 验证单引号包裹的 URL 原样保留内部字符。
func TestParseCurl_SingleQuotes(t *testing.T) {
	req, err := ParseCurl("curl 'https://example.com/path with space?q=a b'")
	if err != nil {
		t.Fatalf("解析失败：%v", err)
	}
	expected := "https://example.com/path with space?q=a b"
	if req.Url != expected {
		t.Errorf("单引号 URL 应原样保留，期望 %q，实际 %q", expected, req.Url)
	}
}

// TestParseCurl_WithHeaders 验证多 -H 的 Authorization 与 Content-Type 解析。
func TestParseCurl_WithHeaders(t *testing.T) {
	cmd := `curl 'https://api.example.com/data' ` +
		`-H 'Authorization: Bearer abc.def.ghi' ` +
		`-H 'Content-Type: application/json' ` +
		`-H 'Accept: application/json'`
	req, err := ParseCurl(cmd)
	if err != nil {
		t.Fatalf("解析失败：%v", err)
	}
	if req.Headers.GetAuthorization() != "Bearer abc.def.ghi" {
		t.Errorf("Authorization 不匹配，实际 %q", req.Headers.GetAuthorization())
	}
	if req.Headers.GetContentType() != "application/json" {
		t.Errorf("Content-Type 不匹配，实际 %q", req.Headers.GetContentType())
	}
	if req.Headers.Get("Accept") != "application/json" {
		t.Errorf("Accept 不匹配，实际 %q", req.Headers.Get("Accept"))
	}
}

// TestParseCurl_PostData 验证 -X POST -d 携带 JSON 请求体。
func TestParseCurl_PostData(t *testing.T) {
	cmd := `curl -X POST 'https://api.example.com/users' ` +
		`-H 'Content-Type: application/json' ` +
		`-d '{"name":"alice","age":30}'`
	req, err := ParseCurl(cmd)
	if err != nil {
		t.Fatalf("解析失败：%v", err)
	}
	if req.Method != "POST" {
		t.Errorf("Method 应为 POST，实际 %q", req.Method)
	}
	expectedBody := `{"name":"alice","age":30}`
	if string(req.Body) != expectedBody {
		t.Errorf("Body 不匹配，期望 %q，实际 %q", expectedBody, string(req.Body))
	}
	// 用户显式设置了 JSON Content-Type，不应被表单默认值覆盖
	if req.Headers.GetContentType() != "application/json" {
		t.Errorf("Content-Type 应保留用户显式值 application/json，实际 %q", req.Headers.GetContentType())
	}
}

// TestParseCurl_DataImpliesPost 验证 -d 无显式 -X 时默认 POST 并补表单 Content-Type。
func TestParseCurl_DataImpliesPost(t *testing.T) {
	cmd := `curl 'https://api.example.com/login' -d 'username=alice&password=secret'`
	req, err := ParseCurl(cmd)
	if err != nil {
		t.Fatalf("解析失败：%v", err)
	}
	if req.Method != "POST" {
		t.Errorf("有 -d 无 -X 时应默认 POST，实际 %q", req.Method)
	}
	if string(req.Body) != "username=alice&password=secret" {
		t.Errorf("Body 不匹配，实际 %q", string(req.Body))
	}
	if req.Headers.GetContentType() != "application/x-www-form-urlencoded" {
		t.Errorf("应补默认表单 Content-Type，实际 %q", req.Headers.GetContentType())
	}
}

// TestParseCurl_LineContinuation 验证反斜杠续行多行命令被正确拼接。
func TestParseCurl_LineContinuation(t *testing.T) {
	cmd := "curl https://example.com/api \\\n" +
		"  -H 'Authorization: Bearer xyz' \\\n" +
		"  -H 'Accept: text/plain'"
	req, err := ParseCurl(cmd)
	if err != nil {
		t.Fatalf("解析失败：%v", err)
	}
	if req.Url != "https://example.com/api" {
		t.Errorf("Url 不匹配，实际 %q", req.Url)
	}
	if req.Headers.GetAuthorization() != "Bearer xyz" {
		t.Errorf("Authorization 不匹配，实际 %q", req.Headers.GetAuthorization())
	}
	if req.Headers.Get("Accept") != "text/plain" {
		t.Errorf("Accept 不匹配，实际 %q", req.Headers.Get("Accept"))
	}
}

// TestParseCurl_CompactFlags 验证 -XPOST 紧凑形式与 --compressed 等无害 flag 跳过。
func TestParseCurl_CompactFlags(t *testing.T) {
	cmd := `curl -XPOST --compressed -s -k -L -i 'https://example.com/submit' -d 'k=v'`
	req, err := ParseCurl(cmd)
	if err != nil {
		t.Fatalf("解析失败：%v", err)
	}
	if req.Method != "POST" {
		t.Errorf("-XPOST 紧凑形式应为 POST，实际 %q", req.Method)
	}
	if req.Url != "https://example.com/submit" {
		t.Errorf("Url 不匹配，实际 %q", req.Url)
	}
	if string(req.Body) != "k=v" {
		t.Errorf("Body 不匹配，实际 %q", string(req.Body))
	}
	// 无害 flag 不应产生任何 header 副作用
	if len(req.Headers) != 1 {
		t.Errorf("应仅含 -d 隐含的 Content-Type header，实际 %v", req.Headers)
	}
	if req.Headers.GetContentType() != "application/x-www-form-urlencoded" {
		t.Errorf("表单 Content-Type 不匹配，实际 %q", req.Headers.GetContentType())
	}
}

// TestParseCurl_DoubleQuotesWithEnv 验证双引号内 $ 变量按字面保留，不展开。
func TestParseCurl_DoubleQuotesWithEnv(t *testing.T) {
	cmd := `curl "https://example.com/api?key=$API_KEY&token=$TOKEN" -H "X-Secret: $SECRET"`
	req, err := ParseCurl(cmd)
	if err != nil {
		t.Fatalf("解析失败：%v", err)
	}
	expectedURL := "https://example.com/api?key=$API_KEY&token=$TOKEN"
	if req.Url != expectedURL {
		t.Errorf("双引号内 $ 变量应原样保留，期望 %q，实际 %q", expectedURL, req.Url)
	}
	if req.Headers.Get("X-Secret") != "$SECRET" {
		t.Errorf("header 中 $ 变量应原样保留，实际 %q", req.Headers.Get("X-Secret"))
	}
}

// TestParseCurl_Errors 表驱动验证各类错误场景。
func TestParseCurl_Errors(t *testing.T) {
	tests := []struct {
		name string
		curl string
	}{
		{
			name: "非 curl 前缀",
			curl: "wget https://example.com",
		},
		{
			name: "无 URL",
			curl: "curl -H 'Content-Type: application/json'",
		},
		{
			name: "未闭合单引号",
			curl: "curl 'https://example.com/unclosed",
		},
		{
			name: "未闭合双引号",
			curl: `curl "https://example.com/unclosed`,
		},
		{
			name: "-H 缺少参数",
			curl: "curl https://example.com -H",
		},
		{
			name: "非法 header 格式（无冒号）",
			curl: "curl https://example.com -H 'InvalidHeader'",
		},
		{
			name: "-X 缺少参数",
			curl: "curl https://example.com -X",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseCurl(tt.curl)
			if err == nil {
				t.Fatalf("期望返回错误，但解析成功")
			}
		})
	}
}

// TestParseCurl_FlagsWithValue 验证带值 flag（--max-time/--connect-timeout/-m 等）
// 的值不会被误当 URL。修复前 --max-time 30 'http://x' 的 URL 被错误设为 "30"。
func TestParseCurl_FlagsWithValue(t *testing.T) {
	cases := []struct {
		name       string
		curl       string
		wantURL    string
		wantMethod string
	}{
		{
			name:       "max-time with value",
			curl:       `curl --max-time 30 'http://api.example.com/users/123'`,
			wantURL:    "http://api.example.com/users/123",
			wantMethod: "GET",
		},
		{
			name:       "connect-timeout + max-time",
			curl:       `curl --connect-timeout 5 --max-time 30 'http://api.example.com/users'`,
			wantURL:    "http://api.example.com/users",
			wantMethod: "GET",
		},
		{
			name:       "retry flags",
			curl:       `curl --retry 3 --retry-delay 2 'http://api.example.com/x'`,
			wantURL:    "http://api.example.com/x",
			wantMethod: "GET",
		},
		{
			name:       "user-agent flag value not consumed as body",
			curl:       `curl -A 'MyAgent/1.0' 'http://api.example.com/x'`,
			wantURL:    "http://api.example.com/x",
			wantMethod: "GET",
		},
		{
			name:       "output flag value not consumed as url",
			curl:       `curl -o /tmp/out 'http://api.example.com/x'`,
			wantURL:    "http://api.example.com/x",
			wantMethod: "GET",
		},
		{
			name:       "value flags before POST with body",
			curl:       `curl --connect-timeout 5 -X POST 'http://api.example.com/users' -d '{"a":1}'`,
			wantURL:    "http://api.example.com/users",
			wantMethod: "POST",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, err := ParseCurl(tc.curl)
			if err != nil {
				t.Fatalf("ParseCurl(%q) err: %v", tc.curl, err)
			}
			if r.GetUrl() != tc.wantURL {
				t.Errorf("URL = %q, want %q", r.GetUrl(), tc.wantURL)
			}
			if r.GetMethod() != tc.wantMethod {
				t.Errorf("Method = %q, want %q", r.GetMethod(), tc.wantMethod)
			}
		})
	}
}

// TestParseCurl_GetFlagAndUrlFlag 验证 -G 把 -d 转 query、--url 显式指定 URL。
func TestParseCurl_GetFlagAndUrlFlag(t *testing.T) {
	// -G：-d 的值作为 query 串附加到 URL，不应作为 body
	r, err := ParseCurl(`curl -G 'http://api.example.com/search' -d 'q=go&page=1'`)
	if err != nil {
		t.Fatalf("ParseCurl -G err: %v", err)
	}
	if got := r.GetUrl(); !strings.Contains(got, "q=go") || !strings.Contains(got, "page=1") {
		t.Errorf("-G URL = %q, 应含 q=go 和 page=1", got)
	}
	if got := r.GetMethod(); got != "GET" {
		t.Errorf("-G Method = %q, want GET", got)
	}
	if len(r.GetBody()) != 0 {
		t.Errorf("-G 不应有 body，实际 %q", string(r.GetBody()))
	}

	// -G 且 URL 已含 query：用 & 拼接
	r2, err := ParseCurl(`curl -G 'http://api.example.com/search?lang=zh' -d 'q=go'`)
	if err != nil {
		t.Fatalf("ParseCurl -G& err: %v", err)
	}
	if !strings.Contains(r2.GetUrl(), "lang=zh") || !strings.Contains(r2.GetUrl(), "&q=go") {
		t.Errorf("-G 拼接 & 失败: %q", r2.GetUrl())
	}

	// --url 显式 URL 优先于位置 URL
	r3, err := ParseCurl(`curl --url 'http://api.example.com/explicit'`)
	if err != nil {
		t.Fatalf("ParseCurl --url err: %v", err)
	}
	if got := r3.GetUrl(); got != "http://api.example.com/explicit" {
		t.Errorf("--url = %q, want http://api.example.com/explicit", got)
	}
}
