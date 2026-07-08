package node

import (
	"fmt"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

// RequestCookieNode 定义了Cookie路由节点
// 某些Web服务会根据Cookie做路由决策，例如：
//   - lang: zh-CN vs en-US → 多语言路由
//   - session: 不同session对应不同用户视图
//   - theme: dark vs light → 不同主题路由
//   - region: us vs eu → 区域路由
//
// 树结构设计（两层）：
//   - 第一层：Cookie名称节点（key=cookieName，如 "lang"）
//   - 第二层：Cookie值节点（key=cookieValue，如 "zh-CN"）
//
// 这样同一个Cookie的不同值会作为兄弟节点挂在Cookie名称节点下，
// 便于后续做变量合并（如多个不同的lang值可能合并为变量）。
type RequestCookieNode struct {
	*BaseNode[NodeContext]
	cookieName string // cookie 名称（如 "lang", "theme"）
}

// NewRequestCookieNode 创建一个新的Cookie路由分组节点
// key 使用 cookieName（如 "lang"），用于在方法节点下查找
func NewRequestCookieNode(cookieName string) *RequestCookieNode {
	context := NewBaseNodeContext()
	baseNode := NewBaseNode[NodeContext]("request_cookie", cookieName, cookieName, context)

	return &RequestCookieNode{
		BaseNode:    baseNode,
		cookieName:  cookieName,
	}
}

// GetCookieName 获取Cookie名称
func (n *RequestCookieNode) GetCookieName() string {
	return n.cookieName
}

// FindOrCreateValueNode 查找或创建Cookie值子节点
func (n *RequestCookieNode) FindOrCreateValueNode(cookieValue string) *RequestCookieValueNode {
	// 查找已有的值节点
	child := n.FindChildByKey(cookieValue)
	if child != nil && child.GetType() == "request_cookie_value" {
		valueNode := child.(*RequestCookieValueNode)
		valueNode.ObserveValue(cookieValue)
		return valueNode
	}

	// 创建新的值节点
	newValueNode := NewRequestCookieValueNode(n.cookieName, cookieValue)
	if err := n.AddChild(newValueNode); err == nil {
		return newValueNode
	}
	return nil
}

// String 返回节点的字符串表示
func (n *RequestCookieNode) String() string {
	return fmt.Sprintf("%s [Cookie]", n.cookieName)
}

// 确保 RequestCookieNode 实现了 Node 接口
var _ Node[NodeContext] = (*RequestCookieNode)(nil)

// RequestCookieValueNode Cookie值节点
// 作为 RequestCookieNode 的子节点，存储具体的Cookie值
type RequestCookieValueNode struct {
	*BaseNode[NodeContext]
	cookieName  string             // 所属的cookie名称
	cookieValue string             // cookie值
	valueMetric *value.ValueMetric // 观察到的 cookie 值统计
}

// NewRequestCookieValueNode 创建一个新的Cookie值节点
func NewRequestCookieValueNode(cookieName, cookieValue string) *RequestCookieValueNode {
	context := NewBaseNodeContext()
	baseNode := NewBaseNode[NodeContext]("request_cookie_value", cookieValue, cookieName, context)

	return &RequestCookieValueNode{
		BaseNode:    baseNode,
		cookieName:  cookieName,
		cookieValue: cookieValue,
		valueMetric: value.NewValueMetric(),
	}
}

// GetCookieName 获取所属Cookie名称
func (n *RequestCookieValueNode) GetCookieName() string {
	return n.cookieName
}

// GetCookieValue 获取Cookie值
func (n *RequestCookieValueNode) GetCookieValue() string {
	return n.cookieValue
}

// IsMatch 判断给定的cookie值是否匹配
func (n *RequestCookieValueNode) IsMatch(cookieValue string) bool {
	return n.cookieValue == cookieValue
}

// ObserveValue 记录观察到的cookie值
func (n *RequestCookieValueNode) ObserveValue(val string) {
	n.valueMetric.AddValue(val)
}

// GetValueMetric 获取值统计信息
func (n *RequestCookieValueNode) GetValueMetric() *value.ValueMetric {
	return n.valueMetric
}

// String 返回节点的字符串表示
func (n *RequestCookieValueNode) String() string {
	return fmt.Sprintf("%s=%s [CookieValue]", n.cookieName, n.cookieValue)
}

// 确保 RequestCookieValueNode 实现了 Node 接口
var _ Node[NodeContext] = (*RequestCookieValueNode)(nil)
