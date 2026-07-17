package request

import (
	"strings"
	"sync"
)

// http请求路径
type HttpRequestPath struct {
	Path string
	// 如果路径段包含 key=value 格式，则解析为路径参数
	// 例如 "action=delete" → pathParamKey="action", pathParamValue="delete"
	pathParamKey   string
	pathParamValue string
	isPathParam    bool
}

// httpRequestPathPool 复用 *HttpRequestPath，避免每个路径段一次堆分配。
// Parse 从池取，调用方用 ReleasePaths 归还。
var httpRequestPathPool = sync.Pool{
	New: func() any {
		return &HttpRequestPath{}
	},
}

// AcquireHttpRequestPath 从池取出一个已重置的 HttpRequestPath 并设置 Path。
func AcquireHttpRequestPath(path string) *HttpRequestPath {
	hp := httpRequestPathPool.Get().(*HttpRequestPath)
	hp.Path = path
	hp.pathParamKey = ""
	hp.pathParamValue = ""
	hp.isPathParam = false
	hp.detectPathParam()
	return hp
}

// ReleasePath 归还单个 HttpRequestPath 到池。归还后禁止再使用该指针。
func ReleasePath(hp *HttpRequestPath) {
	if hp == nil {
		return
	}
	hp.Path = ""
	hp.pathParamKey = ""
	hp.pathParamValue = ""
	hp.isPathParam = false
	httpRequestPathPool.Put(hp)
}

// ReleasePaths 批量归还 paths slice 中的所有 *HttpRequestPath。
// 归还后 paths slice 不再可用，调用方不应再访问其中的元素。
func ReleasePaths(paths []*HttpRequestPath) {
	for i := range paths {
		hp := paths[i]
		hp.Path = ""
		hp.pathParamKey = ""
		hp.pathParamValue = ""
		hp.isPathParam = false
		httpRequestPathPool.Put(hp)
	}
}

// NewHttpRequestPath 创建并初始化一个 HttpRequestPath。
// 注意：返回的对象来自 sync.Pool，用完必须调用 ReleasePath 归还，否则池无效。
// 若需脱离池管理的独立对象，请显式构造 &HttpRequestPath{...}。
func NewHttpRequestPath(path string) *HttpRequestPath {
	return AcquireHttpRequestPath(path)
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
