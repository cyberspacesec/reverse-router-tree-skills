// Package request 提供处理HTTP请求和相关组件的工具。
package request

import "strings"

// Headers 表示HTTP头字段的键值对集合。
// 根据HTTP规范，键名不区分大小写。
type Headers map[string]string

// GetContentType 从请求头中获取Content-Type
// 它处理HTTP头字段名称的大小写不敏感问题。
func (h Headers) GetContentType() string {
	// 处理大小写问题，HTTP 头字段名称不区分大小写
	for k, v := range h {
		if strings.EqualFold(k, "Content-Type") {
			return v
		}
	}
	return ""
}

// String 实现fmt.Stringer接口，将Headers格式化为字符串
// 以"{key1: value1, key2: value2}"的格式返回可读的字符串表示。
// 对于空的headers，返回"{}"。
func (h Headers) String() string {
	if len(h) == 0 {
		return "{}"
	}

	result := "{"
	for key, value := range h {
		result += key + ": " + value + ", "
	}
	// 移除最后的逗号和空格
	result = result[:len(result)-2] + "}"
	return result
}
