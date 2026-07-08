package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

// 定义常见错误
var (
	ErrNilNode        = errors.New("无效的nil节点")
	ErrNodeNotFound   = errors.New("节点不存在")
	ErrDuplicateNode  = errors.New("节点已存在")
	ErrInvalidContext = errors.New("无效的上下文")
	ErrInvalidKey     = errors.New("无效的键")
	ErrInvalidValue   = errors.New("无效的值")
)

// BaseNode 是Node接口的基本实现
// 提供了节点的通用功能和属性
type BaseNode[Context NodeContext] struct {
	// 节点基本属性
	nodeType string
	key      string
	value    string

	// 节点关系
	parent   Node[Context]
	children []Node[Context]

	// 索引加速查找
	childrenByKey  map[string]Node[Context]   // 通过键名索引子节点
	childrenByType map[string][]Node[Context] // 通过类型索引子节点

	// 节点元数据
	context Context

	// 性能统计
	requestCount int64

	// 并发控制 - 更细粒度的锁控制
	propMu    sync.RWMutex // 用于保护节点基本属性 (nodeType, key, value)
	parentMu  sync.RWMutex // 用于保护父节点引用
	childMu   sync.RWMutex // 用于保护子节点集合和索引
	contextMu sync.RWMutex // 用于保护上下文

	// 路径缓存
	cachedRoot      Node[Context]   // 缓存的根节点
	cachedAncestors []Node[Context] // 缓存的祖先节点
	cacheMu         sync.RWMutex    // 用于保护缓存
}

// NewBaseNode 创建一个新的基础节点
// 参数:
//   - nodeType: 节点类型
//   - key: 节点键
//   - value: 节点值
//   - context: 节点上下文
//
// 返回:
//   - *BaseNode[Context]: 新创建的基础节点
func NewBaseNode[Context NodeContext](nodeType, key, value string, context Context) *BaseNode[Context] {
	return &BaseNode[Context]{
		nodeType:       nodeType,
		key:            key,
		value:          value,
		children:       make([]Node[Context], 0),
		childrenByKey:  make(map[string]Node[Context]),
		childrenByType: make(map[string][]Node[Context]),
		context:        context,
		requestCount:   0,
	}
}

// GetType 获取节点的类型
func (n *BaseNode[Context]) GetType() string {
	n.propMu.RLock()
	defer n.propMu.RUnlock()
	return n.nodeType
}

// GetParent 获取节点的父节点
func (n *BaseNode[Context]) GetParent() Node[Context] {
	n.parentMu.RLock()
	defer n.parentMu.RUnlock()
	return n.parent
}

// SetParent 设置节点的父节点
func (n *BaseNode[Context]) SetParent(parent Node[Context]) error {
	n.parentMu.Lock()
	defer n.parentMu.Unlock()

	// 如果父节点发生变化，清除路径缓存
	if n.parent != parent {
		n.clearPathCache()
	}

	n.parent = parent
	return nil
}

// HasParent 检查节点是否有父节点
func (n *BaseNode[Context]) HasParent() bool {
	n.parentMu.RLock()
	defer n.parentMu.RUnlock()
	return n.parent != nil
}

// IsRoot 检查节点是否为根节点
func (n *BaseNode[Context]) IsRoot() bool {
	n.parentMu.RLock()
	defer n.parentMu.RUnlock()
	return n.parent == nil
}

// GetRoot 获取当前节点所在树的根节点
func (n *BaseNode[Context]) GetRoot() Node[Context] {
	// 首先检查缓存
	n.cacheMu.RLock()
	if n.cachedRoot != nil {
		root := n.cachedRoot
		n.cacheMu.RUnlock()
		return root
	}
	n.cacheMu.RUnlock()

	// 缓存未命中，遍历查找根节点
	n.parentMu.RLock()
	current := n.parent
	var root Node[Context] = n
	n.parentMu.RUnlock()

	for current != nil {
		root = current
		current = current.GetParent()
	}

	// 存储到缓存中
	n.cacheMu.Lock()
	n.cachedRoot = root
	n.cacheMu.Unlock()

	return root
}

// GetDepth 获取节点在树中的深度
func (n *BaseNode[Context]) GetDepth() int {
	n.parentMu.RLock()
	current := n.parent
	n.parentMu.RUnlock()

	depth := 0
	for current != nil {
		depth++
		current = current.GetParent()
	}

	return depth
}

