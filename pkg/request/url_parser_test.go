package request

import "testing"

func TestUrlParserParse(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		wantPaths  []*HttpRequestPath
		wantParams []*HttpParam
		wantErr    bool
	}{
		{
			name: "simple path",
			url:  "http://example.com/api/users",
			wantPaths: []*HttpRequestPath{
				NewHttpRequestPath("api"),
				NewHttpRequestPath("users"),
			},
			wantParams: []*HttpParam{},
			wantErr:    false,
		},
		{
			name: "path with trailing slash",
			url:  "http://example.com/api/users/",
			wantPaths: []*HttpRequestPath{
				NewHttpRequestPath("api"),
				NewHttpRequestPath("users"),
			},
			wantParams: []*HttpParam{},
			wantErr:    false,
		},
		{
			name: "path with query parameters",
			url:  "http://example.com/api/users?id=123&filter=active",
			wantPaths: []*HttpRequestPath{
				NewHttpRequestPath("api"),
				NewHttpRequestPath("users"),
			},
			wantParams: []*HttpParam{
				NewHttpParam("id", "123"),
				NewHttpParam("filter", "active"),
			},
			wantErr: false,
		},
		{
			name:       "empty path",
			url:        "http://example.com",
			wantPaths:  []*HttpRequestPath{},
			wantParams: []*HttpParam{},
			wantErr:    false,
		},
		{
			name: "multiple consecutive slashes",
			url:  "http://example.com//api///users//",
			wantPaths: []*HttpRequestPath{
				NewHttpRequestPath("api"),
				NewHttpRequestPath("users"),
			},
			wantParams: []*HttpParam{},
			wantErr:    false,
		},
		{
			name:       "invalid URL",
			url:        "://invalid-url",
			wantPaths:  nil,
			wantParams: nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewUrlParser(tt.url)
			gotPaths, gotParams, err := parser.Parse()

			if (err != nil) != tt.wantErr {
				t.Errorf("UrlParser.Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if len(gotPaths) != len(tt.wantPaths) {
				t.Errorf("UrlParser.Parse() paths length = %d, want %d", len(gotPaths), len(tt.wantPaths))
				return
			}

			for i, path := range gotPaths {
				if path.GetPath() != tt.wantPaths[i].GetPath() {
					t.Errorf("UrlParser.Parse() path[%d] = %s, want %s", i, path.GetPath(), tt.wantPaths[i].GetPath())
				}
			}

			if len(gotParams) != len(tt.wantParams) {
				t.Errorf("UrlParser.Parse() params length = %d, want %d", len(gotParams), len(tt.wantParams))
				return
			}

			// Check if all parameters are present (order may vary)
			paramMap := make(map[string]string)
			for _, param := range gotParams {
				paramMap[param.GetName()] = param.GetValue()
			}

			for _, wantParam := range tt.wantParams {
				if val, ok := paramMap[wantParam.GetName()]; !ok || val != wantParam.GetValue() {
					t.Errorf("UrlParser.Parse() missing or incorrect param %s=%s", wantParam.GetName(), wantParam.GetValue())
				}
			}
		})
	}
}

// 测试路径参数检测
func TestHttpRequestPath_ParamDetection(t *testing.T) {
	// key=value 格式应该被识别为路径参数
	p1 := NewHttpRequestPath("action=delete")
	if !p1.IsPathParam() {
		t.Error("'action=delete' 应该被识别为路径参数")
	}
	if p1.GetPathParamKey() != "action" {
		t.Errorf("参数键应该是 'action'，实际: '%s'", p1.GetPathParamKey())
	}
	if p1.GetPathParamValue() != "delete" {
		t.Errorf("参数值应该是 'delete'，实际: '%s'", p1.GetPathParamValue())
	}

	// 普通路径不应该被识别为路径参数
	p2 := NewHttpRequestPath("users")
	if p2.IsPathParam() {
		t.Error("'users' 不应该被识别为路径参数")
	}

	// 数字路径不应该被识别为路径参数
	p3 := NewHttpRequestPath("123")
	if p3.IsPathParam() {
		t.Error("'123' 不应该被识别为路径参数")
	}

	// 以数字开头的 key 不应该被识别（不合法的参数名）
	p4 := NewHttpRequestPath("1abc=value")
	if p4.IsPathParam() {
		t.Error("'1abc=value' 不应该被识别为路径参数（数字开头的key）")
	}

	// 空值
	p5 := NewHttpRequestPath("flag=")
	if !p5.IsPathParam() {
		t.Error("'flag=' 应该被识别为路径参数")
	}
	if p5.GetPathParamKey() != "flag" {
		t.Errorf("参数键应该是 'flag'，实际: '%s'", p5.GetPathParamKey())
	}

	// 下划线开头的 key
	p6 := NewHttpRequestPath("_type=json")
	if !p6.IsPathParam() {
		t.Error("'_type=json' 应该被识别为路径参数")
	}

	// 包含等号但不是合法参数名
	p7 := NewHttpRequestPath("user@example.com")
	if p7.IsPathParam() {
		t.Error("'user@example.com' 不应该被识别为路径参数")
	}
}
