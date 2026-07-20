package node

import "testing"

// TestRequestHeaderNode_String 覆盖 Header 名称节点的字符串表示。
func TestRequestHeaderNode_String(t *testing.T) {
	n := NewRequestHeaderNode("Accept")
	if got := n.String(); got != "Accept [Header]" {
		t.Errorf("HeaderNode String = %q, want 'Accept [Header]'", got)
	}
}

// TestRequestHeaderValueNode_StringAndMetric 覆盖 Header 值节点的 String 与 GetValueMetric。
func TestRequestHeaderValueNode_StringAndMetric(t *testing.T) {
	v := NewRequestHeaderValueNode("Accept", "application/json")
	if got := v.String(); got != "Accept: application/json [HeaderValue]" {
		t.Errorf("HeaderValueNode String = %q, want 'Accept: application/json [HeaderValue]'", got)
	}
	if v.GetValueMetric() == nil {
		t.Error("GetValueMetric 返回 nil")
	}
	v.ObserveValue("application/json")
	v.ObserveValue("text/plain")
	if v.GetValueMetric().GetUniqueValueCount() != 2 {
		t.Errorf("GetValueMetric uniqueCount = %d, want 2", v.GetValueMetric().GetUniqueValueCount())
	}
}

// TestRequestHeaderValueNode_Accessors 覆盖 GetHeaderName/GetHeaderValue/IsMatch。
func TestRequestHeaderValueNode_Accessors(t *testing.T) {
	v := NewRequestHeaderValueNode("Accept", "application/json")
	if v.GetHeaderName() != "Accept" {
		t.Errorf("GetHeaderName = %q, want Accept", v.GetHeaderName())
	}
	if v.GetHeaderValue() != "application/json" {
		t.Errorf("GetHeaderValue = %q, want application/json", v.GetHeaderValue())
	}
	if !v.IsMatch("application/json") {
		t.Error("IsMatch(application/json) 应为 true")
	}
	if v.IsMatch("text/plain") {
		t.Error("IsMatch(text/plain) 应为 false")
	}
}
