package tree

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

// roundTrip 将树序列化再反序列化，返回新树。
func roundTrip(t *testing.T, tr *Tree) *Tree {
	t.Helper()
	data, err := tr.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON err=%v", err)
	}
	out := NewTree()
	if err := out.FromJSON(data); err != nil {
		t.Fatalf("FromJSON err=%v", err)
	}
	return out
}

// buildLeafTree 构造 root → users → {users_id}(var) → GET → 含参数/header/cookie/值节点的树。
func buildLeafTree() *Tree {
	tr := NewTree()
	users := node.NewRequestPathNode("users")
	root := tr.Root
	_ = root.AddChild(users)

	v := node.NewRequestPathVariableNode("users_id", "[0-9]+")
	v.ObserveValue("123")
	v.ObserveValue("456")
	v.ObserveValue("123")
	v.SetType(value.Type("integer"))
	v.SetLogicalType(value.LogicalType("int"))
	_ = users.AddChild(v)

	m := node.NewRequestMethodNode("GET")
	m.IncrementRequestCount()
	_ = v.AddChild(m)

	p := node.NewRequestParamNode("page", "1", false)
	p.ObserveValue("1")
	p.ObserveValue("2")
	p.IncrementPresenceCount()
	p.SetPresenceCount(5)
	_ = m.AddChild(p)

	h1 := node.NewRequestHeaderNode("Accept")
	h1v := node.NewRequestHeaderValueNode("Accept", "application/json")
	h1v.ObserveValue("application/json")
	_ = h1.AddChild(h1v)
	_ = m.AddChild(h1)

	c1 := node.NewRequestCookieNode("session")
	c1v := node.NewRequestCookieValueNode("session", "abc123")
	c1v.ObserveValue("abc123")
	_ = c1.AddChild(c1v)
	_ = m.AddChild(c1)

	return tr
}

// F01-F04 各类节点 ValueMetric 计数序列化往返不被改动。
func TestF_PathVarValueCountPreserved(t *testing.T) {
	tr := roundTrip(t, buildLeafTree())
	v2 := tr.Root.FindChildByKey("users").GetChildByType("request_path_variable").(*node.RequestPathVariableNode)
	if got := v2.GetValueMetric().GetValueCount("123"); got != 2 {
		t.Errorf("F01 count of 123=%d want 2 (observed 123,456,123)", got)
	}
	if got := v2.GetValueMetric().GetValueCount("456"); got != 1 {
		t.Errorf("F01 count of 456=%d want 1", got)
	}
	if got := v2.GetValueMetric().GetUniqueValueCount(); got != 2 {
		t.Errorf("F01 unique=%d want 2", got)
	}
}

func TestF_ParamValueCountPreserved(t *testing.T) {
	tr := roundTrip(t, buildLeafTree())
	data, _ := tr.ToJSON()
	s := string(data)
	if !containsJSONCount(s, "1") || !containsJSONCount(s, "2") {
		t.Errorf("F02 param value counts not preserved")
	}
	if !contains(s, `"page"`) {
		t.Errorf("F02 param node missing")
	}
}

func TestF_HeaderValueCountPreserved(t *testing.T) {
	tr := roundTrip(t, buildLeafTree())
	data, _ := tr.ToJSON()
	s := string(data)
	if !containsJSONCount(s, "application/json") {
		t.Errorf("F03 header value count not preserved: %s", s)
	}
}

func TestF_CookieValueCountPreserved(t *testing.T) {
	tr := roundTrip(t, buildLeafTree())
	data, _ := tr.ToJSON()
	s := string(data)
	if !containsJSONCount(s, "abc123") {
		t.Errorf("F04 cookie value count not preserved: %s", s)
	}
}

// F05 序列化后继续采集：反序列化树的值计数可累加，不丢先前统计。
func TestF_ContinuationAccumulatesAfterRestore(t *testing.T) {
	tr := NewTree()
	users := node.NewRequestPathNode("users")
	_ = tr.Root.AddChild(users)
	v := node.NewRequestPathVariableNode("users_id", "[0-9]+")
	v.ObserveValue("1")
	v.ObserveValue("2")
	_ = users.AddChild(v)

	restored := roundTrip(t, tr)
	// 找到反序列化后的变量节点并继续观察
	typeFinder := restored.Root
	users2 := typeFinder.FindChildByKey("users")
	v2node := users2.GetChildByType("request_path_variable")
	v2 := v2node.(*node.RequestPathVariableNode)
	v2.ObserveValue("1") // 已存在值 → 计数累加为 2
	v2.ObserveValue("3")

	if got := v2.GetValueMetric().GetValueCount("1"); got != 2 {
		t.Errorf("F05 after restore+fresh ObserveValue, count of %q=%d want 2", "1", got)
	}
	if got := v2.GetValueMetric().GetValueCount("3"); got != 1 {
		t.Errorf("F05 fresh value count=%d want 1", got)
	}
	if got := v2.GetValueMetric().GetUniqueValueCount(); got != 3 {
		t.Errorf("F05 unique after continuation=%d want 3", got)
	}
}

