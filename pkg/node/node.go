package node

// Node 表示路由树上的一个节点，节点可能会有各种不同类型的实现
type Node[Context NodeContext] interface {

	// GetType 获取节点的类型
	// 节点类型用于区分不同功能的节点，如路径节点、变量节点等
	// 返回:
	//   - string: 节点类型的字符串标识符
	GetType() string

	// GetParent 获取节点的父节点
	// 返回:
	//   - Node[Context]: 父节点，如果是根节点则返回nil
	GetParent() Node[Context]

	// SetParent 设置节点的父节点
	// 参数:
	//   - parent: 要设置的父节点
	// 返回:
	//   - error: 操作过程中的错误，如果成功则为nil
	SetParent(parent Node[Context]) error

	// HasParent 检查节点是否有父节点
	// 返回:
	//   - bool: 是否有父节点
	HasParent() bool

	// IsRoot 检查节点是否为根节点
	// 返回:
	//   - bool: 是否为根节点（没有父节点）
	IsRoot() bool

	// GetRoot 获取当前节点所在树的根节点
	// 通过递归向上查找，直到找到没有父节点的节点
	// 返回:
	//   - Node[Context]: 根节点
	GetRoot() Node[Context]

	// GetDepth 获取节点在树中的深度
	// 根节点的深度为0，每向下一层深度加1
	// 返回:
	//   - int: 节点深度
	GetDepth() int

	// GetKey 获取节点的键名
	// 根据不同的节点类型，键名可能有不同的含义，例如路径节点的路径名、变量节点的变量名等
	// 返回:
	//   - string: 节点的键名
	GetKey() string

	// SetKey 设置节点的键名
	// 参数:
	//   - key: 要设置的键名
	// 返回:
	//   - error: 操作过程中的错误，如果成功则为nil
	SetKey(key string) error

	// GetValue 获取节点的值
	// 根据不同的节点类型，值可能有不同的含义，例如变量节点的默认值、路径参数的实际值等
	// 返回:
	//   - string: 节点的值
	GetValue() string

	// SetValue 设置节点的值
	// 参数:
	//   - value: 要设置的值
	// 返回:
	//   - error: 操作过程中的错误，如果成功则为nil
	SetValue(value string) error

	// GetChildren 获取节点的所有子节点
	// 返回当前节点的所有直接子节点列表
	// 返回:
	//   - []Node[Context]: 子节点列表
	GetChildren() []Node[Context]

	// AddChild 添加子节点
	// 将指定节点添加为当前节点的子节点
	// 参数:
	//   - child: 要添加的子节点
	// 返回:
	//   - error: 操作过程中的错误，如果成功则为nil
	AddChild(child Node[Context]) error

	// RemoveChild 移除指定的子节点
	// 从当前节点的子节点列表中移除指定节点
	// 参数:
	//   - child: 要移除的子节点
	// 返回:
	//   - error: 操作过程中的错误，如果成功则为nil
	RemoveChild(child Node[Context]) error

	// RemoveChildByType 移除指定类型的所有子节点
	// 从当前节点的子节点列表中移除所有指定类型的节点
	// 参数:
	//   - nodeType: 要移除的节点类型
	RemoveChildByType(nodeType string)

	// GetChildByType 根据节点类型获取子节点
	// 查找并返回当前节点的第一个指定类型的子节点
	// 参数:
	//   - nodeType: 要查找的节点类型
	// 返回:
	//   - Node[Context]: 找到的子节点，如果不存在则返回nil
	GetChildByType(nodeType string) Node[Context]

	// FindChildByKey 根据键名查找子节点
	// 查找并返回当前节点中键名匹配的第一个子节点
	// 参数:
	//   - key: 要查找的键名
	// 返回:
	//   - Node[Context]: 找到的子节点，如果不存在则返回nil
	FindChildByKey(key string) Node[Context]

	// HasChildren 判断当前节点是否有子节点
	// 返回:
	//   - bool: 如果当前节点有子节点则返回true，否则返回false
	HasChildren() bool

	// ClearChildren 清空所有子节点
	// 删除当前节点的所有子节点
	// 返回:
	//   - error: 操作过程中的错误，如果成功则为nil
	ClearChildren() error

	// GetSiblings 获取同级节点（共享同一父节点的其他节点）
	// 返回与当前节点共享同一个父节点的所有其他节点
	// 如果当前节点是根节点（没有父节点），则返回空切片
	// 返回:
	//   - []Node[Context]: 同级节点列表，不包含当前节点自身
	GetSiblings() []Node[Context]

	// HasSiblings 检查当前节点是否有兄弟节点
	// 如果当前节点有父节点，且父节点有多个子节点，则返回true
	// 如果当前节点是根节点或者是父节点的唯一子节点，则返回false
	// 返回:
	//   - bool: 是否有兄弟节点
	HasSiblings() bool

	// GetSiblingByType 根据节点类型获取特定的兄弟节点
	// 参数:
	//   - nodeType: 要查找的节点类型
	// 返回:
	//   - Node[Context]: 找到的兄弟节点，如果不存在则返回nil
	GetSiblingByType(nodeType string) Node[Context]

	// GetSiblingByKey 根据键名查找兄弟节点
	// 参数:
	//   - key: 要查找的键名
	// 返回:
	//   - Node[Context]: 找到的兄弟节点，如果不存在则返回nil
	GetSiblingByKey(key string) Node[Context]

	// GetSiblingCount 获取兄弟节点的数量
	// 返回与当前节点共享同一父节点的其他节点的数量
	// 返回:
	//   - int: 兄弟节点的数量
	GetSiblingCount() int

	// GetContext 获取节点上下文
	// 返回当前节点关联的上下文对象
	// 返回:
	//   - Context: 节点的上下文对象
	GetContext() Context

	// SetContext 设置节点上下文
	// 为当前节点设置新的上下文对象
	// 参数:
	//   - Context: 要设置的上下文对象
	// 返回:
	//   - error: 设置过程中可能发生的错误，如果成功则返回nil
	SetContext(Context) error

	// GetParentByType 获取特定类型的祖先节点
	// 从当前节点开始向上查找，返回第一个匹配指定类型的祖先节点
	// 参数:
	//   - nodeType: 要查找的节点类型
	// 返回:
	//   - Node[Context]: 找到的祖先节点，如果不存在则返回nil
	GetParentByType(nodeType string) Node[Context]

	// GetAllAncestors 获取从根节点到当前节点的所有祖先节点
	// 返回一个包含所有祖先节点的切片，按照从根节点到当前节点的父节点的顺序排列
	// 返回:
	//   - []Node[Context]: 祖先节点列表
	GetAllAncestors() []Node[Context]

	// IsMatch 判断当前节点是否匹配某个请求路径
	// 根据不同的节点类型，匹配规则可能有所不同
	// 例如路径节点可能需要匹配完整的路径，变量节点可能需要匹配变量名等
	// 参数:
	//   - path: 要匹配的请求路径
	// 返回:
	//   - bool: 如果当前节点匹配请求路径则返回true，否则返回false
	IsMatch(path string) bool

	// IsLeaf 判断当前节点是否为叶子节点（即没有子节点）
	IsLeaf() bool

	// GetChildCount 获取当前节点的子节点数量
	// 返回:
	//   - int: 当前节点的子节点数量
	GetChildCount() int

	// Serialize 序列化节点为字节数组
	Serialize() ([]byte, error)

	// Deserialize 从字节数组反序列化节点
	Deserialize([]byte) error
	// Clone 克隆节点（深拷贝）
	// 创建当前节点的完整副本，包括其所有属性和子节点
	// 返回:
	//   - Node[Context]: 当前节点的深拷贝
	Clone() Node[Context]

	// Equals 判断两个节点是否相等
	// 比较当前节点与另一个节点的所有属性和结构是否完全一致
	// 参数:
	//   - Node[Context]: 要比较的另一个节点
	// 返回:
	//   - bool: 如果两个节点完全相等则返回true，否则返回false
	Equals(Node[Context]) bool

	// MergeWith 与另一个节点合并
	// 将另一个节点的属性和子节点合并到当前节点中
	// 合并规则可能因节点类型而异，通常会保留当前节点的优先级
	// 参数:
	//   - other: 要合并的源节点
	// 返回:
	//   - error: 合并过程中可能发生的错误，如果成功则返回nil
	MergeWith(other Node[Context]) error

	// VisitChildren 使用访问者模式遍历子节点，从当前节点的孩子节点开始
	// 对当前节点的每个子节点应用访问者函数，但不包括当前节点本身
	// 参数:
	//   - visitor: 访问者函数，接收一个节点参数并返回一个布尔值
	//              如果返回true，则继续遍历；如果返回false，则停止遍历
	// 用途:
	//   - 对子树进行局部操作而不影响当前节点
	//   - 实现子树的特定处理逻辑
	VisitChildren(visitor func(Node[Context]) bool)

	// VisitLevelOrder 层序遍历树（按层从上到下，每层从左到右）
	// 首先访问当前节点，然后访问所有子节点，然后是所有孙节点，以此类推
	// 参数:
	//   - visitor: 访问者函数，接收一个节点参数并返回一个布尔值
	//              如果返回true，则继续遍历；如果返回false，则停止遍历
	// 用途:
	//   - 查找最短路径
	//   - 按层级处理节点
	VisitLevelOrder(visitor func(Node[Context]) bool)

	// FindNode 在树中查找满足特定条件的节点
	// 从当前节点开始深度优先搜索，返回第一个满足条件的节点
	// 参数:
	//   - predicate: 判断函数，接收一个节点参数并返回一个布尔值
	//                如果返回true，表示找到了目标节点
	// 返回:
	//   - Node[Context]: 找到的节点，如果不存在则返回nil
	// 用途:
	//   - 查找特定属性的节点
	//   - 实现复杂的节点搜索逻辑
	FindNode(predicate func(Node[Context]) bool) Node[Context]

	// IncrementRequestCount 增加节点的请求计数
	// 每当该节点被请求命中时，调用此方法来增加计数器
	// 此方法应当是线程安全的，以支持并发访问
	// 用途:
	//   - 统计节点的访问频率
	//   - 用于负载均衡或热点分析
	IncrementRequestCount()

	// GetRequestCount 获取当前节点被请求命中的次数
	// 返回:
	//   - int64: 当前节点被请求命中的总次数
	GetRequestCount() int64

	// DeepClone 深度克隆节点及其所有子节点
	// 此方法会复制整个子树结构，包括所有子节点
	// 返回:
	//   - Node[Context]: 当前节点及其所有子树的深拷贝
	DeepClone() Node[Context]

	// IsDynamic 判断当前节点是否为动态节点
	// 动态节点是指节点的值是不固定的，可能会发生各种变化
	// 返回:
	//   - bool: 如果当前节点是动态节点则返回true，否则返回false
	IsDynamic() bool
}
