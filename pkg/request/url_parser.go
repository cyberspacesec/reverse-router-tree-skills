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
				// 路径段规范化
				segment = normalizePathSegment(segment)
				if segment != "" {
					paths = append(paths, NewHttpRequestPath(segment))
				}
			}
		}
	}

	// 解析URL查询参数
	var params []*HttpParam
	for key, values := range u.Query() {
		// 参数名统一小写（HTTP参数名不区分大小写是常见约定）
		normalizedKey := strings.ToLower(key)
		for _, value := range values {
			// URL解码已在 url.Query() 中自动完成
			params = append(params, NewHttpParam(normalizedKey, value))
		}
	}
	return paths, params, nil
}

// normalizePathSegment 规范化路径段
// 处理以下边界条件：
//   - URL解码：将 %XX 编码还原为原始字符
//   - 路径遍历：忽略 . 和 .. 段（安全考虑，不应出现在路由树中）
//   - 尾部斜杠：已在 Parse 中通过 Trim 处理
func normalizePathSegment(segment string) string {
	// URL解码
	decoded, err := url.PathUnescape(segment)
	if err == nil {
		segment = decoded
	}

	// 忽略当前目录标记
	if segment == "." {
		return ""
	}

	// 忽略上级目录标记（路径遍历，安全风险）
	if segment == ".." {
		return ""
	}

	return segment
}
