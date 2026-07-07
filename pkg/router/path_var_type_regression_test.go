package router

import (
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

// TestPathVarTypeNotPolluted_AfterMergeHit 验证路径变量节点在首次合并后，
// 被后续请求命中时物理类型不被逻辑类型串污染。
//
// 复现场景：3 个 uuid 先 GET 合并 → 变量节点物理=string。
// 然后 PUT 带 body 命中同一变量节点 → 修复前物理被污染成 "uuid"，
// 修复后应保持 string。
func TestPathVarTypeNotPolluted_AfterMergeHit(t *testing.T) {
	r := NewReverseRouter()
	uuids := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"550e8400-e29b-41d4-a716-446655440001",
		"550e8400-e29b-41d4-a716-446655440002",
	}
	// 先 GET 3 个 uuid，触发合并
	for _, u := range uuids {
		if err := r.ReverseHttpRequest(request.NewHttpRequest("/api/items/"+u, nil, "GET", nil)); err != nil {
			t.Fatalf("GET 失败: %v", err)
		}
	}
	// 合并后断言物理=string、逻辑=uuid
	items := r.Tree.Root.FindChildByKey("api").FindChildByKey("items")
	varNode := items.GetChildByType("request_path_variable")
	if varNode == nil {
		t.Fatal("期望合并出路径变量节点")
	}
	pv := varNode.(*node.RequestPathVariableNode)
	if got := pv.GetValueType(); got != value.Type(value.PhysicalTypeString) {
		t.Errorf("合并后物理类型期望 string 实际 %q", got)
	}
	if got := pv.GetLogicalType(); got != value.LogicalTypeUUID {
		t.Errorf("合并后逻辑类型期望 uuid 实际 %q", got)
	}

	// 后续 PUT 带 body 命中同一变量节点（关键：触发 ObserveValue 路径）
	h := request.Headers{}
	h.Set("Content-Type", "application/json")
	for _, u := range uuids {
		if err := r.ReverseHttpRequest(request.NewHttpRequest("/api/items/"+u, h, "PUT", []byte(`{"name":"a"}`))); err != nil {
			t.Fatalf("PUT 失败: %v", err)
		}
	}
	r.InferRequiredParams()

	// 修复后：物理仍应为 string，不应被污染成 "uuid"
	if got := pv.GetValueType(); got != value.Type(value.PhysicalTypeString) {
		t.Errorf("后续命中后物理类型期望仍为 string 实际 %q（被逻辑类型污染）", got)
	}
	if got := pv.GetLogicalType(); got != value.LogicalTypeUUID {
		t.Errorf("后续命中后逻辑类型期望仍为 uuid 实际 %q", got)
	}
}

// TestPathVarTypeNotPolluted_PhoneAfterMergeHit phone 物理类型为 integer，
// 后续命中后应保持 integer（不被 "phone" 污染）
func TestPathVarTypeNotPolluted_PhoneAfterMergeHit(t *testing.T) {
	r := NewReverseRouter()
	phones := []string{"13812345678", "13912345678", "15012345678"}
	for _, p := range phones {
		if err := r.ReverseHttpRequest(request.NewHttpRequest("/api/users/"+p, nil, "GET", nil)); err != nil {
			t.Fatalf("GET 失败: %v", err)
		}
	}
	// 后续 DELETE 命中
	for _, p := range phones {
		if err := r.ReverseHttpRequest(request.NewHttpRequest("/api/users/"+p, nil, "DELETE", nil)); err != nil {
			t.Fatalf("DELETE 失败: %v", err)
		}
	}
	r.InferRequiredParams()
	users := r.Tree.Root.FindChildByKey("api").FindChildByKey("users")
	pv := users.GetChildByType("request_path_variable").(*node.RequestPathVariableNode)
	if got := pv.GetValueType(); got != value.Type(value.PhysicalTypeInteger) {
		t.Errorf("phone 后续命中后物理类型期望 integer 实际 %q", got)
	}
	if got := pv.GetLogicalType(); got != value.LogicalTypePhoneNumber {
		t.Errorf("phone 后续命中后逻辑类型期望 phone 实际 %q", got)
	}
}
