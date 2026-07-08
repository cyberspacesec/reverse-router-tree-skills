package node

import (
	"reflect"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"
)

// 测试创建新的BaseNodeContext
func TestNewBaseNodeContext(t *testing.T) {
	context := NewBaseNodeContext()

	if context == nil {
		t.Fatal("NewBaseNodeContext() 应该返回一个非nil对象")
	}

	if context.storageMap == nil {
		t.Error("存储映射不应该为nil")
	}

	if len(context.storageMap) != 0 {
		t.Errorf("初始存储映射应该为空，但包含了 %d 个元素", len(context.storageMap))
	}
}

// 测试设置和获取键值对
func TestBaseNodeContext_SetAndGetKey(t *testing.T) {
	context := NewBaseNodeContext()

	// 测试设置和获取单个键值对
	err, ok := context.SetKey("key1", "value1")
	if err != nil || !ok {
		t.Errorf("SetKey() 返回了错误: %v, ok: %v", err, ok)
	}

	value, exists := context.GetKey("key1")
	if !exists {
		t.Error("GetKey() 表明键不存在，但它应该存在")
	}
	if value != "value1" {
		t.Errorf("GetKey() 返回了 %v, 期望 %v", value, "value1")
	}

	// 测试不存在的键
	value, exists = context.GetKey("nonExistentKey")
	if exists {
		t.Error("GetKey() 表明不存在的键存在")
	}
	if value != nil {
		t.Errorf("GetKey() 对不存在的键返回了非nil值: %v", value)
	}

	// 测试各种类型的值
	testCases := []struct {
		key   string
		value interface{}
	}{
		{"intKey", 123},
		{"floatKey", 123.456},
		{"boolKey", true},
		{"sliceKey", []string{"a", "b", "c"}},
		{"mapKey", map[string]int{"a": 1, "b": 2}},
		{"nilKey", nil},
	}

	for _, tc := range testCases {
		err, ok := context.SetKey(tc.key, tc.value)
		if err != nil || !ok {
			t.Errorf("SetKey(%s, %v) 返回了错误: %v, ok: %v", tc.key, tc.value, err, ok)
		}

		value, exists := context.GetKey(tc.key)
		if !exists {
			t.Errorf("GetKey(%s) 表明键不存在，但它应该存在", tc.key)
		}

		if !reflect.DeepEqual(value, tc.value) {
			t.Errorf("GetKey(%s) 返回了 %v, 期望 %v", tc.key, value, tc.value)
		}
	}

	// 测试覆盖现有键
	err, ok = context.SetKey("key1", "newValue")
	if err != nil || !ok {
		t.Errorf("SetKey() 覆盖时返回了错误: %v, ok: %v", err, ok)
	}

	value, exists = context.GetKey("key1")
	if !exists {
		t.Error("GetKey() 表明键不存在，但它应该存在")
	}
	if value != "newValue" {
		t.Errorf("GetKey() 返回了 %v, 期望 %v", value, "newValue")
	}
}

// 测试删除键
func TestBaseNodeContext_DeleteKey(t *testing.T) {
	context := NewBaseNodeContext()

	// 设置一些初始键值对
	context.SetKey("key1", "value1")
	context.SetKey("key2", "value2")
	context.SetKey("key3", "value3")

	// 测试删除存在的键
	err, ok := context.DeleteKey("key2")
	if err != nil || !ok {
		t.Errorf("DeleteKey() 返回了错误: %v, ok: %v", err, ok)
	}

	// 验证键已被删除
	_, exists := context.GetKey("key2")
	if exists {
		t.Error("DeleteKey() 后键仍然存在")
	}

	// 验证其他键没有受到影响
	value, exists := context.GetKey("key1")
	if !exists || value != "value1" {
		t.Errorf("DeleteKey() 影响了其他键，key1: %v, exists: %v", value, exists)
	}

	value, exists = context.GetKey("key3")
	if !exists || value != "value3" {
		t.Errorf("DeleteKey() 影响了其他键，key3: %v, exists: %v", value, exists)
	}

	// 测试删除不存在的键
	err, ok = context.DeleteKey("nonExistentKey")
	if err != nil || !ok {
		t.Errorf("DeleteKey() 对不存在的键返回了错误: %v, ok: %v", err, ok)
	}
}

