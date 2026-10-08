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
		decodedSegs := make([]string, 0, len(segments))
		for _, seg := range segments {
			// %xx 解码（对齐 net/url 的 PathUnescape，非法 %xx 返回 error 透传）
			decoded, err := fastDecodeSegment(seg)
			if err != nil {
				// 出错也要归还已取的 paths 容器，避免池泄漏
				ReleasePaths(paths)
				return nil, nil, err
			}
			decodedSegs = append(decodedSegs, decoded)
		}
		// 解码后再消解：. 丢弃，.. 弹出上一段（RFC 3986 remove_dot_segments）。
		// 必须先解码，否则 %2e%2e 不会被当成 ..。
		for _, seg := range resolveDotSegments(decodedSegs) {
			// 矩阵参数（;jsessionid=ABC）不参与路由：只保留 ; 前的段名。
			if i := strings.IndexByte(seg, ';'); i >= 0 {
				seg = seg[:i]
			}
			if seg == "" {
				continue
			}
			paths = append(paths, NewHttpRequestPath(seg))
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
			// 参数名统一小写，数组/嵌套括号收成点号（ids[]→ids，filter[status]→filter.status）。
			normalizedKey := CanonicalParamName(key)
			if normalizedKey == "" || IsNoiseParam(normalizedKey) {
				continue
			}
			for _, v := range vals {
				// URL解码已在 url.ParseQuery 中完成
				params = append(params, NewHttpParam(normalizedKey, v))
			}
		}
	}
	return paths, params, nil
}

// resolveDotSegments 按 RFC 3986 remove_dot_segments 消解路径段。
// "." 丢弃；".." 弹出上一段（栈空时忽略，不产生越界段）；其余原样保留。
func resolveDotSegments(segs []string) []string {
	out := make([]string, 0, len(segs))
	for _, seg := range segs {
		switch seg {
		case "", ".":
			continue
		case "..":
			if len(out) > 0 {
				out = out[:len(out)-1]
			}
		default:
			out = append(out, seg)
		}
	}
	return out
}

// CanonicalParamName 把查询参数名收成稳定键：小写，数组下标与空括号去掉，
// 嵌套括号改成点号。
//
//	ids[]          → ids
//	ids[0]         → ids
//	filter[status] → filter.status
//	a[b][c]        → a.b.c
func CanonicalParamName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c != '[' {
			b.WriteByte(c)
			continue
		}
		end := strings.IndexByte(name[i:], ']')
		if end < 0 {
			b.WriteByte(c)
			continue
		}
		inner := name[i+1 : i+end]
		i += end
		if inner == "" || isDigits(inner) {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('.')
		}
		b.WriteString(inner)
	}
	return strings.Trim(b.String(), ".")
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// noiseParamExact 默认丢弃的追踪/会话参数（精确名，已小写）。
var noiseParamExact = map[string]struct{}{
	"fbclid": {}, "gclid": {}, "dclid": {}, "msclkid": {},
	"yclid": {}, "twclid": {}, "_ga": {}, "_gl": {}, "_gid": {},
	"spm": {}, "scm": {},
}

// IsNoiseParam 判断参数名是否为默认噪声（utm_* 前缀，或精确名单）。
// 名字应已由 CanonicalParamName 规范化；这里再做一次小写以容忍直接调用。
func IsNoiseParam(name string) bool {
	name = strings.ToLower(name)
	if name == "" {
		return false
	}
	if strings.HasPrefix(name, "utm_") {
		return true
	}
	_, ok := noiseParamExact[name]
	return ok
}
