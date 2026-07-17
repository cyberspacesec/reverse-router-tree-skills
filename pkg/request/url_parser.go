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
	// 用轻量 fast 解析提取 path + query，避开 net/url.Parse 的全功能开销
	// （net/url.Parse 占 CPU 约 40%，详见吞吐量基线）。
	// 行为与原实现（u.Path + u.Query()）等价，见 TestFastParse_VsOriginalParser。
	pathStr, queryStr, err := fastParseURLPathAndQuery(x.Url)
	if err != nil {
		return nil, nil, err
	}

	// 解析路径段（paths 容器从池复用，避免 append 扩容分配）
	paths := AcquirePaths()
	if pathStr != "" {
		// segments 用临时栈上 slice 切分（≤8 段零分配；超出由 make 扩容，罕见）
		segments := make([]string, 0, 8)
		segments = fastSplitPathSegments(pathStr, segments)
		for _, seg := range segments {
			// %xx 解码（对齐 net/url 的 PathUnescape，非法 %xx 返回 error 透传）
			decoded, err := fastDecodeSegment(seg)
			if err != nil {
				// 出错也要归还已取的 paths 容器，避免池泄漏
				ReleasePaths(paths)
				return nil, nil, err
			}
			// 过滤 . 和 .. 段（路径遍历安全）
			if decoded == "" || decoded == "." || decoded == ".." {
				continue
			}
			paths = append(paths, NewHttpRequestPath(decoded))
		}
	}

	// 解析查询参数（query 解析仍用标准库 ParseQuery，格式复杂非主热点）
	var params []*HttpParam
	if queryStr != "" {
		values, err := url.ParseQuery(queryStr)
		if err != nil {
			return nil, nil, err
		}
		for key, vals := range values {
			// 参数名统一小写（HTTP参数名不区分大小写是常见约定）
			normalizedKey := strings.ToLower(key)
			for _, v := range vals {
				// URL解码已在 url.ParseQuery 中完成
				params = append(params, NewHttpParam(normalizedKey, v))
			}
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
