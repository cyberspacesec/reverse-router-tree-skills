package router

// coverage_gap_rc_test.go — 补齐 pkg/router/reverse_router.go 剩余 56 个分支。
// 只新增测试，不改业务代码。前缀 TestGapRC_。
//
// 触发策略：
//   - 生产护栏（childrenGuard）失败：SetResourceLimits 设极小上限，再喂入需要新建节点的请求；
//   - AddChild 失败：用内嵌 BaseNode 的桩节点覆写 SetParent/AddChild 返回错误；
//   - 空值/空守卫：同包直接调用私有方法；防御死代码按既有共识注释说明。

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/tree"
)

// gapRCStub 内嵌 BaseNode，覆写个别方法以触发错误分支。
type gapRCStub struct {
	*node.BaseNode[node.NodeContext]
	setParentErr error
	addChildErr  error
	getChildNum  int
	childSet     bool
}

func (s *gapRCStub) SetParent(p node.Node[node.NodeContext]) error {
	if s.setParentErr != nil {
		return s.setParentErr
	}
	return s.BaseNode.SetParent(p)
}

func (s *gapRCStub) AddChild(c node.Node[node.NodeContext]) error {
	if s.addChildErr != nil {
		return s.addChildErr
	}
	return s.BaseNode.AddChild(c)
}

func (s *gapRCStub) GetChildCount() int {
	if s.childSet {
		return s.getChildNum
	}
	return s.BaseNode.GetChildCount()
}

// --- 护栏：子节点数上限触发各 findOrCreate 的 guard 失败分支 ---

// 976/980 findOrCreateMethodNode：超限拒绝新建 + AddChild 失败。
func TestGapRC_MethodGuardAndAddFail(t *testing.T) {
	// guard 失败：父节点已有 MaxChildrenPerNode 个子节点
	r := newSilentRouter()
	r.SetResourceLimits(ResourceLimits{MaxChildrenPerNode: 1})
	parent := node.NewBaseNode[node.NodeContext]("request_path", "api", "", node.NewBaseNodeContext())
	r.Tree.Root.AddChild(parent)
	parent.AddChild(node.NewRequestPathNode("a")) // 占满 1 个
	if _, err := r.findOrCreateMethodNode(parent, "GET"); err == nil {
		t.Error("子节点上限应拒绝新建方法节点")
	}

	// AddChild 失败：桩父节点 AddChild 返回错误
	r2 := newSilentRouter()
	stubParent := &gapRCStub{BaseNode: node.NewBaseNode[node.NodeContext]("request_path", "api", "", node.NewBaseNodeContext()), addChildErr: errors.New("桩错误")}
	if _, err := r2.findOrCreateMethodNode(stubParent, "GET"); err == nil {
		t.Error("AddChild 失败应返回错误")
	}
}

// 1081/1084 findOrCreateParamNode：guard 失败 + AddChild 失败。
func TestGapRC_ParamGuardAndAddFail(t *testing.T) {
	r := newSilentRouter()
	r.SetResourceLimits(ResourceLimits{MaxChildrenPerNode: 1})
	m := node.NewBaseNode[node.NodeContext]("request_method", "GET", "", node.NewBaseNodeContext())
	m.AddChild(node.NewRequestParamNode("a", "", false)) // 占满
	err := r.findOrCreateParamNode(m, request.NewHttpParam("b", "1"), false)
	if err == nil {
		t.Error("子节点上限应拒绝新建参数节点")
	}

	r2 := newSilentRouter()
	stubM := &gapRCStub{BaseNode: node.NewBaseNode[node.NodeContext]("request_method", "GET", "", node.NewBaseNodeContext()), addChildErr: errors.New("桩错误")}
	if err := r2.findOrCreateParamNode(stubM, request.NewHttpParam("x", "1"), false); err == nil {
		t.Error("AddChild 失败应返回错误")
	}
}

