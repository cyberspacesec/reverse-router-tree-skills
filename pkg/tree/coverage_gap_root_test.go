package tree

import "testing"

func TestGapRoot_ExportImport(t *testing.T) {
	var tr *Tree
	if tr.ExportRoot() != nil {
		t.Fatal("nil 树导出应为 nil")
	}
	tr.ImportRoot(&RouteNodeJSON{Type: "root", Key: "root"})

	tr = NewTree()
	tr.ImportRoot(nil)
	root := tr.ExportRoot()
	if root == nil || root.Type != "root" {
		t.Fatal("空树应导出 root")
	}
	again := NewTree()
	again.ImportRoot(root)
	if again.Root.GetType() != "root" {
		t.Fatal("导入后应为 root")
	}
}
