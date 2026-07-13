package inference

import (
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

// TestChainTypeInferenceRule_AddRule 覆盖 AddRule（含 nil 守卫）。
// AddRule 是上层项目扩展推断链的公开 API，此前 0% 覆盖率。
func TestChainTypeInferenceRule_AddRule(t *testing.T) {
	// 空链开始，逐条 AddRule 添加
	chain := NewChainTypeInferenceRuleWithRules()

	// nil 规则应被忽略（守卫分支）
	chain.AddRule(nil)

	// 添加一条逻辑规则
	chain.AddRule(NewLogicalTypeInferenceRule())

	pathVarNode := node.NewRequestPathVariableNode("id", "")
	for _, ip := range []string{"192.168.1.1", "10.0.0.1", "172.16.0.1"} {
		pathVarNode.ObserveValue(ip)
	}

	inferred, err := chain.Infer(pathVarNode)
	if err != nil {
		t.Fatalf("AddRule 后 Infer 失败: %v", err)
	}
	if inferred != value.Type(value.LogicalTypeIPAddress) {
		t.Errorf("AddRule 添加的规则应生效，期望 ipaddress，得 %s", inferred)
	}
}

// TestLogicalTypeInference_FromParamNode 覆盖 getMetricFromNode 的 param 命中分支。
// 此前逻辑推断只对 pathVar 节点测试，param 节点路径未覆盖（60% 缺口）。
func TestLogicalTypeInference_FromParamNode(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	// param 节点喂入 IP 值，应走 getMetricFromNode 的 param 分支命中
	paramNode := node.NewRequestParamNode("client_ip", "", false)
	paramNode.ObserveValue("192.168.1.1")
	paramNode.ObserveValue("10.0.0.1")

	inferred, err := rule.Infer(paramNode)
	if err != nil {
		t.Fatalf("param 节点 Infer 失败: %v", err)
	}
	if inferred != value.Type(value.LogicalTypeIPAddress) {
		t.Errorf("param 节点应推断为 ipaddress，得 %s", inferred)
	}
}

// TestLogicalTypeInference_ParamNodeEmptyMetric 覆盖 param 节点 metric 为空时
// 落到 context 分支的路径（metric != nil 但 IsEmpty）。
func TestLogicalTypeInference_ParamNodeEmptyMetric(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	// param 节点无任何观察值，metric 为空
	paramNode := node.NewRequestParamNode("empty", "", false)

	inferred, err := rule.Infer(paramNode)
	if err != nil {
		t.Fatalf("Infer 失败: %v", err)
	}
	// 空节点应回退为 string
	if inferred != value.Type(value.LogicalTypeString) {
		t.Errorf("空 metric param 应推断为 string，得 %s", inferred)
	}
}

// TestLogicalTypeInference_BaseNodeContextFallback 覆盖 getMetricFromNode 的
// context nil 守卫与无 __value_metric__ 回退分支。用纯 BaseNode 触发。
func TestLogicalTypeInference_BaseNodeContextFallback(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	// 纯 BaseNode：非 pathVar 非 param，走 context 分支
	// context 存在但无 __value_metric__ 键，最终回退 string
	baseNode := node.NewBaseNode[node.NodeContext](
		"custom", "k", "", node.NewBaseNodeContext())

	inferred, err := rule.Infer(baseNode)
	if err != nil {
		t.Fatalf("BaseNode Infer 失败: %v", err)
	}
	if inferred != value.Type(value.LogicalTypeString) {
		t.Errorf("无 metric 的 BaseNode 应回退 string，得 %s", inferred)
	}
}

// TestLogicalTypeInference_ContextValueMetricHit 覆盖 getMetricFromNode 的
// context.__value_metric__ 命中分支（此前 80% 缺口）。构造一个在上下文里
// 存有非空 ValueMetric 的 BaseNode，验证推断能用到该 metric。
func TestLogicalTypeInference_ContextValueMetricHit(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	ctx := node.NewBaseNodeContext()
	vm := value.NewValueMetric()
	vm.AddValue("192.168.1.1")
	vm.AddValue("10.0.0.1")
	ctx.SetKey("__value_metric__", vm)

	// 纯 BaseNode 但 context 持有 __value_metric__
	baseNode := node.NewBaseNode[node.NodeContext]("custom", "k", "", ctx)

	inferred, err := rule.Infer(baseNode)
	if err != nil {
		t.Fatalf("Infer 失败: %v", err)
	}
	// 应回退到 context metric 并识别为 IPAddress（验证走到了 metric 读取分支）
	if inferred != value.Type(value.LogicalTypeIPAddress) {
		t.Errorf("context __value_metric__ 命中后应推断为 ipaddress，得 %s", inferred)
	}
}
