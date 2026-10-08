package request

import "testing"

func TestCanonicalHost(t *testing.T) {
	cases := []struct{ raw, scheme, want string }{
		{"http://example.com/a", "", "example.com"},
		{"http://example.com:80/a", "", "example.com"},
		{"https://example.com:443/a", "", "example.com"},
		{"http://example.com:8080/a", "", "example.com:8080"},
		{"https://example.com:8443/a", "", "example.com:8443"},
		{"http://user:pass@example.com/a", "", "example.com"},
		{"http://example.com#frag", "", "example.com"},
		{"HTTP://Example.COM/a", "", "example.com"},
		{"http://[::1]:80/a", "", "[::1]"},
		{"http://[::1]:8080/a", "", "[::1]:8080"},
		{"example.com:80", "http", "example.com"},
		{"example.com:443", "https", "example.com"},
		{"example.com:443", "http", "example.com:443"},
		{"localhost", "", "localhost"},
		{"localhost:80", "http", "localhost"},
		{"/api/users", "", ""},
		{"api/users", "", ""},
		{"", "", ""},
	}
	for _, c := range cases {
		if got := CanonicalHost(c.raw, c.scheme); got != c.want {
			t.Errorf("CanonicalHost(%q, %q)=%q want %q", c.raw, c.scheme, got, c.want)
		}
	}
}

func TestResolveDotSegmentsAndMatrix(t *testing.T) {
	cases := []struct {
		raw  string
		want []string
	}{
		{"/api/./users/../admin/list", []string{"api", "admin", "list"}},
		{"/api/users/../admin", []string{"api", "admin"}},
		{"/../admin", []string{"admin"}},
		{"/api/%2e%2e/admin", []string{"admin"}},
		{"/api/%252e%252e/admin", []string{"api", "%2e%2e", "admin"}},
		{"/api/users;jsessionid=ABC/list", []string{"api", "users", "list"}},
		{"/api/;only", []string{"api"}},
	}
	for _, c := range cases {
		paths, _, err := NewUrlParser(c.raw).Parse()
		if err != nil {
			t.Errorf("parse %q: %v", c.raw, err)
			continue
		}
		got := make([]string, len(paths))
		for i, p := range paths {
			got[i] = p.Path
		}
		ReleasePaths(paths)
		if len(got) != len(c.want) {
			t.Errorf("%q segs=%v want %v", c.raw, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%q seg[%d]=%q want %q", c.raw, i, got[i], c.want[i])
			}
		}
	}
}

func TestCanonicalParamNameAndNoise(t *testing.T) {
	cases := map[string]string{
		"Page":           "page",
		"ids[]":          "ids",
		"ids[0]":         "ids",
		"filter[status]": "filter.status",
		"a[b][c]":        "a.b.c",
		"utm_source":     "utm_source",
	}
	for in, want := range cases {
		if got := CanonicalParamName(in); got != want {
			t.Errorf("CanonicalParamName(%q)=%q want %q", in, got, want)
		}
	}
	if !IsNoiseParam("utm_source") || !IsNoiseParam("fbclid") || !IsNoiseParam("_ga") {
		t.Fatal("追踪参数应判为噪声")
	}
	if IsNoiseParam("page") {
		t.Fatal("page 不是噪声")
	}

	_, params, err := NewUrlParser("/api/list?utm_source=a&page=1&ids[]=1&ids[]=2&filter[status]=open&fbclid=x").Parse()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]int{}
	for _, p := range params {
		got[p.Name]++
	}
	if _, ok := got["utm_source"]; ok {
		t.Error("utm_source 应被丢弃")
	}
	if _, ok := got["fbclid"]; ok {
		t.Error("fbclid 应被丢弃")
	}
	if got["page"] != 1 || got["ids"] != 2 || got["filter.status"] != 1 {
		t.Errorf("参数归并结果=%v", got)
	}
}
