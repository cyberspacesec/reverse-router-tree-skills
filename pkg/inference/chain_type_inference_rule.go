package inference

import (
	"fmt"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

// ChainTypeInferenceRule 链式类型推断规则
// 按顺序执行多个推断规则，每个规则可以覆盖前一个规则的结果。
// 典型用法：先推断物理类型，再推断逻辑类型。
//
// 链式执行逻辑：
// 1. 按顺序执行每个规则
// 2. 如果某个规则返回了更具体的类型（非空且非默认string），则使用该结果
// 3. 如果某个规则返回错误，跳过该规则继续执行
// 4. 如果所有规则都没有返回有效结果，返回默认类型
type ChainTypeInferenceRule struct {
	rules []TypeInferenceRule
	// physicalRule/logicalRule 是 InferPhysicalAndLogical 复用的专属规则实例。
	// 这两个规则的 Infer 是无状态、并发安全的（PhysicalTypeInferenceRule 是空结构体，
	// LogicalTypeInferenceRule 的 patterns 在构造时一次性编译后只读），
	// 故可在 ChainTypeInferenceRule（单实例）中复用，避免每次推断都重建实例、
	// 重复编译 13+ 个正则（这是原本 InferPhysicalAndLogical 的主要 alloc 来源）。
	physicalRule *PhysicalTypeInferenceRule
	logicalRule  *LogicalTypeInferenceRule
}

// 确保 ChainTypeInferenceRule 实现了 TypeInferenceRule 接口
var _ TypeInferenceRule = (*ChainTypeInferenceRule)(nil)

// NewChainTypeInferenceRule 创建链式推断规则
// 默认包含物理类型推断和逻辑类型推断
func NewChainTypeInferenceRule() *ChainTypeInferenceRule {
	physical := NewPhysicalTypeInferenceRule()
	logical := NewLogicalTypeInferenceRule()
	return &ChainTypeInferenceRule{
		rules: []TypeInferenceRule{
			physical,
			logical,
		},
		physicalRule: physical,
		logicalRule:  logical,
	}
}

// NewChainTypeInferenceRuleWithRules 使用自定义规则链创建推断规则
func NewChainTypeInferenceRuleWithRules(rules ...TypeInferenceRule) *ChainTypeInferenceRule {
	return &ChainTypeInferenceRule{
		rules: rules,
	}
}

// AddRule 添加推断规则到链尾
func (c *ChainTypeInferenceRule) AddRule(rule TypeInferenceRule) {
	if rule != nil {
		c.rules = append(c.rules, rule)
	}
}

// Infer 执行链式推断
func (c *ChainTypeInferenceRule) Infer(n node.Node[node.NodeContext]) (value.Type, error) {
	var lastType value.Type
	var lastErr error

	for _, rule := range c.rules {
		inferredType, err := rule.Infer(n)
		if err != nil {
			lastErr = err
			continue
		}

		// 如果推断出了更具体的类型，使用该结果
		if inferredType != "" && inferredType != value.Type(value.PhysicalTypeString) {
			lastType = inferredType
		} else if lastType == "" {
			// 第一个规则的结果作为基础
			lastType = inferredType
		}
	}

	if lastType == "" {
		lastType = value.Type(value.PhysicalTypeString)
	}

	return lastType, lastErr
}

// InferPhysicalAndLogical 分别推断物理类型和逻辑类型
// 返回物理类型和逻辑类型，便于调用者获取更详细的类型信息
func (c *ChainTypeInferenceRule) InferPhysicalAndLogical(n node.Node[node.NodeContext]) (value.PhysicalType, value.LogicalType, error) {
	var physicalType value.PhysicalType
	var logicalType value.LogicalType

	// 复用构造时创建的专属规则实例，避免每次推断都重建实例并重新编译正则。
	// 仅在默认链（NewChainTypeInferenceRule）下复用；自定义链回退到每次新建。
	physicalRule := c.physicalRule
	if physicalRule == nil {
		physicalRule = NewPhysicalTypeInferenceRule()
	}
	pt, err := physicalRule.Infer(n)
	if err != nil {
		return value.PhysicalTypeString, value.LogicalTypeString, fmt.Errorf("物理类型推断失败: %w", err)
	}
	physicalType = value.PhysicalType(pt)

	logicalRule := c.logicalRule
	if logicalRule == nil {
		logicalRule = NewLogicalTypeInferenceRule()
	}
	lt, err := logicalRule.Infer(n)
	if err != nil {
		return physicalType, value.LogicalTypeString, nil
	}
	logicalType = value.LogicalType(lt)

	// 如果逻辑类型和物理类型相同，说明没有推断出更具体的逻辑类型
	// 此时逻辑类型保持为 string（表示没有更具体的语义信息）
	if logicalType == value.LogicalType(physicalType) && physicalType != value.PhysicalTypeString {
		logicalType = value.LogicalTypeString
	}

	return physicalType, logicalType, nil
}