// 测试检查键存在性
func TestBaseNodeContext_HasKey(t *testing.T) {
	context := NewBaseNodeContext()

	// 设置一些初始键值对
	context.SetKey("key1", "value1")
	context.SetKey("emptyValue", "")
	context.SetKey("nilValue", nil)

	// 测试存在的键
	exists, err := context.HasKey("key1")
	if err != nil {
		t.Errorf("HasKey() 返回了错误: %v", err)
	}
	if !exists {
		t.Error("HasKey() 返回了false，但键应该存在")
	}

	// 测试值为空字符串的键
	exists, err = context.HasKey("emptyValue")
	if err != nil || !exists {
		t.Errorf("HasKey() 对值为空字符串的键返回了错误结果: %v, exists: %v", err, exists)
	}

	// 测试值为nil的键
	exists, err = context.HasKey("nilValue")
	if err != nil || !exists {
		t.Errorf("HasKey() 对值为nil的键返回了错误结果: %v, exists: %v", err, exists)
	}

	// 测试不存在的键
	exists, err = context.HasKey("nonExistentKey")
	if err != nil {
		t.Errorf("HasKey() 返回了错误: %v", err)
	}
	if exists {
		t.Error("HasKey() 返回了true，但键不应该存在")
	}
}

// 测试清空上下文
func TestBaseNodeContext_Clear(t *testing.T) {
	context := NewBaseNodeContext()

	// 设置一些初始键值对
	context.SetKey("key1", "value1")
	context.SetKey("key2", "value2")

	// 测试初始状态
	if size := context.Size(); size != 2 {
		t.Errorf("初始大小应该是2，得到 %d", size)
	}

	// 测试Clear
	err := context.Clear()
	if err != nil {
		t.Errorf("Clear() 返回了错误: %v", err)
	}

	// 验证上下文已清空
	if size := context.Size(); size != 0 {
		t.Errorf("Clear() 后大小应该是0，得到 %d", size)
	}

	keys := context.GetAllKeys()
	if len(keys) != 0 {
		t.Errorf("Clear() 后应该没有键，但有 %d 个", len(keys))
	}

	// 测试清空空的上下文
	err = context.Clear()
	if err != nil {
		t.Errorf("Clear() 空上下文返回了错误: %v", err)
	}
}

// 测试获取所有键
func TestBaseNodeContext_GetAllKeys(t *testing.T) {
	context := NewBaseNodeContext()

	// 测试空上下文
	keys := context.GetAllKeys()
	if len(keys) != 0 {
		t.Errorf("空上下文应该返回空键列表，但有 %d 个键", len(keys))
	}

	// 设置一些键值对
	expectedKeys := []string{"a", "b", "c"}
	for _, key := range expectedKeys {
		context.SetKey(key, key+"_value")
	}

	// 获取所有键并排序以便比较
	keys = context.GetAllKeys()
	sort.Strings(keys)

	// 验证键列表
	if len(keys) != len(expectedKeys) {
		t.Errorf("GetAllKeys() 返回了 %d 个键, 期望 %d 个", len(keys), len(expectedKeys))
	}

	// 对期望的键也进行排序，然后比较
	sort.Strings(expectedKeys)
	for i, key := range expectedKeys {
		if i >= len(keys) || keys[i] != key {
			t.Errorf("GetAllKeys()[%d] = %s, 期望 %s", i, keys[i], key)
		}
	}
}

// 测试获取所有值
func TestBaseNodeContext_GetAllValues(t *testing.T) {
	context := NewBaseNodeContext()

	// 测试空上下文
	values := context.GetAllValues()
	if len(values) != 0 {
		t.Errorf("空上下文应该返回空值列表，但有 %d 个值", len(values))
	}

	// 设置一些键值对
	expectedValues := []interface{}{"value_a", "value_b", "value_c"}
	keys := []string{"a", "b", "c"}
	for i, key := range keys {
		context.SetKey(key, expectedValues[i])
	}

	// 获取所有值
	values = context.GetAllValues()

	// 验证值列表长度
	if len(values) != len(expectedValues) {
		t.Errorf("GetAllValues() 返回了 %d 个值, 期望 %d 个", len(values), len(expectedValues))
	}

	// 由于map的迭代顺序不确定，我们需要检查每个期望值是否都在返回的值列表中
	for _, expectedValue := range expectedValues {
		found := false
		for _, value := range values {
			if reflect.DeepEqual(value, expectedValue) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("期望的值 %v 不在GetAllValues()返回的列表中", expectedValue)
		}
	}
}

