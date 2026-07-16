package request

import (
	"fmt"
	"strings"
)

// curlParseError 表示 curl 命令解析过程中发生的错误。
type curlParseError struct {
	// message 错误描述
	message string
}

// Error 实现 error 接口。
func (e *curlParseError) Error() string {
	return e.message
}

// newCurlParseError 构造一个新的 curl 解析错误。
func newCurlParseError(format string, args ...any) error {
	return &curlParseError{message: fmt.Sprintf(format, args...)}
}

// ParseCurl 将一行 curl 命令解析为 HttpRequest。
//
// 支持：
//   - 单引号/双引号包裹的参数，反斜杠转义
//   - 反斜杠行尾续行（\<换行> 被跳过）
//   - -X/--request 指定方法，以及 -XPOST 紧凑形式
//   - -H/--header 添加请求头（格式 "Key: Value"）
//   - -d/--data/--data-raw/--data-binary/--data-ascii 指定请求体，
//     无显式 -X 时默认 POST，无 Content-Type 时默认 application/x-www-form-urlencoded
//   - --compressed/-s/-k/-L/-i/--insecure 等无害 flag 跳过
//
// 零外部依赖，手写 shell-token 切分以精确处理引号与转义。
func ParseCurl(curl string) (*HttpRequest, error) {
	tokens, err := tokenizeCurl(curl)
	if err != nil {
		return nil, err
	}
	return parseCurlTokens(tokens)
}

// tokenizeCurl 对 curl 命令字符串进行 shell 风格的 token 切分。
//
// 处理规则：
//   - 反斜杠行尾续行（"\\\n" 或 "\\\r\n"）：跳过，不产生 token
//   - 单引号：内部所有字符原样保留，直到下一个单引号
//   - 双引号：内部保留字面值，仅识别反斜杠转义（保留 $ 等变量符号原样，不展开）
//   - 反斜杠转义：在非引号或双引号上下文中转义下一个字符
//   - 未闭合引号报错
//   - 连续空白作为分隔符
func tokenizeCurl(curl string) ([]string, error) {
	var tokens []string
	var current strings.Builder
	inToken := false // 当前是否正在构造一个 token

	i := 0
	n := len(curl)
	for i < n {
		c := curl[i]

		// 处理反斜杠续行：反斜杠后紧跟换行符（\n 或 \r\n）时跳过
		if c == '\\' {
			if i+1 < n && curl[i+1] == '\n' {
				i += 2
				continue
			}
			if i+2 < n && curl[i+1] == '\r' && curl[i+2] == '\n' {
				i += 3
				continue
			}
		}

		// 空白字符作为 token 分隔符（仅在未进入引号时）
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			if inToken {
				tokens = append(tokens, current.String())
				current.Reset()
				inToken = false
			}
			i++
			continue
		}

		// 非空白字符：开始一个新 token
		inToken = true

		switch c {
		case '\'':
			// 单引号：原样保留内部内容，直到闭合的单引号
			i++
			closed := false
			for i < n {
				if curl[i] == '\'' {
					closed = true
					i++
					break
				}
				current.WriteByte(curl[i])
				i++
			}
			if !closed {
				return nil, newCurlParseError("未闭合的单引号")
			}
		case '"':
			// 双引号：保留字面值，识别反斜杠转义
			i++
			closed := false
			for i < n {
				ch := curl[i]
				if ch == '"' {
					closed = true
					i++
					break
				}
				if ch == '\\' && i+1 < n {
					// 双引号内仅对特定字符做转义处理，其余反斜杠按字面保留
					next := curl[i+1]
					if next == '"' || next == '\\' || next == '$' || next == '`' {
						current.WriteByte(next)
						i += 2
						continue
					}
				}
				current.WriteByte(ch)
				i++
			}
			if !closed {
				return nil, newCurlParseError("未闭合的双引号")
			}
		case '\\':
			// 非引号上下文中的反斜杠转义：保留下一个字符的字面值
			if i+1 < n {
				current.WriteByte(curl[i+1])
				i += 2
			} else {
				// 行尾孤立反斜杠：按字面保留
				current.WriteByte(c)
				i++
			}
		default:
			current.WriteByte(c)
			i++
		}
	}

	// 收尾：最后一个 token
	if inToken {
		tokens = append(tokens, current.String())
	}

	return tokens, nil
}

