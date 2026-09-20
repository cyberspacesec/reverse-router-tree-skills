package value

// coverage_gap_main_test.go — 主线程补齐 pkg/value 未覆盖分支：
// SetMaxUnique / CappedDropped / AddValue 上限分支 / AddValueCapped / RestoreCounts。

import "testing"

func TestGapMain_SetMaxUnique(t *testing.T) {
	m := NewValueMetric()
	// 空占位键回退默认 "[capped]"
	m.SetMaxUnique(2, "")
	m.AddValue("a")
	m.AddValue("b")
	m.AddValue("c") // 超限 → 占位桶
	if got := m.GetValueCount("[capped]"); got != 1 {
		t.Errorf("占位桶计数=%d want 1", got)
	}
	if got := m.CappedDropped(); got != 1 {
		t.Errorf("CappedDropped=%d want 1", got)
	}
	// 已存在值超限后仍累加原值
	m.AddValue("a")
	if got := m.GetValueCount("a"); got != 2 {
		t.Errorf("a 计数=%d want 2", got)
	}
	// 自定义占位键
	m2 := NewValueMetric()
	m2.SetMaxUnique(1, "OTHER")
	m2.AddValue("x")
	m2.AddValue("y")
	if got := m2.GetValueCount("OTHER"); got != 1 {
		t.Errorf("自定义占位桶=%d want 1", got)
	}
	// 0 表示不限制
	m3 := NewValueMetric()
	m3.SetMaxUnique(0, "")
	for i := 0; i < 10; i++ {
		m3.AddValue(string(rune('a' + i)))
	}
	if got := m3.GetUniqueValueCount(); got != 10 {
		t.Errorf("0=不限制，unique=%d want 10", got)
	}
	if got := m3.CappedDropped(); got != 0 {
		t.Errorf("不限时 CappedDropped=%d want 0", got)
	}
}

func TestGapMain_AddValueCapped(t *testing.T) {
	m := NewValueMetric()
	// 未达上限：原值记录返回 true
	if !m.AddValueCapped("a", 2, "[capped]") {
		t.Error("未超限应返回 true")
	}
	// 已存在值：累加并返回 true（即使已达上限）
	m.AddValueCapped("b", 2, "[capped]")
	if !m.AddValueCapped("a", 2, "[capped]") {
		t.Error("已存在值应返回 true")
	}
	if got := m.GetValueCount("a"); got != 2 {
		t.Errorf("a 计数=%d want 2", got)
	}
	// 新值超限：归入占位桶返回 false
	if m.AddValueCapped("c", 2, "[capped]") {
		t.Error("超限新值应返回 false")
	}
	if got := m.GetValueCount("[capped]"); got != 1 {
		t.Errorf("占位桶=%d want 1", got)
	}
	// 空占位键回退默认
	m2 := NewValueMetric()
	m2.AddValueCapped("x", 1, "")
	if m2.AddValueCapped("y", 1, "") {
		t.Error("超限新值应返回 false")
	}
	if got := m2.GetValueCount("[capped]"); got != 1 {
		t.Errorf("默认占位桶=%d want 1", got)
	}
	// maxUnique<=0 不限制
	m3 := NewValueMetric()
	for i := 0; i < 5; i++ {
		if !m3.AddValueCapped(string(rune('a'+i)), 0, "") {
			t.Errorf("不限时 %q 应返回 true", string(rune('a'+i)))
		}
	}
}

func TestGapMain_RestoreCounts(t *testing.T) {
	m := NewValueMetric()
	m.AddValue("old")
	m.RestoreCounts(map[string]int{"b": 2, "c": 0, "d": -5})
	// 只保留 count>0，且整体替换（old 被清掉）
	if got := m.GetValueCount("b"); got != 2 {
		t.Errorf("b=%d want 2", got)
	}
	if got := m.GetUniqueValueCount(); got != 1 {
		t.Errorf("unique=%d want 1（只保留正计数）", got)
	}
	// 输入被复制：调用方后续修改不影响
	in := map[string]int{"k": 3}
	m.RestoreCounts(in)
	in["k"] = 99
	if got := m.GetValueCount("k"); got != 3 {
		t.Errorf("应复制输入，k=%d want 3", got)
	}
	// 空 map 清空
	m.RestoreCounts(map[string]int{})
	if !m.IsEmpty() {
		t.Error("空 map 恢复后应为空")
	}
}