// GetKey 获取节点的键名
func (n *BaseNode[Context]) GetKey() string {
	n.propMu.RLock()
	defer n.propMu.RUnlock()
	return n.key
}

// SetKey 设置节点的键名
func (n *BaseNode[Context]) SetKey(key string) error {
	n.propMu.Lock()
	defer n.propMu.Unlock()
	n.key = key
	return nil
}

// GetValue 获取节点的值
func (n *BaseNode[Context]) GetValue() string {
	n.propMu.RLock()
	defer n.propMu.RUnlock()
	return n.value
}

// SetValue 设置节点的值
func (n *BaseNode[Context]) SetValue(value string) error {
	n.propMu.Lock()
	defer n.propMu.Unlock()
	n.value = value
	return nil
}

// GetChildren 获取节点的所有子节点
func (n *BaseNode[Context]) GetChildren() []Node[Context] {
	n.childMu.RLock()
	defer n.childMu.RUnlock()

	// 返回一个副本以避免外部修改
	result := make([]Node[Context], len(n.children))
	copy(result, n.children)
	return result
}

// AddChild 添加子节点
func (n *BaseNode[Context]) AddChild(child Node[Context]) error {
	if child == nil {
		return fmt.Errorf("%w: 无法添加nil子节点", ErrNilNode)
	}

	// 防止节点添加自身为子节点，避免循环引用
	if child == n {
		return fmt.Errorf("%w: 无法将节点添加为自身的子节点", ErrDuplicateNode)
	}

	n.childMu.Lock()
	defer n.childMu.Unlock()

	// 设置父子关系
	if err := child.SetParent(n); err != nil {
		return fmt.Errorf("设置父节点失败: %w", err)
	}

	// 检查该子节点是否已经在列表中
	childKey := child.GetKey()
	if existingChild, ok := n.childrenByKey[childKey]; ok && existingChild == child {
		return nil // 子节点已存在，无需再添加
	}

	// 添加到子节点列表
	n.children = append(n.children, child)

	// 更新索引
	n.childrenByKey[childKey] = child

	childType := child.GetType()
	if _, ok := n.childrenByType[childType]; !ok {
		n.childrenByType[childType] = make([]Node[Context], 0, 1)
	}
	n.childrenByType[childType] = append(n.childrenByType[childType], child)

	return nil
}

// RemoveChild 移除指定的子节点
func (n *BaseNode[Context]) RemoveChild(child Node[Context]) error {
	if child == nil {
		return fmt.Errorf("%w: 无法移除nil子节点", ErrNilNode)
	}

	n.childMu.Lock()
	defer n.childMu.Unlock()

	for i, c := range n.children {
		if c == child {
			// 移除父子关系
			if err := child.SetParent(nil); err != nil {
				return fmt.Errorf("解除父子关系失败: %w", err)
			}

			// 从子节点列表中移除
			n.children = append(n.children[:i], n.children[i+1:]...)

			// 从索引中移除
			delete(n.childrenByKey, child.GetKey())

			// 从类型索引中移除
			childType := child.GetType()
			if typeChildren, ok := n.childrenByType[childType]; ok {
				for j, tc := range typeChildren {
					if tc == child {
						// 如果是最后一个该类型的节点，移除整个索引项
						if len(typeChildren) == 1 {
							delete(n.childrenByType, childType)
						} else {
							// 否则移除该节点
							n.childrenByType[childType] = append(typeChildren[:j], typeChildren[j+1:]...)
						}
						break
					}
				}
			}

			return nil
		}
	}

	return fmt.Errorf("%w: 子节点不在当前节点的子节点列表中", ErrNodeNotFound)
}

// RemoveChildByType 移除指定类型的所有子节点
func (n *BaseNode[Context]) RemoveChildByType(nodeType string) {
	n.childMu.Lock()
	defer n.childMu.Unlock()

	var newChildren []Node[Context]

	// 如果存在此类型的子节点，则处理它们
	if children, ok := n.childrenByType[nodeType]; ok {
		// 为所有其他子节点创建新切片
		newChildren = make([]Node[Context], 0, len(n.children)-len(children))

		for _, child := range n.children {
			if child.GetType() != nodeType {
				newChildren = append(newChildren, child)
			} else {
				// 移除父子关系
				child.SetParent(nil)

				// 从键名索引中移除
				delete(n.childrenByKey, child.GetKey())
			}
		}

		// 更新子节点列表和删除类型索引
		n.children = newChildren
		delete(n.childrenByType, nodeType)
	}
}

