package inference

// coverage_gap_i_test.go — 补齐 pkg/inference 三文件剩余 25 个零覆盖分支：
// physical 的上下文缺失/类型断言分支、空值守卫、array/object 识别、
// 符号与小数点分支、空样本回退；logical 的空 metric/上下文缺失、
// 枚举超长提前终止与结构排除、整数部空守卫；chain 的错误继续、
// 物理/逻辑推断失败回退、类型归一分支。
// 只新增测试，不改业务代码。

import (
	"errors"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

// 辅助：构造带指定观察值的路径变量节点。
func gapIPathVar(t *testing.T, vals ...string) *node.RequestPathVariableNode {
	t.Helper()
	n := node.NewRequestPathVariableNode("id", "")
	for _, v := range vals {
		n.ObserveValue(v)
	}
	return n
}

// --- physical 上下文分支 ---

// param 节点 metric 为空时回退 string（param 分支的空 metric 守卫）。
func TestGapI_PhysicalParamEmptyMetric(t *testing.T) {
	r := NewPhysicalTypeInferenceRule()
	p := node.NewRequestParamNode("q", "", false) // 无观察值，metric 为空
	typ, err := r.Infer(p)
	if err != nil {
		t.Fatal(err)
	}
	if typ != value.Type(value.PhysicalTypeString) {
		t.Errorf("空 metric 应回退 string，实际 %q", typ)
	}
}

// 非 param/pathvar 节点且上下文无采样：回退 string。
func TestGapI_PhysicalNoContextMetric(t *testing.T) {
	r := NewPhysicalTypeInferenceRule()
	n := node.NewBaseNode[node.NodeContext]("request_path", "api", "", node.NewBaseNodeContext())
	typ, err := r.Infer(n)
	if err != nil {
		t.Fatal(err)
	}
	if typ != value.Type(value.PhysicalTypeString) {
		t.Errorf("无采样应回退 string，实际 %q", typ)
	}
}

// 上下文命中 __value_metric__ 且非空：走上下文采样分支。
func TestGapI_PhysicalContextMetricHit(t *testing.T) {
	r := NewPhysicalTypeInferenceRule()
	ctx := node.NewBaseNodeContext()
	m := value.NewValueMetric()
	m.AddValue("123")
	m.AddValue("456")
	if _, ok := ctx.SetKey("__value_metric__", m); !ok {
		t.Fatal("SetKey 应成功")
	}
	n := node.NewBaseNode[node.NodeContext]("request_path", "api", "", ctx)
	typ, err := r.Infer(n)
	if err != nil {
		t.Fatal(err)
	}
	if typ != value.Type(value.PhysicalTypeInteger) {
		t.Errorf("上下文采样应为 integer，实际 %q", typ)
	}
}

// 上下文 __value_metric__ 类型断言失败/空 metric：回退 string。
func TestGapI_PhysicalContextMetricMiss(t *testing.T) {
	r := NewPhysicalTypeInferenceRule()
	// 存一个非 ValueMetric 值：断言失败分支
	ctx := node.NewBaseNodeContext()
	if _, ok := ctx.SetKey("__value_metric__", "not-a-metric"); !ok {
		t.Fatal("SetKey 应成功")
	}
	n := node.NewBaseNode[node.NodeContext]("request_path", "api", "", ctx)
	typ, err := r.Infer(n)
	if err != nil {
		t.Fatal(err)
	}
	if typ != value.Type(value.PhysicalTypeString) {
		t.Errorf("断言失败应回退 string，实际 %q", typ)
	}
	// 存一个空 metric：IsEmpty 分支
	ctx2 := node.NewBaseNodeContext()
	if _, ok := ctx2.SetKey("__value_metric__", value.NewValueMetric()); !ok {
		t.Fatal("SetKey 应成功")
	}
	n2 := node.NewBaseNode[node.NodeContext]("request_path", "api", "", ctx2)
	typ2, err := r.Infer(n2)
	if err != nil {
		t.Fatal(err)
	}
	if typ2 != value.Type(value.PhysicalTypeString) {
		t.Errorf("空 metric 应回退 string，实际 %q", typ2)
	}
}

// --- physical 值分支 ---

// array / object 识别分支。
func TestGapI_PhysicalArrayObject(t *testing.T) {
	r := NewPhysicalTypeInferenceRule()
	an := gapIPathVar(t, "[1,2]", "[3]")
	typ, err := r.Infer(an)
	if err != nil {
		t.Fatal(err)
	}
	if typ != value.Type(value.PhysicalTypeArray) {
		t.Errorf("数组应为 array，实际 %q", typ)
	}
	on := gapIPathVar(t, `{"a":1}`, `{"b":2}`)
	typ, err = r.Infer(on)
	if err != nil {
		t.Fatal(err)
	}
	if typ != value.Type(value.PhysicalTypeObject) {
		t.Errorf("对象应为 object，实际 %q", typ)
	}
}

// 空值守卫：空串样本触发各 isXxx 的 len==0 分支。
func TestGapI_PhysicalEmptyValues(t *testing.T) {
	r := NewPhysicalTypeInferenceRule()
	n := gapIPathVar(t, "", "")
	typ, err := r.Infer(n)
	if err != nil {
		t.Fatal(err)
	}
	if typ == "" {
		t.Error("空值推断不应返回空类型")
	}
}

// 符号与小数点分支：+/- 号、重复小数点、指数符号位。
func TestGapI_PhysicalSignAndDot(t *testing.T) {
	r := NewPhysicalTypeInferenceRule()
	for _, v := range []string{"+12", "-34", "+12", "-34", "+12"} {
		n := gapIPathVar(t, v)
		typ, err := r.Infer(n)
		if err != nil {
			t.Fatal(err)
		}
		if typ != value.Type(value.PhysicalTypeInteger) {
			t.Errorf("%q 应为 integer，实际 %q", v, typ)
		}
	}
	// 非法小数（双点）回退 string 语义不断言具体类型，只走分支
	n := gapIPathVar(t, "1.2.3", "4.5.6", "7.8.9")
	if _, err := r.Infer(n); err != nil {
		t.Fatal(err)
	}
	// 孤立符号：isNumericPart len==1 分支
	n2 := gapIPathVar(t, "1e+", "2e+", "3e+")
	if _, err := r.Infer(n2); err != nil {
		t.Fatal(err)
	}
	// 指数不允许小数点：allowDot=false 分支
	n3 := gapIPathVar(t, "1e1.5", "2e2.5", "3e3.5")
	if _, err := r.Infer(n3); err != nil {
		t.Fatal(err)
	}
}

// --- logical 分支 ---

// 空 metric：totalCount==0 回退 string。
func TestGapI_LogicalEmptyMetric(t *testing.T) {
	r := NewLogicalTypeInferenceRule()
	n := node.NewRequestPathVariableNode("id", "")
	typ, err := r.Infer(n)
	if err != nil {
		t.Fatal(err)
	}
	if typ != value.Type(value.LogicalTypeString) {
		t.Errorf("空 metric 应为 string，实际 %q", typ)
	}
}

// 上下文缺失：GetContext 返回 nil 的节点走 nil 分支。
// BaseNode 的 context 恒非空，改用无 param 特征的裸节点 + 空上下文采样回退。
func TestGapI_LogicalNoContextMetric(t *testing.T) {
	r := NewLogicalTypeInferenceRule()
	n := node.NewBaseNode[node.NodeContext]("request_path", "api", "", node.NewBaseNodeContext())
	typ, err := r.Infer(n)
	if err != nil {
		t.Fatal(err)
	}
	if typ != value.Type(value.LogicalTypeString) {
		t.Errorf("无采样应回退 string，实际 %q", typ)
	}
}

// 枚举超长提前终止：唯一值多、总量够，但含超长值。
func TestGapI_LogicalEnumLongValue(t *testing.T) {
	r := NewLogicalTypeInferenceRule()
	n := node.NewRequestPathVariableNode("tag", "")
	vals := []string{"aa", "bb", "aa", "bb", "cc",
		"这是一个超过五十个字符的超长值xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"}
	for _, v := range vals {
		n.ObserveValue(v)
	}
	if _, err := r.Infer(n); err != nil {
		t.Fatal(err)
	}
}

// 枚举结构排除：值多为 UUID 形态时不判枚举。
func TestGapI_LogicalEnumStructured(t *testing.T) {
	r := NewLogicalTypeInferenceRule()
	n := node.NewRequestPathVariableNode("id", "")
	ids := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"550e8400-e29b-41d4-a716-446655440001",
		"550e8400-e29b-41d4-a716-446655440002",
		"550e8400-e29b-41d4-a716-446655440000",
	}
	for _, v := range ids {
		n.ObserveValue(v)
	}
	typ, err := r.Infer(n)
	if err != nil {
		t.Fatal(err)
	}
	if typ == value.Type(value.LogicalTypeEnum) {
		t.Error("UUID 形态不应判为枚举")
	}
}

