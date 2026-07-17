package request

import "strings"

// fastParseURLPathAndQuery 轻量解析 URL，仅提取 path 与 query，避开 net/url.Parse 的全功能开销。
//
// 行为对齐 net/url 的 path/query 提取（但不构造 *url.URL 结构体，不解析 scheme/host 细节）：
//   - 含 scheme:// 的 URL（http://h/p?q）→ 跳过 scheme+host，从首个 / 取 path
//   - host 段后无 / 但有 ?（http://h?q）→ path 为空，提取 query
//   - // 开头无 scheme（//auth/p）→ 视为 authority 形式，跳过 //authority 取 path
//   - 普通 /path 或 path → 整体为 path
//
// 返回归一化后的 pathStr（去首尾斜杠、连续斜杠压缩，未做 %xx 解码），
// 以及原始 query 串（不含 ?）。纯路径（无 scheme 无 query 无 //）走零分配快路径。
// 对空 scheme（如 "://x"，对齐 net/url 的 "missing protocol scheme"）返回 errMissingScheme。
func fastParseURLPathAndQuery(raw string) (pathStr, queryStr string, err error) {
	if raw == "" {
		return "", "", nil
	}

	rest := raw

	// 处理 scheme：// 或裸 //authority
	if schemeEnd := strings.Index(rest, "://"); schemeEnd >= 0 {
		// scheme 为空（"://..."）→ 对齐 net/url 报 "missing protocol scheme"
		if schemeEnd == 0 {
			return "", "", errMissingScheme
		}
		// scheme://host[:port]/path?query → 跳过 scheme://host 段
		hostStart := schemeEnd + 3
		if hostStart >= len(rest) {
			return "", "", nil
		}
		// host 段结束于首个 / 或 ? 或行尾
		slashIdx := strings.IndexByte(rest[hostStart:], '/')
		qIdx := strings.IndexByte(rest[hostStart:], '?')
		// 取 host 段后的起点
		if slashIdx < 0 && qIdx < 0 {
			// 无 path 无 query（如 http://x.com）
			return "", "", nil
		}
		// host 段后首个分隔符位置（相对 raw）
		var sepPos int
		if slashIdx < 0 {
			sepPos = hostStart + qIdx // 仅 query
		} else if qIdx < 0 {
			sepPos = hostStart + slashIdx // 仅 path
		} else {
			// 取更靠前的
			if slashIdx < qIdx {
				sepPos = hostStart + slashIdx
			} else {
				sepPos = hostStart + qIdx
			}
		}
		rest = raw[sepPos:]
	} else if len(rest) >= 2 && rest[0] == '/' && rest[1] == '/' {
		// //authority/path?query（scheme-relative，对齐 net/url）
		authStart := 2
		if authStart >= len(rest) {
			return "", "", nil
		}
		slashIdx := strings.IndexByte(rest[authStart:], '/')
		qIdx := strings.IndexByte(rest[authStart:], '?')
		if slashIdx < 0 && qIdx < 0 {
			return "", "", nil
		}
		var sepPos int
		if slashIdx < 0 {
			sepPos = authStart + qIdx
		} else if qIdx < 0 {
			sepPos = authStart + slashIdx
		} else if slashIdx < qIdx {
			sepPos = authStart + slashIdx
		} else {
			sepPos = authStart + qIdx
		}
		rest = raw[sepPos:]
	}

	// 此时 rest 形如 "/path?query" 或 "?query" 或 "/path" 或 ""
	// 分离 path 与 query（首个 ? 分隔）
	if qIdx := strings.IndexByte(rest, '?'); qIdx >= 0 {
		pathStr = rest[:qIdx]
		queryStr = rest[qIdx+1:]
	} else {
		pathStr = rest
	}

	pathStr = normalizePathFast(pathStr)
	return pathStr, queryStr, nil
}

// normalizePathFast 去首尾斜杠并压缩连续斜杠。
// 对齐 url_parser.go 原 Trim + Contains//ReplaceAll 循环，但无连续斜杠时零分配返回原串。
func normalizePathFast(p string) string {
	// 去首尾斜杠
	for len(p) > 0 && p[0] == '/' {
		p = p[1:]
	}
	for len(p) > 0 && p[len(p)-1] == '/' {
		p = p[:len(p)-1]
	}
	if p == "" {
		return ""
	}
	// 无连续斜杠则零分配返回
	if !strings.Contains(p, "//") {
		return p
	}
	// 有连续斜杠才分配（罕见路径）
	var b strings.Builder
	b.Grow(len(p))
	prevSlash := false
	for i := 0; i < len(p); i++ {
		c := p[i]
		if c == '/' {
			if prevSlash {
				continue
			}
			prevSlash = true
		} else {
			prevSlash = false
		}
		b.WriteByte(c)
	}
	return b.String()
}

// fastSplitPathSegments 将归一化后的 path 按 / 切分为段，复用传入的 out slice 避免分配。
// 空段（normalizePathFast 后理论上不会出现）自动跳过。
func fastSplitPathSegments(pathStr string, out []string) []string {
	if pathStr == "" {
		return out[:0]
	}
	out = out[:0]
	start := 0
	for i := 0; i <= len(pathStr); i++ {
		if i == len(pathStr) || pathStr[i] == '/' {
			if i > start {
				out = append(out, pathStr[start:i])
			}
			start = i + 1
		}
	}
	return out
}

// fastDecodeSegment 对单个路径段做 %xx 解码，行为对齐 net/url 的 PathUnescape。
//
// 仅对含 % 的段解码（无 % 零分配返回原串）。
// 对 '+' 原样保留（与 PathUnescape 一致；QueryUnescape 才把 '+' 解码为空格）。
// 非法 %xx（如 %ZZ、末尾孤立 %）返回 error——对齐 net/url.Parse 的严格契约
// （拒绝畸形输入，UrlParser.Parse 透传 error，FindRouteNode 据此报错）。
func fastDecodeSegment(seg string) (string, error) {
	if !strings.ContainsRune(seg, '%') {
		return seg, nil
	}
	var b strings.Builder
	b.Grow(len(seg))
	for i := 0; i < len(seg); i++ {
		c := seg[i]
		if c == '%' {
			// 需要 2 个 hex 位
			if i+2 >= len(seg) {
				return "", errInvalidEscape
			}
			hi, ok1 := fromHex(seg[i+1])
			lo, ok2 := fromHex(seg[i+2])
			if !ok1 || !ok2 {
				return "", errInvalidEscape
			}
			b.WriteByte(hi<<4 | lo)
			i += 2
			continue
		}
		b.WriteByte(c)
	}
	return b.String(), nil
}

// errInvalidEscape 非法 %xx 转义错误（对齐 net/url 的 "invalid URL escape"）。
var errInvalidEscape = &urlParseError{msg: "invalid URL escape"}

// errMissingScheme 空 scheme 错误（对齐 net/url 的 "missing protocol scheme"，
// 如 "://x"）。
var errMissingScheme = &urlParseError{msg: "missing protocol scheme"}

// fromHex 单字节 hex 解码，非法返回 (0,false)。
func fromHex(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}

// urlParseError URL 解析错误（对齐 net/url 的错误格式）。
type urlParseError struct {
	msg string
}

func (e *urlParseError) Error() string { return e.msg }
