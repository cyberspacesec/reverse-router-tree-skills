package node

import (
	"fmt"

	"github.com/cyberspacesec/go-reverse-router-tree/pkg/value"
)

// RequestPathVariableNode 定义了路径变量节点
// 该节点通过黑盒流量分析动态识别出的路径变量
// 不同于白盒路由定义，这里的路径变量是根据实际请求流量推断出来的
type RequestPathVariableNode struct {
	*BaseNode[NodeContext]

	// 这个路径节点的值
	value value.Value
}

// NewRequestPathVariableNode 创建一个新的路径变量节点
// 该节点初始没有预定义的名称或模式，而是随着流量分析逐步建立模式
//
// 参数:
//   - position: 在路径中的位置标识符（如 "segment2" 表示路径的第二段）
//   - inferFunc: 可选的类型推断函数，如果为nil则使用默认推断
//
// 返回:
//   - *RequestPathVariableNode: 新创建的路径变量节点
func NewRequestPathVariableNode(position string, inferFunc func(node Node[NodeContext]) (value.Type, error)) *RequestPathVariableNode {
	context := NewBaseNodeContext()
	// 初始没有明确的key或value，它们会根据观察到的流量推断
	baseNode := NewBaseNode[NodeContext]("request_path_variable", position, "", context)

	// 如果没有提供推断函数，使用默认的空推断函数
	if inferFunc == nil {
		inferFunc = func(node Node[NodeContext]) (value.Type, error) {
			// 默认简单推断，只返回字符串类型
			return value.Type(value.PhysicalTypeString), nil
		}
	}

	return &RequestPathVariableNode{
		BaseNode:    baseNode,
		valueMetric: value.NewValueMetric(),
		valueType:   value.Type(value.PhysicalTypeString), // 默认为字符串类型
		inferFunc:   inferFunc,
	}
}

// SetTypeInferenceFunc 设置类型推断函数
// 这允许在创建节点后更改类型推断的方式
//
// 参数:
//   - inferFunc: 用于推断值类型的函数
func (n *RequestPathVariableNode) SetTypeInferenceFunc(inferFunc func(node Node[NodeContext]) (value.Type, error)) {
	if inferFunc != nil {
		n.inferFunc = inferFunc
	}
}

// IsMatch 判断请求的路径段是否匹配
// 在黑盒分析模式下，我们通常认为变量节点可以匹配任何非空路径段
//
// 参数:
//   - pathSegment: URL路径的一个段
//
// 返回:
//   - bool: 如果路径段可能是变量则返回true，否则返回false
func (n *RequestPathVariableNode) IsMatch(pathSegment string) bool {
	// 变量路径段不应该为空，并且不应该包含路径分隔符
	if pathSegment == "" || pathSegment == "/" {
		return false
	}

	// 记录观察到的值，用于后续类型推断
	n.ObserveValue(pathSegment)

	return true
}

// ObserveValue 记录观察到的路径段值，用于模式推断
//
// 参数:
//   - value: 观察到的路径段值
func (n *RequestPathVariableNode) ObserveValue(val string) {
	n.valueMetric.AddValue(val)

	// 每当累积足够的样本时，重新推断类型
	if n.ShouldUpdateInference() {
		n.UpdateTypeInference()
	}
}

// ShouldUpdateInference 判断是否应该更新类型推断
//
// 返回:
//   - bool: 如果应该更新类型推断则返回true，否则返回false
func (n *RequestPathVariableNode) ShouldUpdateInference() bool {
	// 这里可以实现更复杂的逻辑，如检查样本数量是否达到阈值
	// 当前简单实现：每次都更新
	return true
}

// UpdateTypeInference 根据已观察的值更新类型推断
func (n *RequestPathVariableNode) UpdateTypeInference() {
	// 使用推断函数确定值类型
	if n.inferFunc != nil {
		inferredType, err := n.inferFunc(n)
		if err != nil {
			// 推断失败时保持当前类型
			return
		}

		// 更新推断的类型
		n.valueType = inferredType
	}
}

// GetValueType 获取推断出的值类型
//
// 返回:
//   - value.Type: 推断出的值类型
func (n *RequestPathVariableNode) GetValueType() value.Type {
	return n.valueType
}

// GetPositionIdentifier 获取变量在路径中的位置标识符
//
// 返回:
//   - string: 变量的位置标识符
func (n *RequestPathVariableNode) GetPositionIdentifier() string {
	return n.GetKey()
}

// GetValueMetric 获取值统计信息
//
// 返回:
//   - *value.ValueMetric: 值统计信息
func (n *RequestPathVariableNode) GetValueMetric() *value.ValueMetric {
	return n.valueMetric
}

// ExtractValue 从路径段中提取变量值并存储在上下文中
//
// 参数:
//   - pathSegment: URL路径的一个段
//
// 返回:
//   - bool: 如果成功提取并存储变量值则返回true，否则返回false
func (n *RequestPathVariableNode) ExtractValue(pathSegment string) bool {
	if n.IsMatch(pathSegment) {
		// 在上下文中存储变量值，键为位置标识符
		varKey := n.GetPositionIdentifier()
		context := n.GetContext()
		context.SetKey(varKey, pathSegment)
		return true
	}
	return false
}

// String 返回变量节点的字符串表示
//
// 返回:
//   - string: 变量节点的字符串表示，格式为 "{POSITION:TYPE}"
func (n *RequestPathVariableNode) String() string {
	position := n.GetPositionIdentifier()
	typeStr := string(n.GetValueType())

	return fmt.Sprintf("{%s:%s}", position, typeStr)
}

// IsDynamic 判断当前节点是否为动态节点
// 路径变量节点始终是动态的
func (n *RequestPathVariableNode) IsDynamic() bool {
	return true
}

// 确保 RequestPathVariableNode 实现了 Node 接口
var _ Node[NodeContext] = (*RequestPathVariableNode)(nil)