// F06 RestoreCounts 只过滤 count>0 的项，并整体替换而非累加。
func TestF_RestoreCountsFiltersAndReplaces(t *testing.T) {
	vm := value.NewValueMetric()
	vm.AddValue("a")
	vm.RestoreCounts(map[string]int{"b": 2, "c": 0, "d": -5})
	if vm.GetValueCount("a") != 0 {
		t.Errorf("F06 restore should replace old counts")
	}
	if vm.GetValueCount("b") != 2 {
		t.Errorf("F06 restored count b=%d want 2", vm.GetValueCount("b"))
	}
	if vm.GetValueCount("c") != 0 || vm.GetValueCount("d") != 0 {
		t.Errorf("F06 zero/negative counts must be filtered out")
	}
	if vm.GetUniqueValueCount() != 1 {
		t.Errorf("F06 unique after restore=%d want 1", vm.GetUniqueValueCount())
	}
}

// F07 多次同一值观察，计数在 JSON 中精确还原；反序列化后重复计数累加。
func TestF_RepeatedCountsPrecise(t *testing.T) {
	tr := NewTree()
	users := node.NewRequestPathNode("users")
	_ = tr.Root.AddChild(users)
	v := node.NewRequestPathVariableNode("users_id", "[0-9]+")
	for i := 0; i < 5; i++ {
		v.ObserveValue("42")
	}
	v.ObserveValue("7")
	_ = users.AddChild(v)

	restored := roundTrip(t, tr)
	users2 := restored.Root.FindChildByKey("users")
	v2 := users2.GetChildByType("request_path_variable").(*node.RequestPathVariableNode)
	if got := v2.GetValueMetric().GetValueCount("42"); got != 5 {
		t.Errorf("F07 count of 42=%d want 5", got)
	}
	if got := v2.GetValueMetric().GetTotalCount(); got != 6 {
		t.Errorf("F07 total=%d want 6", got)
	}
}

// F08 空树序列化往返保持根节点结构。
func TestF_EmptyTreeRoundTrip(t *testing.T) {
	tr := NewTree()
	restored := roundTrip(t, tr)
	if restored.Root == nil || restored.Root.GetType() != "root" {
		t.Errorf("F08 empty tree root lost after roundtrip")
	}
	data, _ := restored.ToJSON()
	if !contains(string(data), `"root"`) {
		t.Errorf("F08 root not serialized")
	}
}

// F09 路径变量 type/logicalType/pattern 往返保持一致。
func TestF_PathVariableFieldsRoundTrip(t *testing.T) {
	tr := NewTree()
	users := node.NewRequestPathNode("users")
	_ = tr.Root.AddChild(users)
	v := node.NewRequestPathVariableNode("users_uuid", "[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}")
	v.ObserveValue("550e8400-e29b-41d4-a716-446655440000")
	v.SetType(value.Type("string"))
	v.SetLogicalType(value.LogicalType("uuid"))
	_ = users.AddChild(v)

	restored := roundTrip(t, tr)
	v2 := restored.Root.FindChildByKey("users").GetChildByType("request_path_variable").(*node.RequestPathVariableNode)
	if string(v2.GetValueType()) != "string" {
		t.Errorf("F09 physical type lost: want string, got %s", string(v2.GetValueType()))
	}
	if string(v2.GetLogicalType()) != "uuid" {
		t.Errorf("F09 logical type lost: want uuid, got %s", string(v2.GetLogicalType()))
	}
	if v2.GetPattern() == nil || !v2.GetPattern().MatchString("550e8400-e29b-41d4-a716-446655440000") {
		t.Errorf("F09 pattern not restored/enforced")
	}
	if v2.GetPattern().MatchString("550e8400-e29b-41d4-a716-44665544000Z") {
		t.Errorf("F09 pattern too loose after restore")
	}
}

// F10 参数节点 required/presence/defaultValue/multiValue 往返一致。
func TestF_ParamKeeperFieldsRoundTrip(t *testing.T) {
	tr := NewTree()
	users := node.NewRequestPathNode("users")
	_ = tr.Root.AddChild(users)
	m := node.NewRequestMethodNode("GET")
	_ = users.AddChild(m)
	req := true
	p := node.NewRequestParamNode("status", "active", req)
	p.ObserveValue("active")
	p.ObserveValue("disabled")
	p.SetPresenceCount(9)
	p.SetMultiValue(true)
	p.SetValueType(value.Type("string"))
	p.SetLogicalType(value.LogicalType("enum"))
	_ = m.AddChild(p)

	restored := roundTrip(t, tr)
	p2node := restored.Root.FindChildByKey("users").FindChildByKey("GET").FindChildByKey("status")
	p2 := p2node.(*node.RequestParamNode)
	if !p2.IsRequired() {
		t.Errorf("F10 required not preserved")
	}
	if p2.GetPresenceCount() != 9 {
		t.Errorf("F10 presence=%d want 9", p2.GetPresenceCount())
	}
	if !p2.IsMultiValue() {
		t.Errorf("F10 multi_value not preserved")
	}
	if p2.GetDefaultValue() != "active" {
		t.Errorf("F10 default_value=%q want active", p2.GetDefaultValue())
	}
	if string(p2.GetLogicalType()) != "enum" {
		t.Errorf("F10 logical type not preserved")
	}
}