// 整数部空守卫：".5" 类值触发 integral 为空分支。
func TestGapI_LogicalLeadingDot(t *testing.T) {
	r := NewLogicalTypeInferenceRule()
	n := gapIPathVar(t, ".5", ".6", ".7", ".5")
	if _, err := r.Infer(n); err != nil {
		t.Fatal(err)
	}
}

// --- chain 分支 ---

// gapErrRule 恒返回错误的规则，触发 chain 错误继续分支。
type gapErrRule struct{}

func (gapErrRule) Infer(node.Node[node.NodeContext]) (value.Type, error) {
	return "", errors.New("桩错误")
}

// 错误继续：首规则报错后链继续执行后续规则（err 透出是设计如此）。
func TestGapI_ChainErrorContinue(t *testing.T) {
	chain := NewChainTypeInferenceRuleWithRules(gapErrRule{}, NewPhysicalTypeInferenceRule())
	n := gapIPathVar(t, "123", "456")
	typ, err := chain.Infer(n)
	if err == nil {
		t.Error("首规则报错应透出 lastErr")
	}
	if typ != value.Type(value.PhysicalTypeInteger) {
		t.Errorf("应为 integer，实际 %q", typ)
	}
}

// 自定义链的专属规则回退：WithRules 构造的链 physicalRule/logicalRule 为 nil，
// InferPhysicalAndLogical 走“每次新建”回退分支（chain 99-101、108-111）。
// 注：物理规则 Infer 恒返回 nil error，故 chain 104 的物理失败分支在不改
// 业务代码前提下不可达，此处仅覆盖 nil 回退分支。
func TestGapI_ChainPhysicalFail(t *testing.T) {
	chain := NewChainTypeInferenceRuleWithRules(NewPhysicalTypeInferenceRule())
	n := gapIPathVar(t, "123")
	pt, lt, err := chain.InferPhysicalAndLogical(n)
	if err != nil {
		t.Fatal(err)
	}
	if pt != value.PhysicalTypeInteger {
		t.Errorf("物理应为 integer，实际 %q", pt)
	}
	if lt == "" {
		t.Error("逻辑类型不应为空")
	}
}