// 测试获取所有键值对
func TestBaseNodeContext_GetAllItems(t *testing.T) {
	context := NewBaseNodeContext()

	// 测试空上下文
	items := context.GetAllItems()
	if len(items) != 0 {
		t.Errorf("空上下文应该返回空映射，但有 %d 个项", len(items))
	}

	// 设置一些键值对
	expectedItems := map[string]interface{}{
		"a": "value_a",
		"b": 123,
		"c": true,
	}
	for key, value := range expectedItems {
		context.SetKey(key, value)
	}

	// 获取所有项
	items = context.GetAllItems()

	// 验证项数量
	if len(items) != len(expectedItems) {
		t.Errorf("GetAllItems() 返回了 %d 个项, 期望 %d 个", len(items), len(expectedItems))
	}

	// 验证每个项
	for key, expectedValue := range expectedItems {
		value, exists := items[key]
		if !exists {
			t.Errorf("键 %s 不在GetAllItems()返回的映射中", key)
		} else if !reflect.DeepEqual(value, expectedValue) {
			t.Errorf("GetAllItems()[%s] = %v, 期望 %v", key, value, expectedValue)
		}
	}
}

// 测试获取上下文大小
func TestBaseNodeContext_Size(t *testing.T) {
	context := NewBaseNodeContext()

	// 测试空上下文
	if size := context.Size(); size != 0 {
		t.Errorf("空上下文的Size()应该返回0，得到 %d", size)
	}

	// 添加一个键值对
	context.SetKey("key1", "value1")
	if size := context.Size(); size != 1 {
		t.Errorf("添加一个键后Size()应该返回1，得到 %d", size)
	}

	// 添加更多键值对
	context.SetKey("key2", "value2")
	context.SetKey("key3", "value3")
	if size := context.Size(); size != 3 {
		t.Errorf("添加三个键后Size()应该返回3，得到 %d", size)
	}

	// 删除一个键
	context.DeleteKey("key2")
	if size := context.Size(); size != 2 {
		t.Errorf("删除一个键后Size()应该返回2，得到 %d", size)
	}

	// 清空上下文
	context.Clear()
	if size := context.Size(); size != 0 {
		t.Errorf("清空后Size()应该返回0，得到 %d", size)
	}
}

// 测试综合场景
func TestBaseNodeContext_ComplexScenario(t *testing.T) {
	context := NewBaseNodeContext()

	// 添加多个键值对
	context.SetKey("string", "hello")
	context.SetKey("int", 42)
	context.SetKey("bool", true)
	context.SetKey("slice", []int{1, 2, 3})
	context.SetKey("map", map[string]string{"k": "v"})

	// 验证大小
	if size := context.Size(); size != 5 {
		t.Errorf("复杂场景中Size()应该返回5，得到 %d", size)
	}

	// 删除一个键
	context.DeleteKey("bool")

	// 验证键已被删除
	exists, _ := context.HasKey("bool")
	if exists {
		t.Error("键'bool'应该已被删除，但HasKey返回true")
	}

	// 修改现有键
	context.SetKey("int", 100)

	// 验证修改成功
	value, _ := context.GetKey("int")
	if value != 100 {
		t.Errorf("修改后的值应该是100，得到 %v", value)
	}

	// 获取所有键并验证数量
	keys := context.GetAllKeys()
	if len(keys) != 4 {
		t.Errorf("应该有4个键，得到 %d 个", len(keys))
	}

	// 清空上下文
	context.Clear()

	// 验证上下文已清空
	if size := context.Size(); size != 0 {
		t.Errorf("清空后Size()应该返回0，得到 %d", size)
	}
}

