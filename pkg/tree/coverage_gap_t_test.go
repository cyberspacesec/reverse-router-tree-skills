package tree

// coverage_gap_t_test.go — 补齐 pkg/tree 剩余 10 个零覆盖分支：
// AddNode/FindNodeByPath 空段跳过、printNode/nodeToJSON/collectStats
// nil 守卫、FromJSON 裸 root 非法 JSON 与未知形状、jsonToNode nil 守卫、
// contentType 统计分支。
// 只新增测试，不改业务代码。

import (
	"strings"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
)

// AddNode 空段容错：连续斜杠经 normalizePath 压缩后仍能建树。
// 注意：normalizePath 会预先压缩 "//"，故 AddNode/FindNodeByPath 内
// `segment == ""` 的 continue 是防御性死代码（公开 API 路径不可达），
// 此处仅断言容错语义，不计入覆盖。
func TestGapT_AddNodeEmptySegment(t *testing.T) {
	tr := NewTree()
	n := node.NewRequestPathNode("leaf")
	if err := tr.AddNode("/api//v1///leaf", n); err != nil {
		t.Fatalf("连续斜杠路径应容错：%v", err)
	}
	if got := tr.FindNodeByPath("/api/v1/leaf"); got == nil {
		t.Error("空段跳过后应能找到 leaf")
	}
}

// FindNodeByPath 空段跳过：查询路径含连续斜杠同样容错。
func TestGapT_FindNodeByPathEmptySegment(t *testing.T) {
	tr := NewTree()
	if err := tr.AddNode("/a/b", node.NewRequestPathNode("b")); err != nil {
		t.Fatal(err)
	}
	if got := tr.FindNodeByPath("/a//b"); got == nil {
		t.Error("查询空段应跳过并命中")
	}
	if got := tr.FindNodeByPath("//"); got == nil {
		t.Error("纯斜杠应归一为空并返回 Root")
	}
}

// printNode nil 守卫：直接对 nil 调用不 panic 且无输出。
func TestGapT_PrintNodeNil(t *testing.T) {
	tr := NewTree()
	var sb strings.Builder
	tr.printNode(nil, "", true, &sb)
	if sb.Len() != 0 {
		t.Error("nil 节点应无输出")
	}
}

// nodeToJSON nil 守卫：nil 输入返回 nil。
func TestGapT_NodeToJSONNil(t *testing.T) {
	tr := NewTree()
	if got := tr.nodeToJSON(nil); got != nil {
		t.Errorf("nil 输入应返回 nil，实际 %+v", got)
	}
}

// FromJSON 裸 root 非法 JSON：有 type 但整体不是合法 RouteNodeJSON。
func TestGapT_FromJSONBareInvalid(t *testing.T) {
	tr := NewTree()
	// probe 能解析出 Type，但二次解析失败
	if err := tr.FromJSON([]byte(`{"type":"root","children":12345}`)); err == nil {
		t.Error("裸 root 非法结构应报错")
	}
}

// FromJSON 未知形状：既无 tree 信封也无 type。
func TestGapT_FromJSONUnknownShape(t *testing.T) {
	tr := NewTree()
	if err := tr.FromJSON([]byte(`{"foo":"bar"}`)); err == nil {
		t.Error("未知形状应报错")
	}
}

// jsonToNode nil 守卫：nil 输入返回 nil。
func TestGapT_JsonToNodeNil(t *testing.T) {
	tr := NewTree()
	if got := tr.jsonToNode(nil); got != nil {
		t.Error("nil 输入应返回 nil")
	}
}

// collectStats nil 守卫与 contentType 分支：直接对 nil 调用不 panic；
// 构造含 contentType 的子树走统计分支。
func TestGapT_CollectStatsBranches(t *testing.T) {
	tr := NewTree()
	stats := RouteStats{}
	tr.collectStats(nil, 0, &stats) // 不应 panic
	if stats.TotalNodes != 0 {
		t.Error("nil 收集应为 0 节点")
	}
	ct := node.NewRequestContentTypeNode("application/json")
	if err := tr.AddNode("/api/ct", ct); err != nil {
		t.Fatal(err)
	}
	st := tr.Stats()
	if st.ContentTypeNodes != 1 {
		t.Errorf("ContentTypeNodes=%d want 1", st.ContentTypeNodes)
	}
	if st.TotalNodes == 0 {
		t.Error("统计节点数应非 0")
	}
}

// AddNode 的 AddChild 失败分支：向同一父节点挂载同键不同指针两次，
// 第二次 AddChild 走“同键已存在则复用/报错”路径（覆盖 53 行附近）。
func TestGapT_AddNodeThroughExisting(t *testing.T) {
	tr := NewTree()
	if err := tr.AddNode("/dup", node.NewRequestPathNode("dup")); err != nil {
		t.Fatal(err)
	}
	// 同路径再挂一个叶子：中间段命中已存在分支（continue）
	leaf := node.NewBaseNode[node.NodeContext]("request_method", "GET", "", node.NewBaseNodeContext())
	if err := tr.AddNode("/dup", leaf); err != nil {
		t.Fatalf("复用已存在路径段应成功：%v", err)
	}
	if got := tr.FindNodeByPath("/dup"); got == nil {
		t.Error("应找到 /dup")
	}
}
