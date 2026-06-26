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
//
// 使用示例:
//
//	rule := NewPhysicalTypeInferenceRule()
//	inferredType, err := rule.Infer(someNode)
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
func (r *PhysicalTypeInferenceRule) Infer(node node.Node[node.NodeContext]) (value.Type, error) {
	// 获取节点上下文中的值采样
	context := node.GetContext()
	if context == nil {
		// 没有上下文信息，无法推断类型
		return "", nil
	}

	// 假设我们从上下文中获取值采样
	// 在实际实现中，需要从节点上下文中提取值采样数据
	// TODO: 从节点上下文中提取实际的值采样数据
	valueMetric := value.NewValueMetric()

	// 基于采样数据推断物理类型
	return r.inferFromSamples(valueMetric)
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
//
// 特殊情况:
//   - 如果度量对象为nil或没有采样值，返回null类型
//   - 如果无法确定主导类型，默认返回string类型（最安全的选择）
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
//
// 为所有支持的物理类型创建一个初始化为零的计数映射，用于后续统计每种类型的匹配次数。
//
// 返回值:
//   - map[value.PhysicalType]int: 初始化的类型计数映射
func (r *PhysicalTypeInferenceRule) initializeTypeMatches() map[value.PhysicalType]int {
	return map[value.PhysicalType]int{
		value.PhysicalTypeInteger: 0, // 整数类型
		value.PhysicalTypeFloat:   0, // 浮点数类型
		value.PhysicalTypeBoolean: 0, // 布尔类型
		value.PhysicalTypeString:  0, // 字符串类型
		value.PhysicalTypeObject:  0, // 对象类型
		value.PhysicalTypeArray:   0, // 数组类型
		value.PhysicalTypeNull:    0, // null类型
	}
}

// analyzeValues 分析所有值并统计每种类型的匹配次数
//
// 遍历所有采样值，对每个值执行类型匹配，并累计每种类型的匹配计数。
// 同时返回总样本数，用于后续计算比例和确定主导类型。
//
// 参数:
//   - metric: 包含值采样数据的度量对象
//   - typeMatches: 用于累计各类型匹配次数的映射
//
// 返回值:
//   - int: 处理的总样本数
func (r *PhysicalTypeInferenceRule) analyzeValues(metric *value.ValueMetric, typeMatches map[value.PhysicalType]int) int {
	totalSamples := 0

	// 遍历所有采样值并尝试匹配类型
	for val, count := range metric.GetAllValues() {
		totalSamples += count
		// 对每个唯一的值执行类型匹配，并加权计数
		r.matchValueType(val, count, typeMatches)
	}

	return totalSamples
}

// matchValueType 匹配单个值的类型并更新计数
//
// 该方法按照一定的优先级顺序检测值的类型：
// 1. 首先检测空值（null）
// 2. 然后检测布尔值
// 3. 然后检测整数值
// 4. 然后检测浮点数值
// 5. 然后检测数组和对象
// 6. 如果以上都不匹配，则归类为字符串
//
// 注意: 顺序很重要，因为某些类型检测可能存在重叠（例如，整数也可能匹配浮点数格式）
//
// 参数:
//   - val: 要分析的单个值
//   - count: 该值在样本中出现的次数
//   - typeMatches: 用于累计各类型匹配次数的映射
func (r *PhysicalTypeInferenceRule) matchValueType(val string, count int, typeMatches map[value.PhysicalType]int) {
	// 检测空值（优先级最高）
	if r.isNull(val) {
		typeMatches[value.PhysicalTypeNull] += count
		return
	}

	// 检测布尔值（优先级次之）
	if r.isBoolean(val) {
		typeMatches[value.PhysicalTypeBoolean] += count
		return
	}

	// 检测整数（优先于浮点数，因为整数可以表示为浮点数，但反之不一定）
	if r.isInteger(val) {
		typeMatches[value.PhysicalTypeInteger] += count
		return
	}

	// 检测浮点数
	if r.isFloat(val) {
		typeMatches[value.PhysicalTypeFloat] += count
		return
	}

	// 检测数组和对象（基于简单的语法检测）
	if r.isArray(val) {
		typeMatches[value.PhysicalTypeArray] += count
		return
	}

	if r.isObject(val) {
		typeMatches[value.PhysicalTypeObject] += count
		return
	}

	// 默认为字符串（兜底类型，如果以上所有类型都不匹配）
	typeMatches[value.PhysicalTypeString] += count
}

// isNull 检测值是否为空
//
// 判断一个字符串值是否代表null值。
// 以下情况被视为null:
// - 空字符串 ("")
// - 字符串 "null" (不区分大小写)
//
// 参数:
//   - val: 要检查的字符串值
//
// 返回值:
//   - bool: 如果值表示null则返回true，否则返回false
func (r *PhysicalTypeInferenceRule) isNull(val string) bool {
	return val == "" || val == "null" || val == "NULL"
}

// isBoolean 检测值是否为布尔值
//
// 判断一个字符串值是否代表布尔值。
// 仅"true"和"false"（不区分大小写）被视为布尔值。
//
// 参数:
//   - val: 要检查的字符串值
//
// 返回值:
//   - bool: 如果值表示布尔值则返回true，否则返回false
//
// 示例:
//
//	"true" -> 布尔值
//	"TRUE" -> 布尔值
//	"false" -> 布尔值
//	"False" -> 不是布尔值（当前实现不支持首字母大写）
func (r *PhysicalTypeInferenceRule) isBoolean(val string) bool {
	return val == "true" || val == "false" || val == "TRUE" || val == "FALSE"
}

// isInteger 检测值是否为整数
//
// 判断一个字符串值是否代表整数。
// 整数定义为可选的符号(+/-)后跟一个或多个数字(0-9)。
//
// 参数:
//   - val: 要检查的字符串值
//
// 返回值:
//   - bool: 如果值表示整数则返回true，否则返回false
//
// 示例:
//
//	"123" -> 整数
//	"+123" -> 整数
//	"-123" -> 整数
//	"0" -> 整数
//	"123.0" -> 不是整数（有小数点）
//	"a123" -> 不是整数（含非数字字符）
//	"" -> 不是整数（空字符串）
func (r *PhysicalTypeInferenceRule) isInteger(val string) bool {
	if len(val) == 0 {
		// 空字符串不是整数
		return false
	}

	for i, c := range val {
		if i == 0 && (c == '+' || c == '-') {
			// 第一个字符是符号位（+或-）是允许的
			continue
		}
		if c < '0' || c > '9' {
			// 只允许数字字符0-9
			return false
		}
	}
	return true
}

// isFloat 检测值是否为浮点数
//
// 判断一个字符串值是否代表浮点数。
// 浮点数定义为可选的符号(+/-)后跟数字序列，中间包含一个小数点。
//
// 参数:
//   - val: 要检查的字符串值
//
// 返回值:
//   - bool: 如果值表示浮点数则返回true，否则返回false
//
// 示例:
//
//	"123.45" -> 浮点数
//	"+123.45" -> 浮点数
//	"-123.45" -> 浮点数
//	"123" -> 不是浮点数（没有小数点）
//	"123." -> 浮点数（虽然小数部分为空，但有小数点）
//	".45" -> 浮点数（整数部分为空，但有小数点）
//	"123.45.67" -> 不是浮点数（多个小数点）
//	"a123.45" -> 不是浮点数（含非法字符）
//	"" -> 不是浮点数（空字符串）
func (r *PhysicalTypeInferenceRule) isFloat(val string) bool {
	if len(val) == 0 {
		// 空字符串不是浮点数
		return false
	}

	hasDot := false
	for i, c := range val {
		if i == 0 && (c == '+' || c == '-') {
			// 第一个字符是符号位（+或-）是允许的
			continue
		}
		if c == '.' {
			if hasDot {
				// 已经有一个小数点了，多个小数点是不允许的
				return false
			}
			hasDot = true
			continue
		}
		if c < '0' || c > '9' {
			// 只允许数字字符0-9和前面处理过的小数点
			return false
		}
	}

	// 必须有小数点才认为是浮点数（这是区分整数和浮点数的关键）
	return hasDot
}

// isArray 检测值是否为数组
//
// 判断一个字符串值是否代表JSON数组。
// 目前仅做简单的语法检测：以'['开头，以']'结尾的字符串。
// 注意：这是一个简化的检测，不验证内部结构的有效性。
//
// 参数:
//   - val: 要检查的字符串值
//
// 返回值:
//   - bool: 如果值看起来像数组则返回true，否则返回false
//
// 示例:
//
//	"[]" -> 数组
//	"[1,2,3]" -> 数组
//	"[\"a\",\"b\"]" -> 数组
//	"[" -> 不是数组（没有结束括号）
//	"a[]" -> 不是数组（有前缀字符）
//	"" -> 不是数组（空字符串）
//
// TODO: 考虑实现更严格的JSON数组验证
func (r *PhysicalTypeInferenceRule) isArray(val string) bool {
	return len(val) >= 2 && val[0] == '[' && val[len(val)-1] == ']'
}

// isObject 检测值是否为对象
//
// 判断一个字符串值是否代表JSON对象。
// 目前仅做简单的语法检测：以'{'开头，以'}'结尾的字符串。
// 注意：这是一个简化的检测，不验证内部结构的有效性。
//
// 参数:
//   - val: 要检查的字符串值
//
// 返回值:
//   - bool: 如果值看起来像对象则返回true，否则返回false
//
// 示例:
//
//	"{}" -> 对象
//	"{\"a\":1}" -> 对象
//	"{\"a\":{\"b\":2}}" -> 对象
//	"{" -> 不是对象（没有结束括号）
//	"a{}" -> 不是对象（有前缀字符）
//	"" -> 不是对象（空字符串）
//
// TODO: 考虑实现更严格的JSON对象验证
func (r *PhysicalTypeInferenceRule) isObject(val string) bool {
	return len(val) >= 2 && val[0] == '{' && val[len(val)-1] == '}'
}

// findDominantType 找出匹配次数最多的类型
//
// 分析类型匹配统计，确定占主导地位的数据类型。
// 如果有多个类型具有相同的匹配次数，将选择在typeMatches映射中先出现的类型。
// 如果没有匹配或无法确定，默认返回字符串类型（作为最通用的类型）。
//
// 参数:
//   - typeMatches: 各类型的匹配计数映射
//   - totalSamples: 处理的总样本数
//
// 返回值:
//   - value.PhysicalType: 确定的主导类型
//
// 特殊情况:
//   - 如果totalSamples为0（没有样本），返回string类型
//   - 如果dominantType为空（可能因为映射为空），返回string类型
func (r *PhysicalTypeInferenceRule) findDominantType(typeMatches map[value.PhysicalType]int, totalSamples int) value.PhysicalType {
	var dominantType value.PhysicalType
	maxMatches := 0

	// 遍历所有类型匹配计数，找出匹配次数最多的类型
	for pType, matches := range typeMatches {
		if matches > maxMatches {
			maxMatches = matches
			dominantType = pType
		}
		// 注意：如果有相同匹配次数的类型，将保留先找到的类型
	}

	// 安全防护：如果没有样本或无法确定类型，默认为字符串类型
	// 字符串类型是最通用的类型，可以表示任何值，所以是最安全的默认选择
	if totalSamples == 0 || dominantType == "" {
		dominantType = value.PhysicalTypeString
	}

	return dominantType
}
