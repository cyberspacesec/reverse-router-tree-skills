package inference

import (
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

func TestPhysicalTypeInference_Integer(t *testing.T) {
	rule := NewPhysicalTypeInferenceRule()

	// 创建路径变量节点并观察整数值
	varNode := node.NewRequestPathVariableNode("id", "[0-9]+")
	varNode.ObserveValue("123")
	varNode.ObserveValue("456")
	varNode.ObserveValue("789")

	inferredType, err := rule.Infer(varNode)
	if err != nil {
		t.Fatalf("Infer 失败: %v", err)
	}

	if inferredType != value.Type(value.PhysicalTypeInteger) {
		t.Errorf("推断类型错误，期望 'integer'，得到 '%s'", inferredType)
	}
}

func TestPhysicalTypeInference_Float(t *testing.T) {
	rule := NewPhysicalTypeInferenceRule()

	varNode := node.NewRequestPathVariableNode("price", "")
	varNode.ObserveValue("12.34")
	varNode.ObserveValue("56.78")
	varNode.ObserveValue("90.12")

	inferredType, err := rule.Infer(varNode)
	if err != nil {
		t.Fatalf("Infer 失败: %v", err)
	}

	if inferredType != value.Type(value.PhysicalTypeFloat) {
		t.Errorf("推断类型错误，期望 'float'，得到 '%s'", inferredType)
	}
}

func TestPhysicalTypeInference_Boolean(t *testing.T) {
	rule := NewPhysicalTypeInferenceRule()

	varNode := node.NewRequestPathVariableNode("active", "")
	varNode.ObserveValue("true")
	varNode.ObserveValue("false")
	varNode.ObserveValue("true")

	inferredType, err := rule.Infer(varNode)
	if err != nil {
		t.Fatalf("Infer 失败: %v", err)
	}

	if inferredType != value.Type(value.PhysicalTypeBoolean) {
		t.Errorf("推断类型错误，期望 'boolean'，得到 '%s'", inferredType)
	}
}

func TestPhysicalTypeInference_String(t *testing.T) {
	rule := NewPhysicalTypeInferenceRule()

	varNode := node.NewRequestPathVariableNode("name", "")
	varNode.ObserveValue("alice")
	varNode.ObserveValue("bob")
	varNode.ObserveValue("charlie")

	inferredType, err := rule.Infer(varNode)
	if err != nil {
		t.Fatalf("Infer 失败: %v", err)
	}

	if inferredType != value.Type(value.PhysicalTypeString) {
		t.Errorf("推断类型错误，期望 'string'，得到 '%s'", inferredType)
	}
}

func TestPhysicalTypeInference_MixedTypes(t *testing.T) {
	rule := NewPhysicalTypeInferenceRule()

	varNode := node.NewRequestPathVariableNode("id", "")
	// 大部分是整数，少量是字符串
	varNode.ObserveValue("123")
	varNode.ObserveValue("456")
	varNode.ObserveValue("abc")
	varNode.ObserveValue("789")

	inferredType, err := rule.Infer(varNode)
	if err != nil {
		t.Fatalf("Infer 失败: %v", err)
	}

	// 整数占多数，应该推断为整数
	if inferredType != value.Type(value.PhysicalTypeInteger) {
		t.Errorf("混合类型推断错误，期望 'integer'（多数类型），得到 '%s'", inferredType)
	}
}

func TestPhysicalTypeInference_EmptyNode(t *testing.T) {
	rule := NewPhysicalTypeInferenceRule()

	// 空的路径变量节点
	varNode := node.NewRequestPathVariableNode("id", "")

	inferredType, err := rule.Infer(varNode)
	if err != nil {
		t.Fatalf("Infer 失败: %v", err)
	}

	// 没有值采样，默认返回 string
	if inferredType != value.Type(value.PhysicalTypeString) {
		t.Errorf("空节点推断类型错误，期望 'string'，得到 '%s'", inferredType)
	}
}

func TestPhysicalTypeInference_NilNode(t *testing.T) {
	rule := NewPhysicalTypeInferenceRule()

	// 普通节点（没有 ValueMetric）
	normalNode := node.NewRequestPathNode("test")

	inferredType, err := rule.Infer(normalNode)
	if err != nil {
		t.Fatalf("Infer 失败: %v", err)
	}

	// 普通节点没有值采样，默认返回 string
	if inferredType != value.Type(value.PhysicalTypeString) {
		t.Errorf("普通节点推断类型错误，期望 'string'，得到 '%s'", inferredType)
	}
}

func TestPhysicalTypeInference_NullValues(t *testing.T) {
	rule := NewPhysicalTypeInferenceRule()

	varNode := node.NewRequestPathVariableNode("val", "")
	varNode.ObserveValue("null")
	varNode.ObserveValue("NULL")
	varNode.ObserveValue("")

	inferredType, err := rule.Infer(varNode)
	if err != nil {
		t.Fatalf("Infer 失败: %v", err)
	}

	if inferredType != value.Type(value.PhysicalTypeNull) {
		t.Errorf("null值推断类型错误，期望 'null'，得到 '%s'", inferredType)
	}
}

// 测试从 RequestParamNode 推断物理类型
func TestPhysicalTypeInference_FromParamNode(t *testing.T) {
	rule := NewPhysicalTypeInferenceRule()

	// 整数参数
	paramNode := node.NewRequestParamNode("page", "1", false)
	paramNode.ObserveValue("1")
	paramNode.ObserveValue("2")
	paramNode.ObserveValue("3")

	inferredType, err := rule.Infer(paramNode)
	if err != nil {
		t.Fatalf("Infer 失败: %v", err)
	}

	if inferredType != value.Type(value.PhysicalTypeInteger) {
		t.Errorf("整数参数应该推断为 'integer'，实际: '%s'", inferredType)
	}

	// 浮点数参数
	floatParam := node.NewRequestParamNode("price", "0", false)
	floatParam.ObserveValue("1.5")
	floatParam.ObserveValue("2.5")
	floatParam.ObserveValue("3.5")

	inferredType, err = rule.Infer(floatParam)
	if err != nil {
		t.Fatalf("Infer 失败: %v", err)
	}

	if inferredType != value.Type(value.PhysicalTypeFloat) {
		t.Errorf("浮点参数应该推断为 'float'，实际: '%s'", inferredType)
	}
}

// 测试从 RequestParamNode 推断物理类型 - 字符串
func TestPhysicalTypeInference_FromParamNodeString(t *testing.T) {
	rule := NewPhysicalTypeInferenceRule()

	paramNode := node.NewRequestParamNode("name", "", false)
	paramNode.ObserveValue("alice")
	paramNode.ObserveValue("bob")
	paramNode.ObserveValue("charlie")

	inferredType, err := rule.Infer(paramNode)
	if err != nil {
		t.Fatalf("Infer 失败: %v", err)
	}

	if inferredType != value.Type(value.PhysicalTypeString) {
		t.Errorf("字符串参数应该推断为 'string'，实际: '%s'", inferredType)
	}
}

// 测试长数字串降级为 string
// 16位及以上纯数字串（银行卡号、身份证号、超长业务ID）应识别为 string 而非 integer，
// 因为这些值是标识符语义，业务系统普遍以 string 存储，且存在 int64 溢出风险。
func TestPhysicalTypeInference_LongDigitString(t *testing.T) {
	rule := NewPhysicalTypeInferenceRule()

	cases := []struct {
		name   string
		values []string
	}{
		// 16位：银行卡号最小长度
		{"16位数字串", []string{"6222021234567890", "6225887654321098", "6217001234567890"}},
		// 18位：身份证号长度
		{"18位数字串", []string{"110101199003072314", "310101198506153214", "440101199012251234"}},
		// 19位：银行卡号最大长度
		{"19位数字串", []string{"6222021234567890123", "6225887654321098765", "6217001234567890123"}},
		// >19位：超长业务ID
		{"25位超长数字串", []string{"1234567890123456789012345", "9876543210987654321098765"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			varNode := node.NewRequestPathVariableNode("id", "")
			for _, v := range c.values {
				varNode.ObserveValue(v)
			}

			inferredType, err := rule.Infer(varNode)
			if err != nil {
				t.Fatalf("Infer 失败: %v", err)
			}

			if inferredType != value.Type(value.PhysicalTypeString) {
				t.Errorf("%s 应降级为 'string'（标识符语义），实际: '%s'", c.name, inferredType)
			}
		})
	}
}

// 测试15位以下数字仍识别为 integer
// 15位是身份证旧版长度，但也能放进 int64，且作为算术整数的可能性更高
func TestPhysicalTypeInference_ShortDigitInteger(t *testing.T) {
	rule := NewPhysicalTypeInferenceRule()

	cases := []struct {
		name   string
		values []string
	}{
		{"1位数字", []string{"1", "2", "3"}},
		{"4位数字", []string{"1001", "1002", "1003"}},
		{"11位手机号", []string{"13812345678", "15912345678", "18612345678"}},
		{"15位数字", []string{"123456789012345", "987654321098765"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			varNode := node.NewRequestPathVariableNode("id", "")
			for _, v := range c.values {
				varNode.ObserveValue(v)
			}

			inferredType, err := rule.Infer(varNode)
			if err != nil {
				t.Fatalf("Infer 失败: %v", err)
			}

			if inferredType != value.Type(value.PhysicalTypeInteger) {
				t.Errorf("%s 应识别为 'integer'，实际: '%s'", c.name, inferredType)
			}
		})
	}
}

// 测试科学计数法识别为 float
// 科学计数法（1e5, 1.5e3, 2E-3）本质是数值，应识别为 float 而非 string。
func TestPhysicalTypeInference_ScientificNotation(t *testing.T) {
	rule := NewPhysicalTypeInferenceRule()

	cases := []struct {
		name   string
		values []string
	}{
		{"简单科学计数法", []string{"1e5", "2e3", "3e4"}},
		{"带小数尾数", []string{"1.5e3", "2.5e4", "3.14e2"}},
		{"大写E", []string{"1E5", "2E3", "3E4"}},
		{"负指数", []string{"2e-3", "3e-4", "5e-2"}},
		{"正指数", []string{"1.5e+3", "2.5e+4", "3.14e+2"}},
		{"负数科学计数法", []string{"-3.14e2", "-1.5e3", "-2.5e4"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			varNode := node.NewRequestPathVariableNode("val", "")
			for _, v := range c.values {
				varNode.ObserveValue(v)
			}

			inferredType, err := rule.Infer(varNode)
			if err != nil {
				t.Fatalf("Infer 失败: %v", err)
			}

			if inferredType != value.Type(value.PhysicalTypeFloat) {
				t.Errorf("%s 应识别为 'float'，实际: '%s'", c.name, inferredType)
			}
		})
	}
}

// 测试非科学计数法不被误判
func TestPhysicalTypeInference_NotScientificNotation(t *testing.T) {
	rule := NewPhysicalTypeInferenceRule()

	// 这些看起来像科学计数法但实际不是合法数值
	cases := []struct {
		name   string
		values []string
		expect value.PhysicalType
	}{
		// e 在末位，无指数
		{"e无指数", []string{"1e", "2e", "3e"}, value.PhysicalTypeString},
		// e 在首位，无尾数
		{"e无尾数", []string{"e5", "e3", "e4"}, value.PhysicalTypeString},
		// 多个e
		{"多个e", []string{"1e2e3", "2e3e4"}, value.PhysicalTypeString},
		// 纯字母字符串
		{"纯字母", []string{"abc", "def", "ghi"}, value.PhysicalTypeString},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			varNode := node.NewRequestPathVariableNode("val", "")
			for _, v := range c.values {
				varNode.ObserveValue(v)
			}

			inferredType, err := rule.Infer(varNode)
			if err != nil {
				t.Fatalf("Infer 失败: %v", err)
			}

			if inferredType != value.Type(c.expect) {
				t.Errorf("%s 应为 '%s'，实际: '%s'", c.name, c.expect, inferredType)
			}
		})
	}
}

