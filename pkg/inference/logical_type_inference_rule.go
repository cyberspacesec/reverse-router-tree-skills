package inference

import (
	"net"
	"regexp"
	"strings"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

// LogicalTypeInferenceRule 实现了逻辑类型推断规则
// 在物理类型推断的基础上，进一步推断值的语义类型。
// 例如：物理类型是 "string"，但逻辑类型可能是 "date"、"email"、"uuid" 等。
//
// 推断优先级（从高到低）：
// 1. UUID - UUID格式有非常明确的结构
// 2. IP地址 - IPv4格式明确
// 3. 邮箱 - 包含@和域名
// 4. 日期/时间/日期时间 - 多种日期格式
// 5. URL - 包含协议头
// 6. JSON - 以{或[开头
// 7. XML - 以<开头
// 8. 正则表达式 - 特殊字符组合
// 9. 枚举值 - 有限集合的字符串
// 10. 货币/百分比/精确小数 - 数值扩展类型
type LogicalTypeInferenceRule struct {
	// EnumThreshold 枚举值判定阈值
	// 当唯一值数量占总采样数量的比例不超过此值时，判定为枚举类型
	EnumThreshold float64

	// patterns 预编译的正则表达式
	patterns []*logicalPattern
}

// logicalPattern 逻辑类型模式
type logicalPattern struct {
	logicalType value.LogicalType
	regex       *regexp.Regexp
}

// 确保 LogicalTypeInferenceRule 实现了 TypeInferenceRule 接口
var _ TypeInferenceRule = (*LogicalTypeInferenceRule)(nil)

// NewLogicalTypeInferenceRule 创建一个新的逻辑类型推断规则实例
func NewLogicalTypeInferenceRule() *LogicalTypeInferenceRule {
	rule := &LogicalTypeInferenceRule{
		EnumThreshold: 0.3, // 唯一值占比≤30%时可能是枚举
	}

	rule.initPatterns()
	return rule
}

// initPatterns 初始化模式匹配规则
// 推断优先级（从高到低）：
// 1. UUID - UUID格式有非常明确的结构
// 2. IP地址 - IPv4格式明确
// 3. 手机号 - 中国手机号有明确前缀和长度
// 4. 身份证号 - 18位/15位有校验位结构
// 5. 银行卡号 - 16-19位纯数字，特定BIN前缀
// 6. 车牌号 - 中文+字母+数字组合
// 7. 邮箱 - 包含@和域名
// 8. 日期/时间/日期时间 - 多种日期格式
// 9. URL - 包含协议头
// 10. JSON - 以{或[开头
// 11. XML - 以<开头
//
// 注意：邮政编码(postalcode)不在此列表中，因为6位纯数字无法与普通数字ID、
// 验证码等可靠区分，纯正则识别误判率太高。如需识别邮政编码，应结合参数名语义。
func (r *LogicalTypeInferenceRule) initPatterns() {
	r.patterns = []*logicalPattern{
		// UUID 格式
		{value.LogicalTypeUUID, regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)},

		// IP 地址（IPv4 正则粗筛；IPv6 的 :: 压缩/嵌入 IPv4 等形式由 isIPAddress
		// 用 net.ParseIP 兜底，正则难以穷尽且易误判）
		{value.LogicalTypeIPAddress, regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)},

		// 中国电话号码（手机号 + 座机号）
		// 手机号：11位，1开头第二位3-9，支持+86/0086国际前缀
		// 座机号：区号(3-4位，0开头) + 号码(7-8位)，归一化后为10-12位0开头数字
		//   - 3位区号：010/021/022/023/...+8位号码 → 11位
		//   - 4位区号：0755/0991/...+7-8位号码 → 11-12位
		// 归一化（stripPhoneSeparators）会先去除空格/横线/括号，故此处只匹配纯数字形式
		{value.LogicalTypePhoneNumber, regexp.MustCompile(
			`^(?:\+?86|0086)?1[3-9]\d{9}$` + // 手机号
				`|^0\d{2,3}[1-9]\d{6,7}$`)}, // 座机号：0+2-3位区号+7-8位号码

		// 中国身份证号
		// 18位：6位地区码 + 8位出生日期 + 3位顺序码 + 1位校验位（数字或X）
		// 15位：6位地区码 + 6位出生日期 + 3位顺序码（旧版）
		{value.LogicalTypeIDCard, regexp.MustCompile(`^[1-9]\d{5}(?:19|20)\d{2}(?:0[1-9]|1[0-2])(?:0[1-9]|[12]\d|3[01])\d{3}[\dXx]$|^[1-9]\d{5}\d{2}(?:0[1-9]|1[0-2])(?:0[1-9]|[12]\d|3[01])\d{3}$`)},

		// 银行卡号
		// 16-19位纯数字，首位非0
		// 注意：银行卡号必须与身份证号区分，身份证号有明确的日期结构
		{value.LogicalTypeBankCard, regexp.MustCompile(`^[3-6]\d{15,18}$`)},

		// 中国车牌号
		// 格式：省份汉字+字母+5位字母数字（新能源6位）
		{value.LogicalTypePlateNumber, regexp.MustCompile(`^[\p{Han}][A-Z][A-Z0-9]{5,6}$`)},

		// 邮箱
		{value.LogicalTypeEmail, regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)},

		// 日期时间 ISO 8601
		{value.LogicalTypeDateTime, regexp.MustCompile(`^\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}`)},

		// 日期
		{value.LogicalTypeDate, regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)},

		// 时间
		{value.LogicalTypeTime, regexp.MustCompile(`^\d{2}:\d{2}:\d{2}`)},

		// URL
		{value.LogicalTypeURL, regexp.MustCompile(`^https?://`)},

		// JSON
		{value.LogicalTypeJSON, regexp.MustCompile(`^\s*\{.*\}\s*$|^\s*\[.*\]\s*$`)},

		// XML
		{value.LogicalTypeXML, regexp.MustCompile(`^\s*<`)},
	}
}