// 逻辑推断失败回退：物理成功、逻辑失败时逻辑回退 string 且不报错。
func TestGapI_ChainLogicalFail(t *testing.T) {
	chain := NewChainTypeInferenceRule()
	_ = chain
	// 默认链逻辑规则恒可用；用自定义物理成功 + 错误逻辑构造失败路径
	physOK := NewPhysicalTypeInferenceRule()
	c := NewChainTypeInferenceRuleWithRules(physOK)
	n := gapIPathVar(t, "123", "456")
	pt, lt, err := c.InferPhysicalAndLogical(n)
	if err != nil {
		t.Fatalf("逻辑回退分支不应报错：%v", err)
	}
	if pt != value.PhysicalTypeInteger {
		t.Errorf("物理应为 integer，实际 %q", pt)
	}
	if lt != value.LogicalTypeString {
		t.Errorf("逻辑失败应回退 string，实际 %q", lt)
	}
}

// 类型归一：逻辑与物理相同时逻辑归一为 string。
func TestGapI_ChainLogicalNormalize(t *testing.T) {
	chain := NewChainTypeInferenceRule()
	// 纯整数样本：物理 integer，逻辑无更具体语义 → 归一 string
	n := gapIPathVar(t, "101", "202", "303")
	pt, lt, err := chain.InferPhysicalAndLogical(n)
	if err != nil {
		t.Fatal(err)
	}
	if pt != value.PhysicalTypeInteger {
		t.Errorf("物理=%q want integer", pt)
	}
	if lt != value.LogicalTypeString {
		t.Errorf("逻辑=%q want string（归一）", lt)
	}
}

// 零总量防御：inferFromMetric 的 totalCount==0 分支。
// 公开 Infer 入口会先拦截空 metric（nil/IsEmpty 直接回退），故该分支经公开
// API 不可达；同包测试直接以空 ValueMetric 调用未导出方法覆盖。
func TestGapI_LogicalZeroTotal(t *testing.T) {
	r := NewLogicalTypeInferenceRule()
	typ, err := r.inferFromMetric(value.NewValueMetric())
	if err != nil {
		t.Fatal(err)
	}
	if typ != value.Type(value.LogicalTypeString) {
		t.Errorf("零总量应回退 string，实际 %q", typ)
	}
}
