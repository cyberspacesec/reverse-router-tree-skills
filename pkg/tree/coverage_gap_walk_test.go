package tree

import (
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
)

func TestGapWalk_EmptySegment(t *testing.T) {
	root := node.NewBaseNode[node.NodeContext]("root", "root", "", node.NewBaseNodeContext())
	got, err := walkSegments(root, []string{"", "api", ""}, true)
	if err != nil || got.GetKey() != "api" {
		t.Fatalf("空段应跳过，got=%v err=%v", got, err)
	}
	found, err := walkSegments(root, []string{"", "missing"}, false)
	if err != nil || found != nil {
		t.Fatalf("缺失段应返回 nil，found=%v err=%v", found, err)
	}
	again, err := walkSegments(root, []string{"api", "users"}, true)
	if err != nil || again.GetKey() != "users" {
		t.Fatalf("应沿已有 api 继续建 users，got=%v err=%v", again, err)
	}
	// 查找命中已建路径。
	hit, _ := walkSegments(root, []string{"api", "users"}, false)
	if hit == nil || hit.GetKey() != "users" {
		t.Fatal("已建路径应能查到")
	}
}
