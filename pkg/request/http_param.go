package request

// 表示一个http请求参数
type HttpParam struct {
	Name  string
	Value string
}

func NewHttpParam(name string, value string) *HttpParam {
	return &HttpParam{Name: name, Value: value}
}

func (x *HttpParam) GetValue() string {
	return x.Value
}

func (x *HttpParam) SetValue(value string) {
	x.Value = value
}

func (x *HttpParam) GetName() string {
	return x.Name
}

func (x *HttpParam) SetName(name string) {
	x.Name = name
}

func (x *HttpParam) String() string {
	return x.Name + "=" + x.Value
}
