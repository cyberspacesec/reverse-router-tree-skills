package generator

import "testing"

// TestFirstNonEmpty 覆盖 firstNonEmpty 的空切片与多元素含空串分支。
func TestFirstNonEmpty(t *testing.T) {
	cases := []struct {
		vals []string
		want string
	}{
		{nil, ""},
		{[]string{}, ""},
		{[]string{"", "", ""}, ""},
		{[]string{"", "a", "b"}, "a"},
		{[]string{"x", "y"}, "x"},
	}
	for _, c := range cases {
		if got := firstNonEmpty(c.vals); got != c.want {
			t.Errorf("firstNonEmpty(%v) = %q, want %q", c.vals, got, c.want)
		}
	}
}

// TestIsIntegerLike 覆盖 isIntegerLike 的空串/超长/带符号/非数字分支。
func TestIsIntegerLike(t *testing.T) {
	cases := []struct {
		val  string
		want bool
	}{
		{"", false},                // 空串
		{"123", true},              // 正常
		{"+456", true},             // 带正号（首位）
		{"-789", true},             // 带负号（首位）
		{"12a3", false},            // 含非数字
		{"1234567890123456", false}, // 16 位降级
		{"123456789012345", true},  // 15 位仍 integer
		{"+-123", false},           // 符号后非数字
		{"-", true},               // 仅符号：实现现状（符号后无数字，循环不执行返回 true）
	}
	for _, c := range cases {
		if got := isIntegerLike(c.val); got != c.want {
			t.Errorf("isIntegerLike(%q) = %v, want %v", c.val, got, c.want)
		}
	}
}

// TestInferPhysicalFromValue 覆盖 integer/string 两条返回。
func TestInferPhysicalFromValue(t *testing.T) {
	if got := inferPhysicalFromValue("123"); got != "integer" {
		t.Errorf("inferPhysicalFromValue(123) = %v, want integer", got)
	}
	if got := inferPhysicalFromValue("abc"); got != "string" {
		t.Errorf("inferPhysicalFromValue(abc) = %v, want string", got)
	}
	if got := inferPhysicalFromValue(""); got != "string" {
		t.Errorf("inferPhysicalFromValue(空) = %v, want string", got)
	}
}