// GetChildByType 根据节点类型获取子节点
func (n *BaseNode[Context]) GetChildByType(nodeType string) Node[Context] {
	n.childMu.RLock()
	defer n.childMu.RUnlock()

	// 使用类型索引快速查找
	if children, ok := n.childrenByType[nodeType]; ok && len(children) > 0 {
		return children[0]
	}

	return nil
}

// FindChildByKey 根据键名查找子节点
func (n *BaseNode[Context]) FindChildByKey(key string) Node[Context] {
	n.childMu.RLock()
	defer n.childMu.RUnlock()

	// 使用键名索引快速查找
	if child, ok := n.childrenByKey[key]; ok {
		return child
	}

	return nil
}

// GetSiblings 获取同级节点（共享同一父节点的其他节点）
func (n *BaseNode[Context]) GetSiblings() []Node[Context] {
	parent := n.GetParent()
	if parent == nil {
		return []Node[Context]{}
	}

	siblings := make([]Node[Context], 0)
	for _, child := range parent.GetChildren() {
		if child != n {
			siblings = append(siblings, child)
		}
	}

	return siblings
}

// HasSiblings 检查当前节点是否有兄弟节点
func (n *BaseNode[Context]) HasSiblings() bool {
	return n.GetSiblingCount() > 0
}

// GetSiblingByType 根据节点类型获取特定的兄弟节点
func (n *BaseNode[Context]) GetSiblingByType(nodeType string) Node[Context] {
	for _, sibling := range n.GetSiblings() {
		if sibling.GetType() == nodeType {
			return sibling
		}
	}

	return nil
}

// GetSiblingByKey 根据键名查找兄弟节点
func (n *BaseNode[Context]) GetSiblingByKey(key string) Node[Context] {
	for _, sibling := range n.GetSiblings() {
		if sibling.GetKey() == key {
			return sibling
		}
	}

	return nil
}

// GetSiblingCount 获取兄弟节点的数量
func (n *BaseNode[Context]) GetSiblingCount() int {
	parent := n.GetParent()
	if parent == nil {
		return 0
	}

	return parent.GetChildCount() - 1
}

// GetContext 获取节点上下文
func (n *BaseNode[Context]) GetContext() Context {
	n.contextMu.RLock()
	defer n.contextMu.RUnlock()
	return n.context
}

// SetContext 设置节点上下文
func (n *BaseNode[Context]) SetContext(context Context) error {
	// 由于 Context 是泛型参数，我们不能直接和 nil 比较
	// 如果需要验证上下文有效性，可以在具体实现中添加额外逻辑
	n.contextMu.Lock()
	defer n.contextMu.Unlock()
	n.context = context
	return nil
}

// GetParentByType 获取特定类型的祖先节点
func (n *BaseNode[Context]) GetParentByType(nodeType string) Node[Context] {
	current := n.GetParent()

	for current != nil {
		if current.GetType() == nodeType {
			return current
		}
		current = current.GetParent()
	}

	return nil
}

// GetAllAncestors 获取从根节点到当前节点的所有祖先节点
func (n *BaseNode[Context]) GetAllAncestors() []Node[Context] {
	// 首先检查缓存
	n.cacheMu.RLock()
	if n.cachedAncestors != nil {
		// 返回缓存副本以避免外部修改
		result := make([]Node[Context], len(n.cachedAncestors))
		copy(result, n.cachedAncestors)
		n.cacheMu.RUnlock()
		return result
	}
	n.cacheMu.RUnlock()

	// 缓存未命中，收集所有祖先节点
	ancestors := make([]Node[Context], 0)
	current := n.GetParent()

	for current != nil {
		ancestors = append(ancestors, current)
		current = current.GetParent()
	}

	// 反转列表，使其从根到父的顺序
	if len(ancestors) > 1 {
		for i, j := 0, len(ancestors)-1; i < j; i, j = i+1, j-1 {
			ancestors[i], ancestors[j] = ancestors[j], ancestors[i]
		}
	}

	// 存储到缓存中
	n.cacheMu.Lock()
	n.cachedAncestors = make([]Node[Context], len(ancestors))
	copy(n.cachedAncestors, ancestors)
	n.cacheMu.Unlock()

	return ancestors
}