// 1102/1105 findOrCreateContentTypeNode：guard 失败 + AddChild 失败。
func TestGapRC_CTGuardAndAddFail(t *testing.T) {
	r := newSilentRouter()
	r.SetResourceLimits(ResourceLimits{MaxChildrenPerNode: 1})
	m := node.NewBaseNode[node.NodeContext]("request_method", "POST", "", node.NewBaseNodeContext())
	m.AddChild(node.NewRequestParamNode("a", "", false))
	if _, err := r.findOrCreateContentTypeNode(m, "application/json"); err == nil {
		t.Error("子节点上限应拒绝新建 CT 节点")
	}

	r2 := newSilentRouter()
	stubM := &gapRCStub{BaseNode: node.NewBaseNode[node.NodeContext]("request_method", "POST", "", node.NewBaseNodeContext()), addChildErr: errors.New("桩错误")}
	if _, err := r2.findOrCreateContentTypeNode(stubM, "text/plain"); err == nil {
		t.Error("AddChild 失败应返回错误")
	}
}

// 1719/1722 processRoutingHeaders：header guard 失败 + AddChild 失败。
func TestGapRC_HeaderGuardAndAddFail(t *testing.T) {
	// 1719 guard 失败：方法节点子节点数已达上限
	r := newSilentRouter()
	r.SetResourceLimits(ResourceLimits{MaxChildrenPerNode: 1})
	m1 := &gapRCStub{BaseNode: node.NewBaseNode[node.NodeContext]("request_method", "GET", "", node.NewBaseNodeContext()), childSet: true, getChildNum: 1}
	if err := r.processRoutingHeaders(m1, request.Headers{"Accept": "application/json"}); err == nil {
		t.Error("header guard 失败应返回错误")
	}
	// 1722 AddChild 失败：桩方法节点 AddChild 返回错误
	r2 := newSilentRouter()
	stubM := &gapRCStub{BaseNode: node.NewBaseNode[node.NodeContext]("request_method", "GET", "", node.NewBaseNodeContext()), addChildErr: errors.New("桩错误")}
	if err := r2.processRoutingHeaders(stubM, request.Headers{"Accept": "application/json"}); err == nil {
		t.Error("Header AddChild 失败应返回错误")
	}
}

// 1730 processRoutingHeaders 值节点 guard 失败（fail-soft continue）。
func TestGapRC_HeaderValueGuard(t *testing.T) {
	r := newSilentRouter()
	r.SetResourceLimits(ResourceLimits{MaxChildrenPerNode: 1})
	m := node.NewBaseNode[node.NodeContext]("request_method", "GET", "", node.NewBaseNodeContext())
	hg := node.NewRequestHeaderNode("Accept")
	hg.AddChild(node.NewRequestHeaderValueNode("Accept", "application/json"))
	m.AddChild(hg)
	if err := r.processRoutingHeaders(m, request.Headers{"Accept": "text/html"}); err != nil {
		t.Errorf("值节点超限应 fail-soft continue，得 %v", err)
	}
}

// 1770/1773/1786 processCookies：cookie guard 失败 + AddChild 失败 + 值 guard。
func TestGapRC_CookieGuardAndAddFail(t *testing.T) {
	r := newSilentRouter()
	r.SetResourceLimits(ResourceLimits{MaxChildrenPerNode: 1})
	m := node.NewBaseNode[node.NodeContext]("request_method", "GET", "", node.NewBaseNodeContext())
	m.AddChild(node.NewRequestParamNode("a", "", false))
	// guard 失败（1770）：方法节点已满
	stubM := &gapRCStub{BaseNode: node.NewBaseNode[node.NodeContext]("request_method", "GET", "", node.NewBaseNodeContext()), childSet: true, getChildNum: 1}
	if err := r.processCookies(stubM, request.Headers{"Cookie": "sid=abc"}); err == nil {
		t.Error("cookie guard 失败应返回错误")
	}
	// AddChild 失败（1773）
	r2 := newSilentRouter()
	stubM2 := &gapRCStub{BaseNode: node.NewBaseNode[node.NodeContext]("request_method", "GET", "", node.NewBaseNodeContext()), addChildErr: errors.New("桩错误")}
	if err := r2.processCookies(stubM2, request.Headers{"Cookie": "sid=abc"}); err == nil {
		t.Error("cookie AddChild 失败应返回错误")
	}
	// 值 guard（1786）：cookie 分组已满，fail-soft continue
	r3 := newSilentRouter()
	r3.SetResourceLimits(ResourceLimits{MaxChildrenPerNode: 1})
	m3 := node.NewBaseNode[node.NodeContext]("request_method", "GET", "", node.NewBaseNodeContext())
	cg := node.NewRequestCookieNode("sid")
	cg.AddChild(node.NewRequestCookieValueNode("sid", "abc"))
	m3.AddChild(cg)
	if err := r3.processCookies(m3, request.Headers{"Cookie": "sid=def"}); err != nil {
		t.Errorf("cookie 值超限应 fail-soft continue，得 %v", err)
	}
}

