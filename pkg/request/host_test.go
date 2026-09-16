package request

import "testing"

func TestExtractHost(t *testing.T) {
	cases := map[string]string{
		"http://example.com/api/users": "example.com",
		"https://example.com:8443/api": "example.com:8443",
		"//example.com/path":           "example.com",
		"http://example.com?x=1":       "example.com",
		"/api/users/1":                 "",
		"api/users/1":                  "",
		"":                             "",
	}
	for raw, want := range cases {
		if got := ExtractHost(raw); got != want {
			t.Errorf("ExtractHost(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestHttpRequestHostAccessors(t *testing.T) {
	req := NewHttpRequest("/api", nil, "GET", nil)
	req.SetHost("Example.COM:443")
	if got := req.GetHost(); got != "Example.COM:443" {
		t.Fatalf("GetHost() = %q", got)
	}
}
