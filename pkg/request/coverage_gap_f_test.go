package request

import (
	"strings"
	"testing"
)

// TestGapF_ 补齐 pkg/request 覆盖缺口的集中测试（url/body/fasturl/headers/path/curl）。
// 函数名统一 TestGapF_ 前缀，避免与其他并行补测任务冲突。

// TestGapF_UrlParser_Branches 覆盖 UrlParser.Parse 的异常与空分支。
func TestGapF_UrlParser_Branches(t *testing.T) {
	// 空 URL：path/query 均空，无错误
	p := NewUrlParser("")
	paths, params, err := p.Parse()
	if err != nil {
		t.Fatalf("空 URL 不应报错: %v", err)
	}
	if len(paths) != 0 || len(params) != 0 {
		t.Errorf("空 URL 应返回空结果，实际 paths=%d params=%d", len(paths), len(params))
	}

	// 非法百分号转义：走解码错误分支（触发 ReleasePaths 后透传 error）
	p2 := NewUrlParser("/api/%ZZ")
	if _, _, err := p2.Parse(); err == nil {
		t.Error("非法百分号转义应返回解码错误")
	}

	// 非法 query 转义：走 url.ParseQuery 错误分支
	p3 := NewUrlParser("/api?a=%zz")
	if _, _, err := p3.Parse(); err == nil {
		t.Error("非法 query 转义应返回错误")
	}

	// 无 path 无 query：覆盖 pathStr=="" 且 queryStr=="" 跳过分支
	p4 := NewUrlParser("http://x.com")
	paths4, params4, err := p4.Parse()
	if err != nil {
		t.Fatalf("无 path URL 不应报错: %v", err)
	}
	if len(paths4) != 0 || len(params4) != 0 {
		t.Errorf("无 path 应返回空结果，实际 paths=%d params=%d", len(paths4), len(params4))
	}
	ReleasePaths(paths4)

	// 纯 query 无 path：覆盖 path 为空但 query 非空分支
	p5 := NewUrlParser("http://x.com?q=1")
	_, params5, err := p5.Parse()
	if err != nil {
		t.Fatalf("纯 query URL 不应报错: %v", err)
	}
	if len(params5) != 1 {
		t.Errorf("纯 query 应解析出 1 个参数，实际 %d", len(params5))
	}
}

// TestGapF_BodyParser_ParseBranches 覆盖 BodyParser.Parse 的各类 body 类型与错误分支。
func TestGapF_BodyParser_ParseBranches(t *testing.T) {
	p := NewBodyParser()

	// 空 Content-Type：mime 为空直接返回 nil,nil
	got, err := p.Parse("", []byte("a=1"))
	if err != nil || len(got) != 0 {
		t.Errorf("空 Content-Type 应返回空无错，实际 got=%v err=%v", got, err)
	}

	// multipart 缺 boundary：覆盖 parseMultipart 错误透传分支
	if _, err := p.Parse("multipart/form-data", []byte("xxx")); err == nil {
		t.Error("multipart 缺 boundary 应返回错误")
	}

	// form 非法分隔符：覆盖 parseFormUrlencoded 错误分支
	if _, err := p.Parse("application/x-www-form-urlencoded", []byte("a=1;b=2")); err == nil {
		t.Error("form 非法分号分隔应返回错误")
	}

	// JSON 顶层标量：prefix 为空不追加参数，覆盖 flattenJSON default 空前缀分支
	got2, err := p.Parse("application/json", []byte(`"topscalar"`))
	if err != nil {
		t.Fatalf("顶层标量 JSON 不应报错: %v", err)
	}
	if len(got2) != 0 {
		t.Errorf("顶层标量 prefix 为空应无参数，实际 %+v", got2)
	}

	// JSON 顶层数组：覆盖 flattenJSON 数组空前缀分支（".0" 形式）
	got3, err := p.Parse("application/json", []byte(`["a","b"]`))
	if err != nil {
		t.Fatalf("顶层数组 JSON 不应报错: %v", err)
	}
	if len(got3) != 2 {
		t.Errorf("顶层数组应解析出 2 个参数，实际 %+v", got3)
	}
}

// TestGapF_BodyParser_FormatScalar 直接覆盖 formatJSONScalar 全部分支。
func TestGapF_BodyParser_FormatScalar(t *testing.T) {
	cases := []struct {
		name string
		in   interface{}
		want string
	}{
		{"nil", nil, ""},
		{"字符串", "hi", "hi"},
		{"真", true, "true"},
		{"假", false, "false"},
		{"整数浮点", float64(100), "100"},
		{"小数", float64(99.5), "99.5"},
		{"默认分支int", int(42), "42"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := formatJSONScalar(c.in); got != c.want {
				t.Errorf("formatJSONScalar(%v) = %q, 期望 %q", c.in, got, c.want)
			}
		})
	}
}

