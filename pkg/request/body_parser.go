package request

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// BodyParser 解析HTTP请求体，将其转换为统一的 HttpParam 列表。
//
// 支持的 Content-Type：
//   - application/x-www-form-urlencoded：表单编码，复用 net/url.ParseQuery
//   - application/json：扁平化为 name=value（嵌套用 parent.child 点号表示，数组用索引）
//   - multipart/form-data：解析表单字段值（不含文件内容）
//
// 参数名统一小写，与查询参数处理保持一致。
type BodyParser struct {
	// MaxParams 解析出的最大参数数量，防止恶意超大 body 导致参数爆炸。
	// 0 表示不限制。
	MaxParams int
}

// NewBodyParser 创建默认配置的 BodyParser。
func NewBodyParser() *BodyParser {
	return &BodyParser{MaxParams: 1000}
}

// Parse 根据 Content-Type 解析请求体。
// 返回的 HttpParam 列表与查询参数同构，可直接复用 processParams。
// 若 Content-Type 不支持或 body 为空，返回空列表和 nil 错误。
func (p *BodyParser) Parse(contentType string, body []byte) ([]*HttpParam, error) {
	if len(body) == 0 {
		return nil, nil
	}

	mime := normalizeContentType(contentType)
	if mime == "" {
		return nil, nil
	}

	var params []*HttpParam
	var err error
	switch {
	case mime == "application/x-www-form-urlencoded":
		params, err = p.parseFormUrlencoded(body)
	case mime == "application/json":
		params, err = p.parseJSON(body)
	case strings.HasPrefix(mime, "multipart/form-data"):
		params, err = p.parseMultipart(contentType, body)
	default:
		// 不支持的类型（如 text/plain、application/octet-stream）不解析
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	// 参数名小写化（与查询参数一致）
	for _, param := range params {
		param.Name = strings.ToLower(param.Name)
	}

	return p.truncate(params), nil
}

// truncate 按上限截断参数数量，防止参数爆炸。
func (p *BodyParser) truncate(params []*HttpParam) []*HttpParam {
	if p.MaxParams > 0 && len(params) > p.MaxParams {
		return params[:p.MaxParams]
	}
	return params
}

// parseFormUrlencoded 解析 application/x-www-form-urlencoded 格式。
func (p *BodyParser) parseFormUrlencoded(body []byte) ([]*HttpParam, error) {
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, fmt.Errorf("解析表单编码失败: %w", err)
	}

	var params []*HttpParam
	for name, vals := range values {
		for _, v := range vals {
			params = append(params, NewHttpParam(name, v))
		}
	}
	return params, nil
}

// parseJSON 解析 application/json 格式，扁平化为参数列表。
// 嵌套对象用点号连接（user.name），数组用索引（items.0）。
func (p *BodyParser) parseJSON(body []byte) ([]*HttpParam, error) {
	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %w", err)
	}

	var params []*HttpParam
	flattenJSON("", data, &params)
	return params, nil
}

// flattenJSON 递归扁平化 JSON 数据。
func flattenJSON(prefix string, data interface{}, params *[]*HttpParam) {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, val := range v {
			newPrefix := key
			if prefix != "" {
				newPrefix = prefix + "." + key
			}
			flattenJSON(newPrefix, val, params)
		}
	case []interface{}:
		for i, val := range v {
			newPrefix := fmt.Sprintf("%s.%d", prefix, i)
			flattenJSON(newPrefix, val, params)
		}
	default:
		// 标量值：string/number/bool/null
		if prefix != "" {
			*params = append(*params, NewHttpParam(prefix, formatJSONScalar(data)))
		}
	}
}

// formatJSONScalar 将 JSON 标量值格式化为字符串。
func formatJSONScalar(data interface{}) string {
	if data == nil {
		return ""
	}
	switch v := data.(type) {
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case float64:
		// JSON 数字统一解析为 float64，整数无小数部分时不显示 .0
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%g", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// parseMultipart 解析 multipart/form-data 格式。
// 仅提取文本字段值，文件字段记录文件名作为值（不读取文件内容）。
func (p *BodyParser) parseMultipart(contentType string, body []byte) ([]*HttpParam, error) {
	boundary := extractBoundary(contentType)
	if boundary == "" {
		return nil, fmt.Errorf("multipart/form-data 缺少 boundary 参数")
	}

	var params []*HttpParam
	// 用 \r\n 分割各部分
	segments := strings.Split(string(body), "--"+boundary)
	for _, seg := range segments {
		seg = strings.Trim(seg, "\r\n")
		if seg == "" || seg == "--" {
			continue
		}
		// 每个 part 由 header 和 body 用 \r\n\r\n 分隔
		headerBody := strings.SplitN(seg, "\r\n\r\n", 2)
		if len(headerBody) < 2 {
			continue
		}
		headerPart := headerBody[0]
		valuePart := headerBody[1]

		name := extractContentDispositionName(headerPart)
		if name == "" {
			continue
		}

		// 若是文件字段（有 filename），值用文件名
		if filename := extractContentDispositionField(headerPart, "filename"); filename != "" {
			params = append(params, NewHttpParam(name, filename))
		} else {
			params = append(params, NewHttpParam(name, valuePart))
		}
	}
	return params, nil
}

// extractBoundary 从 Content-Type 中提取 boundary 参数值。
func extractBoundary(contentType string) string {
	return extractParam(contentType, "boundary")
}

// normalizeContentType 提取 Content-Type 的主类型（去掉参数和空格，转小写）。
// "application/json; charset=utf-8" → "application/json"
func normalizeContentType(contentType string) string {
	ct := strings.TrimSpace(contentType)
	if ct == "" {
		return ""
	}
	if idx := strings.Index(ct, ";"); idx >= 0 {
		ct = strings.TrimSpace(ct[:idx])
	}
	return strings.ToLower(ct)
}

// extractContentDispositionName 从 multipart part header 提取 name 字段。
func extractContentDispositionName(headerPart string) string {
	return extractContentDispositionField(headerPart, "name")
}

// extractContentDispositionField 从 Content-Disposition header 提取指定字段。
func extractContentDispositionField(headerPart, field string) string {
	for _, line := range strings.Split(headerPart, "\r\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(strings.ToLower(line), "content-disposition:") {
			continue
		}
		// 形如: Content-Disposition: form-data; name="field"; filename="a.txt"
		rest := line[len("content-disposition:"):]
		for _, part := range strings.Split(rest, ";") {
			part = strings.TrimSpace(part)
			prefix := field + "="
			if strings.HasPrefix(part, prefix) {
				val := strings.Trim(part[len(prefix):], "\"")
				return val
			}
		}
	}
	return ""
}

// extractParam 从 "key1=val1; key2=val2" 形式的字符串提取指定 key 的值。
func extractParam(s, key string) string {
	for _, part := range strings.Split(s, ";") {
		part = strings.TrimSpace(part)
		prefix := key + "="
		if strings.HasPrefix(part, prefix) {
			return strings.Trim(part[len(prefix):], "\"")
		}
	}
	return ""
}
