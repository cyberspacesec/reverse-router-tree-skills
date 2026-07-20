package request

import "testing"

// TestReleasePath_Nil 覆盖 nil 守卫分支（不应 panic）。
func TestReleasePath_Nil(t *testing.T) {
	// nil 入参直接返回，不 panic
	ReleasePath(nil)
}

// TestReleasePath_Reuse 覆盖正常归还后池可复用，字段被清零。
func TestReleasePath_Reuse(t *testing.T) {
	hp := AcquireHttpRequestPath("/users/123")
	if hp == nil {
		t.Fatal("AcquireHttpRequestPath 返回 nil")
	}
	ReleasePath(hp)

	// 再次取出应得到已清零的实例
	hp2 := AcquireHttpRequestPath("/x")
	if hp2.Path != "/x" {
		t.Errorf("复用后 Path = %q, want /x", hp2.Path)
	}
	if hp2.isPathParam {
		t.Error("复用后 isPathParam 应为 false")
	}
	ReleasePath(hp2)
}