// Infer 根据节点上下文推断逻辑类型
func (r *LogicalTypeInferenceRule) Infer(n node.Node[node.NodeContext]) (value.Type, error) {
	metric := r.getMetricFromNode(n)
	if metric == nil || metric.IsEmpty() {
		return value.Type(value.LogicalTypeString), nil
	}

	return r.inferFromMetric(metric)
}

// inferFromMetric 根据值采样推断逻辑类型
func (r *LogicalTypeInferenceRule) inferFromMetric(metric *value.ValueMetric) (value.Type, error) {
	totalCount := metric.GetTotalCount()
	uniqueCount := metric.GetUniqueValueCount()

	if totalCount == 0 {
		return value.Type(value.LogicalTypeString), nil
	}

	// 各匹配函数直接用 metric.ForEachValue 遍历，避免 GetAllValues 拷贝整个 map
	// （合并后的路径变量节点可能累积上千个值，拷贝开销显著）。

	// 先尝试结构化模式匹配（UUID、IP、邮箱、日期等）
	if logicalType, found := r.matchStructuredPatterns(metric, totalCount); found {
		return value.Type(logicalType), nil
	}

	// 尝试数值扩展类型匹配（货币、百分比、精确小数）
	if logicalType, found := r.matchNumericPatterns(metric, totalCount); found {
		return value.Type(logicalType), nil
	}

	// 检查是否为枚举类型
	// 枚举值特征：唯一值数量少且占比低
	if r.isEnumLike(uniqueCount, totalCount, metric) {
		return value.Type(value.LogicalTypeEnum), nil
	}

	// 无法确定更具体的逻辑类型，返回通用的 string
	return value.Type(value.LogicalTypeString), nil
}

// getMetricFromNode 从节点中获取 ValueMetric
func (r *LogicalTypeInferenceRule) getMetricFromNode(n node.Node[node.NodeContext]) *value.ValueMetric {
	// 首先尝试从 RequestPathVariableNode 获取
	if pathVarNode, ok := n.(*node.RequestPathVariableNode); ok {
		metric := pathVarNode.GetValueMetric()
		if metric != nil && !metric.IsEmpty() {
			return metric
		}
	}

	// 从 RequestParamNode 获取（参数值类型推断）
	if paramNode, ok := n.(*node.RequestParamNode); ok {
		metric := paramNode.GetValueMetric()
		if metric != nil && !metric.IsEmpty() {
			return metric
		}
	}

	// 从上下文中获取
	context := n.GetContext()
	if context == nil {
		return nil
	}

	if vm, exists := context.GetKey("__value_metric__"); exists {
		if metric, ok := vm.(*value.ValueMetric); ok && metric != nil && !metric.IsEmpty() {
			return metric
		}
	}

	return nil
}

