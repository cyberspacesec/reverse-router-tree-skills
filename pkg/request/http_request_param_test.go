package request

import "testing"

// TestHttpRequestParam_Accessors 覆盖 NewHttpRequestParam 构造与全部读写方法。
func TestHttpRequestParam_Accessors(t *testing.T) {
	p := NewHttpRequestParam("page", "1", true)
	if p.GetName() != "page" {
		t.Errorf("GetName = %q, want page", p.GetName())
	}
	if p.GetValue() != "1" {
		t.Errorf("GetValue = %q, want 1", p.GetValue())
	}
	if !p.IsRequired() {
		t.Error("IsRequired = false, want true")
	}
	if p.String() != "page=1" {
		t.Errorf("String = %q, want page=1", p.String())
	}

	// setter 往返
	p.SetName("size")
	p.SetValue("20")
	p.SetRequired(false)
	if p.GetName() != "size" || p.GetValue() != "20" || p.IsRequired() {
		t.Errorf("setter 后状态异常: name=%q value=%q required=%v", p.GetName(), p.GetValue(), p.IsRequired())
	}
	if p.String() != "size=20" {
		t.Errorf("setter 后 String = %q, want size=20", p.String())
	}
}

// TestHttpRequestParam_EmptyValue 覆盖空值边界。
func TestHttpRequestParam_EmptyValue(t *testing.T) {
	p := NewHttpRequestParam("", "", false)
	if p.String() != "=" {
		t.Errorf("空值 String = %q, want =", p.String())
	}
}
