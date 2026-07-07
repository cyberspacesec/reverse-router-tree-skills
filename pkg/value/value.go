package value

import "sync"

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
	mu       sync.RWMutex
	valueMap map[string]int
}

// NewValueMetric 创建一个新的值度量对象
func NewValueMetric() *ValueMetric {
	return &ValueMetric{
		valueMap: make(map[string]int),
	}
}

// AddValue 添加一个值到度量中
func (v *ValueMetric) AddValue(val string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.valueMap[val]++
}

// IsEmpty 检查度量是否为空
func (v *ValueMetric) IsEmpty() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.valueMap) == 0
}

// GetValueCount 获取特定值的计数
func (v *ValueMetric) GetValueCount(val string) int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.valueMap[val]
}

// GetAllValues 获取所有值及其计数的映射
//
// 注意：本方法返回 valueMap 的完整拷贝，适用于需要长期持有或随机访问的场景。
// 若只需遍历一次（如类型推断），优先用 ForEachValue——它持读锁遍历，零拷贝，
// 在值数量大时（合并后的路径变量节点可能累积上千个值）可显著减少分配。
func (v *ValueMetric) GetAllValues() map[string]int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	result := make(map[string]int, len(v.valueMap))
	for k, val := range v.valueMap {
		result[k] = val
	}
	return result
}

// ForEachValue 持读锁遍历所有值及其计数，零拷贝。
// 适用于只需单次遍历的场景（如类型推断里的模式匹配）。
// 在 fn 返回 false 时提前终止遍历。fn 内不可调用同一 ValueMetric 的写方法（会死锁）。
func (v *ValueMetric) ForEachValue(fn func(val string, count int) bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	for k, c := range v.valueMap {
		if !fn(k, c) {
			return
		}
	}
}

// GetUniqueValueCount 获取不同值的数量
func (v *ValueMetric) GetUniqueValueCount() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.valueMap)
}

// GetTotalCount 获取所有值的总计数
func (v *ValueMetric) GetTotalCount() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	total := 0
	for _, count := range v.valueMap {
		total += count
	}
	return total
}
