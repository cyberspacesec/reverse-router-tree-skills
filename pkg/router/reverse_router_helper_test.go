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