// IsMatch 判断当前节点是否匹配某个请求路径
// 基本实现仅检查路径是否与节点的键完全匹配
// 子类可以重写此方法以提供更复杂的匹配逻辑
func (n *BaseNode[Context]) IsMatch(path string) bool {
	n.propMu.RLock()
	defer n.propMu.RUnlock()
	return n.key == path
}

// HasChildren 判断当前节点是否有子节点
func (n *BaseNode[Context]) HasChildren() bool {
	n.childMu.RLock()
	defer n.childMu.RUnlock()
	return len(n.children) > 0
}

// FindChildByPath 根据路径查找子节点
// 基本实现是简单地检查直接子节点是否有匹配路径的
// 子类可以重写此方法以提供更复杂的路径查找逻辑
func (n *BaseNode[Context]) FindChildByPath(path string) Node[Context] {
	n.childMu.RLock()
	children := n.children
	n.childMu.RUnlock()

	for _, child := range children {
		if child.IsMatch(path) {
			return child
		}
	}

	return nil
}

// ClearChildren 清空所有子节点
func (n *BaseNode[Context]) ClearChildren() error {
	n.childMu.Lock()
	defer n.childMu.Unlock()

	// 解除所有子节点的父子关系
	for _, child := range n.children {
		child.SetParent(nil)
	}

	// 清空子节点列表和索引
	n.children = make([]Node[Context], 0)
	n.childrenByKey = make(map[string]Node[Context])
	n.childrenByType = make(map[string][]Node[Context])

	return nil
}

// IsLeaf 判断当前节点是否为叶子节点（即没有子节点）
func (n *BaseNode[Context]) IsLeaf() bool {
	n.childMu.RLock()
	defer n.childMu.RUnlock()
	return len(n.children) == 0
}

// GetChildCount 获取当前节点的子节点数量
func (n *BaseNode[Context]) GetChildCount() int {
	n.childMu.RLock()
	defer n.childMu.RUnlock()
	return len(n.children)
}

// Serialize 序列化节点为字节数组
// 基本实现仅序列化节点本身的基本属性，不包括子节点和上下文
// 子类可以重写此方法以提供完整的序列化功能
func (n *BaseNode[Context]) Serialize() ([]byte, error) {
	n.propMu.RLock()
	defer n.propMu.RUnlock()

	type nodeData struct {
		Type  string `json:"type"`
		Key   string `json:"key"`
		Value string `json:"value"`
	}

	data := nodeData{
		Type:  n.nodeType,
		Key:   n.key,
		Value: n.value,
	}

	return json.Marshal(data)
}

// Deserialize 从字节数组反序列化节点
// 基本实现仅反序列化节点本身的基本属性
// 子类可以重写此方法以提供完整的反序列化功能
func (n *BaseNode[Context]) Deserialize(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("反序列化失败: 输入数据为空")
	}

	type nodeData struct {
		Type  string `json:"type"`
		Key   string `json:"key"`
		Value string `json:"value"`
	}

	var parsed nodeData
	if err := json.Unmarshal(data, &parsed); err != nil {
		return fmt.Errorf("JSON解析失败: %w", err)
	}

	n.propMu.Lock()
	defer n.propMu.Unlock()

	n.nodeType = parsed.Type
	n.key = parsed.Key
	n.value = parsed.Value

	return nil
}

// Clone 克隆节点（深拷贝）
// 基本实现创建一个新的节点，复制基本属性，但不复制子节点和父节点关系
// 对于需要完整克隆整个子树的情况，请使用 DeepClone 方法或在子类中重写此方法
func (n *BaseNode[Context]) Clone() Node[Context] {
	// 获取所需属性，减少锁的持有时间
	n.propMu.RLock()
	nodeType := n.nodeType
	key := n.key
	value := n.value
	n.propMu.RUnlock()

	n.contextMu.RLock()
	contextClone := n.context
	n.contextMu.RUnlock()

	// 创建新的节点
	clone := &BaseNode[Context]{
		nodeType:       nodeType,
		key:            key,
		value:          value,
		context:        contextClone,
		children:       make([]Node[Context], 0),
		childrenByKey:  make(map[string]Node[Context]),
		childrenByType: make(map[string][]Node[Context]),
		requestCount:   0, // 不复制请求计数，因为这是一个新节点
	}

	return clone
}

