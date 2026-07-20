package request

import "testing"

// TestCurlParseError_Error 覆盖 Error() 返回 message。
func TestCurlParseError_Error(t *testing.T) {
	err := newCurlParseError("解析失败：%s", "缺 URL")
	if got := err.Error(); got != "解析失败：缺 URL" {
		t.Errorf("Error() = %q, want 解析失败：缺 URL", got)
	}
}

// TestCurlParseError_NoArgs 覆盖无格式化参数场景。
func TestCurlParseError_NoArgs(t *testing.T) {
	err := newCurlParseError("空命令")
	if got := err.Error(); got != "空命令" {
		t.Errorf("Error() = %q, want 空命令", got)
	}
}
