package router

import (
	"strings"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

// TestCurlE2E_BasicGet 验证 curl GET 命令端到端还原路由树。
// 完整 URL（含 host）经 net/url 解析后 host 不进路径，仅 /users/123 还原。
func TestCurlE2E_BasicGet(t *testing.T) {
	r := newSilentRouter()
	req, err := request.ParseCurl(`curl 'http://api.example.com/users/123'`)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.ReverseHttpRequest(req); err != nil {
		t.Fatal(err)
	}
	// host 不应进入路径树，只应有 users 段
	if r.Tree.Root.FindChildByKey("users") == nil {
		t.Errorf("应还原出 users 路径段，树:\n%s", r.Tree.String())
	}
	// api.example.com 不应作为路径段（它是 host）
	if r.Tree.Root.FindChildByKey("api.example.com") != nil {
		t.Error("host 不应作为路径段进入路由树")
	}
}

// TestCurlE2E_PostWithBody 验证 curl POST + JSON body 端到端参数解析。
func TestCurlE2E_PostWithBody(t *testing.T) {
	r := newSilentRouter()
	curl := `curl 'http://api.example.com/users' -X POST -H 'Content-Type: application/json' -d '{"name":"bob","age":25}'`
	req, err := request.ParseCurl(curl)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.ReverseHttpRequest(req); err != nil {
		t.Fatal(err)
	}
	s := r.GetStats()
	if s.BodyParamsParsed != 2 {
		t.Errorf("应解析 2 个 body 参数（name, age），实际 %d", s.BodyParamsParsed)
	}
	// 验证 age 参数被识别为 integer
	getNode := r.Tree.Root.FindChildByKey("users").FindChildByKey("POST")
	if getNode == nil {
		t.Fatal("应存在 POST 方法节点")
	}
	ageNode := getNode.FindChildByKey("age")
	if ageNode == nil {
		t.Fatal("应存在 age 参数节点")
	}
	if p, ok := ageNode.(*node.RequestParamNode); ok {
		if p.GetValueType() != value.Type(value.PhysicalTypeInteger) {
			t.Errorf("age 应为 integer，实际 %s", p.GetValueType())
		}
	}
}

// TestCurlE2E_MultipleRequestsMerge 验证多条 curl 命令触发路径变量合并。
func TestCurlE2E_MultipleRequestsMerge(t *testing.T) {
	r := newSilentRouter()
	curls := []string{
		`curl 'http://api.example.com/users/101'`,
		`curl 'http://api.example.com/users/102'`,
		`curl 'http://api.example.com/users/103'`,
	}
	for _, c := range curls {
		req, err := request.ParseCurl(c)
		if err != nil {
			t.Fatal(err)
		}
		r.ReverseHttpRequest(req)
	}
	s := r.GetStats()
	if s.PathVariablesIdentified != 1 {
		t.Errorf("应识别 1 个路径变量（合并 101/102/103），实际 %d", s.PathVariablesIdentified)
	}
}

// TestCurlE2E_BearerAuth 验证 curl Authorization 头被规范化（Bearer xxx → Bearer）。
func TestCurlE2E_BearerAuth(t *testing.T) {
	r := newSilentRouter()
	curl := `curl 'http://api.example.com/users/123' -H 'Authorization: Bearer eyJtoken123'`
	req, err := request.ParseCurl(curl)
	if err != nil {
		t.Fatal(err)
	}
	r.ReverseHttpRequest(req)
	treeStr := r.Tree.String()
	// Bearer 应被规范化保留（具体 token 值不进树），Authorization header 节点应存在
	if !strings.Contains(treeStr, "Authorization") && !strings.Contains(treeStr, "authorization") {
		t.Errorf("应还原出 Authorization header 节点，树:\n%s", treeStr)
	}
}

// TestCurlE2E_LineContinuation 验证多行 curl（反斜杠续行）端到端还原。
func TestCurlE2E_LineContinuation(t *testing.T) {
	r := newSilentRouter()
	curl := `curl 'http://api.example.com/orders/555' \
  -H 'Authorization: Bearer token' \
  -H 'Accept: application/json'`
	req, err := request.ParseCurl(curl)
	if err != nil {
		t.Fatal(err)
	}
	r.ReverseHttpRequest(req)
	if r.Tree.Root.FindChildByKey("orders") == nil {
		t.Errorf("应还原出 orders 路径段，树:\n%s", r.Tree.String())
	}
	s := r.GetStats()
	if s.RequestsProcessed != 1 {
		t.Errorf("应处理 1 个请求，实际 %d", s.RequestsProcessed)
	}
}
