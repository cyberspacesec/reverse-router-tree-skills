package router

import (
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/tree"
)

// TestNewReverseRouterWithTree 覆盖 NewReverseRouterWithTree（此前 0%）：
// 验证它复用传入的树而非新建，且能正常喂数据。
func TestNewReverseRouterWithTree(t *testing.T) {
	// 预先构造一棵带数据的树
	existing := tree.NewTree()
	existing.AddNode("api/items", node.NewRequestMethodNode("GET"))

	r := NewReverseRouterWithTree(existing)
	if r.Tree != existing {
		t.Error("NewReverseRouterWithTree 应复用传入的树，而非新建")
	}
	// 验证内部组件已初始化
	if r.inferenceRule == nil || r.chainRule == nil || r.bodyParser == nil || r.stats == nil {
		t.Error("NewReverseRouterWithTree 应初始化所有内部组件")
	}
	if r.pathRouter == nil || r.paramRouter == nil || r.ctRouter == nil {
		t.Error("NewReverseRouterWithTree 应初始化 3 个查询 router")
	}
	// 复用的树应已含预置数据
	if r.Tree.Root.FindChildByKey("api") == nil {
		t.Error("复用的树应保留预置的 api 节点")
	}
	// 继续喂数据应正常工作
	r.ReverseHttpRequest(request.NewHttpRequest("/api/items/42", nil, "GET", nil))
	if r.Tree.Root.FindChildByKey("api").FindChildByKey("items").FindChildByKey("GET") == nil {
		t.Error("继续喂请求后应能定位 GET 方法节点")
	}
}

// TestStatsSnapshot_NilReceiver 覆盖 StatsSnapshot 的 nil 守卫分支（Snapshot 66.7%→更高）。
func TestStatsSnapshot_NilReceiver(t *testing.T) {
	var s *RouterStats
	snap := s.Snapshot()
	// nil 接收者应返回零值快照，不 panic
	if snap.RequestsProcessed != 0 || snap.Errors != 0 {
		t.Errorf("nil RouterStats.Snapshot 应返回零值，实际 %+v", snap)
	}
	// nil Reset 也不应 panic
	s.Reset()
}

// === 辅助纯函数测试 ===

