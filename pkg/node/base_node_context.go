package node

import (
	"sync"
)

// ------------------------------------------------------------------------------------

// BaseNodeContext 节点上下文实现的基础逻辑
type BaseNodeContext struct {
	storageMap map[string]any
	mutex      sync.RWMutex // 添加读写锁以保证并发安全
}

var _ NodeContext = &BaseNodeContext{}

// NewBaseNodeContext 创建一个新的基础节点上下文
func NewBaseNodeContext() *BaseNodeContext {
	return &BaseNodeContext{
		storageMap: make(map[string]any),
	}
}

// SetKey 设置节点上下文的键值对
func (x *BaseNodeContext) SetKey(key string, value any) (error, bool) {
	x.mutex.Lock()
	defer x.mutex.Unlock()

	x.storageMap[key] = value
	return nil, true
}

// GetKey 获取节点上下文的键值对
func (x *BaseNodeContext) GetKey(key string) (any, bool) {
	x.mutex.RLock()
	defer x.mutex.RUnlock()

	value, exists := x.storageMap[key]
	return value, exists
}

// DeleteKey 删除节点上下文的键值对
func (x *BaseNodeContext) DeleteKey(key string) (error, bool) {
	x.mutex.Lock()
	defer x.mutex.Unlock()

	delete(x.storageMap, key)
	return nil, true
}

// HasKey 检查节点上下文是否存在某个键
func (x *BaseNodeContext) HasKey(key string) (bool, error) {
	x.mutex.RLock()
	defer x.mutex.RUnlock()

	_, exists := x.storageMap[key]
	return exists, nil
}

// Clear 清空节点上下文
func (x *BaseNodeContext) Clear() error {
	x.mutex.Lock()
	defer x.mutex.Unlock()

	x.storageMap = make(map[string]any)
	return nil
}

// GetAllKeys 获取所有键
func (x *BaseNodeContext) GetAllKeys() []string {
	x.mutex.RLock()
	defer x.mutex.RUnlock()

	keys := make([]string, 0, len(x.storageMap))
	for key := range x.storageMap {
		keys = append(keys, key)
	}
	return keys
}

// GetAllValues 获取所有值
func (x *BaseNodeContext) GetAllValues() []any {
	x.mutex.RLock()
	defer x.mutex.RUnlock()

	values := make([]any, 0, len(x.storageMap))
	for _, value := range x.storageMap {
		values = append(values, value)
	}
	return values
}

// GetAllItems 获取所有键值对
func (x *BaseNodeContext) GetAllItems() map[string]any {
	x.mutex.RLock()
	defer x.mutex.RUnlock()

	// 创建一个新的map返回，避免对原始map的直接访问
	result := make(map[string]any, len(x.storageMap))
	for k, v := range x.storageMap {
		result[k] = v
	}
	return result
}

// Size 获取节点上下文的大小
func (x *BaseNodeContext) Size() int {
	x.mutex.RLock()
	defer x.mutex.RUnlock()

	return len(x.storageMap)
}

// ------------------------------------------------------------------------------------
