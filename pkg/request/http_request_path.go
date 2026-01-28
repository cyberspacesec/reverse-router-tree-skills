package request

// http请求路径
type HttpRequestPath struct {
	Path string
}

func NewHttpRequestPath(path string) *HttpRequestPath {
	return &HttpRequestPath{Path: path}
}

func (x *HttpRequestPath) GetPath() string {
	return x.Path
}

func (x *HttpRequestPath) SetPath(path string) {
	x.Path = path
}

func (x *HttpRequestPath) String() string {
	return x.Path
}
