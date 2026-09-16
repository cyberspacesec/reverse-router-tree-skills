package request

import (
	"testing"
)

// 解析 URL 路径段的忠实断言：确切断言各段、isPathParam、路径参数 key/value。
func parseSegs(t *testing.T, url string) ([]string, []bool, error) {
	t.Helper()
	paths, _, err := NewUrlParser(url).Parse()
	if err != nil {
		return nil, nil, err
	}
	segs := make([]string, len(paths))
	flags := make([]bool, len(paths))
	for i, p := range paths {
		segs[i] = p.GetPath()
		flags[i] = p.IsPathParam()
	}
	ReleasePaths(paths)
	return segs, flags, nil
}

func assertParity(t *testing.T, url string) {
	t.Helper()
	if _, _, err := NewUrlParser(url).Parse(); err != nil {
		t.Fatalf("parsing %q: %v", url, err)
	}
}

type hostCase struct {
	raw  string
	want string
}

func TestExtractHostExtremes(t *testing.T) {
	cases := []hostCase{
		{"http://Example.COM:8080/api/users/1", "Example.COM:8080"},
		{"HTTPS://example.com/", "example.com"},
		{"//example.com/path?x=1", "example.com"},
		{"http://192.168.0.1:8443/x", "192.168.0.1:8443"},
		{"http://[::1]:9090/api", "[::1]:9090"},
		{"http://user:pass@example.com/x", "user:pass@example.com"},
		{"http://example.com/x#frag", "example.com"},
		{"http://example.com?x=1", "example.com"},
		{"http://example.com", "example.com"},
		{"//example.com", "example.com"},
		{"/api/users/1", ""},
		{"api/users", ""},
		{"", ""},
		{"about:blank", ""},
	}
	for i, c := range cases {
		got := ExtractHost(c.raw)
		if got != c.want {
			t.Errorf("[A%02d] ExtractHost(%q)=%q, want %q", i+1, c.raw, got, c.want)
		}
	}
}

func TestParsePathStructure(t *testing.T) {
	cases := []struct {
		raw      string
		wantSegs []string
		wantPP   []bool
	}{
		{"/", []string{}, []bool{}},
		{"/api", []string{"api"}, []bool{false}},
		{"api/users", []string{"api", "users"}, []bool{false, false}},
		{"//host/api/users", []string{"api", "users"}, []bool{false, false}},
		{"http://h/api/users", []string{"api", "users"}, []bool{false, false}},
		{"/api//users", []string{"api", "users"}, []bool{false, false}},
		{"/api/users/", []string{"api", "users"}, []bool{false, false}},
		{"/api/users.json", []string{"api", "users.json"}, []bool{false, false}},
		{"/api/filter=1", []string{"api", "filter=1"}, []bool{false, true}},
		{"/a/../b", []string{"a", "b"}, []bool{false, false}},
		{"/a/./b", []string{"a", "b"}, []bool{false, false}},
		{"/a b/c", []string{"a b", "c"}, []bool{false, false}},
	}
	for i, c := range cases {
		segs, pp, err := parseSegs(t, c.raw)
		if err != nil {
			t.Errorf("[B%02d] parse %q err=%v", i+1, c.raw, err)
			continue
		}
		if len(segs) != len(c.wantSegs) {
			t.Errorf("[B%02d] %q segs=%v want=%v", i+1, c.raw, segs, c.wantSegs)
			continue
		}
		for j := range segs {
			if segs[j] != c.wantSegs[j] || pp[j] != c.wantPP[j] {
				t.Errorf("[B%02d] %q seg%d=%q pp=%v want %q/%v", i+1, c.raw, j, segs[j], pp[j], c.wantSegs[j], c.wantPP[j])
			}
		}
	}
}

func TestParseQueryParameters(t *testing.T) {
	cases := []struct {
		raw  string
		want int
	}{
		{"/api?page=1&size=20", 2},
		{"/api?PAGE=1", 1},
		{"/api?empty=", 1},
		{"/api?flag", 1}, // 无 = 号的裸键，net/url ParseQuery 视为 1 个空值参数
		{"/api?a=1&a=2", 2},
		{"/api?a=hello%20world", 1},
		{"/api?k=v v", 1},
		{"/api?#frag", 0},    // fragment 剥离，query 为空
		{"/api?x=1#frag", 1}, // fragment 不产生伪查询参数
	}
	for i, c := range cases {
		_, params, err := NewUrlParser(c.raw).Parse()
		if err != nil {
			t.Errorf("[D%02d] parse %q err=%v", i+1, c.raw, err)
			continue
		}
		if len(params) != c.want {
			t.Errorf("[D%02d] %q params=%v want %d", i+1, c.raw, len(params), c.want)
		}
	}
}

func TestParseIncludedMethods(t *testing.T) {
	for _, m := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "TRACE", "CONNECT", "PROPFIND", "custom"} {
		req := NewHttpRequest("/api/x", nil, m, nil)
		if req.Method != m {
			t.Errorf("method %q not preserved", m)
		}
	}
}

// 确保解析不 panic 且有干净行为。
func TestParseResourceBoundaries(t *testing.T) {
	uris := []string{
		"http://h/" + repeatStr("a/", 2000),
		"http://h/x?" + repeatStr("k=v&", 2000),
		"http://h/" + repeatStr("%20", 3000),
	}
	for i, u := range uris {
		if _, _, err := NewUrlParser(u).Parse(); err != nil {
			t.Errorf("[R%02d] large uri err=%v", i+1, err)
		}
	}
}

func repeatStr(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}