// matchStructuredPatterns 匹配结构化模式（UUID、IP、邮箱、日期等）
func (r *LogicalTypeInferenceRule) matchStructuredPatterns(metric *value.ValueMetric, totalCount int) (value.LogicalType, bool) {
	// 统计每种模式匹配的样本数
	typeMatches := make(map[value.LogicalType]int)

	metric.ForEachValue(func(val string, count int) bool {
		for _, pattern := range r.patterns {
			// 对需要归一化的模式，先做格式归一化再匹配
			// 例如手机号 138-1234-5678 / 138 1234 5678 归一化后匹配
			normalized := r.normalizeForPattern(val, pattern.logicalType)
			if pattern.regex.MatchString(normalized) {
				typeMatches[pattern.logicalType] += count
				break // 一个值只匹配最优先的模式
			}
			// IP 地址用 net.ParseIP 兜底：IPv6 的 :: 压缩、嵌入 IPv4 等形式
			// 正则难以穷尽且易误判，标准库权威解析更可靠。
			if pattern.logicalType == value.LogicalTypeIPAddress && isIPAddress(normalized) {
				typeMatches[pattern.logicalType] += count
				break
			}
		}
		return true
	})

	// 找出匹配率最高的模式
	for _, pattern := range r.patterns {
		if matches, ok := typeMatches[pattern.logicalType]; ok {
			ratio := float64(matches) / float64(totalCount)
			if ratio >= 0.6 { // 60%以上的值匹配该模式
				return pattern.logicalType, true
			}
		}
	}

	return "", false
}

// normalizeForPattern 针对特定逻辑类型的值归一化
// normalizeForPattern 针对特定逻辑类型的值归一化
// 某些格式在现实中常以分隔符形式出现（如手机号 138-1234-5678），
// 归一化后能统一识别，提升异常/不规范数据的兼容性。
func (r *LogicalTypeInferenceRule) normalizeForPattern(val string, logicalType value.LogicalType) string {
	switch logicalType {
	case value.LogicalTypePhoneNumber:
		// 手机号：去除空格、横线、括号等分隔符
		// 138-1234-5678 → 13812345678
		// 138 1234 5678 → 13812345678
		// (+86)138-1234-5678 → +8613812345678
		return stripPhoneSeparators(val)
	default:
		return val
	}
}

// isIPAddress 用 net.ParseIP 判定字符串是否为合法 IPv4 或 IPv6 地址。
// IPv6 的 :: 压缩、嵌入 IPv4（::ffff:1.2.3.4）等形式正则难以穷尽且易误判，
// 标准库权威解析更可靠。net.ParseIP 对形如 "123" 的纯数字返回非 nil（视为
// IPv4 等价），故仅在本函数被调用时（即值已含 . 或 : 的 IP 候选场景）兜底。
func isIPAddress(val string) bool {
	// 快速排除：IP 地址必含 . 或 :，避免对纯数字/普通字符串无谓调用
	if !strings.ContainsAny(val, ".:") {
		return false
	}
	return net.ParseIP(val) != nil
}

// stripPhoneSeparators 去除手机号中常见的分隔符
// 保留数字和 + 号（国际前缀），去除空格、横线、括号、点等
func stripPhoneSeparators(val string) string {
	// 快路径：逐字节扫描，若值只含 ASCII 数字和 +（无任何分隔符），
	// 直接返回原串，零分配。手机号场景的值绝大多数为纯数字，走快路径。
	// 用字节循环而非 rune 循环：数字/+ 是 ASCII 单字节，避免 rune 解码开销。
	for i := 0; i < len(val); i++ {
		c := val[i]
		if (c >= '0' && c <= '9') || c == '+' {
			continue
		}
		// 遇到非数字非 + 字节（分隔符或多字节 rune 首字节）→ 走慢路径
		var b strings.Builder
		b.Grow(len(val))
		// 先把已确认是数字/+ 的前缀写入
		b.WriteString(val[:i])
		// 从当前位置继续按 rune 过滤（多字节 rune 场景用 rune 更安全）
		for _, c := range val[i:] {
			if c >= '0' && c <= '9' || c == '+' {
				b.WriteRune(c)
			}
		}
		return b.String()
	}
	// 全程无分隔符，直接返回原串
	return val
}

