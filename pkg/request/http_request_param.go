package request

// HttpRequestParam 表示HTTP请求参数
type HttpRequestParam struct {
	// 参数名称
	Name string
	// 参数值
	Value string
	// 是否为必需参数
	Required bool
}

// NewHttpRequestParam 创建一个新的HTTP请求参数对象
func NewHttpRequestParam(name string, value string, required bool) *HttpRequestParam {
	return &HttpRequestParam{
		Name:     name,
		Value:    value,
		Required: required,
	}
}

// GetName 获取参数名称
func (x *HttpRequestParam) GetName() string {
	return x.Name
}

// SetName 设置参数名称
func (x *HttpRequestParam) SetName(name string) {
	x.Name = name
}

// GetValue 获取参数值
func (x *HttpRequestParam) GetValue() string {
	return x.Value
}

// SetValue 设置参数值
func (x *HttpRequestParam) SetValue(value string) {
	x.Value = value
}

// IsRequired 检查参数是否必需
func (x *HttpRequestParam) IsRequired() bool {
	return x.Required
}

// SetRequired 设置参数是否必需
func (x *HttpRequestParam) SetRequired(required bool) {
	x.Required = required
}

// String 返回参数的字符串表示
func (x *HttpRequestParam) String() string {
	return x.Name + "=" + x.Value
}