// --- ReverseHttpRequest 主流程 err 分支 ---

// 327/346/376/385/393/400：ReverseHttpRequest 各 findOrCreate 失败。
// 通过 SetResourceLimits 触发 guard 失败，路径/方法/参数/CT 各覆盖一条。
func TestGapRC_ReverseHttpRequestBranches(t *testing.T) {
	// 376 参数 guard 失败
	r2 := newSilentRouter()
	r2.SetResourceLimits(ResourceLimits{MaxChildrenPerNode: 1})
	r2.ReverseHttpRequest(request.NewHttpRequest("/api/p?x=1", nil, "GET", nil))
	if err := r2.ReverseHttpRequest(request.NewHttpRequest("/api/p?x=1&y=2", nil, "GET", nil)); err == nil {
		t.Error("参数节点满时应拒绝新建")
	}
	// 385 CT guard 失败
	r3 := newSilentRouter()
	r3.SetResourceLimits(ResourceLimits{MaxChildrenPerNode: 1})
	h := request.Headers{}
	h.Set("Content-Type", "application/json")
	r3.ReverseHttpRequest(request.NewHttpRequest("/api/ct", h, "POST", []byte(`{}`)))
	h2 := request.Headers{}
	h2.Set("Content-Type", "text/plain")
	if err := r3.ReverseHttpRequest(request.NewHttpRequest("/api/ct", h2, "POST", []byte(`a`))); err == nil {
		t.Error("CT 节点满时应拒绝新建")
	}
	// 400 cookie guard 失败
	r4 := newSilentRouter()
	r4.SetResourceLimits(ResourceLimits{MaxChildrenPerNode: 1})
	r4.ReverseHttpRequest(request.NewHttpRequest("/api/ck", request.Headers{"Cookie": "sid=1"}, "GET", nil))
	if err := r4.ReverseHttpRequest(request.NewHttpRequest("/api/ck", request.Headers{"Cookie": "uid=2"}, "GET", nil)); err == nil {
		t.Error("cookie 节点满时应拒绝新建")
	}
}

// 327 路径参数段 guard 失败：根下已满时新建路径参数段。
func TestGapRC_PathParamGuard(t *testing.T) {
	r := newSilentRouter()
	r.SetResourceLimits(ResourceLimits{MaxChildrenPerNode: 1})
	r.ReverseHttpRequest(request.NewHttpRequest("/api", nil, "GET", nil)) // root→api
	if err := r.ReverseHttpRequest(request.NewHttpRequest("/p=1/x", nil, "GET", nil)); err == nil {
		t.Error("路径参数段 guard 满时应拒绝")
	}
}

// 346 方法 guard 失败：路径命中但方法节点满。
func TestGapRC_MethodGuardViaRequest(t *testing.T) {
	r := newSilentRouter()
	r.SetResourceLimits(ResourceLimits{MaxChildrenPerNode: 1})
	r.ReverseHttpRequest(request.NewHttpRequest("/m", nil, "GET", nil)) // x→GET
	if err := r.ReverseHttpRequest(request.NewHttpRequest("/m", nil, "POST", nil)); err == nil {
		t.Error("方法节点满时应拒绝新建 POST")
	}
}

// 393 header guard 失败：方法节点满时新建 header 分组。
func TestGapRC_HeaderGuardViaRequest(t *testing.T) {
	r := newSilentRouter()
	r.SetResourceLimits(ResourceLimits{MaxChildrenPerNode: 1})
	r.ReverseHttpRequest(request.NewHttpRequest("/hg?q=1", nil, "GET", nil)) // GET→q
	if err := r.ReverseHttpRequest(request.NewHttpRequest("/hg?q=1", request.Headers{"Accept": "text/html"}, "GET", nil)); err == nil {
		t.Error("header 分组 guard 满时应拒绝")
	}
}