// 测试十六进制识别为 integer
// 十六进制（0x/0X 前缀）是明确的数值表示，应识别为 integer。
func TestPhysicalTypeInference_HexInteger(t *testing.T) {
	rule := NewPhysicalTypeInferenceRule()

	cases := []struct {
		name   string
		values []string
	}{
		{"小写0x前缀", []string{"0x1a", "0x2b", "0x3c"}},
		{"大写0X前缀", []string{"0X1A", "0X2B", "0X3C"}},
		{"长十六进制", []string{"0xDEADBEEF", "0xCAFEBABE", "0x12345678"}},
		{"颜色值", []string{"0xFF0000", "0x00FF00", "0x0000FF"}},
		{"单个十六进制", []string{"0xF", "0xA", "0x0"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			varNode := node.NewRequestPathVariableNode("val", "")
			for _, v := range c.values {
				varNode.ObserveValue(v)
			}

			inferredType, err := rule.Infer(varNode)
			if err != nil {
				t.Fatalf("Infer 失败: %v", err)
			}

			if inferredType != value.Type(value.PhysicalTypeInteger) {
				t.Errorf("%s 十六进制应识别为 'integer'，实际: '%s'", c.name, inferredType)
			}
		})
	}
}

// 测试非十六进制不被误判
func TestPhysicalTypeInference_NotHexInteger(t *testing.T) {
	rule := NewPhysicalTypeInferenceRule()

	cases := []struct {
		name   string
		values []string
		expect value.PhysicalType
	}{
		// 0x 但无后续数字
		{"0x无数字", []string{"0x", "0X", "0xG"}, value.PhysicalTypeString},
		// 纯0x前缀字符串
		{"x开头非十六进制", []string{"xyz", "x1a", "x2b"}, value.PhysicalTypeString},
		// 普通字符串
		{"普通字符串", []string{"hello", "world", "test"}, value.PhysicalTypeString},
		// 正常数字仍为integer
		{"正常数字", []string{"123", "456", "789"}, value.PhysicalTypeInteger},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			varNode := node.NewRequestPathVariableNode("val", "")
			for _, v := range c.values {
				varNode.ObserveValue(v)
			}

			inferredType, err := rule.Infer(varNode)
			if err != nil {
				t.Fatalf("Infer 失败: %v", err)
			}

			if inferredType != value.Type(c.expect) {
				t.Errorf("%s 应为 '%s'，实际: '%s'", c.name, c.expect, inferredType)
			}
		})
	}
}
