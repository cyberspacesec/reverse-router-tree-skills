package node

import "testing"

// TestRequestCookieNode_String 覆盖 Cookie 名称节点的字符串表示。
func TestRequestCookieNode_String(t *testing.T) {
	n := NewRequestCookieNode("lang")
	if got := n.String(); got != "lang [Cookie]" {
		t.Errorf("CookieNode String = %q, want 'lang [Cookie]'", got)
	}
}

// TestRequestCookieValueNode_StringAndMetric 覆盖 Cookie 值节点的 String 与 GetValueMetric。
func TestRequestCookieValueNode_StringAndMetric(t *testing.T) {
	v := NewRequestCookieValueNode("lang", "zh-CN")
	if got := v.String(); got != "lang=zh-CN [CookieValue]" {
		t.Errorf("CookieValueNode String = %q, want 'lang=zh-CN [CookieValue]'", got)
	}
	if v.GetValueMetric() == nil {
		t.Error("GetValueMetric 返回 nil")
	}
	v.ObserveValue("zh-CN")
	v.ObserveValue("en-US")
	if v.GetValueMetric().GetUniqueValueCount() != 2 {
		t.Errorf("GetValueMetric uniqueCount = %d, want 2", v.GetValueMetric().GetUniqueValueCount())
	}
}

// TestRequestCookieValueNode_Accessors 覆盖 GetCookieName/GetCookieValue/IsMatch。
func TestRequestCookieValueNode_Accessors(t *testing.T) {
	v := NewRequestCookieValueNode("lang", "zh-CN")
	if v.GetCookieName() != "lang" {
		t.Errorf("GetCookieName = %q, want lang", v.GetCookieName())
	}
	if v.GetCookieValue() != "zh-CN" {
		t.Errorf("GetCookieValue = %q, want zh-CN", v.GetCookieValue())
	}
	if !v.IsMatch("zh-CN") {
		t.Error("IsMatch(zh-CN) 应为 true")
	}
	if v.IsMatch("en-US") {
		t.Error("IsMatch(en-US) 应为 false")
	}
}
