package request

import (
	"net/url"
	"strings"
)

type UrlParser struct {
	Url string
}

func NewUrlParser(url string) *UrlParser {
	return &UrlParser{Url: url}
}

func (x *UrlParser) Parse() ([]*HttpRequestPath, []*HttpParam, error) {
	u, err := url.Parse(x.Url)
	if err != nil {
		return nil, nil, err
	}
	// 解析URL路径，按照路径分隔符分割成有序的HttpRequestPath数组
	var paths []*HttpRequestPath
	if u.Path != "" {
		// 统一化路径，处理连续多个分隔符的情况
		normalizedPath := strings.Trim(u.Path, "/")
		// 处理连续的多个分隔符
		for strings.Contains(normalizedPath, "//") {
			normalizedPath = strings.ReplaceAll(normalizedPath, "//", "/")
		}

		// 分割路径并创建HttpRequestPath对象
		segments := strings.Split(normalizedPath, "/")
		for _, segment := range segments {
			if segment != "" {
				paths = append(paths, NewHttpRequestPath(segment))
			}
		}
	}

	// 解析URL查询参数
	var params []*HttpParam
	for key, values := range u.Query() {
		for _, value := range values {
			params = append(params, NewHttpParam(key, value))
		}
	}
	return paths, params, nil
}
