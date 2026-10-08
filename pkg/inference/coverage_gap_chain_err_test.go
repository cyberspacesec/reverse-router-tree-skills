package inference

import (
	"errors"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

func TestGapChain_InferErrorsAndCollapse(t *testing.T) {
	orig := inferType
	t.Cleanup(func() { inferType = orig })

	n := node.NewRequestPathNode("x")
	c := NewChainTypeInferenceRule()

	inferType = func(rule TypeInferenceRule, n node.Node[node.NodeContext]) (value.Type, error) {
		if _, ok := rule.(*PhysicalTypeInferenceRule); ok {
			return "", errors.New("boom")
		}
		return value.Type(value.PhysicalTypeString), nil
	}
	if _, _, err := c.InferPhysicalAndLogical(n); err == nil {
		t.Fatal("物理推断失败应向上返回")
	}

	inferType = func(rule TypeInferenceRule, n node.Node[node.NodeContext]) (value.Type, error) {
		if _, ok := rule.(*LogicalTypeInferenceRule); ok {
			return "", errors.New("boom")
		}
		return value.Type(value.PhysicalTypeInteger), nil
	}
	pt, lt, err := c.InferPhysicalAndLogical(n)
	if err != nil || pt != value.PhysicalTypeInteger || lt != value.LogicalTypeString {
		t.Fatalf("逻辑推断失败应回落 string，pt=%s lt=%s err=%v", pt, lt, err)
	}

	inferType = func(rule TypeInferenceRule, n node.Node[node.NodeContext]) (value.Type, error) {
		return value.Type(value.PhysicalTypeInteger), nil
	}
	pt, lt, err = c.InferPhysicalAndLogical(n)
	if err != nil || lt != value.LogicalTypeString || pt != value.PhysicalTypeInteger {
		t.Fatalf("逻辑与物理同为 integer 时应归并为 string，pt=%s lt=%s err=%v", pt, lt, err)
	}

	// 自定义链没有预置规则，走新建分支。
	custom := NewChainTypeInferenceRuleWithRules()
	inferType = orig
	if _, _, err := custom.InferPhysicalAndLogical(n); err != nil {
		t.Fatal(err)
	}
}