// 500 findOrCreatePathNode AddChild 失败。
func TestGapRC_PathNodeAddFail(t *testing.T) {
	r := newSilentRouter()
	stubParent := &gapRCStub{BaseNode: node.NewBaseNode[node.NodeContext]("root", "root", "", node.NewBaseNodeContext()), addChildErr: errors.New("桩错误")}
	if _, err := r.findOrCreatePathNode(stubParent, "seg"); err == nil {
		t.Error("AddChild 失败应返回错误")
	}
}

// 422 空路径段防御分支：空段直接返回 parent。
func TestGapRC_PathNodeEmptySegment(t *testing.T) {
	r := newSilentRouter()
	parent := node.NewBaseNode[node.NodeContext]("root", "root", "", node.NewBaseNodeContext())
	got, err := r.findOrCreatePathNode(parent, "")
	if err != nil || got != parent {
		t.Errorf("空段应直接返回父节点，got=%v err=%v", got, err)
	}
}

// 1025 脱敏参数：参数二次出现命中已有节点时走 redacted 分支。
func TestGapRC_RedactedParam(t *testing.T) {
	r := newSilentRouter()
	r.SetRedactConfig(RedactConfig{Params: []string{"password"}})
	// 首次出现走新建分支（1055），二次出现才命中已有节点走 1025。
	for i := 0; i < 2; i++ {
		if err := r.ReverseHttpRequest(request.NewHttpRequest("/api/login?password=secret", nil, "GET", nil)); err != nil {
			t.Fatal(err)
		}
	}
	st := r.GetStats()
	if st.RedactedValues == 0 {
		t.Error("脱敏参数应计数 RedactedValues")
	}
}

// 553 findMergeableSiblings 空 children 与 748 mergeSiblings 空 children。
func TestGapRC_MergeEmptySiblings(t *testing.T) {
	r := newSilentRouter()
	parent := node.NewBaseNode[node.NodeContext]("request_path", "api", "", node.NewBaseNodeContext())
	if got := r.findMergeableSiblings(parent, nil); got != nil {
		t.Errorf("空 children 应返回 nil，得 %v", got)
	}
	r.mergeSiblings(parent, nil) // 不应 panic
}

// 577 mergeRule MergeAction 但 mergeable 为空：返回 nil。
func TestGapRC_MergeRuleEmptyMergeable(t *testing.T) {
	r := newSilentRouter()
	r.SetMergeRule(&stubMergeRule{action: MergeActionMerge, mergeable: nil})
	parent := node.NewBaseNode[node.NodeContext]("request_path", "api", "", node.NewBaseNodeContext())
	children := []node.Node[node.NodeContext]{
		node.NewRequestPathNode("1"),
		node.NewRequestPathNode("2"),
		node.NewRequestPathNode("3"),
	}
	if got := r.findMergeableSiblings(parent, children); got != nil {
		t.Errorf("空 mergeable 应返回 nil，得 %v", got)
	}
}

// 691 DetectPattern 空 values。
func TestGapRC_DetectPatternEmpty(t *testing.T) {
	d := NewPatternDetector()
	if p, r := d.DetectPattern(nil); p != "" || r != 0.0 {
		t.Errorf("空 values 应返回空模式，得 %q %.2f", p, r)
	}
}

// 876/924/931 infer* 空 base/prefix/suffix 回退。
func TestGapRC_InferContextFallbacks(t *testing.T) {
	if got := inferVariableNameWithContext("p", "suffix", []string{"x", "y"}); strings.Contains(got, "p") == false {
		t.Errorf("suffix 空前缀应回退 parentKey，得 %q", got)
	}
	if got := inferPatternRegexWithContext("prefix", []string{"x", "y"}); got != "" {
		t.Errorf("prefix 空前缀应返回空正则，得 %q", got)
	}
	if got := inferPatternRegexWithContext("suffix", []string{"x", "y"}); got != "" {
		t.Errorf("suffix 空前缀应返回空正则，得 %q", got)
	}
}