// TestGapF_BodyParser_MultipartBranches 覆盖 parseMultipart 的跳过与文件分支。
func TestGapF_BodyParser_MultipartBranches(t *testing.T) {
	p := NewBodyParser()
	ct := "multipart/form-data; boundary=b"

	// part 无头体分隔符：跳过不报错
	body := "--b\r\nno-headers-here\r\n--b--\r\n"
	got, err := p.Parse(ct, []byte(body))
	if err != nil {
		t.Fatalf("无分隔符 part 应跳过不报错: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("无分隔符 part 应被跳过，实际 %+v", got)
	}

	// part 无 name：跳过；同时覆盖普通文本字段分支
	body2 := "--b\r\n" +
		"Content-Disposition: form-data; filename=\"a.txt\"\r\n\r\n" +
		"xxx\r\n" +
		"--b\r\n" +
		"Content-Type: text/plain\r\n" +
		"Content-Disposition: form-data; name=\"note\"\r\n\r\n" +
		"hello\r\n" +
		"--b--\r\n"
	got2, err := p.Parse(ct, []byte(body2))
	if err != nil {
		t.Fatalf("multipart 解析不应报错: %v", err)
	}
	if len(got2) != 1 || got2[0].Name != "note" || got2[0].Value != "hello" {
		t.Errorf("无 name part 应跳过、文本字段应保留，实际 %+v", got2)
	}
}

// TestGapF_BodyParser_DispositionField 覆盖 extractContentDispositionField 的行过滤分支。
func TestGapF_BodyParser_DispositionField(t *testing.T) {
	// 非 disposition 行跳过；大小写不敏感匹配
	hdr := "Content-Type: text/plain\r\nContent-Disposition: form-data; name=\"f\""
	if got := extractContentDispositionField(hdr, "name"); got != "f" {
		t.Errorf("应提取 name=f，实际 %q", got)
	}
	// 不存在的字段返回空
	if got := extractContentDispositionField(hdr, "filename"); got != "" {
		t.Errorf("不存在字段应返回空，实际 %q", got)
	}
	// 无 disposition 行返回空
	if got := extractContentDispositionField("Content-Type: text/plain", "name"); got != "" {
		t.Errorf("无 disposition 行应返回空，实际 %q", got)
	}
}

// TestGapF_FastParse_Branches 覆盖 fastParseURLPathAndQuery 的各类 URL 形态分支。
func TestGapF_FastParse_Branches(t *testing.T) {
	cases := []struct {
		raw       string
		wantPath  string
		wantQuery string
		wantErr   bool
	}{
		{"", "", "", false},                           // 空输入
		{"http://", "", "", false},                    // scheme 后无 host
		{"http://x.com", "", "", false},               // 无 path 无 query
		{"http://x.com?q=1", "", "q=1", false},        // 仅 query
		{"http://x.com/p", "p", "", false},            // 仅 path
		{"http://x.com/p?q=1", "p", "q=1", false},     // path+query
		{"http://x.com?q=1/p", "", "q=1/p", false},    // query 先于 path（取更靠前分隔符）
		{"//auth", "", "", false},                     // scheme-relative 无后续
		{"//", "", "", false},                         // 纯双斜杠：防御分支直接返回
		{"//auth/p?q=1", "p", "q=1", false},           // scheme-relative path 先于 query
		{"//auth?q=1/p", "", "q=1/p", false},          // scheme-relative query 先于 path
		{"//auth?q=1", "", "q=1", false},              // scheme-relative 仅 query
		{"//auth/p#f", "p", "", false},                // scheme-relative + fragment 截断
		{"/p#frag", "p", "", false},                   // fragment 不进 path/query
		{"/p?a=1#frag", "p", "a=1", false},            // fragment 截断保留 query
		{"relative/path", "relative/path", "", false}, // 纯相对路径
		{"://x", "", "", true},                        // 空 scheme 报错
		{"http://x.com/", "", "", false},              // 根斜杠归一为空
		{"?", "", "", false},                          // 孤立问号
		{"#frag", "", "", false},                      // 纯 fragment 截断为空
	}
	for _, c := range cases {
		gotPath, gotQuery, err := fastParseURLPathAndQuery(c.raw)
		if (err != nil) != c.wantErr {
			t.Errorf("fastParse(%q) err=%v, wantErr=%v", c.raw, err, c.wantErr)
			continue
		}
		if c.wantErr {
			continue
		}
		if gotPath != c.wantPath || gotQuery != c.wantQuery {
			t.Errorf("fastParse(%q) = (%q,%q), 期望 (%q,%q)", c.raw, gotPath, gotQuery, c.wantPath, c.wantQuery)
		}
	}
}

// TestGapF_FromHex 覆盖 fromHex 的数字/大小写/非法分支。
func TestGapF_FromHex(t *testing.T) {
	cases := []struct {
		in   byte
		want byte
		ok   bool
	}{
		{'0', 0, true},
		{'9', 9, true},
		{'a', 10, true},
		{'f', 15, true},
		{'A', 10, true},
		{'F', 15, true},
		{'g', 0, false},
		{'%', 0, false},
		{' ', 0, false},
	}
	for _, c := range cases {
		got, ok := fromHex(c.in)
		if ok != c.ok || got != c.want {
			t.Errorf("fromHex(%q) = (%d,%v), 期望 (%d,%v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

// TestGapF_UrlParseError 覆盖 urlParseError.Error 方法。
func TestGapF_UrlParseError(t *testing.T) {
	if got := errMissingScheme.Error(); got != "missing protocol scheme" {
		t.Errorf("errMissingScheme.Error() = %q", got)
	}
	if got := errInvalidEscape.Error(); got != "invalid URL escape" {
		t.Errorf("errInvalidEscape.Error() = %q", got)
	}
	if !strings.Contains((&urlParseError{msg: "自定义"}).Error(), "自定义") {
		t.Error("urlParseError.Error 应返回 msg")
	}
}

// TestGapF_Headers_AuthScheme 覆盖 GetAuthScheme 的无空格分支。
func TestGapF_Headers_AuthScheme(t *testing.T) {
	// 无空格单个 token：SplitN 返回单元素，取 parts[0]
	h := Headers{"Authorization": "TokenOnly"}
	if got := h.GetAuthScheme(); got != "TokenOnly" {
		t.Errorf("单个 token scheme 应为 TokenOnly，实际 %q", got)
	}
	// 空值分支
	if got := (Headers{}).GetAuthScheme(); got != "" {
		t.Errorf("空 Authorization 应返回空，实际 %q", got)
	}
	// 常规 Bearer（回归确认）
	h2 := Headers{"Authorization": "Bearer abc"}
	if got := h2.GetAuthScheme(); got != "Bearer" {
		t.Errorf("Bearer scheme 解析失败，实际 %q", got)
	}
}

// TestGapF_ParseCookies_Gaps 覆盖 ParseCookies 的空段跳过分支。
func TestGapF_ParseCookies_Gaps(t *testing.T) {
	// 连续分号产生空段，应跳过
	c := ParseCookies("a=1;;b=2")
	if len(c) != 2 || c.Get("a") != "1" || c.Get("b") != "2" {
		t.Errorf("空段应跳过，实际 %v", c)
	}
	// 纯分隔符与空白：无有效 cookie
	c2 := ParseCookies(" ; ")
	if len(c2) != 0 {
		t.Errorf("纯分隔符应返回空，实际 %v", c2)
	}
	// 段间空白段跳过
	c3 := ParseCookies("a=1; ;b=2")
	if len(c3) != 2 {
		t.Errorf("空白段应跳过，实际 %v", c3)
	}
}

// TestGapF_IsValidParamName 直接覆盖 isValidParamName 全部分支。
func TestGapF_IsValidParamName(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", false},     // 空名
		{"a", true},     // 单字母
		{"_a1", true},   // 下划线开头含数字
		{"a1_", true},   // 后续数字下划线
		{"A_Z09", true}, // 大写混合
		{"1a", false},   // 数字开头
		{"-a", false},   // 非字母下划线开头
		{"a-b", false},  // 后续非法字符 -
		{"a b", false},  // 后续空格
		{"a.b", false},  // 后续点号
		{"a@b", false},  // 后续 @
	}
	for _, c := range cases {
		if got := isValidParamName(c.in); got != c.want {
			t.Errorf("isValidParamName(%q) = %v, 期望 %v", c.in, got, c.want)
		}
	}
}

// TestGapF_CurlTokenize 覆盖 tokenizeCurl 的续行/转义/分隔符分支。
func TestGapF_CurlTokenize(t *testing.T) {
	// CRLF 续行跳过
	toks, err := tokenizeCurl("curl \\\r\n https://example.com")
	if err != nil {
		t.Fatalf("CRLF 续行不应报错: %v", err)
	}
	if len(toks) != 2 || toks[1] != "https://example.com" {
		t.Errorf("CRLF 续行切分不符，实际 %q", toks)
	}

	// 双引号内转义：\" \\ \$ \` 还原，\x 保留反斜杠原样
	toks2, err := tokenizeCurl("curl \"a\\\"b\\\\c\\$d`e\\fx\" https://x")
	if err != nil {
		t.Fatalf("双引号转义不应报错: %v", err)
	}
	if len(toks2) != 3 {
		t.Fatalf("双引号转义 token 数应为 3，实际 %q", toks2)
	}
	want := "a\"b\\c$d`e\\fx"
	if toks2[1] != want {
		t.Errorf("双引号转义结果不符，期望 %q，实际 %q", want, toks2[1])
	}

	// 非引号上下文转义空格：并入同一 token
	toks3, err := tokenizeCurl(`curl https://example.com/a\ b`)
	if err != nil {
		t.Fatalf("转义空格不应报错: %v", err)
	}
	if len(toks3) != 2 || toks3[1] != "https://example.com/a b" {
		t.Errorf("转义空格应并入 token，实际 %q", toks3)
	}

	// 行尾孤立反斜杠：按字面保留
	toks4, err := tokenizeCurl(`curl abc\`)
	if err != nil {
		t.Fatalf("行尾反斜杠不应报错: %v", err)
	}
	if len(toks4) != 2 || toks4[1] != `abc\` {
		t.Errorf("行尾反斜杠应字面保留，实际 %q", toks4)
	}

	// Tab 与 CR 作为分隔符
	toks5, err := tokenizeCurl("curl\thttps://x\rhttps://y")
	if err != nil {
		t.Fatalf("Tab/CR 分隔不应报错: %v", err)
	}
	if len(toks5) != 3 {
		t.Errorf("Tab/CR 应切分出 3 个 token，实际 %q", toks5)
	}

	// 纯空白：返回空 token 无错误
	toks6, err := tokenizeCurl("   ")
	if err != nil || len(toks6) != 0 {
		t.Errorf("纯空白应返回空 token 无错，实际 %q err=%v", toks6, err)
	}
}

// TestGapF_CurlParseTokens 覆盖 parseCurlTokens 的 flag 与 URL 分支。
func TestGapF_CurlParseTokens(t *testing.T) {
	// 空 token：覆盖 len==0 分支
	if _, err := parseCurlTokens(nil); err == nil {
		t.Error("空 token 应返回空命令错误")
	}

	// 带值 flag 缺参数
	if _, err := ParseCurl("curl --max-time"); err == nil {
		t.Error("--max-time 缺参数应报错")
	}

	// -d 缺参数
	if _, err := ParseCurl("curl https://x -d"); err == nil {
		t.Error("-d 缺参数应报错")
	}

	// --request 长形式小写方法转大写
	r, err := ParseCurl("curl --request delete https://x")
	if err != nil {
		t.Fatalf("--request 不应报错: %v", err)
	}
	if r.GetMethod() != "DELETE" {
		t.Errorf("--request 方法应转大写 DELETE，实际 %q", r.GetMethod())
	}

	// 未知长/短 flag 跳过，不影响 URL
	r2, err := ParseCurl("curl https://x --unknown-thing -Z")
	if err != nil {
		t.Fatalf("未知 flag 不应报错: %v", err)
	}
	if r2.GetUrl() != "https://x" {
		t.Errorf("未知 flag 应跳过，URL 实际 %q", r2.GetUrl())
	}

	// 多个位置 URL 只取第一个
	r3, err := ParseCurl("curl https://a https://b")
	if err != nil {
		t.Fatalf("多 URL 不应报错: %v", err)
	}
	if r3.GetUrl() != "https://a" {
		t.Errorf("多 URL 应取第一个，实际 %q", r3.GetUrl())
	}

	// --url 覆盖位置 URL
	r4, err := ParseCurl("curl https://a --url https://b")
	if err != nil {
		t.Fatalf("--url 不应报错: %v", err)
	}
	if r4.GetUrl() != "https://b" {
		t.Errorf("--url 应优先，实际 %q", r4.GetUrl())
	}

	// -G 无 body：保持 GET 且 URL 不变
	r5, err := ParseCurl("curl -G https://x")
	if err != nil {
		t.Fatalf("-G 无 body 不应报错: %v", err)
	}
	if r5.GetMethod() != "GET" || r5.GetUrl() != "https://x" {
		t.Errorf("-G 无 body 应保持 GET 与原 URL，实际 %v %q", r5.GetMethod(), r5.GetUrl())
	}
}

// TestGapF_CurlApplyHeader 覆盖 applyHeader 的空名与空值分支。
func TestGapF_CurlApplyHeader(t *testing.T) {
	// 空 header 名报错
	if err := applyHeader(Headers{}, ": value"); err == nil {
		t.Error("空 header 名应报错")
	}
	// 空值合法
	h := Headers{}
	if err := applyHeader(h, "X-A:"); err != nil {
		t.Fatalf("空值 header 不应报错: %v", err)
	}
	if h.Get("X-A") != "" {
		t.Errorf("空值 header 值应为空，实际 %q", h.Get("X-A"))
	}
	// 前后空格裁剪
	h2 := Headers{}
	if err := applyHeader(h2, "  K  :  V  "); err != nil {
		t.Fatalf("空格裁剪不应报错: %v", err)
	}
	if h2.Get("K") != "V" {
		t.Errorf("header 空格应裁剪，实际 %v", h2)
	}
}
