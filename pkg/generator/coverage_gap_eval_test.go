package generator

import (
	"strings"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
)

func TestGapEval_ExpectedTemplatesEdges(t *testing.T) {
	var s *Spec
	if s.ExpectedTemplates() != nil {
		t.Fatal("nil spec 期望应为空")
	}
	report, err := Evaluate(nil, nil)
	if err != nil || !report.Passed() {
		t.Fatalf("nil spec 应视为通过，err=%v report=%s", err, report)
	}
	if !strings.Contains(report.String(), "recall=") {
		t.Fatal("摘要应含 recall")
	}

	spec := &Spec{Resources: []*Resource{
		nil,
		{Name: "users", Operations: []*Operation{
			nil,
			{Method: "", PathVar: nil},
			{Method: "GET", PathVar: &PathVarSpec{Values: nil}},
			{Method: "GET", PathVar: &PathVarSpec{ExpectMerge: true, Values: []string{"1", "2"}}},
			{Method: "POST", PathVar: &PathVarSpec{ExpectMerge: true, ExpectVarName: "users_id", Values: []string{"1"}}},
			{Method: "PUT", PathVar: &PathVarSpec{Values: []string{"a", "a"}}},
		}},
	}}
	got := spec.ExpectedTemplates()
	want := []string{"GET /users", "GET /users/{var}", "POST /users/{users_id}", "PUT /users/a"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("模板不符：\n got %v\nwant %v", got, want)
	}
}

func TestGapEval_DiffAndRatio(t *testing.T) {
	missing, extra := diffKeys([]string{"a", "b"}, []string{"b", "c"})
	if strings.Join(missing, ",") != "a" || strings.Join(extra, ",") != "c" {
		t.Fatalf("diff 不符 missing=%v extra=%v", missing, extra)
	}
	if ratio(0, 0) != 1 {
		t.Fatal("空对应空应为 1")
	}
	if ratio(1, 0) != 0 {
		t.Fatal("有命中但总数为 0 应为 0")
	}

	// 喂入会解析失败的请求，Evaluate 应返回错误。
	spec := &Spec{Resources: []*Resource{{Name: "x", Operations: []*Operation{{Method: "GET"}}}}}
	spec.Resources[0].Operations[0].Repeat = 0
	// Requests 为空时两边都空，比率应为 1。
	report, err := Evaluate(spec, nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.Recall != 1 || report.Precision != 1 {
		t.Fatalf("空操作应全中，%s", report)
	}
}

func TestGapEval_RequestError(t *testing.T) {
	spec := &Spec{Resources: []*Resource{{
		Name: "users",
		Operations: []*Operation{{
			Method: "GET", Repeat: 1,
			PathVar: &PathVarSpec{Values: []string{"%zz"}},
		}},
	}}}
	if _, err := Evaluate(spec, nil); err == nil {
		t.Fatal("非法转义应让 Evaluate 返回错误")
	}
}

func TestGapCookie_MissingValueLogs(t *testing.T) {
	root := node.NewBaseNode[node.NodeContext]("root", "root", "", node.NewBaseNodeContext())
	api := node.NewRequestPathNode("api")
	root.AddChild(api)
	method := node.NewRequestMethodNode("GET")
	api.AddChild(method)
	method.AddChild(node.NewRequestCookieNode("lang"))
	a := &CookieAssertion{PathSegments: []string{"api"}, Method: "GET", CookieName: "lang", ExpectValues: []string{"zh-CN"}}
	a.Check(t, root)
}
