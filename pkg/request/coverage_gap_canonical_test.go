package request

import "testing"

func TestGapCanonical_HostEdges(t *testing.T) {
	// ExtractHost 拿到空 authority：http:// 后面什么都没有。
	if got := CanonicalHost("http://", ""); got != "" {
		t.Errorf("空 authority 应为空，got %q", got)
	}
	// userinfo 剥完只剩空白。
	if got := CanonicalHost("   @   ", ""); got != "" {
		t.Errorf("纯空白 host 应为空，got %q", got)
	}
	// IPv6 缺右括号：不拆端口，原样小写返回。
	if got := CanonicalHost("[::1", ""); got != "[::1" {
		t.Errorf("缺右括号应原样返回，got %q", got)
	}
	// 只有方括号、没有端口。
	if got := CanonicalHost("[::1]", ""); got != "[::1]" {
		t.Errorf("无端口 IPv6 应原样返回，got %q", got)
	}
	// 方括号后只有一个冒号，端口为空。
	if got := CanonicalHost("[::1]:", ""); got != "[::1]:" {
		t.Errorf("空端口应保留原串，got %q", got)
	}
	// 裸 IPv6（多个冒号、无方括号）不拆端口。
	if got := CanonicalHost("::1", ""); got != "::1" {
		t.Errorf("裸 IPv6 不应拆端口，got %q", got)
	}
	// 非 http/https 的默认端口判断走 default。
	if got := CanonicalHost("example.com:80", "ftp"); got != "example.com:80" {
		t.Errorf("ftp 不应去掉 :80，got %q", got)
	}
}

func TestGapCanonical_URLScheme(t *testing.T) {
	if got := URLScheme("HTTP://example.com/a"); got != "http" {
		t.Errorf("scheme 应小写，got %q", got)
	}
	if got := URLScheme("/api/users"); got != "" {
		t.Errorf("无 scheme 应为空，got %q", got)
	}
	if got := URLScheme(""); got != "" {
		t.Errorf("空串应为空，got %q", got)
	}
}

func TestGapCanonical_ParamNameEdges(t *testing.T) {
	if got := CanonicalParamName("   "); got != "" {
		t.Errorf("空白参数名应为空，got %q", got)
	}
	if got := CanonicalParamName("foo[bar"); got != "foo[bar" {
		t.Errorf("缺右括号应原样保留，got %q", got)
	}
	if got := CanonicalParamName("[status]"); got != "status" {
		t.Errorf("前导括号应去掉点号，got %q", got)
	}
	if isDigits("") {
		t.Error("空串不是数字")
	}
	if IsNoiseParam("") {
		t.Error("空参数名不是噪声")
	}
	if !IsNoiseParam("UTM_Source") {
		t.Error("utm_ 前缀应大小写不敏感")
	}
}
