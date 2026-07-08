package inference

import (
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

// PhysicalTypeInferenceRule 实现了物理类型推断规则
// 该规则基于对值的统计分析来确定最可能的物理数据类型。
// 它支持基本类型的检测，包括：整数、浮点数、布尔值、字符串、数组和对象。
//
// 工作原理：
// 1. 收集值的样本
// 2. 分析每个样本可能属于的类型
// 3. 统计每种类型的匹配次数
// 4. 选择匹配次数最多的类型作为推断结果
type PhysicalTypeInferenceRule struct {
	// 可以添加配置参数，如阈值等
	// 例如：MinSampleSize - 进行推断所需的最小样本数量
	// 例如：StringThreshold - 将值归类为字符串的阈值百分比
}

// 确保 PhysicalTypeInferenceRule 实现了 TypeInferenceRule 接口
var _ TypeInferenceRule = (*PhysicalTypeInferenceRule)(nil)

// NewPhysicalTypeInferenceRule 创建一个新的物理类型推断规则实例
//
// 返回值:
//   - *PhysicalTypeInferenceRule: 已初始化的推断规则对象
func NewPhysicalTypeInferenceRule() *PhysicalTypeInferenceRule {
	return &PhysicalTypeInferenceRule{}
}

// Infer 根据节点上下文和值采样推断物理类型
//
// 该方法分析节点上下文中包含的值采样，并推断出最可能的物理数据类型。
// 如果节点上下文为空，将返回空类型。
//
// 参数:
//   - node: 包含上下文和值采样的节点
//
// 返回值:
//   - value.Type: 推断出的数据类型
//   - error: 如果推断过程中发生错误，则返回错误信息
func (r *PhysicalTypeInferenceRule) Infer(n node.Node[node.NodeContext]) (value.Type, error) {
	// 首先尝试从 RequestPathVariableNode 获取 ValueMetric
	if pathVarNode, ok := n.(*node.RequestPathVariableNode); ok {
		metric := pathVarNode.GetValueMetric()
		if metric != nil && !metric.IsEmpty() {
			return r.inferFromSamples(metric)
		}
	}

	// 从 RequestParamNode 获取（参数值类型推断）
	if paramNode, ok := n.(*node.RequestParamNode); ok {
		metric := paramNode.GetValueMetric()
		if metric != nil && !metric.IsEmpty() {
			return r.inferFromSamples(metric)
		}
	}

	// 对于其他节点类型，从上下文中获取值采样
	context := n.GetContext()
	if context == nil {
		// 没有上下文信息，无法推断类型
		return "", nil
	}

	// 尝试从上下文中获取 ValueMetric
	// 上下文中可能存储了键为 "__value_metric__" 的 ValueMetric 对象
	if vm, exists := context.GetKey("__value_metric__"); exists {
		if metric, ok := vm.(*value.ValueMetric); ok && metric != nil && !metric.IsEmpty() {
			return r.inferFromSamples(metric)
		}
	}

	// 没有可用的值采样数据
	return value.Type(value.PhysicalTypeString), nil
}

// inferFromSamples 根据值采样推断物理类型
//
// 该方法是类型推断的核心，它分析值采样数据并确定最可能的数据类型。
// 推断过程分为三个主要步骤：
// 1. 初始化类型匹配计数
// 2. 分析所有值并统计各类型的匹配次数
// 3. 确定占主导地位的类型
//
// 参数:
//   - metric: 包含值采样数据的度量对象
//
// 返回值:
//   - value.Type: 推断出的数据类型
//   - error: 如果推断过程中发生错误，则返回错误信息
func (r *PhysicalTypeInferenceRule) inferFromSamples(metric *value.ValueMetric) (value.Type, error) {
	if metric == nil || metric.IsEmpty() {
		// 没有样本数据，无法进行有效推断，返回null类型
		return value.Type(value.PhysicalTypeNull), nil
	}

	// 初始化类型匹配计数
	typeMatches := r.initializeTypeMatches()

	// 分析所有值并统计匹配
	totalSamples := r.analyzeValues(metric, typeMatches)

	// 找出最优的类型
	dominantType := r.findDominantType(typeMatches, totalSamples)

	return value.Type(dominantType), nil
}

// initializeTypeMatches 初始化类型匹配计数映射
func (r *PhysicalTypeInferenceRule) initializeTypeMatches() map[value.PhysicalType]int {
	return map[value.PhysicalType]int{
		value.PhysicalTypeInteger: 0,
		value.PhysicalTypeFloat:   0,
		value.PhysicalTypeBoolean: 0,
		value.PhysicalTypeString:  0,
		value.PhysicalTypeObject:  0,
		value.PhysicalTypeArray:   0,
		value.PhysicalTypeNull:    0,
	}
}

// analyzeValues 分析所有值并统计每种类型的匹配次数
func (r *PhysicalTypeInferenceRule) analyzeValues(metric *value.ValueMetric, typeMatches map[value.PhysicalType]int) int {
	totalSamples := 0

	metric.ForEachValue(func(val string, count int) bool {
		totalSamples += count
		r.matchValueType(val, count, typeMatches)
		return true
	})

	return totalSamples
}

// matchValueType 匹配单个值的类型并更新计数
func (r *PhysicalTypeInferenceRule) matchValueType(val string, count int, typeMatches map[value.PhysicalType]int) {
	if r.isNull(val) {
		typeMatches[value.PhysicalTypeNull] += count
		return
	}

	if r.isBoolean(val) {
		typeMatches[value.PhysicalTypeBoolean] += count
		return
	}

	if r.isInteger(val) {
		typeMatches[value.PhysicalTypeInteger] += count
		return
	}

	// 十六进制数值（0x/0X 前缀）识别为 integer
	// 例如：0x1A, 0xFF, 0xDEADBEEF
	if r.isHexInteger(val) {
		typeMatches[value.PhysicalTypeInteger] += count
		return
	}

	if r.isFloat(val) {
		typeMatches[value.PhysicalTypeFloat] += count
		return
	}

	if r.isArray(val) {
		typeMatches[value.PhysicalTypeArray] += count
		return
	}

	if r.isObject(val) {
		typeMatches[value.PhysicalTypeObject] += count
		return
	}

	typeMatches[value.PhysicalTypeString] += count
}

// isHexInteger 检测值是否为十六进制整数
// 格式：0x 或 0X 前缀 + 至少一位十六进制数字（0-9, a-f, A-F）
// 例如：0x1A, 0XFF, 0xDEADBEEF
func (r *PhysicalTypeInferenceRule) isHexInteger(val string) bool {
	if len(val) < 3 { // 至少 "0x" + 1位数字
		return false
	}

	// 必须以 0x 或 0X 开头
	if val[0] != '0' || (val[1] != 'x' && val[1] != 'X') {
		return false
	}

	// 剩余部分必须是十六进制数字
	for _, c := range val[2:] {
		if !isHexDigit(c) {
			return false
		}
	}
	return true
}

// isHexDigit 判断字符是否为十六进制数字
func isHexDigit(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// isNull 检测值是否为空
func (r *PhysicalTypeInferenceRule) isNull(val string) bool {
	return val == "" || val == "null" || val == "NULL"
}

// isBoolean 检测值是否为布尔值
func (r *PhysicalTypeInferenceRule) isBoolean(val string) bool {
	return val == "true" || val == "false" || val == "TRUE" || val == "FALSE"
}

// isInteger 检测值是否为整数
//
// 长度上限规则：纯数字串长度 >= 16 位时降级为 string，不再识别为 integer。
// 理由：
//   - 16-19 位是银行卡号长度，18 位是身份证号长度，>19 位是超长业务ID
//   - 这些长度的数字串本质是标识符而非算术数值，业务系统普遍以 string 存储
//   - int64 最大值 9223372036854775807（19位），16位以上数字串存在溢出风险
//   - 逻辑层（LogicalTypeInferenceRule）仍会识别 idcard/bankcard 等语义类型
//
// 因此 16 位是"算术整数"与"标识符数字串"的合理分界线。
func (r *PhysicalTypeInferenceRule) isInteger(val string) bool {
	if len(val) == 0 {
		return false
	}

	// 超长数字串降级为 string（标识符语义，非算术整数）
	if len(val) >= 16 {
		return false
	}

	for i, c := range val {
		if i == 0 && (c == '+' || c == '-') {
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// isFloat 检测值是否为浮点数
// 支持以下格式：
//   - 标准小数：12.34, .5, 3.
//   - 科学计数法：1e5, 1.5e3, 2E-3, 1.5E+10
func (r *PhysicalTypeInferenceRule) isFloat(val string) bool {
	if len(val) == 0 {
		return false
	}

	// 先尝试科学计数法：包含 e 或 E，且整体是合法数值
	if r.isScientificNotation(val) {
		return true
	}

	hasDot := false
	for i, c := range val {
		if i == 0 && (c == '+' || c == '-') {
			continue
		}
		if c == '.' {
			if hasDot {
				return false
			}
			hasDot = true
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}

	return hasDot
}

// isScientificNotation 检测值是否为科学计数法表示的数值
// 格式：[+-]?数字[.数字]?[eE][+-]?数字
// 例如：1e5, 1.5e3, 2E-3, 1.5E+10, -3.14e2
func (r *PhysicalTypeInferenceRule) isScientificNotation(val string) bool {
	if len(val) == 0 {
		return false
	}

	// 查找 e 或 E 的位置（不在首位和末位）
	eIndex := -1
	for i, c := range val {
		if c == 'e' || c == 'E' {
			if eIndex != -1 {
				return false // 多个 e/E
			}
			eIndex = i
		}
	}
	if eIndex <= 0 || eIndex >= len(val)-1 {
		return false // e/E 必须在中间
	}

	// e 前面部分：[+-]?数字[.数字]?（可以是整数或小数）
	mantissa := val[:eIndex]
	if !r.isNumericPart(mantissa, true) {
		return false
	}

	// e 后面部分：[+-]?数字（必须是整数）
	exponent := val[eIndex+1:]
	return r.isNumericPart(exponent, false)
}

// isNumericPart 检查数值的某一部分是否合法
// allowDot: 是否允许小数点（尾数允许，指数不允许）
func (r *PhysicalTypeInferenceRule) isNumericPart(s string, allowDot bool) bool {
	if len(s) == 0 {
		return false
	}

	hasDigit := false
	hasDot := false
	for i, c := range s {
		if i == 0 && (c == '+' || c == '-') {
			// 符号只能在首位，且后面必须有内容
			if len(s) == 1 {
				return false
			}
			continue
		}
		if c == '.' {
			if !allowDot || hasDot {
				return false
			}
			hasDot = true
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
		hasDigit = true
	}
	return hasDigit
}

// isArray 检测值是否为数组
func (r *PhysicalTypeInferenceRule) isArray(val string) bool {
	return len(val) >= 2 && val[0] == '[' && val[len(val)-1] == ']'
}

// isObject 检测值是否为对象
func (r *PhysicalTypeInferenceRule) isObject(val string) bool {
	return len(val) >= 2 && val[0] == '{' && val[len(val)-1] == '}'
}

// findDominantType 找出匹配次数最多的类型
func (r *PhysicalTypeInferenceRule) findDominantType(typeMatches map[value.PhysicalType]int, totalSamples int) value.PhysicalType {
	var dominantType value.PhysicalType
	maxMatches := 0

	for pType, matches := range typeMatches {
		if matches > maxMatches {
			maxMatches = matches
			dominantType = pType
		}
	}

	if totalSamples == 0 || dominantType == "" {
		dominantType = value.PhysicalTypeString
	}

	return dominantType
}