// parseCurlTokens 解析 token 序列并构造 HttpRequest。
func parseCurlTokens(tokens []string) (*HttpRequest, error) {
	if len(tokens) == 0 {
		return nil, newCurlParseError("空命令")
	}

	// 首 token 必须是 curl
	if tokens[0] != "curl" {
		return nil, newCurlParseError("非 curl 命令：%q", tokens[0])
	}

	headers := Headers{}
	method := "GET"
	var url string
	var body []byte
	hasBody := false
	urlSet := false

	// 无害 flag 直接跳过（不消费后续参数）
	harmlessFlags := map[string]bool{
		"--compressed": true,
		"-s":           true,
		"--silent":     true,
		"-k":           true,
		"--insecure":   true,
		"-L":           true,
		"--location":   true,
		"-i":           true,
		"--include":    true,
		"-S":           true,
		"--show-error": true,
		"-f":           true,
		"--fail":       true,
		"--fail-with-body": true,
		"-v":           true,
		"--verbose":    true,
		"-q":           true,
	}

	i := 1
	for i < len(tokens) {
		tok := tokens[i]

		// 处理 -XPOST / -XPOST 紧凑形式（flag 与值连写）
		if strings.HasPrefix(tok, "-X") && len(tok) > 2 {
			method = strings.ToUpper(tok[2:])
			i++
			continue
		}

		switch tok {
		case "-X", "--request":
			if i+1 >= len(tokens) {
				return nil, newCurlParseError("%s 缺少参数", tok)
			}
			i++
			method = strings.ToUpper(tokens[i])
		case "-H", "--header":
			if i+1 >= len(tokens) {
				return nil, newCurlParseError("%s 缺少参数", tok)
			}
			i++
			if err := applyHeader(headers, tokens[i]); err != nil {
				return nil, err
			}
		case "-d", "--data", "--data-raw", "--data-binary", "--data-ascii":
			if i+1 >= len(tokens) {
				return nil, newCurlParseError("%s 缺少参数", tok)
			}
			i++
			body = []byte(tokens[i])
			hasBody = true
		default:
			// 无害 flag 直接跳过
			if harmlessFlags[tok] {
				// 不消费参数
				break
			}
			// 未知的长 flag（--xxx）整体跳过
			if strings.HasPrefix(tok, "--") {
				break
			}
			// 未知短 flag（-x）整体跳过
			if strings.HasPrefix(tok, "-") && len(tok) > 1 {
				break
			}
			// 非 flag token 视为 URL（取第一个）
			if !urlSet {
				url = tok
				urlSet = true
			}
		}
		i++
	}

	if !urlSet {
		return nil, newCurlParseError("缺少 URL")
	}

	// -d 隐含 POST：未显式指定方法且存在请求体时默认 POST
	if hasBody && method == "GET" {
		method = "POST"
	}
	// -d 隐含表单 Content-Type：用户未显式设置时补默认值
	if hasBody && headers.GetContentType() == "" {
		headers.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	return NewHttpRequest(url, headers, method, body), nil
}

// applyHeader 解析单个 "Key: Value" 格式的 header 并写入 headers。
// 非法格式（缺少冒号分隔）返回错误。
func applyHeader(headers Headers, raw string) error {
	idx := strings.Index(raw, ":")
	if idx < 0 {
		return newCurlParseError("非法 header 格式（缺少冒号分隔）：%q", raw)
	}
	key := strings.TrimSpace(raw[:idx])
	value := strings.TrimSpace(raw[idx+1:])
	if key == "" {
		return newCurlParseError("非法 header 格式（空的 header 名）：%q", raw)
	}
	headers.Set(key, value)
	return nil
}