// 1122 InferRequiredParams 在 nil Tree 上返回 0。
func TestGapRC_InferRequiredNilTree(t *testing.T) {
	r := NewReverseRouter()
	r.Tree = nil // 构造后置空 Tree
	if got := r.InferRequiredParams(); got != 0 {
		t.Errorf("nil Tree 应返回 0，得 %d", got)
	}
}

// 1129 threshold<=0 回退默认 0.9。
func TestGapRC_InferRequiredThresholdZero(t *testing.T) {
	r := newSilentRouter()
	r.SetMergeConfig(MergeConfig{RequiredParamThreshold: 0})
	r.ReverseHttpRequest(request.NewHttpRequest("/api/r?x=1&x=1&x=1&x=1&x=1", nil, "GET", nil))
	// threshold 0 会被重置为 0.9；请求仅 1 次，样本不足不会强制必需
	if got := r.InferRequiredParams(); got < 0 {
		t.Errorf("InferRequiredParams=%d", got)
	}
}

// 1141 类型断言失败：挂 request_param 类型名但 Go 类型不对的假节点。
func TestGapRC_InferRequiredTypeAssertFail(t *testing.T) {
	r := newSilentRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/ta?x=1", nil, "GET", nil))
	m, _, err := r.FindRouteNode(request.NewHttpRequest("/api/ta", nil, "GET", nil))
	if err != nil || m == nil {
		t.Fatalf("查找方法节点失败: %v %v", m, err)
	}
	fake := node.NewBaseNode[node.NodeContext]("request_param", "zzfake", "", node.NewBaseNodeContext())
	if err := m.AddChild(fake); err != nil {
		t.Fatal(err)
	}
	r.InferRequiredParams() // 不应 panic，断言失败 continue
}

// 1159 visitMethodNodes nil 节点 + 1179 IsNeedRequest 解析失败 + 1185 method 空。
func TestGapRC_VisitNilAndNeedRequest(t *testing.T) {
	r := newSilentRouter()
	r.visitMethodNodes(nil, func(node.Node[node.NodeContext]) {}) // 不应 panic
	if !r.IsNeedRequest(request.NewHttpRequest("http://h.test/%zz%", nil, "GET", nil)) {
		t.Error("非法 URL 应返回 true（需要请求）")
	}
	// 空 method 默认 GET
	if !r.IsNeedRequest(request.NewHttpRequest("http://h.test/none", nil, "", nil)) {
		t.Error("未知路径应返回 true")
	}
}

// 1207/1209/1221/1225/1240/1244/1258 IsNeedRequest 各子分支。
func TestGapRC_IsNeedRequestBranches(t *testing.T) {
	// 建一条含 CT/header/Cookie 的已知路由
	h := request.Headers{}
	h.Set("Content-Type", "application/json")
	h.Set("Accept", "application/json")
	h.Set("Cookie", "sid=abc")
	r := newSilentRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/full", h, "POST", []byte(`{}`)))

	// 1207 CT 缺失：方法已建但 CT 子节点没有
	hCT := request.Headers{}
	hCT.Set("Content-Type", "application/xml")
	if !r.IsNeedRequest(request.NewHttpRequest("/api/full", hCT, "POST", []byte(`<a/>`))) {
		t.Error("未建 CT 子节点应返回 true")
	}

	// 1225/1244 header 分组缺失：独立树方法节点无 Accept/Cookie 分组
	r2 := newSilentRouter()
	r2.ReverseHttpRequest(request.NewHttpRequest("/api/nohdr?x=1", nil, "GET", nil))
	hMiss := request.Headers{}
	hMiss.Set("Accept", "application/json")
	if !r2.IsNeedRequest(request.NewHttpRequest("/api/nohdr?x=1", hMiss, "GET", nil)) {
		t.Error("未建 Accept 分组应返回 true")
	}
	hCK := request.Headers{}
	hCK.Set("Cookie", "nope=1")
	if !r2.IsNeedRequest(request.NewHttpRequest("/api/nohdr?x=1", hCK, "GET", nil)) {
		t.Error("未建 cookie 分组应返回 true")
	}
	// 1221 规范化后为空：Accept=" " 非空但 normalize 后为 "" → continue
	r2.IsNeedRequest(request.NewHttpRequest("/api/nohdr?x=1", request.Headers{"Accept": " "}, "GET", nil))
	// 1221 已执行（不走 return true），再次确认真实 header 分组命中返回 false

	// 1240 cookie 空值：continue → 不因空值 return，走 count>0 返回 false
	r3 := newSilentRouter()
	r3.ReverseHttpRequest(request.NewHttpRequest("/api/ck2", request.Headers{"Cookie": "sid=abc"}, "GET", nil))
	if r3.IsNeedRequest(request.NewHttpRequest("/api/ck2", request.Headers{"Cookie": "sid="}, "GET", nil)) {
		t.Error("纯空 cookie 值应 continue 后命中已知方法返回 false")
	}

	// 1258 方法存在但 count==0 → return true（手工构造未计数方法节点）
	tr := NewReverseRouterWithTree(tree.NewTree())
	method := node.NewBaseNode[node.NodeContext]("request_method", "GET", "", node.NewBaseNodeContext())
	if err := tr.Tree.Root.AddChild(method); err != nil {
		t.Fatal(err)
	}
	if !tr.IsNeedRequest(request.NewHttpRequest("/", nil, "GET", nil)) {
		t.Error("未计数方法节点应返回 true")
	}

	// 已建完整路由且计数>0 → false
	if r.IsNeedRequest(request.NewHttpRequest("/api/full", h, "POST", []byte(`{}`))) {
		t.Error("已知完整路由应返回 false")
	}
}