// 测试边界情况的键名
func TestBaseNodeContext_EdgeCaseKeys(t *testing.T) {
	context := NewBaseNodeContext()

	// 测试空键名
	err, ok := context.SetKey("", "empty key value")
	if err != nil || !ok {
		t.Errorf("SetKey() 空键名返回了错误: %v, ok: %v", err, ok)
	}

	value, exists := context.GetKey("")
	if !exists {
		t.Error("GetKey() 空键名表明键不存在，但它应该存在")
	}
	if value != "empty key value" {
		t.Errorf("GetKey() 空键名返回了 %v, 期望 %v", value, "empty key value")
	}

	// 测试包含特殊字符的键名
	specialKeys := []string{
		"key with spaces",
		"key-with-dashes",
		"key_with_underscores",
		"key.with.dots",
		"中文键名",
		"!@#$%^&*()",
		"very long key name " + string(make([]byte, 1000)),
	}

	for _, key := range specialKeys {
		err, ok := context.SetKey(key, "value for "+key)
		if err != nil || !ok {
			t.Errorf("SetKey() 特殊键名 '%s' 返回了错误: %v, ok: %v", key, err, ok)
		}

		value, exists := context.GetKey(key)
		if !exists {
			t.Errorf("GetKey() 特殊键名 '%s' 表明键不存在，但它应该存在", key)
		}
		if value != "value for "+key {
			t.Errorf("GetKey() 特殊键名 '%s' 返回了 %v, 期望 %v", key, value, "value for "+key)
		}
	}
}

// 测试复杂数据结构作为值
func TestBaseNodeContext_ComplexValues(t *testing.T) {
	context := NewBaseNodeContext()

	// 测试嵌套的数据结构
	type NestedStruct struct {
		Name  string
		Value int
		Data  map[string]interface{}
	}

	nestedValue := NestedStruct{
		Name:  "test",
		Value: 42,
		Data: map[string]interface{}{
			"a": 1,
			"b": "string",
			"c": []float64{1.1, 2.2, 3.3},
		},
	}

	// 设置复杂值
	err, ok := context.SetKey("nested", nestedValue)
	if err != nil || !ok {
		t.Errorf("SetKey() 复杂值返回了错误: %v, ok: %v", err, ok)
	}

	// 获取复杂值
	value, exists := context.GetKey("nested")
	if !exists {
		t.Error("GetKey() 复杂值表明键不存在，但它应该存在")
	}

	// 验证复杂值的完整性
	retrievedValue, ok := value.(NestedStruct)
	if !ok {
		t.Errorf("无法将获取的值转换回原始类型: %T", value)
	} else {
		if retrievedValue.Name != nestedValue.Name {
			t.Errorf("获取的复杂值中Name = %s, 期望 %s", retrievedValue.Name, nestedValue.Name)
		}
		if retrievedValue.Value != nestedValue.Value {
			t.Errorf("获取的复杂值中Value = %d, 期望 %d", retrievedValue.Value, nestedValue.Value)
		}
		if !reflect.DeepEqual(retrievedValue.Data, nestedValue.Data) {
			t.Errorf("获取的复杂值中Data不匹配: %v, 期望 %v", retrievedValue.Data, nestedValue.Data)
		}
	}

	// 测试通道、函数等不可序列化的值
	ch := make(chan int)
	fn := func() {}

	context.SetKey("channel", ch)
	context.SetKey("function", fn)

	// 验证可以存储和检索这些类型
	_, exists = context.GetKey("channel")
	if !exists {
		t.Error("无法检索通道值")
	}

	_, exists = context.GetKey("function")
	if !exists {
		t.Error("无法检索函数值")
	}
}

// 测试并发操作
func TestBaseNodeContext_ConcurrentAccess(t *testing.T) {
	context := NewBaseNodeContext()
	const goroutines = 10
	const operationsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines * 2) // 读写各goroutines个

	// 写入goroutines
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := "key" + strconv.Itoa(id) + strconv.Itoa(j)
				value := id*1000 + j
				context.SetKey(key, value)
				// 短暂延迟，增加并发冲突的可能性
				time.Sleep(time.Microsecond)
			}
		}(i)
	}

	// 读取goroutines
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				index := j % goroutines // 随机取一些其他goroutine写入的值
				key := "key" + strconv.Itoa(index) + strconv.Itoa(j%operationsPerGoroutine)
				context.GetKey(key)
				// 有时删除键
				if j%10 == 0 {
					context.DeleteKey(key)
				}
				// 有时获取所有键或值
				if j%20 == 0 {
					context.GetAllKeys()
				}
				if j%30 == 0 {
					context.GetAllValues()
				}
				// 短暂延迟，增加并发冲突的可能性
				time.Sleep(time.Microsecond)
			}
		}(i)
	}

	// 等待所有操作完成
	wg.Wait()

	// 如果没有panic，则认为并发测试通过
	// 我们无法确切知道最终的键数量，因为有随机删除操作
}

