package tree

import (
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
)

func TestJSONRoundTripPreservesValueMetric(t *testing.T) {
	tree := NewTree()
	if err := tree.AddNode("api/users", node.NewRequestMethodNode("GET")); err != nil {
		t.Fatal(err)
	}
	param := node.NewRequestParamNode("page", "", false)
	param.ObserveValue("1")
	param.ObserveValue("1")
	param.ObserveValue("2")
	method := tree.Root.FindChildByKey("api").FindChildByKey("users").FindChildByKey("GET")
	if err := method.AddChild(param); err != nil {
		t.Fatal(err)
	}

	data, err := tree.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	restored := NewTree()
	if err := restored.FromJSON(data); err != nil {
		t.Fatal(err)
	}
	restoredParam, ok := restored.Root.FindChildByKey("api").FindChildByKey("users").FindChildByKey("GET").FindChildByKey("page").(*node.RequestParamNode)
	if !ok {
		t.Fatal("反序列化后未找到 page 参数")
	}
	if got := restoredParam.GetValueMetric().GetValueCount("1"); got != 2 {
		t.Errorf("值 1 计数 = %d, want 2", got)
	}
	if got := restoredParam.GetValueMetric().GetTotalCount(); got != 3 {
		t.Errorf("总计数 = %d, want 3", got)
	}

	// 续喂必须在原计数上累加，而不是覆盖或从零开始。
	restoredParam.ObserveValue("1")
	if got := restoredParam.GetValueMetric().GetValueCount("1"); got != 3 {
		t.Errorf("续喂后值 1 计数 = %d, want 3", got)
	}
}

func TestJSONWithoutValueCountsIsCompatible(t *testing.T) {
	tree := NewTree()
	if err := tree.FromJSON([]byte(`{"type":"root","key":"root","children":[]}`)); err != nil {
		t.Fatal(err)
	}
	if tree.Root == nil {
		t.Fatal("旧 JSON 应可正常导入")
	}
}

func TestVersionedJSONRoundTrip(t *testing.T) {
	tree := NewTree()
	if err := tree.AddNode("api/users", node.NewRequestMethodNode("GET")); err != nil {
		t.Fatal(err)
	}
	data, err := tree.ToVersionedJSON()
	if err != nil {
		t.Fatal(err)
	}
	restored := NewTree()
	if err := restored.FromJSON(data); err != nil {
		t.Fatalf("版本信封应可导入: %v", err)
	}
	if restored.Root.FindChildByKey("api") == nil {
		t.Fatal("版本信封往返后应保留 api 节点")
	}
}

func TestVersionedJSONRejectsFutureVersion(t *testing.T) {
	tree := NewTree()
	future := []byte(`{"version":999,"tree":{"type":"root","key":"root","children":[]}}`)
	if err := tree.FromJSON(future); err == nil {
		t.Fatal("未知高版本应返回明确错误，而非静默误读")
	}
}

func TestVersionedJSONEnvelopeWithoutVersion(t *testing.T) {
	// 无 version 字段的信封形状按 v0 兼容读，不报错。
	tree := NewTree()
	data := []byte(`{"tree":{"type":"root","key":"root","children":[]}}`)
	if err := tree.FromJSON(data); err != nil {
		t.Fatalf("无版本信封应兼容读: %v", err)
	}
}