// DeepClone 深度克隆节点及其所有子节点
// 此方法会复制整个子树结构，包括所有子节点
// 注意: 上下文对象仍然是浅拷贝的
func (n *BaseNode[Context]) DeepClone() Node[Context] {
	// 创建基础克隆
	clone := n.Clone()

	// 获取子节点的本地副本，减少锁的持有时间
	n.childMu.RLock()
	localChildren := make([]Node[Context], len(n.children))
	copy(localChildren, n.children)
	n.childMu.RUnlock()

	// 克隆所有子节点并添加到新节点，并行处理
	if len(localChildren) > 10 { // 只有当子节点数量足够大时才使用并行处理
		var wg sync.WaitGroup
		childClones := make([]Node[Context], len(localChildren))

		for i, child := range localChildren {
			wg.Add(1)
			go func(index int, childNode Node[Context]) {
				defer wg.Done()
				childClones[index] = childNode.DeepClone() // 使用DeepClone而不是Clone来实现深度复制
			}(i, child)
		}

		wg.Wait()

		// 依次添加子节点，保持添加顺序
		for _, childClone := range childClones {
			clone.AddChild(childClone)
		}
	} else {
		// 子节点较少时直接串行处理更高效
		for _, child := range localChildren {
			childClone := child.DeepClone() // 使用DeepClone而不是Clone来实现深度复制
			clone.AddChild(childClone)
		}
	}

	return clone
}

// Equals 判断两个节点是否相等
// 基本实现仅比较节点的基本属性
// 子类可以重写此方法以提供更复杂的相等性比较逻辑
func (n *BaseNode[Context]) Equals(other Node[Context]) bool {
	if other == nil {
		return false
	}

	n.propMu.RLock()
	defer n.propMu.RUnlock()

	return n.nodeType == other.GetType() &&
		n.key == other.GetKey() &&
		n.value == other.GetValue()
}

// MergeWith 与另一个节点合并
// 将另一个节点的子节点合并到当前节点中
// 合并策略：如果存在键名相同的子节点，则保留当前节点的子节点
func (n *BaseNode[Context]) MergeWith(other Node[Context]) error {
	if other == nil {
		return fmt.Errorf("%w: 无法与nil节点合并", ErrNilNode)
	}

	// 防止节点与自身合并
	if other == n {
		return fmt.Errorf("%w: 无法将节点与自身合并", ErrDuplicateNode)
	}

	// 获取当前节点的所有子节点键名
	n.childMu.RLock()
	existingKeys := make(map[string]bool, len(n.childrenByKey))
	for k := range n.childrenByKey {
		existingKeys[k] = true
	}
	n.childMu.RUnlock()

	// 合并子节点，跳过键名已存在的子节点
	otherChildren := other.GetChildren()
	for _, child := range otherChildren {
		// 如果键名已存在，则跳过
		if existingKeys[child.GetKey()] {
			continue
		}

		// 否则添加克隆的子节点
		childClone := child.Clone()
		if err := n.AddChild(childClone); err != nil {
			return fmt.Errorf("合并子节点失败: %w", err)
		}
	}

	return nil
}

// VisitChildren 使用访问者模式遍历子节点
func (n *BaseNode[Context]) VisitChildren(visitor func(Node[Context]) bool) {
	// 获取一个子节点副本以避免并发修改问题，同时减少锁的持有时间
	n.childMu.RLock()
	children := make([]Node[Context], len(n.children))
	copy(children, n.children)
	n.childMu.RUnlock()

	// 批量处理子节点，减少访问者函数的调用次数
	const batchSize = 10
	for i := 0; i < len(children); i += batchSize {
		end := i + batchSize
		if end > len(children) {
			end = len(children)
		}

		// 如果是小批次，直接处理
		if end-i <= 3 {
			for j := i; j < end; j++ {
				if !visitor(children[j]) {
					return
				}
			}
		} else {
			// 对于较大的批次，考虑并行处理
			results := make([]bool, end-i)
			var wg sync.WaitGroup

			for j := i; j < end; j++ {
				wg.Add(1)
				go func(index int, node Node[Context]) {
					defer wg.Done()
					results[index-i] = visitor(node)
				}(j, children[j])
			}

			wg.Wait()

			// 检查是否有任何访问者返回了false
			for _, result := range results {
				if !result {
					return
				}
			}
		}
	}
}