// 测试与BaseNode集成
func TestBaseNodeContext_IntegrationWithBaseNode(t *testing.T) {
	// 创建上下文并设置一些值
	context := NewBaseNodeContext()
	context.SetKey("nodeType", "special")
	context.SetKey("priority", 10)
	context.SetKey("metadata", map[string]string{"created": "today"})

	// 创建使用该上下文的节点
	node := NewBaseNode[NodeContext]("test", "key1", "value1", context)

	// 验证通过节点可以访问上下文中的值
	nodeContext := node.GetContext()
	if nodeContext == nil {
		t.Fatal("节点的上下文为nil")
	}

	// 验证可以获取上下文中的值
	nodeType, exists := nodeContext.GetKey("nodeType")
	if !exists || nodeType != "special" {
		t.Errorf("上下文中的nodeType = %v, 期望 %v, exists = %v", nodeType, "special", exists)
	}

	priority, exists := nodeContext.GetKey("priority")
	if !exists || priority != 10 {
		t.Errorf("上下文中的priority = %v, 期望 %v, exists = %v", priority, 10, exists)
	}

	// 验证节点方法不影响上下文
	if err := node.SetKey("newKey"); err != nil {
		t.Errorf("SetKey() 错误 = %v", err)
	}

	// 上下文的键不应该被更改
	unchangedValue, exists := nodeContext.GetKey("nodeType")
	if !exists || unchangedValue != "special" {
		t.Errorf("节点操作后，上下文值被意外修改: %v, exists = %v", unchangedValue, exists)
	}

	// 测试修改上下文对象
	nodeContext.SetKey("nodeType", "modified")

	// 通过节点再次获取上下文，验证修改已生效
	updatedContext := node.GetContext()
	updatedValue, exists := updatedContext.GetKey("nodeType")
	if !exists || updatedValue != "modified" {
		t.Errorf("上下文修改未在节点中体现: %v, exists = %v", updatedValue, exists)
	}
}

// 测试边界条件下的所有方法行为
func TestBaseNodeContext_EdgeCases(t *testing.T) {
	context := NewBaseNodeContext()

	// 1. 在空map上调用所有方法
	if _, exists := context.GetKey("anyKey"); exists {
		t.Error("空上下文的GetKey应返回exists=false")
	}

	if exists, _ := context.HasKey("anyKey"); exists {
		t.Error("空上下文的HasKey应返回false")
	}

	if keys := context.GetAllKeys(); len(keys) != 0 {
		t.Error("空上下文的GetAllKeys应返回空切片")
	}

	if values := context.GetAllValues(); len(values) != 0 {
		t.Error("空上下文的GetAllValues应返回空切片")
	}

	if items := context.GetAllItems(); len(items) != 0 {
		t.Error("空上下文的GetAllItems应返回空map")
	}

	// 2. 对同一键重复操作
	context.SetKey("testKey", "value1")
	context.SetKey("testKey", "value2")
	if value, _ := context.GetKey("testKey"); value != "value2" {
		t.Errorf("重复SetKey后，值应该是最后一次设置的, 得到 %v", value)
	}

	// 3. 删除后再添加同一键
	context.DeleteKey("testKey")
	context.SetKey("testKey", "value3")
	if value, _ := context.GetKey("testKey"); value != "value3" {
		t.Errorf("删除后重新添加的键，值应该是新设置的, 得到 %v", value)
	}

	// 4. 清空后检查所有方法
	context.SetKey("key1", "value1")
	context.Clear()

	if _, exists := context.GetKey("key1"); exists {
		t.Error("Clear后GetKey应返回exists=false")
	}

	if exists, _ := context.HasKey("key1"); exists {
		t.Error("Clear后HasKey应返回false")
	}

	if keys := context.GetAllKeys(); len(keys) != 0 {
		t.Error("Clear后GetAllKeys应返回空切片")
	}

	if size := context.Size(); size != 0 {
		t.Errorf("Clear后Size应返回0, 得到 %d", size)
	}
}