func TestTrimLeadingDigits(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"00_user", "_user"}, // 去前导数字
		{"123abc", "abc"},
		{"abc", "abc"},       // 无前导数字原样返回
		{"", ""},             // 空串
		{"123", ""},          // 全数字
		{"0", ""},
	}
	for _, c := range cases {
		if got := trimLeadingDigits(c.in); got != c.want {
			t.Errorf("trimLeadingDigits(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestIsAlphanumeric(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"abc123", true},   // 字母+数字
		{"a1", true},
		{"123", false},     // 纯数字无字母
		{"abc", false},     // 纯字母无数字
		{"", false},        // 空串
		{"ab-12", false},   // 含非字母数字字符
		{"a_b1", false},    // 含下划线
		{"A1b2", true},     // 大小写混合
	}
	for _, c := range cases {
		if got := isAlphanumeric(c.in); got != c.want {
			t.Errorf("isAlphanumeric(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestIsInteger(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"123", true},
		{"0", true},
		{"+42", true},     // 正号前缀
		{"-42", true},     // 负号前缀
		{"", false},       // 空串
		{"12a", false},    // 含字母
		{"3.14", false},   // 含小数点
		{"++1", false},    // 第二个字符非数字
		{"1+2", false},    // 中间符号
		{" 12", false},    // 含空格
	}
	for _, c := range cases {
		if got := isInteger(c.in); got != c.want {
			t.Errorf("isInteger(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// TestDetectPrefixPattern 覆盖 detectPrefixPattern 各分支：样本不足返回 0、
// 无公共前缀、前缀裁剪后为空、值长度<=前缀长度 continue、变量部分非整数/字母数字不计数。
func TestDetectPrefixPattern(t *testing.T) {
	// 样本不足（<3）→ 0
	if got := detectPrefixPattern([]string{"user_001", "user_002"}); got != 0 {
		t.Errorf("样本<3 应返回 0，实际 %v", got)
	}
	// 无公共前缀 → 0
	if got := detectPrefixPattern([]string{"abc", "def", "ghi"}); got != 0 {
		t.Errorf("无公共前缀应返回 0，实际 %v", got)
	}
	// 正常前缀模式：user_001/user_002/user_003，变量部分 001/002/003 是整数 → 3/3=1.0
	got := detectPrefixPattern([]string{"user_001", "user_002", "user_003"})
	if got != 1.0 {
		t.Errorf("前缀模式匹配率应为 1.0，实际 %v", got)
	}
	// 部分匹配：变量部分有整数也有纯字母（纯字母不满足 isAlphanumeric，需字母+数字）
	// user_001(整数)/user_002(整数)/user_abc(纯字母，isAlphanumeric 要求含字母且含数字 → false) → 2/3
	got = detectPrefixPattern([]string{"user_001", "user_002", "user_abc"})
	if got < 0.66 || got > 0.67 {
		t.Errorf("纯字母变量不匹配，应 2/3≈0.667，实际 %v", got)
	}
	// 变量部分为字母+数字组合（isAlphanumeric true）→ 全匹配
	got = detectPrefixPattern([]string{"user_a1", "user_b2", "user_c3"})
	if got != 1.0 {
		t.Errorf("字母+数字变量应全匹配 1.0，实际 %v", got)
	}
	// 变量部分非整数非字母数字（含特殊字符）→ 不计数
	// user_001/user_002/user_a-b（含连字符，isAlphanumeric false、isInteger false）→ 2/3
	got = detectPrefixPattern([]string{"user_001", "user_002", "user_a-b"})
	if got < 0.66 || got > 0.67 {
		t.Errorf("含特殊字符变量部分应 2/3≈0.667，实际 %v", got)
	}
	// 值长度<=前缀长度：某值等于前缀本身，无变量部分 → continue
	// 前缀 "user_00"（trimTrailingDigits 后 "user_"），user_ 本身长度<=前缀长度 → continue
	got = detectPrefixPattern([]string{"user_1", "user_2", "user_"})
	// user_1/user_2 变量部分 1/2 是整数，user_ 长度<=前缀长度 continue → 2/3
	if got < 0.66 || got > 0.67 {
		t.Errorf("值长度<=前缀长度应 continue，匹配率 2/3≈0.667，实际 %v", got)
	}
}

// TestDetectSuffixPattern 覆盖 detectSuffixPattern 各分支（与前缀对称）。
func TestDetectSuffixPattern(t *testing.T) {
	// 样本不足 → 0
	if got := detectSuffixPattern([]string{"001_user", "002_user"}); got != 0 {
		t.Errorf("样本<3 应返回 0，实际 %v", got)
	}
	// 无公共后缀 → 0
	if got := detectSuffixPattern([]string{"abc", "def", "ghi"}); got != 0 {
		t.Errorf("无公共后缀应返回 0，实际 %v", got)
	}
	// 正常后缀模式：001_user/002_user/003_user，变量部分 001/002/003 整数 → 1.0
	got := detectSuffixPattern([]string{"001_user", "002_user", "003_user"})
	if got != 1.0 {
		t.Errorf("后缀模式匹配率应为 1.0，实际 %v", got)
	}
	// 变量部分含特殊字符不计数：001_user/002_user/a-b_user → 2/3
	got = detectSuffixPattern([]string{"001_user", "002_user", "a-b_user"})
	if got < 0.66 || got > 0.67 {
		t.Errorf("含特殊字符变量部分应 2/3≈0.667，实际 %v", got)
	}
	// 公共后缀经 trimLeadingDigits 裁剪后为空 → 0
	// 如三个值的公共后缀全是数字（"123"），trimLeadingDigits 后为空 → 0
	got = detectSuffixPattern([]string{"a123", "b123", "c123"})
	if got != 0 {
		t.Errorf("公共后缀全为数字裁剪后为空应返回 0，实际 %v", got)
	}
}

// TestLongestCommonPrefixSuffix 覆盖 longestCommonPrefix/Suffix 的空切片与单元素分支。
func TestLongestCommonPrefixSuffix(t *testing.T) {
	// 空切片
	if got := longestCommonPrefix(nil); got != "" {
		t.Errorf("longestCommonPrefix(nil) 应为空串，实际 %q", got)
	}
	if got := longestCommonSuffix(nil); got != "" {
		t.Errorf("longestCommonSuffix(nil) 应为空串，实际 %q", got)
	}
	// 单元素
	if got := longestCommonPrefix([]string{"abc"}); got != "abc" {
		t.Errorf("单元素前缀应为自身，实际 %q", got)
	}
	if got := longestCommonSuffix([]string{"abc"}); got != "abc" {
		t.Errorf("单元素后缀应为自身，实际 %q", got)
	}
	// 公共前缀中途变空提前返回
	if got := longestCommonPrefix([]string{"abc", "", "abc"}); got != "" {
		t.Errorf("含空串应返回空，实际 %q", got)
	}
}
