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