// F11 海量样本计数序列化不丢、不重复。
func TestF_LargeValueCountsRoundTrip(t *testing.T) {
	tr := NewTree()
	users := node.NewRequestPathNode("items")
	_ = tr.Root.AddChild(users)
	v := node.NewRequestPathVariableNode("items_id", "[0-9]+")
	for i := 0; i < 3000; i++ {
		v.ObserveValue(strconv.Itoa(i % 500)) // 500 个不同值，每个约 6 次
	}
	_ = users.AddChild(v)

	restored := roundTrip(t, tr)
	v2 := restored.Root.FindChildByKey("items").GetChildByType("request_path_variable").(*node.RequestPathVariableNode)
	if got := v2.GetValueMetric().GetUniqueValueCount(); got != 500 {
		t.Errorf("F11 unique=%d want 500", got)
	}
	if got := v2.GetValueMetric().GetTotalCount(); got != 3000 {
		t.Errorf("F11 total=%d want 3000", got)
	}
	if v2.GetValueMetric().GetValueCount("0") != 6 {
		t.Errorf("F11 count of '0'=%d want 6", v2.GetValueMetric().GetValueCount("0"))
	}
}

// F12 缺失可选字段的 JSON（旧版）仍能反序列化，不报错。
func TestF_MissingOptionalFieldsCompatible(t *testing.T) {
	old := `{
		"type":"root","key":"root","value":"",
		"children":[{
			"type":"request_path","key":"users",
			"children":[
				{"type":"request_path_variable","key":"users_id","pattern":"[0-9]+"},
				{"type":"request_method","key":"GET",
					"children":[
						{"type":"request_param","key":"page","value":"1"}
					]}
			]
		}]
	}`
	tr := NewTree()
	if err := tr.FromJSON([]byte(old)); err != nil {
		t.Fatalf("F12 old JSON parse err=%v", err)
	}
	users := tr.Root.FindChildByKey("users")
	if users == nil {
		t.Fatalf("F12 users node missing")
	}
	if tr.Root.FindChildByKey("users").FindChildByKey("GET").FindChildByKey("page") == nil {
		t.Errorf("F12 param page missing after old-JSON load")
	}
	// 无 value_counts → valueMap 空，不 panic
	vnode := users.GetChildByType("request_path_variable").(*node.RequestPathVariableNode)
	if vnode.GetValueMetric().GetUniqueValueCount() != 0 {
		t.Errorf("F12 absent counts should be empty")
	}
}

// F13 未知节点类型保持原样往返（向前兼容）。
func TestF_UnknownNodeTypeRoundTrip(t *testing.T) {
	data := []byte(`{
		"type":"root","key":"root","value":"",
		"children":[{"type":"future_node","key":"x","value":"v","requests":3}]
	}`)
	tr := NewTree()
	if err := tr.FromJSON(data); err != nil {
		t.Fatalf("F13 parse err=%v", err)
	}
	child := tr.Root.FindChildByKey("x")
	if child == nil || child.GetType() != "future_node" {
		t.Errorf("F13 unknown node type not preserved")
	}
	if child.GetValue() != "v" {
		t.Errorf("F13 unknown node value not preserved")
	}
}

// F14 无效 JSON 报错而不崩溃。
func TestF_InvalidJSONReturnsError(t *testing.T) {
	tr := NewTree()
	if err := tr.FromJSON([]byte(`{"type":`)); err == nil {
		t.Errorf("F14 malformed JSON should error")
	}
	if err := tr.FromJSON([]byte("not json")); err == nil {
		t.Errorf("F14 non-JSON should error")
	}
	if err := tr.FromJSON(nil); err == nil {
		t.Errorf("F14 empty input should error")
	}
}

// F15 同一方法下同时出现参数、header、cookie 值，序列化往返后 Stat 计数一致。
func TestF_StatsStableAfterRoundTrip(t *testing.T) {
	tr := buildLeafTree()
	s1 := tr.Stats()
	restored := roundTrip(t, tr)
	s2 := restored.Stats()
	if s1.TotalNodes != s2.TotalNodes || s1.ParamNodes != s2.ParamNodes ||
		s1.HeaderValueNodes != s2.HeaderValueNodes || s1.CookieValueNodes != s2.CookieValueNodes ||
		s1.PathVariableNodes != s2.PathVariableNodes {
		t.Errorf("F15 stats changed after roundtrip: before=%+v after=%+v", s1, s2)
	}
}

// ---- 工具函数 ----

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (indexOf(s, sub) >= 0))
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// containsJSONCount 检查 JSON 中某个值作为 key 出现在 value_counts 附近。
func containsJSONCount(s, val string) bool {
	quoted, err := json.Marshal(val)
	if err != nil {
		return false
	}
	return contains(s, string(quoted))
}
