package router

import (
	"strings"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/inference"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
)

func TestGapSnapshot_NilAndBadInput(t *testing.T) {
	var rs *RouterSet
	data, err := rs.ExportJSON()
	if err != nil || !strings.Contains(string(data), `"hosts"`) {
		t.Fatalf("nil RouterSet 应导出空快照，err=%v data=%s", err, data)
	}
	if err := rs.ImportJSON([]byte(`{}`)); err == nil {
		t.Fatal("nil RouterSet 导入应报错")
	}

	var m *ProjectManager
	data, err = m.ExportJSON()
	if err != nil || !strings.Contains(string(data), `"projects"`) {
		t.Fatalf("nil ProjectManager 应导出空快照，err=%v data=%s", err, data)
	}
	if err := m.ImportJSON([]byte(`{}`)); err == nil {
		t.Fatal("nil ProjectManager 导入应报错")
	}

	live := NewRouterSet()
	if err := live.ImportJSON([]byte(`{`)); err == nil {
		t.Fatal("坏 JSON 应报错")
	}
	pm := NewProjectManager()
	if err := pm.ImportJSON([]byte(`{`)); err == nil {
		t.Fatal("项目坏 JSON 应报错")
	}
	if err := pm.ImportJSON([]byte(`{"version":99}`)); err == nil {
		t.Fatal("项目更高版本应报错")
	}
	if _, ok := pm.NormalizeURLString("p", "http://a.com/x", "GET"); ok {
		t.Fatal("失败导入不应建出数据")
	}
}

func TestGapSnapshot_SkipsEmptyAndRejectsTreeVersion(t *testing.T) {
	rs := NewRouterSet()
	rs.routers["empty"] = &ReverseRouter{}
	rs.routers["blank"] = nil
	data, err := rs.ExportJSON()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "empty") {
		t.Fatal("没有树的 host 不应导出")
	}

	bad := []byte(`{"version":1,"hosts":{"a.com":null,"b.com":{"version":1},"c.com":{"version":9,"tree":{"type":"root","key":"root"}}}}`)
	if err := NewRouterSet().ImportJSON(bad); err == nil {
		t.Fatal("单棵树版本过高应报错")
	}
}

func TestGapSnapshot_ConfiguredRouterInherits(t *testing.T) {
	rs := NewRouterSet()
	rs.SetInferenceRule(inference.NewChainTypeInferenceRule())
	rs.SetMergeRule(&stubMergeRule{action: MergeActionSkip})
	rs.SetLogger(NewRouterLoggerWithWriter(&strings.Builder{}))
	rs.SetLogLevel(LogLevelError)
	r := rs.newConfiguredRouter()
	if r.inferenceRule == nil || r.mergeRule == nil || r.logger == nil {
		t.Fatal("新路由器应继承推断规则、合并规则和日志器")
	}

	m := NewProjectManager()
	m.SetInferenceRule(inference.NewChainTypeInferenceRule())
	m.SetMergeRule(&stubMergeRule{action: MergeActionSkip})
	m.SetLogger(NewRouterLoggerWithWriter(&strings.Builder{}))
	m.SetLogLevel(LogLevelWarn)
	set := m.newConfiguredSet()
	if set.rule == nil || set.mergeRule == nil || !set.loggerConfigured || set.logLevel == nil {
		t.Fatal("新集合应继承管理器的规则与日志配置")
	}
}

func TestGapSnapshot_ProjectImportTreeVersionError(t *testing.T) {
	m := NewProjectManager()
	raw := []byte(`{"version":1,"projects":{"p":{"h":{"version":9,"tree":{"type":"root","key":"root"}}}}}`)
	if err := m.ImportJSON(raw); err == nil {
		t.Fatal("项目内路由树版本过高应报错")
	}
}

func TestGapAbsorbAndSignature(t *testing.T) {
	absorbRequestCount(nil, nil)
	dst := node.NewRequestMethodNode("GET")
	other := node.NewRequestMethodNode("GET")
	absorbRequestCount(dst, other)
	if dst.GetRequestCount() != 0 {
		t.Fatal("零计数不应改写")
	}
	other.SetRequestCount(3)
	other.SetValue("http://a/1")
	absorbRequestCount(dst, other)
	if dst.GetRequestCount() != 3 || dst.GetValue() != "http://a/1" {
		t.Fatalf("应吸收计数与样本，count=%d value=%s", dst.GetRequestCount(), dst.GetValue())
	}
	// 已有样本不被覆盖。
	other.SetValue("http://b/2")
	absorbRequestCount(dst, other)
	if dst.GetValue() != "http://a/1" {
		t.Fatal("已有样本 URL 不应被覆盖")
	}

	empty := NormalizedRoute{Method: "GET", Template: "/api"}
	if empty.SignatureKey() != empty.AssetKey() {
		t.Fatal("无参数时签名应等于资产键")
	}
	with := NormalizedRoute{Method: "GET", Template: "/api", QueryParams: []string{"page", "size"}}
	if with.SignatureKey() != "GET /api?page&size" {
		t.Fatalf("签名不符：%s", with.SignatureKey())
	}
}

func TestGapNormalize_VariableNodeWrongConcreteType(t *testing.T) {
	r := NewReverseRouter()
	// 手工塞一个类型名是路径变量、实际却是 BaseNode 的节点，覆盖类型断言失败后仍折叠的分支。
	fake := node.NewBaseNode[node.NodeContext]("request_path_variable", "users_id", "", node.NewBaseNodeContext())
	api := node.NewRequestPathNode("api")
	r.Tree.Root.AddChild(api)
	api.AddChild(fake)
	paths := []*request.HttpRequestPath{request.NewHttpRequestPath("api"), request.NewHttpRequestPath("123")}
	_, segs, params, ok := r.normalizePathSegments(paths)
	if !ok || len(segs) != 2 || segs[1] != "{users_id}" || len(params) != 1 || params[0] != "users_id" {
		t.Fatalf("非具体类型的变量节点仍应折叠，ok=%v segs=%v params=%v", ok, segs, params)
	}
}

func TestGapNormalize_VariablePatternMismatch(t *testing.T) {
	r := NewReverseRouter()
	api := node.NewRequestPathNode("api")
	r.Tree.Root.AddChild(api)
	variable := node.NewRequestPathVariableNode("users_id", "[0-9]+")
	api.AddChild(variable)
	paths := []*request.HttpRequestPath{request.NewHttpRequestPath("api"), request.NewHttpRequestPath("profile")}
	if _, _, _, ok := r.normalizePathSegments(paths); ok {
		t.Fatal("字母段不应匹配数字模式的变量")
	}
}
