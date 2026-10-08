package node

import (
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

func TestGapClone_TypedDeepCloneKeepsTypeAndCount(t *testing.T) {
	path := NewRequestPathNode("users")
	path.SetValue("v")
	path.IncrementRequestCount()
	if got := path.DeepClone(); got.GetType() != "request_path" || got.GetRequestCount() != 1 || got.GetValue() != "v" {
		t.Fatalf("路径节点克隆丢字段：type=%s count=%d value=%s", got.GetType(), got.GetRequestCount(), got.GetValue())
	}

	method := NewRequestMethodNode("GET")
	method.SetValue("http://a/b")
	method.SetRequestCount(4)
	if got := method.DeepClone(); got.GetType() != "request_method" || got.GetRequestCount() != 4 || got.GetValue() != "http://a/b" {
		t.Fatalf("方法节点克隆丢字段：%s %d %s", got.GetType(), got.GetRequestCount(), got.GetValue())
	}

	ct := NewRequestContentTypeNode("application/json")
	ct.SetValue("json")
	if got := ct.DeepClone(); got.GetType() != "request_content_type" || got.GetValue() != "json" {
		t.Fatalf("content-type 克隆丢字段：%s %s", got.GetType(), got.GetValue())
	}

	pv := NewRequestPathVariableNode("users_id", "[0-9]+")
	pv.SetValue("sample")
	pv.SetType(value.Type(value.PhysicalTypeInteger))
	pv.SetLogicalType(value.LogicalTypeString)
	pv.SetLastInferredUniqueCount(3)
	pv.SetRequestCount(2)
	cloned := pv.DeepClone().(*RequestPathVariableNode)
	if cloned.GetPattern() == nil || cloned.GetPattern().String() != "[0-9]+" {
		t.Fatal("路径变量克隆应保留模式")
	}
	if cloned.GetValueType() != value.Type(value.PhysicalTypeInteger) || cloned.GetLastInferredUniqueCount() != 3 || cloned.GetRequestCount() != 2 {
		t.Fatalf("路径变量克隆丢推断状态：type=%s last=%d count=%d", cloned.GetValueType(), cloned.GetLastInferredUniqueCount(), cloned.GetRequestCount())
	}

	// 无模式的变量节点。
	plain := NewRequestPathVariableNode("var_x", "")
	if got := plain.Clone().(*RequestPathVariableNode); got.GetPattern() != nil {
		t.Fatal("无模式变量克隆后不应凭空出现模式")
	}

	param := NewRequestParamNode("page", "1", true)
	param.SetValueType(value.Type(value.PhysicalTypeInteger))
	param.SetLogicalType(value.LogicalTypeString)
	param.SetMultiValue(true)
	param.SetPresenceCount(5)
	param.SetLastInferredUniqueCount(2)
	pc := param.DeepClone().(*RequestParamNode)
	if !pc.IsRequired() || !pc.IsMultiValue() || pc.GetPresenceCount() != 5 || pc.GetDefaultValue() != "1" {
		t.Fatalf("参数克隆丢状态：required=%v multi=%v presence=%d def=%s", pc.IsRequired(), pc.IsMultiValue(), pc.GetPresenceCount(), pc.GetDefaultValue())
	}
	if pc.GetValueType() != value.Type(value.PhysicalTypeInteger) || pc.GetLastInferredUniqueCount() != 2 {
		t.Fatal("参数克隆丢类型与推断计数")
	}

	header := NewRequestHeaderNode("Accept")
	hv := NewRequestHeaderValueNode("Accept", "application/json")
	header.AddChild(hv)
	hc := header.DeepClone()
	if hc.GetType() != "request_header" || hc.FindChildByKey("application/json").GetType() != "request_header_value" {
		t.Fatal("header 克隆应保留分组与值节点类型")
	}

	cookie := NewRequestCookieNode("lang")
	cv := NewRequestCookieValueNode("lang", "zh-CN")
	cookie.AddChild(cv)
	cc := cookie.DeepClone()
	if cc.GetType() != "request_cookie" || cc.FindChildByKey("zh-CN").GetType() != "request_cookie_value" {
		t.Fatal("cookie 克隆应保留分组与值节点类型")
	}
}

func TestGapClone_MergeWithAccumulatesCount(t *testing.T) {
	dst := NewRequestPathNode("users")
	dstMethod := NewRequestMethodNode("GET")
	dstMethod.SetRequestCount(2)
	dst.AddChild(dstMethod)

	other := NewRequestPathNode("users")
	otherMethod := NewRequestMethodNode("GET")
	otherMethod.SetRequestCount(3)
	otherMethod.SetValue("http://a/1")
	other.AddChild(otherMethod)

	if err := dst.MergeWith(other); err != nil {
		t.Fatal(err)
	}
	got := dst.FindChildByKey("GET")
	if got.GetRequestCount() != 5 {
		t.Fatalf("同键子节点计数应累加为 5，got %d", got.GetRequestCount())
	}
}

// rejectCloneNode 深克隆出的节点拒绝被设置父节点，用来触发 MergeWith 的挂接失败。
type rejectCloneNode struct {
	*BaseNode[NodeContext]
	reject bool
}

func (n *rejectCloneNode) SetParent(Node[NodeContext]) error {
	if n.reject {
		return ErrInvalidContext
	}
	return n.BaseNode.SetParent(nil)
}

func (n *rejectCloneNode) DeepClone() Node[NodeContext] {
	return &rejectCloneNode{
		BaseNode: NewBaseNode[NodeContext](n.GetType(), n.GetKey(), "", NewBaseNodeContext()),
		reject:   true,
	}
}

func (n *rejectCloneNode) MergeWith(Node[NodeContext]) error {
	return ErrDuplicateNode
}

func TestGapClone_MergeWithAddChildError(t *testing.T) {
	dst := NewRequestPathNode("users")
	other := NewRequestPathNode("other")
	bad := &rejectCloneNode{BaseNode: NewBaseNode[NodeContext]("request_path", "bad", "", NewBaseNodeContext())}
	other.AddChild(bad)
	if err := dst.MergeWith(other); err == nil {
		t.Fatal("子节点拒绝设置父节点时合并应报错")
	}
}

func TestGapClone_MergeWithChildError(t *testing.T) {
	dst := NewRequestPathNode("users")
	bad := &rejectCloneNode{BaseNode: NewBaseNode[NodeContext]("request_path", "loop", "", NewBaseNodeContext())}
	dst.AddChild(bad)
	other := NewRequestPathNode("users")
	other.AddChild(NewRequestPathNode("loop"))
	if err := dst.MergeWith(other); err == nil {
		t.Fatal("同键子节点合并失败应向上返回")
	}
}

func TestGapClone_SetRequestCount(t *testing.T) {
	n := NewRequestPathNode("a")
	n.SetRequestCount(9)
	if n.GetRequestCount() != 9 {
		t.Fatalf("SetRequestCount 未生效，got %d", n.GetRequestCount())
	}
}

func TestGapClone_DeepCloneManyChildrenParallel(t *testing.T) {
	root := NewRequestPathNode("root")
	root.SetRequestCount(1)
	for i := 0; i < 12; i++ {
		c := NewRequestPathNode(string(rune('a' + i)))
		c.SetRequestCount(int64(i + 1))
		root.AddChild(c)
	}
	cloned := root.DeepClone()
	if cloned.GetChildCount() != 12 || cloned.GetRequestCount() != 1 {
		t.Fatalf("并行深克隆丢子节点或计数：children=%d count=%d", cloned.GetChildCount(), cloned.GetRequestCount())
	}
	if cloned.FindChildByKey("a").GetRequestCount() != 1 {
		t.Fatal("并行深克隆应带回子节点计数")
	}
}
