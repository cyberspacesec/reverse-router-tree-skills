package node

import (
	"sync/atomic"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

// === RequestPathVariableNode 未覆盖的 getter/setter ===

// TestPathVariableNode_TypeGettersSetters 覆盖 SetType/GetValueType 往返与默认值。
// GetValueType/SetType 此前 0% 覆盖率（类型推断结果落库的核心 setter/getter）。
func TestPathVariableNode_TypeGettersSetters(t *testing.T) {
	n := NewRequestPathVariableNode("id", "[0-9]+")

	// 默认物理类型为 string
	if got := n.GetValueType(); got != value.Type(value.PhysicalTypeString) {
		t.Errorf("默认物理类型应为 string，得 %s", got)
	}

	// SetType 后 GetValueType 应返回新值
	n.SetType(value.Type(value.PhysicalTypeInteger))
	if got := n.GetValueType(); got != value.Type(value.PhysicalTypeInteger) {
		t.Errorf("SetType 后应为 integer，得 %s", got)
	}
}

// TestPathVariableNode_LogicalTypeGettersSetters 覆盖 GetLogicalType/SetLogicalType。
func TestPathVariableNode_LogicalTypeGettersSetters(t *testing.T) {
	n := NewRequestPathVariableNode("id", "[0-9]+")

	if got := n.GetLogicalType(); got != value.LogicalTypeString {
		t.Errorf("默认逻辑类型应为 string，得 %s", got)
	}

	n.SetLogicalType(value.LogicalTypeInteger)
	if got := n.GetLogicalType(); got != value.LogicalTypeInteger {
		t.Errorf("SetLogicalType 后应为 integer，得 %s", got)
	}
}

// TestPathVariableNode_ValueMetric 覆盖 GetValueMetric（ObserveValue 落库的统计器）。
func TestPathVariableNode_ValueMetric(t *testing.T) {
	n := NewRequestPathVariableNode("id", "[0-9]+")
	vm := n.GetValueMetric()
	if vm == nil {
		t.Fatal("GetValueMetric 不应返回 nil")
	}
	// ObserveValue 应反映到同一 ValueMetric
	n.ObserveValue("123")
	if vm.GetTotalCount() != 1 {
		t.Errorf("ObserveValue 后 GetTotalCount 应为 1，得 %d", vm.GetTotalCount())
	}
}

// TestPathVariableNode_GetPattern 覆盖 GetPattern（序列化 pattern 往返依赖的 API）。
func TestPathVariableNode_GetPattern(t *testing.T) {
	// 带模式
	n := NewRequestPathVariableNode("id", "[0-9]+")
	p := n.GetPattern()
	if p == nil {
		t.Fatal("带模式节点的 GetPattern 不应返回 nil")
	}
	if p.String() != "[0-9]+" {
		t.Errorf("pattern 应为 [0-9]+，得 %s", p.String())
	}

	// 无模式
	anyNode := NewRequestPathVariableNode("any", "")
	if anyNode.GetPattern() != nil {
		t.Error("无模式节点的 GetPattern 应返回 nil")
	}
}

// TestPathVariableNode_IsDynamic 路径变量节点始终是动态的。
func TestPathVariableNode_IsDynamic(t *testing.T) {
	n := NewRequestPathVariableNode("id", "[0-9]+")
	if !n.IsDynamic() {
		t.Error("路径变量节点的 IsDynamic 应返回 true")
	}
}

// === RequestParamNode 未覆盖的 getter/setter ===

// TestParamNode_RequiredSetter 覆盖 SetRequired（IsRequired 已被现有测试间接覆盖，SetRequired 未覆盖）。
func TestParamNode_RequiredSetter(t *testing.T) {
	n := NewRequestParamNode("page", "1", false)
	if n.IsRequired() {
		t.Error("构造 required=false 时 IsRequired 应为 false")
	}
	n.SetRequired(true)
	if !n.IsRequired() {
		t.Error("SetRequired(true) 后 IsRequired 应为 true")
	}
	n.SetRequired(false)
	if n.IsRequired() {
		t.Error("SetRequired(false) 后 IsRequired 应为 false")
	}
}

// TestParamNode_PresenceCountSetter 覆盖 SetPresenceCount（从持久化恢复状态用）。
func TestParamNode_PresenceCountSetter(t *testing.T) {
	n := NewRequestParamNode("page", "1", false)
	if n.GetPresenceCount() != 0 {
		t.Error("初始 presenceCount 应为 0")
	}
	n.SetPresenceCount(42)
	if n.GetPresenceCount() != 42 {
		t.Errorf("SetPresenceCount(42) 后应为 42，得 %d", n.GetPresenceCount())
	}
}

// TestParamNode_TypeGettersSetters 覆盖 GetValueType/SetValueType 往返。
func TestParamNode_TypeGettersSetters(t *testing.T) {
	n := NewRequestParamNode("page", "1", false)
	// 默认值由构造函数决定，这里只验证 Set/Get 往返
	n.SetValueType(value.Type(value.PhysicalTypeInteger))
	if got := n.GetValueType(); got != value.Type(value.PhysicalTypeInteger) {
		t.Errorf("SetValueType 后应为 integer，得 %s", got)
	}
}

// TestParamNode_LogicalTypeGettersSetters 覆盖 GetLogicalType/SetLogicalType。
func TestParamNode_LogicalTypeGettersSetters(t *testing.T) {
	n := NewRequestParamNode("page", "1", false)
	n.SetLogicalType(value.LogicalTypeInteger)
	if got := n.GetLogicalType(); got != value.LogicalTypeInteger {
		t.Errorf("SetLogicalType 后应为 integer，得 %s", got)
	}
}

// TestParamNode_ValueMetric 覆盖 GetValueMetric + ObserveValue 往返。
func TestParamNode_ValueMetric(t *testing.T) {
	n := NewRequestParamNode("page", "1", false)
	vm := n.GetValueMetric()
	if vm == nil {
		t.Fatal("GetValueMetric 不应返回 nil")
	}
	n.ObserveValue("2")
	n.ObserveValue("2")
	if vm.GetValueCount("2") != 2 {
		t.Errorf("ObserveValue 两次后 page=2 计数应为 2，得 %d", vm.GetValueCount("2"))
	}
}

// === BaseNode 未覆盖项 ===

// TestBaseNode_IsDynamicDefault BaseNode 默认非动态（子类覆写）。
// 此前 0% 覆盖率。
func TestBaseNode_IsDynamicDefault(t *testing.T) {
	ctx := NewBaseNodeContext()
	n := NewBaseNode[NodeContext]("test", "k", "v", ctx)
	if n.IsDynamic() {
		t.Error("BaseNode.IsDynamic 默认应为 false")
	}
}

// TestBaseNode_DeepClone_Parallel 覆盖 DeepClone 的并行分支（子节点 >10）。
// 此前 50% 覆盖率，缺并行分支。
func TestBaseNode_DeepClone_Parallel(t *testing.T) {
	ctx := NewBaseNodeContext()
	root := NewBaseNode[NodeContext]("root", "root", "", ctx)

	// 添加 15 个直接子节点，触发并行克隆分支（>10）
	for i := 0; i < 15; i++ {
		child := NewBaseNode[NodeContext]("child", "child"+itoa(i), "v", ctx)
		root.AddChild(child)
	}

	clone := root.DeepClone()
	if clone == root {
		t.Fatal("DeepClone 返回了相同引用")
	}
	children := clone.GetChildren()
	if len(children) != 15 {
		t.Errorf("克隆后子节点数应为 15，得 %d", len(children))
	}
	// 验证克隆是深拷贝：修改原树不影响克隆
	root.GetChildren()[0].SetValue("mutated")
	if clone.GetChildren()[0].GetValue() == "mutated" {
		t.Error("深克隆应独立，原树修改不应影响克隆")
	}
}

// TestBaseNode_VisitChildren_SmallBatch 覆盖小批次串行分支与提前终止。
func TestBaseNode_VisitChildren_SmallBatch(t *testing.T) {
	ctx := NewBaseNodeContext()
	root := NewBaseNode[NodeContext]("root", "root", "", ctx)
	for i := 0; i < 3; i++ {
		root.AddChild(NewBaseNode[NodeContext]("c", "c"+itoa(i), "", ctx))
	}

	visited := 0
	root.VisitChildren(func(n Node[NodeContext]) bool {
		visited++
		return true
	})
	if visited != 3 {
		t.Errorf("小批次应遍历全部 3 个，得 %d", visited)
	}

	// 提前终止
	visited = 0
	root.VisitChildren(func(n Node[NodeContext]) bool {
		visited++
		return false // 第 1 个就终止
	})
	if visited != 1 {
		t.Errorf("提前终止应只访问 1 个，得 %d", visited)
	}
}

// TestBaseNode_VisitChildren_LargeBatch 覆盖大批量并行分支与聚合终止。
func TestBaseNode_VisitChildren_LargeBatch(t *testing.T) {
	ctx := NewBaseNodeContext()
	root := NewBaseNode[NodeContext]("root", "root", "", ctx)
	// batchSize=10，单批 >3 走并行分支；建 12 个子节点形成一批 10+一批 2
	for i := 0; i < 12; i++ {
		root.AddChild(NewBaseNode[NodeContext]("c", "c"+itoa(i), "", ctx))
	}

	visited := int64(0)
	root.VisitChildren(func(n Node[NodeContext]) bool {
		atomic.AddInt64(&visited, 1)
		return true
	})
	if visited != 12 {
		t.Errorf("大批量应遍历全部 12 个，得 %d", visited)
	}

	// 并行批次内的聚合终止：visitor 对所有返回 false 应整体终止
	visited = 0
	root.VisitChildren(func(n Node[NodeContext]) bool {
		atomic.AddInt64(&visited, 1)
		return false
	})
	// 第一个批次（10 个并行）全部返回 false，聚合后终止，应只访问第一批
	got := atomic.LoadInt64(&visited)
	if got == 0 || got > 10 {
		t.Errorf("聚合终止应只访问第一批（≤10），得 %d", got)
	}
}

// itoa 避免引入 strconv（与现有测试风格一致，base_node_test.go 已 import strconv，
// 但本文件不 import，用轻量本地实现生成唯一 key）。
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	buf := []byte{}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	for i > 0 {
		buf = append([]byte{byte('0' + i%10)}, buf...)
		i /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}
