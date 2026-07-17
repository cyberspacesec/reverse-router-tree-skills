package request

import (
	"net/url"
	"testing"
)

// TestFastParse_Equivalence 用 net/url.Parse 作 oracle 验证 path+query 等价。
// pathStr 对比用"未解码 raw path 归一化"（fast 返回未解码 path，解码在切分后做）。
func TestFastParse_Equivalence(t *testing.T) {
	cases := []string{
		"/api/users/123",
		"/api/users/123?page=1&size=10",
		"http://api.example.com/users/123",
		"http://api.example.com/users/123?page=1",
		"https://x.com/api/list?tag=go&tag=web",
		"/api/users/%E7%94%A8%E6%88%B7", // 中文（未解码）
		"/api//users///123",             // 连续斜杠
		"/api/users/123/",               // 尾部斜杠
		"//api/users",                   // 首部斜杠
		"http://x.com",                  // 无 path
		"http://x.com?q=1",              // 仅 query 无 path
		"/api/users/123?a=1&b=2&c=3",    // 多参数
		"relative/path",                 // 相对路径无 scheme
		"/api/users/100%25",             // %25 = %
		"://invalid-url",                // 空 scheme（对齐 net/url 报错）
	}
	for _, raw := range cases {
		u, err := url.Parse(raw)
		// fast 与 net/url 报错行为必须一致
		_, _, fastErr := fastParseURLPathAndQuery(raw)
		if (err != nil) != (fastErr != nil) {
			t.Errorf("error mismatch %q: net/url_err=%v fast_err=%v", raw, err, fastErr)
			continue
		}
		if err != nil {
			continue // 都报错，等价
		}
		// fast 返回未解码的归一化 path；oracle 用 u.EscapedPath()（未解码 raw path）归一化
		wantPath := normalizePathFast(u.EscapedPath())
		wantQuery := u.RawQuery

		gotPath, gotQuery, _ := fastParseURLPathAndQuery(raw)
		if gotPath != wantPath {
			t.Errorf("path mismatch %q: got=%q want=%q", raw, gotPath, wantPath)
		}
		if gotQuery != wantQuery {
			t.Errorf("query mismatch %q: got=%q want=%q", raw, gotQuery, wantQuery)
		}
	}
}

// TestFastParse_VsOriginalParser 用原 UrlParser.Parse 的最终路径段作 oracle，
// 验证 fast 解析 + 切分 + 解码链路产出与原实现完全一致（这才是行为等价）。
func TestFastParse_VsOriginalParser(t *testing.T) {
	cases := []string{
		"/api/users/123",
		"/api/users/123?page=1&size=10",
		"http://api.example.com/users/123",
		"/api/users/%E7%94%A8%E6%88%B7",
		"/api//users///123",
		"/api/users/123/",
		"//api/users",
		"/api/users/100%25",
		"/api/./users/../final", // . 与 .. 过滤
	}
	for _, raw := range cases {
		// 原解析器
		origParser := NewUrlParser(raw)
		origPaths, origParams, err := origParser.Parse()
		// origPaths/fastPaths 来自 Pool，本轮比较后立即归还（避免污染后续用例的池对象，
		// 也避免 defer-in-loop 捕获循环变量的问题）
		// fast 解析 + 切分 + 解码 + 过滤
		pathStr, queryStr, _ := fastParseURLPathAndQuery(raw)
		segs := make([]string, 0, 8)
		segs = fastSplitPathSegments(pathStr, segs)
		var fastPaths []*HttpRequestPath
		var fastErr error
		for _, seg := range segs {
			decoded, derr := fastDecodeSegment(seg)
			if derr != nil {
				fastErr = derr
				break
			}
			if decoded == "" || decoded == "." || decoded == ".." {
				continue
			}
			fastPaths = append(fastPaths, NewHttpRequestPath(decoded))
		}
		// 两者要么都报错，要么都不报错
		if (err != nil) != (fastErr != nil) {
			ReleasePaths(origPaths)
			ReleasePaths(fastPaths)
			t.Errorf("error mismatch %q: orig_err=%v fast_err=%v", raw, err, fastErr)
			continue
		}
		if err != nil {
			ReleasePaths(origPaths)
			ReleasePaths(fastPaths)
			continue // 都报错，等价
		}
		// 比较路径段
		if len(fastPaths) != len(origPaths) {
			ReleasePaths(origPaths)
			ReleasePaths(fastPaths)
			t.Errorf("path count mismatch %q: fast=%d orig=%d", raw, len(fastPaths), len(origPaths))
			continue
		}
		for i := range fastPaths {
			if fastPaths[i].Path != origPaths[i].Path {
				t.Errorf("path seg[%d] mismatch %q: fast=%q orig=%q", i, raw, fastPaths[i].Path, origPaths[i].Path)
			}
		}
		// query 参数数量（fast 用 url.ParseQuery，与原一致）
		fastParamCount := 0
		if queryStr != "" {
			vals, _ := url.ParseQuery(queryStr)
			for _, vs := range vals {
				fastParamCount += len(vs)
			}
		}
		if fastParamCount != len(origParams) {
			t.Errorf("param count mismatch %q: fast=%d orig=%d", raw, fastParamCount, len(origParams))
		}
		ReleasePaths(origPaths)
		ReleasePaths(fastPaths)
	}
}

// TestFastSplitSegments 验证路径段切分（含复用 out slice）。
func TestFastSplitSegments(t *testing.T) {
	out := make([]string, 0, 8)
	cases := []struct {
		path string
		want []string
	}{
		{"api/users/123", []string{"api", "users", "123"}},
		{"api", []string{"api"}},
		{"", []string{}},
		{"api/users", []string{"api", "users"}},
	}
	for _, c := range cases {
		got := fastSplitPathSegments(c.path, out)
		if len(got) != len(c.want) {
			t.Errorf("split(%q) len=%d want=%d", c.path, len(got), len(c.want))
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("split(%q)[%d]=%q want=%q", c.path, i, got[i], c.want[i])
			}
		}
	}
}

// TestFastDecodeSegment 验证 %xx 解码等价 net/url 的 PathUnescape。
func TestFastDecodeSegment(t *testing.T) {
	cases := []struct {
		in        string
		want      string
		wantError bool
	}{
		{"123", "123", false},               // 无 %
		{"%E7%94%A8%E6%88%B7", "用户", false}, // 中文
		{"abc%20def", "abc def", false},      // 空格
		{"100%25", "100%", false},            // %25 = %
		{"%2F", "/", false},                  // %2F = /
		{"%xx", "", true},                    // 非法 %xx 报错
		{"%", "", true},                      // 末尾孤立 % 报错
		{"a+b", "a+b", false},                // + 原样保留（PathUnescape 行为）
	}
	for _, c := range cases {
		got, err := fastDecodeSegment(c.in)
		if c.wantError {
			if err == nil {
				t.Errorf("decode(%q) 应报错，实际 got=%q err=nil", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("decode(%q) 不应报错，实际 err=%v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("decode(%q)=%q want=%q", c.in, got, c.want)
		}
	}
}

// TestFastParse_ZeroAlloc 验证无 query 的纯路径零分配（快路径）。
func TestFastParse_ZeroAlloc(t *testing.T) {
	allocs := testing.AllocsPerRun(100, func() {
		_, _, _ = fastParseURLPathAndQuery("/api/users/123")
	})
	if allocs != 0 {
		t.Errorf("纯路径应零分配，实际 %v allocs/op", allocs)
	}
}
