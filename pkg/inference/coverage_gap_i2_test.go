package inference

// coverage_gap_i2_test.go — inference 第二轮补齐（12 个可达分支）：
// physical nil 上下文、inferFromSamples 空守卫（直接调用）、isInteger/isFloat/
// isScientificNotation/isNumericPart 空串守卫（直接调用）、isFloat 符号位、
// findDominantType 回退（直接调用）；logical nil 上下文、isEnumLike 模式命中
// 循环、结构排除、isDecimalLike 符号位。
// 剩余 4 块为防御死代码（chain 103 物理失败、chain 113 逻辑失败、
// chain 120 逻辑物理相等归一、logical 142 totalCount==0），公开 API 不可达。
// 只新增测试，不改业务代码。

import (
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

// nil 上下文节点：physical 走 64-67 空串分支，logical 走 189-191 nil 分支。
func gapI2NilContextNode() *node.BaseNode[node.NodeContext] {
	return node.NewBaseNode[node.NodeContext]("request_path", "api", "", nil)
}

func TestGapI2_PhysicalNilContext(t *testing.T) {
	r := NewPhysicalTypeInferenceRule()
	typ, err := r.Infer(gapI2NilContextNode())
	if err != nil {
		t.Fatal(err)
	}
	if typ != "" {
		t.Errorf("nil 上下文应返回空类型，实际 %q", typ)
	}
}

func TestGapI2_LogicalNilContext(t *testing.T) {
	r := NewLogicalTypeInferenceRule()
	typ, err := r.Infer(gapI2NilContextNode())
	if err != nil {
		t.Fatal(err)
	}
	if typ != value.Type(value.LogicalTypeString) {
		t.Errorf("nil 上下文应回退 string，实际 %q", typ)
	}
}

// inferFromSamples 空守卫：Infer 有前置守卫调不到，直接调用覆盖 96-99。
func TestGapI2_InferFromSamplesEmpty(t *testing.T) {
	r := NewPhysicalTypeInferenceRule()
	typ, err := r.inferFromSamples(nil)
	if err != nil {
		t.Fatal(err)
	}
	if typ != value.Type(value.PhysicalTypeNull) {
		t.Errorf("nil metric 应为 null，实际 %q", typ)
	}
	typ, err = r.inferFromSamples(value.NewValueMetric())
	if err != nil {
		t.Fatal(err)
	}
	if typ != value.Type(value.PhysicalTypeNull) {
		t.Errorf("空 metric 应为 null，实际 %q", typ)
	}
}

// 空串守卫三件套：经 Infer 调不到（空串先被 isNull 截获），直接调用。
func TestGapI2_EmptyGuards(t *testing.T) {
	r := NewPhysicalTypeInferenceRule()
	if r.isInteger("") {
		t.Error("空串不应判为整数")
	}
	if r.isFloat("") {
		t.Error("空串不应判为浮点")
	}
	if r.isScientificNotation("") {
		t.Error("空串不应判为科学计数法")
	}
	if r.isNumericPart("", true) {
		t.Error("空串不应判为合法数值部")
	}
	if r.isNumericPart("", false) {
		t.Error("空串不应判为合法指数部")
	}
}

// isFloat 符号位："+12.34" 非整数但为浮点，走 265 continue。
func TestGapI2_FloatSign(t *testing.T) {
	r := NewPhysicalTypeInferenceRule()
	n := gapIPathVar(t, "+12.34", "+56.78", "+90.12")
	typ, err := r.Infer(n)
	if err != nil {
		t.Fatal(err)
	}
	if typ != value.Type(value.PhysicalTypeFloat) {
		t.Errorf("+12.34 类应为 float，实际 %q", typ)
	}
}

// findDominantType 回退：经 Infer 调不到（样本恒非空），直接调用。
func TestGapI2_DominantFallback(t *testing.T) {
	r := NewPhysicalTypeInferenceRule()
	if got := r.findDominantType(map[value.PhysicalType]int{}, 0); got != value.PhysicalTypeString {
		t.Errorf("空样本应回退 string，实际 %q", got)
	}
}

// isEnumLike 模式命中循环：低占比双值走 ForEach，覆盖 363。
func TestGapI2_EnumPatternLoop(t *testing.T) {
	r := NewLogicalTypeInferenceRule()
	n := node.NewRequestPathVariableNode("tag", "")
	for i := 0; i < 5; i++ {
		n.ObserveValue("aa")
		n.ObserveValue("bb")
	}
	typ, err := r.Infer(n)
	if err != nil {
		t.Fatal(err)
	}
	if typ != value.Type(value.LogicalTypeEnum) {
		t.Errorf("aa/bb 高频重复应判枚举，实际 %q", typ)
	}
}

// 结构排除：3 个低频 UUID + 1 个高频普通值，结构化匹配率不足但
// 枚举内模式占比超半，覆盖 377-379。
func TestGapI2_EnumStructuredExclude(t *testing.T) {
	r := NewLogicalTypeInferenceRule()
	n := node.NewRequestPathVariableNode("id", "")
	for _, id := range []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"550e8400-e29b-41d4-a716-446655440001",
		"550e8400-e29b-41d4-a716-446655440002",
	} {
		n.ObserveValue(id)
	}
	for i := 0; i < 11; i++ {
		n.ObserveValue("aa")
	}
	typ, err := r.Infer(n)
	if err != nil {
		t.Fatal(err)
	}
	if typ == value.Type(value.LogicalTypeEnum) {
		t.Error("UUID 占比超半不应判枚举")
	}
}

// isDecimalLike 符号位："+12.34" 整数部符号跳过，覆盖 411-412。
func TestGapI2_DecimalSign(t *testing.T) {
	r := NewLogicalTypeInferenceRule()
	n := node.NewRequestPathVariableNode("amount", "")
	for i := 0; i < 5; i++ {
		n.ObserveValue("+12.34")
	}
	typ, err := r.Infer(n)
	if err != nil {
		t.Fatal(err)
	}
	if typ != value.Type(value.LogicalTypeDecimal) {
		t.Errorf("+12.34 应判 decimal，实际 %q", typ)
	}
}