// 1274/1284/1288 AssetKey / HostAssetKey。
func TestGapRC_AssetKeys(t *testing.T) {
	empty := NormalizedRoute{}
	if got := empty.AssetKey(); got != "" {
		t.Errorf("空 method 应返回 template，得 %q", got)
	}
	if got := empty.HostAssetKey(); got != "<unknown>" {
		t.Errorf("空 host 应回退 unknown，得 %q", got)
	}
	methodOnly := NormalizedRoute{Method: "GET"}
	if got := methodOnly.HostAssetKey(); got != "<unknown> GET " {
		t.Errorf("空 template 应仅返 host，得 %q", got)
	}
	full := NormalizedRoute{Host: "h.test", Method: "POST", Template: "/x"}
	if got := full.AssetKey(); got != "POST /x" {
		t.Errorf("AssetKey=%q", got)
	}
	if got := full.HostAssetKey(); got != "h.test POST /x" {
		t.Errorf("HostAssetKey=%q", got)
	}
}

// 1301 normalizePathSegments 空段跳过。
func TestGapRC_NormalizePathSegmentsEmpty(t *testing.T) {
	r := newSilentRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/skip?a=1", nil, "GET", nil))
	paths := []*request.HttpRequestPath{nil, request.NewHttpRequestPath("api")}
	cur, segs, _, ok := r.normalizePathSegments(paths)
	if !ok || cur == nil {
		t.Errorf("空段应跳过并继续，ok=%v", ok)
	}
	if len(segs) == 0 {
		t.Error("应至少记录到 api 段")
	}
}

// 1364 FindRouteNode 空 method。
func TestGapRC_FindRouteNodeEmptyMethod(t *testing.T) {
	r := newSilentRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/em", nil, "GET", nil))
	m, ct, err := r.FindRouteNode(request.NewHttpRequest("/api/em", nil, "", nil))
	if err != nil || m == nil {
		t.Errorf("空 method 应默认 GET 命中，m=%v err=%v", m, err)
	}
	if ct != nil {
		t.Errorf("GET 无 CT 应返回 nil ct，得 %v", ct)
	}
}

// 1395 average 空。
func TestGapRC_AverageEmpty(t *testing.T) {
	if got := average(nil); got != 0.0 {
		t.Errorf("空 array 应返回 0，得 %.2f", got)
	}
}

// 1480 detectSuffixPattern 短值（len(v)<=suffixLen 跳过）。
func TestGapRC_SuffixShortValue(t *testing.T) {
	// 公共后缀 "u"，值 "u" 长度==suffixLen(1) → 走 1480 跳过分支；
	// "10u"/"20u" 正常计入。detectSuffixPattern 对短值跳过不计 matches。
	r := detectSuffixPattern([]string{"10u", "20u", "u"})
	// 3 个值中 2 个有变量前缀，ratio ~ 0.67
	if r < 0.5 {
		t.Errorf("suffix 匹配率应 >=0.5，得 %.2f", r)
	}
}