// VisitLevelOrder 层序遍历树
func (n *BaseNode[Context]) VisitLevelOrder(visitor func(Node[Context]) bool) {
	// 使用队列实现层序遍历
	queue := []Node[Context]{n}
	queueIndex := 0 // 使用索引而不是slice操作，减少内存分配

	// 预分配适当大小的队列，避免频繁扩容
	n.childMu.RLock()
	queueCapacity := len(n.children) * 4 // 估计队列大小
	if queueCapacity < 64 {
		queueCapacity = 64 // 最小容量
	}
	n.childMu.RUnlock()

	if cap(queue) < queueCapacity {
		newQueue := make([]Node[Context], 1, queueCapacity)
		newQueue[0] = n
		queue = newQueue
	}

	for queueIndex < len(queue) {
		// 取出当前节点
		current := queue[queueIndex]
		queueIndex++

		// 访问当前节点，如果返回false则提前终止
		if !visitor(current) {
			break
		}

		// 将当前节点的子节点加入队列
		// 直接使用children字段而不是调用GetChildren方法可以减少复制
		if childNode, ok := current.(*BaseNode[Context]); ok {
			childNode.childMu.RLock()
			childrenCount := len(childNode.children)
			// 确保队列有足够的容量
			if len(queue)+childrenCount > cap(queue) {
				// 需要扩容
				newCapacity := cap(queue) * 2
				if newCapacity < len(queue)+childrenCount {
					newCapacity = len(queue) + childrenCount
				}
				newQueue := make([]Node[Context], len(queue), newCapacity)
				copy(newQueue, queue)
				queue = newQueue
			}

			// 直接将子节点加入队列
			queue = append(queue, childNode.children...)
			childNode.childMu.RUnlock()
		} else {
			// 如果不是BaseNode，则使用接口方法
			queue = append(queue, current.GetChildren()...)
		}
	}
}

// FindNode 在树中查找满足特定条件的节点
func (n *BaseNode[Context]) FindNode(predicate func(Node[Context]) bool) Node[Context] {
	// 使用栈实现深度优先搜索，而不是递归，减少函数调用开销
	stack := []Node[Context]{n}

	// 对于一些特殊情况，使用更高效的算法
	// 例如，可以首先检查当前节点及其直接子节点
	if predicate(n) {
		return n
	}

	// 检查直接子节点
	n.childMu.RLock()
	children := make([]Node[Context], len(n.children))
	copy(children, n.children)
	n.childMu.RUnlock()

	for _, child := range children {
		if predicate(child) {
			return child
		}
		stack = append(stack, child)
	}

	// 如果直接子节点中没有找到，继续深度优先搜索
	visited := make(map[Node[Context]]bool)
	visited[n] = true
	for _, child := range children {
		visited[child] = true
	}

	for len(stack) > 0 {
		// 取出栈顶节点
		lastIndex := len(stack) - 1
		current := stack[lastIndex]
		stack = stack[:lastIndex]

		// 将其子节点入栈
		for _, child := range current.GetChildren() {
			if visited[child] {
				continue
			}

			if predicate(child) {
				return child
			}

			visited[child] = true
			stack = append(stack, child)
		}
	}

	return nil
}

// IncrementRequestCount 增加节点的请求计数
func (n *BaseNode[Context]) IncrementRequestCount() {
	atomic.AddInt64(&n.requestCount, 1)
}

// GetRequestCount 获取当前节点被请求命中的次数
func (n *BaseNode[Context]) GetRequestCount() int64 {
	return atomic.LoadInt64(&n.requestCount)
}

// 清除路径缓存
func (n *BaseNode[Context]) clearPathCache() {
	n.cacheMu.Lock()
	defer n.cacheMu.Unlock()
	n.cachedRoot = nil
	n.cachedAncestors = nil
}

// IsDynamic 判断当前节点是否为动态节点
func (n *BaseNode[Context]) IsDynamic() bool {
	// 默认不是动态节点，子节点需要覆写此方法
	return false
}
