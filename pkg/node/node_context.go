package node

// NodeContext 节点上下文接口，用于存储节点的元数据
// 该接口定义了节点上下文的基本操作，包括键值对的增删改查、清空操作等
// 实现该接口的结构可以作为节点的上下文存储，用于保存节点相关的状态信息
type NodeContext interface {

	// SetKey 设置节点上下文的键值对
	// 参数:
	//   - key: 键名
	//   - value: 键值，可以是任意类型
	// 返回:
	//   - error: 操作过程中的错误，如果成功则为nil
	//   - bool: 操作是否成功
	SetKey(key string, value any) (error, bool)

	// GetKey 获取节点上下文的键值对
	// 参数:
	//   - key: 键名
	// 返回:
	//   - any: 获取到的键值，如果键不存在则为nil
	//   - bool: 键是否存在
	GetKey(key string) (any, bool)

	// DeleteKey 删除节点上下文的键值对
	// 参数:
	//   - key: 要删除的键名
	// 返回:
	//   - error: 操作过程中的错误，如果成功则为nil
	//   - bool: 操作是否成功
	DeleteKey(key string) (error, bool)

	// HasKey 检查节点上下文是否存在某个键
	// 参数:
	//   - key: 要检查的键名
	// 返回:
	//   - bool: 键是否存在
	//   - error: 操作过程中的错误，如果成功则为nil
	HasKey(key string) (bool, error)

	// Clear 清空节点上下文
	// 删除所有的键值对，重置上下文状态
	// 返回:
	//   - error: 操作过程中的错误，如果成功则为nil
	Clear() error

	// GetAllKeys 获取所有键
	// 返回上下文中所有的键列表
	// 返回:
	//   - []string: 键的字符串列表
	GetAllKeys() []string

	// GetAllValues 获取所有值
	// 返回上下文中所有的值列表
	// 返回:
	//   - []any: 值列表，可能包含不同类型的元素
	GetAllValues() []any

	// GetAllItems 获取所有键值对
	// 返回上下文中的所有键值对映射
	// 返回:
	//   - map[string]any: 包含所有键值对的映射
	GetAllItems() map[string]any

	// Size 获取节点上下文的大小
	// 返回上下文中键值对的数量
	// 返回:
	//   - int: 键值对数量
	Size() int
}
