package value

// 用于表示识别系统中的一个数值
type Value struct {

	// 实际的值的采样
	ValueMetric *ValueMetric

	// 物理类型
	PhysicalType PhysicalType

	// 逻辑类型
	LogicalType LogicalType
}

// 对实际的值的采样
type ValueMetric struct {
	valueMap map[string]int
}

// NewValueMetric 创建一个新的值度量对象
func NewValueMetric() *ValueMetric {
	return &ValueMetric{
		valueMap: make(map[string]int),
	}
}

// AddValue 添加一个值到度量中
func (v *ValueMetric) AddValue(value string) {
	v.valueMap[value]++
}

// IsEmpty 检查度量是否为空
func (v *ValueMetric) IsEmpty() bool {
	return len(v.valueMap) == 0
}

// GetValueCount 获取特定值的计数
func (v *ValueMetric) GetValueCount(value string) int {
	return v.valueMap[value]
}

// GetAllValues 获取所有值及其计数的映射
func (v *ValueMetric) GetAllValues() map[string]int {
	result := make(map[string]int, len(v.valueMap))
	for k, v := range v.valueMap {
		result[k] = v
	}
	return result
}
