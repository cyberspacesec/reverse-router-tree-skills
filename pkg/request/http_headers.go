package request

import (
	"net/http"
	"strings"
)

// Headers 表示HTTP头字段的键值对集合。
// 根据HTTP规范，键名不区分大小写。
type Headers map[string]string

// GetContentType 从请求头中获取Content-Type
func (h Headers) GetContentType() string {
	return h.Get("Content-Type")
}

// Get 获取指定header的值（大小写不敏感）
func (h Headers) Get(key string) string {
	for k, v := range h {
		if strings.EqualFold(k, key) {
			return v
		}
	}
	return ""
}

// Set 设置header值
func (h Headers) Set(key, value string) {
	h[key] = value
}

// Has 检查是否包含指定header（大小写不敏感）
func (h Headers) Has(key string) bool {
	for k := range h {
		if strings.EqualFold(k, key) {
			return true
		}
	}
	return false
}

// GetAll 获取所有header键值对
func (h Headers) GetAll() map[string]string {
	result := make(map[string]string, len(h))
	for k, v := range h {
		result[k] = v
	}
	return result
}

// GetAccept 获取 Accept header
func (h Headers) GetAccept() string {
	return h.Get("Accept")
}

// GetAuthorization 获取 Authorization header
func (h Headers) GetAuthorization() string {
	return h.Get("Authorization")
}

// GetAuthScheme 获取 Authorization 的认证方案（如 Bearer、Basic、Token）
func (h Headers) GetAuthScheme() string {
	auth := h.GetAuthorization()
	if auth == "" {
		return ""
	}
	// Authorization: Bearer xxx, Basic xxx, Token xxx
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// GetXRequestedWith 获取 X-Requested-With header（常用于 AJAX 请求识别）
func (h Headers) GetXRequestedWith() string {
	return h.Get("X-Requested-With")
}

// GetXForwardedFor 获取 X-Forwarded-For header
func (h Headers) GetXForwardedFor() string {
	return h.Get("X-Forwarded-For")
}

// GetXApiVersion 获取 X-Api-Version header（某些 API 使用此 header 做版本路由）
func (h Headers) GetXApiVersion() string {
	return h.Get("X-Api-Version")
}

// GetAcceptLanguage 获取 Accept-Language header
func (h Headers) GetAcceptLanguage() string {
	return h.Get("Accept-Language")
}

// IsAjax 判断是否为 AJAX 请求
func (h Headers) IsAjax() bool {
	return h.GetXRequestedWith() == "XMLHttpRequest"
}

// String 实现fmt.Stringer接口
func (h Headers) String() string {
	if len(h) == 0 {
		return "{}"
	}

	result := "{"
	for key, value := range h {
		result += key + ": " + value + ", "
	}
	result = result[:len(result)-2] + "}"
	return result
}

// Cookies 表示 HTTP Cookie 键值对集合
type Cookies map[string]string

// ParseCookies 从 Cookie header 字符串中解析出键值对
// 格式: "name1=value1; name2=value2"
func ParseCookies(cookieHeader string) Cookies {
	cookies := make(Cookies)
	if cookieHeader == "" {
		return cookies
	}

	// 使用 http.ParseCookie 解析
	// 但先做简单的 split 处理，兼容性更好
	pairs := strings.Split(cookieHeader, ";")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			name := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])
			cookies[name] = value
		} else if len(kv) == 1 {
			// Cookie 没有值的标志
			name := strings.TrimSpace(kv[0])
			cookies[name] = ""
		}
	}

	return cookies
}

// Get 获取指定 cookie 值
func (c Cookies) Get(key string) string {
	return c[key]
}

// Has 检查是否包含指定 cookie
func (c Cookies) Has(key string) bool {
	_, ok := c[key]
	return ok
}

// GetAll 获取所有 cookie 键值对
func (c Cookies) GetAll() map[string]string {
	result := make(map[string]string, len(c))
	for k, v := range c {
		result[k] = v
	}
	return result
}

// String 返回 cookie 字符串表示
func (c Cookies) String() string {
	if len(c) == 0 {
		return "{}"
	}
	result := "{"
	for k, v := range c {
		result += k + "=" + v + ", "
	}
	result = result[:len(result)-2] + "}"
	return result
}

// 确保 Headers 和 Cookies 可以用于标准 http.Header 转换
// ToHttpHeader 将 Headers 转换为标准 http.Header
func (h Headers) ToHttpHeader() http.Header {
	header := make(http.Header)
	for k, v := range h {
		header.Set(k, v)
	}
	return header
}
