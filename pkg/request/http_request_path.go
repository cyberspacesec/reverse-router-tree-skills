package request

import "strings"

// http请求路径
type HttpRequestPath struct {
	Path string
	// 如果路径段包含 key=value 格式，则解析为路径参数
	// 例如 "action=delete" → pathParamKey="action", pathParamValue="delete"
	pathParamKey   string
	pathParamValue string
	isPathParam    bool
}

func NewHttpRequestPath(path string) *HttpRequestPath {
	hp := &HttpRequestPath{Path: path}
	hp.detectPathParam()
	return hp
}

// detectPathParam 检测路径段是否为 key=value 格式的路径参数
func (x *HttpRequestPath) detectPathParam() {
	// 检查是否包含 = 号
	eqIndex := strings.Index(x.Path, "=")
	if eqIndex <= 0 {
		// 没有 = 号，或者 = 号在开头，不是路径参数
		return
	}

	key := x.Path[:eqIndex]
	value := x.Path[eqIndex+1:]

	// key 不能为空，且应该是合法的参数名（字母/下划线开头，包含字母数字下划线）
	if !isValidParamName(key) {
		return
	}

	x.pathParamKey = key
	x.pathParamValue = value
	x.isPathParam = true
}

// isValidParamName 检查是否为合法的路径参数名
func isValidParamName(name string) bool {
	if len(name) == 0 {
		return false
	}
	// 第一个字符必须是字母或下划线
	first := name[0]
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_') {
		return false
	}
	// 后续字符可以是字母、数字、下划线
	for _, c := range name[1:] {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}

func (x *HttpRequestPath) GetPath() string {
	return x.Path
}

func (x *HttpRequestPath) SetPath(path string) {
	x.Path = path
	x.detectPathParam()
}

func (x *HttpRequestPath) String() string {
	return x.Path
}

// IsPathParam 判断该路径段是否为路径参数（key=value格式）
func (x *HttpRequestPath) IsPathParam() bool {
	return x.isPathParam
}

// GetPathParamKey 获取路径参数的键名
// 仅当 IsPathParam() 返回 true 时有效
func (x *HttpRequestPath) GetPathParamKey() string {
	return x.pathParamKey
}

// GetPathParamValue 获取路径参数的值
// 仅当 IsPathParam() 返回 true 时有效
func (x *HttpRequestPath) GetPathParamValue() string {
	return x.pathParamValue
}
