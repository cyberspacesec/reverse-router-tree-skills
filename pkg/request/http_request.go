package request

// 表示一个http请求
type HttpRequest struct {
	// 一次请求的唯一标识
	RequestId string
	// 请求的URL地址
	Url string
	// 请求头信息
	Headers Headers
	// HTTP方法（GET、POST等）
	Method string
	// 请求体内容
	Body []byte
}

func NewHttpRequest(url string, headers Headers, method string, body []byte) *HttpRequest {
	return &HttpRequest{Url: url, Headers: headers, Method: method, Body: body}
}

func (x *HttpRequest) GetUrl() string {
	return x.Url
}

func (x *HttpRequest) SetUrl(url string) {
	x.Url = url
}

func (x *HttpRequest) GetHeaders() Headers {
	return x.Headers
}

func (x *HttpRequest) SetHeaders(headers Headers) {
	x.Headers = headers
}

func (x *HttpRequest) GetMethod() string {
	return x.Method
}

func (x *HttpRequest) SetMethod(method string) {
	x.Method = method
}

func (x *HttpRequest) GetBody() []byte {
	return x.Body
}

func (x *HttpRequest) SetBody(body []byte) {
	x.Body = body
}

func (x *HttpRequest) String() string {
	return "Url: " + x.Url + "\n" +
		"Headers: " + x.Headers.String() + "\n" +
		"Method: " + x.Method + "\n" +
		"Body: " + string(x.Body)
}