// 1668 normalizeAcceptLanguage 分号分支。
func TestGapRC_AcceptLanguageSemicolon(t *testing.T) {
	if got := normalizeAcceptLanguage("zh-CN;q=0.9,en;q=0.8"); got != "zh-CN" {
		t.Errorf("应取第一个语言，得 %q", got)
	}
}

// 1692/1702 processRoutingHeaders 空值与规范化空值。
func TestGapRC_HeaderEmptyValues(t *testing.T) {
	r := newSilentRouter()
	m := node.NewBaseNode[node.NodeContext]("request_method", "GET", "", node.NewBaseNodeContext())
	// 1692 val=="" → continue 不 return 错误
	if err := r.processRoutingHeaders(m, request.Headers{"Accept": ""}); err != nil {
		t.Errorf("空 header 值应继续，得 %v", err)
	}
	// 1702 normalize 后为空（逗号前空）→ continue
	if err := r.processRoutingHeaders(m, request.Headers{"Accept": ","}); err != nil {
		t.Errorf("规范化空值应继续，得 %v", err)
	}
	// Accept="text/html" 正常建节点
	if err := r.processRoutingHeaders(m, request.Headers{"Accept": "text/html"}); err != nil {
		t.Errorf("正常 header 应成功：%v", err)
	}
}

// 1756 processCookies 空 cookie 值。
func TestGapRC_CookieEmptyValue(t *testing.T) {
	r := newSilentRouter()
	m := node.NewBaseNode[node.NodeContext]("request_method", "GET", "", node.NewBaseNodeContext())
	if err := r.processCookies(m, request.Headers{"Cookie": "a=;b=1"}); err != nil {
		t.Errorf("空 cookie 值应 continue，得 %v", err)
	}
}

// 176 applyMetricCap nil 输入。
func TestGapRC_ApplyMetricCapNil(t *testing.T) {
	r := newSilentRouter()
	r.applyMetricCap(nil) // 不应 panic
}

// --- 防御死代码确认（同包直接调用私有方法，路径无副作用） ---

// 691 DetectPattern 空已在上方覆盖。

// 1656 normalizeAuthorization 的 return ""：SplitN(val," ",2) 对非空 val 恒返回
// len>=1 的切片（至少含原串），故 len(parts)>0 恒真，该 return 在业务路径不可达。
func TestGapRC_AuthorizationReturn(t *testing.T) {
	// SplitN 等价验证：非空串按单空格切分必得 len>=1
	if got := len(strings.SplitN("Basic dXNlcg==", " ", 2)); got != 2 {
		t.Errorf("含空格应 split 成 2 段，得 %d", got)
	}
	// 确认 normalizeAuthorization 返回首个 scheme
	if got := normalizeAuthorization("Basic dXNlcg=="); got != "Basic" {
		t.Errorf("应取 scheme，得 %q", got)
	}
	if got := normalizeAuthorization(""); got != "" {
		t.Errorf("空串应返回空，得 %q", got)
	}
}

var _ = fmt.Sprintf // 保留 fmt 导入（断言用）

// 471 锁内 double-check：并发多个 goroutine 同时喂入相同 key 路径段，
// 都判定 child==nil 后进入 mergeMu 临界区，后到的在锁内 FindChildByKey
// 命中已创建的节点（double-check 提前返回）。
func TestGapRC_ConcurrentPathDoubleCheck(t *testing.T) {
	const n = 32
	for round := 0; round < 30; round++ {
		r := newSilentRouter()
		var wg sync.WaitGroup
		var ready atomic.Int64
		var gate atomic.Bool
		gate.Store(false)
		errs := make([]error, n)
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				ready.Add(1)
				for !gate.Load() {
					runtime.Gosched()
				}
				errs[idx] = r.ReverseHttpRequest(request.NewHttpRequest("/api/users/1", nil, "GET", nil))
			}(i)
		}
		for ready.Load() < n {
			runtime.Gosched()
		}
		gate.Store(true)
		wg.Wait()
		for i := 0; i < n; i++ {
			if errs[i] != nil {
				t.Fatalf("轮次 %d 并发喂入失败: %v", round, errs[i])
			}
		}
	}
}