// matchNumericPatterns 匹配数值扩展类型（货币、百分比、精确小数）
func (r *LogicalTypeInferenceRule) matchNumericPatterns(metric *value.ValueMetric, totalCount int) (value.LogicalType, bool) {
	percentageCount := 0
	currencyCount := 0
	decimalCount := 0

	metric.ForEachValue(func(val string, count int) bool {
		// 百分比：以 % 结尾或值在 0-100/0.0-100.0 范围内
		if strings.HasSuffix(val, "%") {
			percentageCount += count
			return true
		}

		// 货币：带有货币符号前缀
		if strings.HasPrefix(val, "$") || strings.HasPrefix(val, "€") ||
			strings.HasPrefix(val, "£") || strings.HasPrefix(val, "¥") ||
			strings.HasPrefix(val, "￥") {
			currencyCount += count
			return true
		}

		// 精确小数：小数位数为2位的浮点数（可能是金额）
		if isDecimalLike(val) {
			decimalCount += count
		}
		return true
	})

	total := float64(totalCount)

	if float64(percentageCount)/total >= 0.6 {
		return value.LogicalTypePercentage, true
	}

	if float64(currencyCount)/total >= 0.6 {
		return value.LogicalTypeCurrency, true
	}

	if float64(decimalCount)/total >= 0.6 {
		return value.LogicalTypeDecimal, true
	}

	return "", false
}

// isEnumLike 判断值集合是否像枚举类型
func (r *LogicalTypeInferenceRule) isEnumLike(uniqueCount, totalCount int, metric *value.ValueMetric) bool {
	if uniqueCount < 2 || totalCount < 3 {
		return false
	}

	// 唯一值占比低 → 可能是枚举
	ratio := float64(uniqueCount) / float64(totalCount)
	if ratio > r.EnumThreshold {
		return false
	}

	// 枚举值的长度不应该太长（通常是短字符串），且不应包含特殊结构（UUID、邮箱等）。
	// 用一次 ForEachValue 同时收集：是否有超长值 + 各模式命中数。
	longValues := false
	patternMatches := make(map[value.LogicalType]int, len(r.patterns))
	metric.ForEachValue(func(val string, count int) bool {
		if len(val) > 50 {
			longValues = true
			return false // 发现超长值，提前终止（原逻辑此时即判定非枚举）
		}
		for _, pattern := range r.patterns {
			if pattern.regex.MatchString(r.normalizeForPattern(val, pattern.logicalType)) {
				patternMatches[pattern.logicalType]++
			}
		}
		return true
	})

	if longValues {
		return false
	}

	// 枚举值不应该包含特殊结构（UUID、邮箱等）
	for _, pattern := range r.patterns {
		matchCount := patternMatches[pattern.logicalType]
		if matchCount > 0 && float64(matchCount)/float64(uniqueCount) > 0.5 {
			return false
		}
	}

	return true
}

// isDecimalLike 判断值是否像精确小数（2位小数，可能是金额）
func isDecimalLike(val string) bool {
	dotIndex := strings.LastIndex(val, ".")
	if dotIndex == -1 {
		return false
	}

	// 小数部分恰好2位
	fractional := val[dotIndex+1:]
	if len(fractional) != 2 {
		return false
	}

	for _, c := range fractional {
		if c < '0' || c > '9' {
			return false
		}
	}

	// 整数部分应该是数字
	integral := val[:dotIndex]
	if len(integral) == 0 {
		return false
	}

	for i, c := range integral {
		if i == 0 && (c == '+' || c == '-') {
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}

	return true
}
