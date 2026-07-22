package inference

import (
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

// TestInferPhysicalAndLogical_CustomChain 覆盖 physicalRule==nil 时回退新建规则的分支。
// NewChainTypeInferenceRuleWithRules 构造的链 physicalRule/logicalRule 为 nil，
// InferPhysicalAndLogical 走每次 NewXxx 回退路径。
func TestInferPhysicalAndLogical_CustomChain(t *testing.T) {
	chain := NewChainTypeInferenceRuleWithRules()
	pathVarNode := node.NewRequestPathVariableNode("id", "[0-9]+")
	for _, id := range []string{"123", "456", "789"} {
		pathVarNode.ObserveValue(id)
	}
	pt, lt, err := chain.InferPhysicalAndLogical(pathVarNode)
	if err != nil {
		t.Fatalf("自定义链 InferPhysicalAndLogical 报错: %v", err)
	}
	// 回退新建的 PhysicalTypeInferenceRule 行为应与默认链一致
	if pt != value.PhysicalTypeInteger {
		t.Errorf("自定义链物理类型 = %q, want integer", pt)
	}
	if lt != value.LogicalTypeString {
		t.Errorf("自定义链逻辑类型 = %q, want string", lt)
	}
}

// TestInferPhysicalAndLogical_EmptyMetric 覆盖 metric 为空时回退到 string 的分支。
func TestInferPhysicalAndLogical_EmptyMetric(t *testing.T) {
	chain := NewChainTypeInferenceRule()
	// 无任何观察值，metric 为空
	pathVarNode := node.NewRequestPathVariableNode("id", "")
	pt, lt, err := chain.InferPhysicalAndLogical(pathVarNode)
	if err != nil {
		t.Fatalf("空 metric 不应报错: %v", err)
	}
	if pt != value.PhysicalTypeString {
		t.Errorf("空 metric 物理类型 = %q, want string", pt)
	}
	if lt != value.LogicalTypeString {
		t.Errorf("空 metric 逻辑类型 = %q, want string", lt)
	}
}

// TestChainInfer_FallbackToString 覆盖所有规则返回空类型时回退到 string 的分支。
// ChainTypeInferenceRule.Infer 在所有规则返回空类型时，lastType 为空，
// 应回退到 value.Type(value.PhysicalTypeString)。
func TestChainInfer_FallbackToString(t *testing.T) {
	// 空规则链：无任何规则，Infer 直接回退到 string
	chain := NewChainTypeInferenceRuleWithRules()
	pathVarNode := node.NewRequestPathVariableNode("id", "")
	typ, err := chain.Infer(pathVarNode)
	if err != nil {
		t.Fatalf("空链 Infer 不应报错: %v", err)
	}
	if typ != value.Type(value.PhysicalTypeString) {
		t.Errorf("空链回退类型 = %q, want string", typ)
	}
}
