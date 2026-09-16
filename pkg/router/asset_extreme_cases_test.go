package router

// asset_extreme_cases_test.go — 网络空间测绘 URL 资产归一化的 B~E 类极端 case。
//
// 分组：
//   B: 路径结构与匹配语义
//   C: 变量识别与选择性合并（含复杂模式在 3/6 样本下的合并边界）
//   D: Query/Header/Cookie/Body 维度与必需参数阈值边界
//   E: 多目标资产身份与 RouterSet
//
// 本文件只断言确定的、可复现的契约行为；对已知设计边界（如复杂模式需
// ≥6 个相似长度样本才合并）以"文档化边界"断言，而非掩盖实现问题。

import (
	"strings"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
)

// ---------------------------------------------------------------------------
// 通用小工具（本文件私有，避免与其他测试文件重名）
// ---------------------------------------------------------------------------

// assetFeed 逐条喂入 GET 请求。
func assetFeed(t *testing.T, r *ReverseRouter, urls ...string) {
	t.Helper()
	for _, u := range urls {
		if err := r.ReverseHttpRequest(request.NewHttpRequest(u, nil, "GET", nil)); err != nil {
			t.Fatalf("喂入 %q 失败: %v", u, err)
		}
	}
}

// assetPathVar 沿固定段下钻后返回路径变量节点；不存在则 fatal。
func assetPathVar(t *testing.T, r *ReverseRouter, fixed ...string) *node.RequestPathVariableNode {
	t.Helper()
	cur := r.Tree.Root
	for _, s := range fixed {
		cur = cur.FindChildByKey(s)
		if cur == nil {
			t.Fatalf("固定段 %q 缺失", s)
		}
	}
	v := cur.GetChildByType("request_path_variable")
	if v == nil {
		t.Fatalf("在 %v 下未找到路径变量", fixed)
	}
	return v.(*node.RequestPathVariableNode)
}

// assetFixed 沿固定段下钻确认某固定子节点存在。
func assetFixed(t *testing.T, r *ReverseRouter, fixed []string, child string) bool {
	t.Helper()
	cur := r.Tree.Root
	for _, s := range fixed {
		cur = cur.FindChildByKey(s)
		if cur == nil {
			return false
		}
	}
	return cur.FindChildByKey(child) != nil
}

// ---------------------------------------------------------------------------
// B 类：路径结构与匹配语义
// ---------------------------------------------------------------------------

func TestAssetB01_RootPath(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/")
	m := r.Tree.Root.FindChildByKey("GET")
	if m == nil {
		t.Fatal("B01 根路径应建 GET 方法节点")
	}
	if n, ok := r.NormalizeURL(request.NewHttpRequest("/", nil, "GET", nil)); !ok || n.Template != "/" {
		t.Fatalf("B01 NormalizeURL('/')=%+v ok=%v, want Template=/ ok=true", n, ok)
	}
}

func TestAssetB02_MultiSegment(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/api/users/1", "/api/users/2", "/api/users/3")
	pv := assetPathVar(t, r, "api", "users")
	if pv.GetKey() != "users_id" {
		t.Errorf("B02 变量名=%s want users_id", pv.GetKey())
	}
}

func TestAssetB03_TrailingSlashEquivalence(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/api/users/1", "/api/users/2/", "/api/users/3")
	pv := assetPathVar(t, r, "api", "users")
	if pv == nil {
		t.Fatal("B03 尾斜杠应与无尾斜杠等价，仍合并")
	}
}

func TestAssetB04_ConsecutiveSlashes(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/api//users/1", "/api/users/2", "/api/users/3")
	pv := assetPathVar(t, r, "api", "users")
	if pv == nil {
		t.Fatal("B04 连续斜杠应压缩，仍能合并")
	}
}

func TestAssetB05_FixedCoexistsWithVariable(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/api/users/1", "/api/users/2", "/api/users/3", "/api/users/list")
	if !assetFixed(t, r, []string{"api", "users"}, "list") {
		t.Error("B05 固定路径 list 应保留")
	}
	if assetPathVar(t, r, "api", "users") == nil {
		t.Error("B05 数字应合并为变量")
	}
}

func TestAssetB06_FixedPathPriority(t *testing.T) {
	// 固定路径优先：先喂固定 list，再喂变量数字，两者共存且各走各的
	r := newSilentRouter()
	assetFeed(t, r, "/api/users/list")
	// 数字段不应被并入 list
	r.ReverseHttpRequest(request.NewHttpRequest("/api/users/42", nil, "GET", nil))
	users := r.Tree.Root.FindChildByKey("api").FindChildByKey("users")
	if users.FindChildByKey("list") == nil {
		t.Error("B06 list 应保留为固定路径")
	}
	// 单条数字不达阈值，不合并
	if users.GetChildByType("request_path_variable") != nil {
		t.Error("B06 单条数字不应合并")
	}
}

func TestAssetB07_VariableThenDeepLayers(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r,
		"/api/users/1/posts/10", "/api/users/2/posts/20", "/api/users/3/posts/30",
		"/api/users/1/posts/10/comments/100", "/api/users/1/posts/10/comments/200", "/api/users/1/posts/10/comments/300")
	users := r.Tree.Root.FindChildByKey("api").FindChildByKey("users")
	usersVar := users.GetChildByType("request_path_variable")
	if usersVar == nil {
		t.Fatal("B07 users 下应合并出变量")
	}
	posts := usersVar.FindChildByKey("posts")
	if posts == nil {
		t.Fatal("B07 users 变量下应保留固定段 posts")
	}
	postsVar := posts.GetChildByType("request_path_variable")
	if postsVar == nil {
		t.Fatal("B07 posts 下详情数字应合并出变量")
	}
	comments := postsVar.FindChildByKey("comments")
	if comments == nil {
		t.Fatal("B07 变量下应保留固定段 comments")
	}
	if comments.GetChildByType("request_path_variable") == nil {
		t.Fatal("B07 三层级联 comments 下应合并变量")
	}
}

func TestAssetB08_PathKeyValueSegment(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/api/filter=1", "/api/filter=2", "/api/filter=3")
	// key=value 段作为参数键节点（filter），方法节点下生成参数
	f := r.Tree.Root.FindChildByKey("api").FindChildByKey("filter")
	if f == nil {
		t.Fatal("B08 key=value 段应生成 filter 路径节点")
	}
	if f.FindChildByKey("GET").FindChildByKey("filter") == nil {
		t.Error("B08 filter 值应作为参数节点")
	}
}

func TestAssetB09_MethodCaseInsensitiveUpper(t *testing.T) {
	r := newSilentRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/x", nil, "get", nil))
	r.ReverseHttpRequest(request.NewHttpRequest("/api/x", nil, "GET", nil))
	m := r.Tree.Root.FindChildByKey("api").FindChildByKey("x").FindChildByKey("GET")
	if m == nil {
		t.Fatal("B09 小写方法 get 应归一化为 GET")
	}
	children := r.Tree.Root.FindChildByKey("api").FindChildByKey("x").GetChildren()
	// 只应有一个方法节点
	count := 0
	for _, c := range children {
		if c.GetType() == "request_method" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("B09 方法节点数=%d want 1", count)
	}
}

func TestAssetB10_EmptyMethodDefaultsGet(t *testing.T) {
	r := newSilentRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/x", nil, "", nil))
	if r.Tree.Root.FindChildByKey("api").FindChildByKey("x").FindChildByKey("GET") == nil {
		t.Fatal("B10 空方法应默认 GET")
	}
}

func TestAssetB11_SamePathMultiMethod(t *testing.T) {
	r := newSilentRouter()
	for _, m := range []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/resource", nil, m, nil))
	}
	res := r.Tree.Root.FindChildByKey("api").FindChildByKey("resource")
	for _, m := range []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"} {
		if res.FindChildByKey(m) == nil {
			t.Errorf("B11 同路径缺少方法 %s", m)
		}
	}
}

func TestAssetB12_PercentEncodedPath(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/api/%E6%B5%8B%E8%AF%95/1", "/api/%E6%B5%8B%E8%AF%95/2", "/api/%E6%B5%8B%E8%AF%95/3")
	// 解码后的固定段：api → 测试
	if r.Tree.Root.FindChildByKey("api").FindChildByKey("测试") == nil {
		t.Fatal("B12 百分号编码应被解码为固定段 测试")
	}
}

func TestAssetB13_UnknownRouteNotNormalized(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/api/known/1", "/api/known/2", "/api/known/3")
	_, ok := r.NormalizeURL(request.NewHttpRequest("/api/never/seen", nil, "GET", nil))
	if ok {
		t.Error("B13 未收录路由不应归一化成功")
	}
}

func TestAssetB14_DotSegmentsFiltered(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/api/./users/1", "/api/users/2", "/api/users/3")
	pv := assetPathVar(t, r, "api", "users")
	if pv == nil {
		t.Fatal("B14 . 段应被过滤，不影响后续合并")
	}
}

func TestAssetB15_TwoFixedSiblingResources(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/api/users", "/api/orders", "/api/products", "/api/settings", "/api/roles", "/api/logs")
	api := r.Tree.Root.FindChildByKey("api")
	// 固定资源名长度相近，但默认 SimilarLengthBreakThreshold=6，恰好 6 个 → 会合并为变量
	// （这是文档化边界：≥6 个相似长度字符串视为变量集合）
	if api.GetChildByType("request_path_variable") == nil {
		t.Error("B15 6 个相似长度资源名按类似长度突破规则合并")
	}
}

func TestAssetB16_OneFixedResourceNoMerge(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/api/users")
	api := r.Tree.Root.FindChildByKey("api")
	if api.GetChildByType("request_path_variable") != nil {
		t.Error("B16 单个固定资源不应合并")
	}
}

// ---------------------------------------------------------------------------
// C 类：变量识别与选择性合并
// ---------------------------------------------------------------------------

func TestAssetC01_IntegerMerge(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/api/users/101", "/api/users/202", "/api/users/303")
	pv := assetPathVar(t, r, "api", "users")
	if pv.GetKey() != "users_id" || pv.GetPattern() == nil {
		t.Errorf("C01 变量名=%s pattern=%v want users_id + 整数正则", pv.GetKey(), pv.GetPattern())
	}
}

func TestAssetC02_PrefixPatternMerge(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/api/items/user_001", "/api/items/user_002", "/api/items/user_003")
	pv := assetPathVar(t, r, "api", "items")
	if !strings.HasPrefix(pv.GetKey(), "user") && !strings.HasPrefix(pv.GetKey(), "items") {
		t.Errorf("C02 前缀模式变量名=%s", pv.GetKey())
	}
}

func TestAssetC03_UuidMerge(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r,
		"/api/things/550e8400-e29b-41d4-a716-446655440000",
		"/api/things/6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		"/api/things/7c9e6679-7425-40de-944b-e07fc1f90ae7")
	pv := assetPathVar(t, r, "api", "things")
	if pv.GetKey() != "things_uuid" {
		t.Errorf("C03 uuid 变量名=%s want things_uuid", pv.GetKey())
	}
}

func TestAssetC04_HexObjectIDMerge(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r,
		"/api/docs/507f1f77bcf86cd799439011",
		"/api/docs/507f1f77bcf86cd799439012",
		"/api/docs/507f1f77bcf86cd799439013")
	pv := assetPathVar(t, r, "api", "docs")
	if pv == nil {
		t.Fatal("C04 24 位 hex ObjectId 应合并")
	}
}

func TestAssetC05_FloatMerge(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/api/coord/1.5", "/api/coord/2.25", "/api/coord/3.75")
	pv := assetPathVar(t, r, "api", "coord")
	if pv.GetKey() != "coord_value" {
		t.Errorf("C05 float 变量名=%s want coord_value", pv.GetKey())
	}
}

func TestAssetC06_VersionMerge(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/api/app/v1", "/api/app/v2", "/api/app/v3")
	pv := assetPathVar(t, r, "api", "app")
	if pv.GetKey() != "app_version" {
		t.Errorf("C06 version 变量名=%s want app_version", pv.GetKey())
	}
}

func TestAssetC07_MinorityOutlierPreserved(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/api/users/101", "/api/users/102", "/api/users/103", "/api/users/admin")
	users := r.Tree.Root.FindChildByKey("api").FindChildByKey("users")
	if users.FindChildByKey("admin") == nil {
		t.Error("C07 少数异类值 admin 应保留为固定路径")
	}
	if users.GetChildByType("request_path_variable") == nil {
		t.Error("C07 数字多数派应合并")
	}
}

func TestAssetC08_MixedPatternsPartialMerge(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/api/ent/1", "/api/ent/2", "/api/ent/3", "/api/ent/abc-1", "/api/ent/abc-2")
	ent := r.Tree.Root.FindChildByKey("api").FindChildByKey("ent")
	// 3 个整数 + 2 个前缀 → 至少数字应合并
	if ent.GetChildByType("request_path_variable") == nil {
		t.Error("C08 应能选出整数多数派合并")
	}
}

func TestAssetC09_ComplexPatternNoMergeAtThree(t *testing.T) {
	// 文档化边界：mac/semver/latlong 等无内建模式，3 个样本只算相似长度但未达阈值 → 不合并
	r := newSilentRouter()
	assetFeed(t, r, "/api/nodes/AA:BB:CC:DD:EE:FF", "/api/nodes/00:1A:2B:3C:4D:5E", "/api/nodes/11:22:33:44:55:66")
	nodes := r.Tree.Root.FindChildByKey("api").FindChildByKey("nodes")
	if nodes.GetChildByType("request_path_variable") != nil {
		t.Error("C09 复杂模式(MAC)3 个样本不应合并（需≥6）")
	}
}

func TestAssetC10_ComplexPatternMergeAtSix(t *testing.T) {
	r := newSilentRouter()
	macs := []string{
		"/api/nodes/AA:BB:CC:DD:EE:FF", "/api/nodes/00:1A:2B:3C:4D:5E",
		"/api/nodes/11:22:33:44:55:66", "/api/nodes/ab:cd:ef:01:23:45",
		"/api/nodes/12:34:56:78:9a:bc", "/api/nodes/fe:dc:ba:98:76:54",
	}
	assetFeed(t, r, macs...)
	nodes := r.Tree.Root.FindChildByKey("api").FindChildByKey("nodes")
	if nodes.GetChildByType("request_path_variable") == nil {
		t.Error("C10 6 个相似长度复杂模式应按突破规则合并")
	}
}

func TestAssetC11_DateMerge(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/api/events/2024-01-15", "/api/events/2024-02-20", "/api/events/2024-03-25")
	pv := assetPathVar(t, r, "api", "events")
	if pv.GetKey() != "events_date" {
		t.Errorf("C11 date 变量名=%s want events_date", pv.GetKey())
	}
}

func TestAssetC12_ExistingSubtreePreservedOnMerge(t *testing.T) {
	r := newSilentRouter()
	// 先喂一个带深层子树的变量形态，再合并
	assetFeed(t, r, "/api/users/1/profile", "/api/users/2/profile", "/api/users/3/profile")
	pv := assetPathVar(t, r, "api", "users")
	if pv.FindChildByKey("profile") == nil {
		t.Error("C12 合并后变量下应保留 profile 子树")
	}
}

func TestAssetC13_NewValueHitsExistingVariable(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/api/users/101", "/api/users/202", "/api/users/303")
	r.ReverseHttpRequest(request.NewHttpRequest("/api/users/404", nil, "GET", nil))
	pv := assetPathVar(t, r, "api", "users")
	if pv.GetKey() != "users_id" {
		t.Error("C13 新值应命中已有变量而不重建")
	}
	// 仍只有 1 个变量节点
	users := r.Tree.Root.FindChildByKey("api").FindChildByKey("users")
	vc := 0
	for _, c := range users.GetChildren() {
		if c.GetType() == "request_path_variable" {
			vc++
		}
	}
	if vc != 1 {
		t.Errorf("C13 变量节点数=%d want 1", vc)
	}
}

func TestAssetC14_ThreeLevelCascade(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r,
		"/api/org/1/team/10/member/100",
		"/api/org/2/team/20/member/200",
		"/api/org/3/team/30/member/300")
	org := assetPathVar(t, r, "api", "org")
	if org == nil {
		t.Fatal("C14 第一层未合并")
	}
	team := org.FindChildByKey("team")
	if team == nil {
		t.Fatal("C14 变量下应保留固定段 team")
	}
	teamVar := team.GetChildByType("request_path_variable")
	if teamVar == nil {
		t.Fatal("C14 第二层 team 详情未级联合并")
	}
	member := teamVar.FindChildByKey("member")
	if member == nil {
		t.Fatal("C14 team 变量下应保留固定段 member")
	}
	if member.GetChildByType("request_path_variable") == nil {
		t.Fatal("C14 第三层 member 详情未级联合并")
	}
}

// ---------------------------------------------------------------------------
// D 类：Query/Header/Cookie/Body 维度与必需参数阈值
// ---------------------------------------------------------------------------

func TestAssetD01_ParamCaseInsensitive(t *testing.T) {
	r := newSilentRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/x?Page=1", nil, "GET", nil))
	r.ReverseHttpRequest(request.NewHttpRequest("/api/x?page=2", nil, "GET", nil))
	m := r.Tree.Root.FindChildByKey("api").FindChildByKey("x").FindChildByKey("GET")
	if m.FindChildByKey("page") == nil {
		t.Fatal("D01 参数名大小写应归一化为小写 page")
	}
	if m.FindChildByKey("Page") != nil {
		t.Error("D01 不应保留大写 Page 节点")
	}
}

func TestAssetD02_MultiValueParam(t *testing.T) {
	r := newSilentRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/x?tag=a&tag=b", nil, "GET", nil))
	m := r.Tree.Root.FindChildByKey("api").FindChildByKey("x").FindChildByKey("GET")
	p := m.FindChildByKey("tag")
	if p == nil {
		t.Fatal("D02 tag 参数缺失")
	}
	pn := p.(*node.RequestParamNode)
	if !pn.IsMultiValue() {
		t.Error("D02 重复参数应标记 multi_value")
	}
}

func TestAssetD03_EmptyValueParam(t *testing.T) {
	r := newSilentRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/x?flag=", nil, "GET", nil))
	m := r.Tree.Root.FindChildByKey("api").FindChildByKey("x").FindChildByKey("GET")
	if m.FindChildByKey("flag") == nil {
		t.Error("D03 空值参数应建节点")
	}
}

func TestAssetD04_ParamOrderIrrelevant(t *testing.T) {
	r := newSilentRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/x?a=1&b=2", nil, "GET", nil))
	r.ReverseHttpRequest(request.NewHttpRequest("/api/x?b=9&a=8", nil, "GET", nil))
	m := r.Tree.Root.FindChildByKey("api").FindChildByKey("x").FindChildByKey("GET")
	if m.FindChildByKey("a") == nil || m.FindChildByKey("b") == nil {
		t.Error("D04 参数顺序不影响节点存在")
	}
}

func TestAssetD05_RequiredThresholdBoundary(t *testing.T) {
	r := newSilentRouter()
	// 9/10 带 page → 0.9 ≥ 0.9 → 必需
	for i := 0; i < 9; i++ {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/x?page=1", nil, "GET", nil))
	}
	r.ReverseHttpRequest(request.NewHttpRequest("/api/x", nil, "GET", nil))
	r.InferRequiredParams()
	m := r.Tree.Root.FindChildByKey("api").FindChildByKey("x").FindChildByKey("GET")
	pn := m.FindChildByKey("page").(*node.RequestParamNode)
	if !pn.IsRequired() {
		t.Error("D05 出现率 0.9 应为必需")
	}
}

func TestAssetD06_RequiredBelowThreshold(t *testing.T) {
	r := newSilentRouter()
	// 8/10 带 page → 0.8 < 0.9 → 非必需
	for i := 0; i < 8; i++ {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/x?page=1", nil, "GET", nil))
	}
	r.ReverseHttpRequest(request.NewHttpRequest("/api/x", nil, "GET", nil))
	r.ReverseHttpRequest(request.NewHttpRequest("/api/x", nil, "GET", nil))
	r.InferRequiredParams()
	m := r.Tree.Root.FindChildByKey("api").FindChildByKey("x").FindChildByKey("GET")
	pn := m.FindChildByKey("page").(*node.RequestParamNode)
	if pn.IsRequired() {
		t.Error("D06 出现率 0.8 不应为必需")
	}
}

func TestAssetD07_RequiredAllPresent(t *testing.T) {
	r := newSilentRouter()
	for i := 0; i < 5; i++ {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/x?page=1", nil, "GET", nil))
	}
	r.InferRequiredParams()
	m := r.Tree.Root.FindChildByKey("api").FindChildByKey("x").FindChildByKey("GET")
	if !m.FindChildByKey("page").(*node.RequestParamNode).IsRequired() {
		t.Error("D07 全部请求都带 page 应为必需")
	}
}

func TestAssetD08_CustomThresholdHalf(t *testing.T) {
	r := newSilentRouter()
	r.SetMergeConfig(MergeConfig{SiblingMergeThreshold: 3, PatternSimilarityThreshold: 0.6, SimilarLengthBreakThreshold: 6, RequiredParamThreshold: 0.5})
	// 6/10 带 q → 0.6 ≥ 0.5 → 必需
	for i := 0; i < 6; i++ {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/x?q=1", nil, "GET", nil))
	}
	for i := 0; i < 4; i++ {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/x", nil, "GET", nil))
	}
	r.InferRequiredParams()
	m := r.Tree.Root.FindChildByKey("api").FindChildByKey("x").FindChildByKey("GET")
	if !m.FindChildByKey("q").(*node.RequestParamNode).IsRequired() {
		t.Error("D08 阈值 0.5，出现率 0.6 应为必需")
	}
}

func TestAssetD09_JSONNestedBody(t *testing.T) {
	r := newSilentRouter()
	h := request.Headers{}
	h.Set("Content-Type", "application/json")
	r.ReverseHttpRequest(request.NewHttpRequest("/api/users", h, "POST", []byte(`{"name":"alice","profile":{"age":30}}`)))
	m := r.Tree.Root.FindChildByKey("api").FindChildByKey("users").FindChildByKey("POST")
	if m.FindChildByKey("name") == nil {
		t.Error("D09 JSON 顶层字段 name 应建参数")
	}
	if m.FindChildByKey("profile.age") == nil {
		t.Error("D09 JSON 嵌套字段应点号扁平化为 profile.age")
	}
}

func TestAssetD10_JSONArrayBody(t *testing.T) {
	r := newSilentRouter()
	h := request.Headers{}
	h.Set("Content-Type", "application/json")
	r.ReverseHttpRequest(request.NewHttpRequest("/api/batch", h, "POST", []byte(`{"ids":[1,2,3],"meta":null}`)))
	m := r.Tree.Root.FindChildByKey("api").FindChildByKey("batch").FindChildByKey("POST")
	// JSON 数组按索引扁平化为 ids.0/ids.1/ids.2；null 字段 meta 也建节点
	if m.FindChildByKey("ids.0") == nil || m.FindChildByKey("ids.1") == nil || m.FindChildByKey("ids.2") == nil {
		t.Error("D10 数组字段应按索引扁平化为 ids.0/1/2")
	}
	if m.FindChildByKey("meta") == nil {
		t.Error("D10 null 字段应建参数 meta")
	}
}

func TestAssetD11_InvalidJSONBody(t *testing.T) {
	r := newSilentRouter()
	h := request.Headers{}
	h.Set("Content-Type", "application/json")
	if err := r.ReverseHttpRequest(request.NewHttpRequest("/api/bad", h, "POST", []byte(`{not json`))); err == nil {
		// 非法 JSON 可报错；不报错也不应 panic。二者皆可，但必须不崩溃
		m := r.Tree.Root.FindChildByKey("api").FindChildByKey("bad")
		if m == nil {
			t.Log("D11 非法 JSON 报错且未建节点，可接受")
		}
	}
}

func TestAssetD12_FormUrlencodedBody(t *testing.T) {
	r := newSilentRouter()
	h := request.Headers{}
	h.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ReverseHttpRequest(request.NewHttpRequest("/api/form", h, "POST", []byte("a=1&b=hello")))
	m := r.Tree.Root.FindChildByKey("api").FindChildByKey("form").FindChildByKey("POST")
	if m.FindChildByKey("a") == nil || m.FindChildByKey("b") == nil {
		t.Error("D12 表单字段 a/b 应建参数")
	}
}

func TestAssetD13_ContentTypeCaseInsensitive(t *testing.T) {
	r := newSilentRouter()
	h1 := request.Headers{}
	h1.Set("Content-Type", "Application/JSON")
	r.ReverseHttpRequest(request.NewHttpRequest("/api/x", h1, "POST", []byte(`{"a":1}`)))
	ct := r.Tree.Root.FindChildByKey("api").FindChildByKey("x").FindChildByKey("POST").GetChildByType("request_content_type")
	if ct == nil {
		t.Error("D13 大写 Content-Type 应识别并建 CT 节点")
	}
}

func TestAssetD14_AuthorizationSchemeRouting(t *testing.T) {
	r := newSilentRouter()
	h := request.Headers{}
	h.Set("Authorization", "Bearer eyJhbGciOiJIUzI1NiJ9")
	r.ReverseHttpRequest(request.NewHttpRequest("/api/sec", h, "GET", nil))
	m := r.Tree.Root.FindChildByKey("api").FindChildByKey("sec").FindChildByKey("GET")
	auth := m.FindChildByKey("Authorization")
	if auth == nil || auth.FindChildByKey("Bearer") == nil {
		t.Error("D14 Authorization 应规范化为 Bearer 值节点")
	}
}

func TestAssetD15_CookieValueRouting(t *testing.T) {
	r := newSilentRouter()
	h := request.Headers{}
	h.Set("Cookie", "session=abc123; theme=dark")
	r.ReverseHttpRequest(request.NewHttpRequest("/api/home", h, "GET", nil))
	m := r.Tree.Root.FindChildByKey("api").FindChildByKey("home").FindChildByKey("GET")
	sess := m.FindChildByKey("session")
	if sess == nil || sess.FindChildByKey("abc123") == nil {
		t.Error("D15 Cookie session=abc123 应建双层节点")
	}
}

func TestAssetD16_SamePathDifferentContentType(t *testing.T) {
	r := newSilentRouter()
	hj := request.Headers{}
	hj.Set("Content-Type", "application/json")
	r.ReverseHttpRequest(request.NewHttpRequest("/api/data", hj, "POST", []byte(`{"a":1}`)))
	hx := request.Headers{}
	hx.Set("Content-Type", "application/xml")
	r.ReverseHttpRequest(request.NewHttpRequest("/api/data", hx, "POST", []byte("<a>1</a>")))
	post := r.Tree.Root.FindChildByKey("api").FindChildByKey("data").FindChildByKey("POST")
	// 两种 CT 应共存
	if post.GetChildByType("request_content_type") == nil {
		t.Error("D16 应至少存在 CT 路由")
	}
}

// ---------------------------------------------------------------------------
// E 类：多目标资产身份与 RouterSet
// ---------------------------------------------------------------------------

func TestAssetE01_HostCaseInsensitive(t *testing.T) {
	rs := NewRouterSet()
	req := request.NewHttpRequest("http://Example.COM/api/x", nil, "GET", nil)
	req.Host = "Example.COM"
	if err := rs.ReverseHttpRequest(req); err != nil {
		t.Fatal(err)
	}
	hosts := rs.Hosts()
	if len(hosts) != 1 || hosts[0] != "example.com" {
		t.Errorf("E01 Hosts=%v want [example.com]（小写归一）", hosts)
	}
}

func TestAssetE02_PortDistinctHosts(t *testing.T) {
	rs := NewRouterSet()
	r1 := request.NewHttpRequest("http://a.com/api/x", nil, "GET", nil)
	r1.Host = "a.com"
	r2 := request.NewHttpRequest("http://a.com:8080/api/x", nil, "GET", nil)
	r2.Host = "a.com:8080"
	rs.ReverseHttpRequest(r1)
	rs.ReverseHttpRequest(r2)
	if len(rs.Hosts()) != 2 {
		t.Errorf("E02 带端口与不带端口应视为不同 host，Hosts=%v", rs.Hosts())
	}
}

func TestAssetE03_ExplicitHostOverridesURL(t *testing.T) {
	rs := NewRouterSet()
	req := request.NewHttpRequest("http://other.com/path", nil, "GET", nil)
	req.Host = "api.target.com"
	rs.ReverseHttpRequest(req)
	if len(rs.Hosts()) != 1 || rs.Hosts()[0] != "api.target.com" {
		t.Errorf("E03 显式 Host 应优先于 URL，Hosts=%v", rs.Hosts())
	}
}

func TestAssetE04_URLHostFallback(t *testing.T) {
	rs := NewRouterSet()
	req := request.NewHttpRequest("http://z.example.com/api/x", nil, "GET", nil) // 无显式 Host
	rs.ReverseHttpRequest(req)
	if len(rs.Hosts()) != 1 || rs.Hosts()[0] != "z.example.com" {
		t.Errorf("E04 应回退 URL host，Hosts=%v", rs.Hosts())
	}
}

func TestAssetE05_DifferentHostsSamePathIsolated(t *testing.T) {
	rs := NewRouterSet()
	for _, h := range []string{"a.com", "b.com"} {
		for _, id := range []string{"1", "2", "3"} {
			req := request.NewHttpRequest("http://"+h+"/api/users/"+id, nil, "GET", nil)
			req.Host = h
			rs.ReverseHttpRequest(req)
		}
	}
	// 各自只收录自己 host 的路径
	for _, h := range []string{"a.com", "b.com"} {
		probe := request.NewHttpRequest("http://"+h+"/api/users/99", nil, "GET", nil)
		probe.Host = h
		// 99 是未知 ID，但 /api/users/{id} 已合并 → 应能归一化
		if _, ok := rs.NormalizeURL(probe); !ok {
			t.Errorf("E05 host %s 应能归一化同模板未知 ID", h)
		}
	}
	// 反向：b.com 不该收录 a.com 专属路由
	cross := request.NewHttpRequest("http://b.com/api/only-a/1", nil, "GET", nil)
	cross.Host = "b.com"
	if _, ok := rs.NormalizeURL(cross); ok {
		t.Error("E05 跨 host 污染：b.com 不应命中 only-a")
	}
}

func TestAssetE06_HostAssetKeyDistinct(t *testing.T) {
	rs := NewRouterSet()
	for _, h := range []string{"a.com", "b.com"} {
		for _, id := range []string{"1", "2", "3"} {
			req := request.NewHttpRequest("http://"+h+"/api/users/"+id, nil, "GET", nil)
			req.Host = h
			rs.ReverseHttpRequest(req)
		}
	}
	// 两个 host 的同路径同模板 → HostAssetKey 不碰撞
	reqA := request.NewHttpRequest("http://a.com/api/users/7", nil, "GET", nil)
	reqA.Host = "a.com"
	reqB := request.NewHttpRequest("http://b.com/api/users/7", nil, "GET", nil)
	reqB.Host = "b.com"
	na, _ := rs.NormalizeURL(reqA)
	nb, _ := rs.NormalizeURL(reqB)
	if na.HostAssetKey() == nb.HostAssetKey() {
		t.Errorf("E06 不同 host 资产键碰撞: %s == %s", na.HostAssetKey(), nb.HostAssetKey())
	}
	if !strings.HasPrefix(na.HostAssetKey(), "a.com ") || !strings.HasPrefix(nb.HostAssetKey(), "b.com ") {
		t.Errorf("E06 HostAssetKey 应含 host 前缀: %s | %s", na.HostAssetKey(), nb.HostAssetKey())
	}
}

func TestAssetE07_SameHostDifferentMethodKeys(t *testing.T) {
	rs := NewRouterSet()
	req := request.NewHttpRequest("http://a.com/api/res", nil, "GET", nil)
	req.Host = "a.com"
	rs.ReverseHttpRequest(req)
	req2 := request.NewHttpRequest("http://a.com/api/res", nil, "POST", nil)
	req2.Host = "a.com"
	rs.ReverseHttpRequest(req2)
	ng, _ := rs.NormalizeURL(req)
	np, _ := rs.NormalizeURL(req2)
	if ng.AssetKey() == np.AssetKey() {
		t.Errorf("E07 不同方法的资产键不应相同")
	}
	if ng.AssetKey() != "GET /api/res" || np.AssetKey() != "POST /api/res" {
		t.Errorf("E07 AssetKey 格式: %q | %q", ng.AssetKey(), np.AssetKey())
	}
}

func TestAssetE08_NormalizeAssetsBucketed(t *testing.T) {
	rs := NewRouterSet()
	for _, id := range []string{"1", "2", "3"} {
		req := request.NewHttpRequest("http://a.com/api/users/"+id, nil, "GET", nil)
		req.Host = "a.com"
		rs.ReverseHttpRequest(req)
	}
	in := []*request.HttpRequest{
		{Url: "http://a.com/api/users/5", Host: "a.com", Method: "GET"},
		{Url: "http://a.com/api/users/6", Host: "a.com", Method: "GET"},
	}
	assets := rs.NormalizeAssets(in)
	key := "a.com GET /api/users/{users_id}"
	if len(assets[key]) != 2 {
		t.Errorf("E08 NormalizeAssets 应聚合到 %q，got %v", key, assets)
	}
}

func TestAssetE09_UnknownRouteNoBadAsset(t *testing.T) {
	rs := NewRouterSet()
	req := request.NewHttpRequest("http://a.com/api/x", nil, "GET", nil)
	req.Host = "a.com"
	rs.ReverseHttpRequest(req)
	unknown := request.NewHttpRequest("http://a.com/api/never", nil, "GET", nil)
	unknown.Host = "a.com"
	if _, ok := rs.NormalizeURL(unknown); ok {
		t.Error("E09 未收录路由不应归一化")
	}
}

func TestAssetE10_EmptyHostBucket(t *testing.T) {
	rs := NewRouterSet()
	// 相对路径无 host → 空 host 桶
	req := request.NewHttpRequest("/api/local/1", nil, "GET", nil)
	rs.ReverseHttpRequest(req)
	if len(rs.Hosts()) != 1 || rs.Hosts()[0] != "" {
		t.Errorf("E10 无 host 应落入空桶，Hosts=%v", rs.Hosts())
	}
}

func TestAssetE11_ConfigPropagationToNewBucket(t *testing.T) {
	rs := NewRouterSet()
	cfg := DefaultMergeConfig
	cfg.SiblingMergeThreshold = 2
	rs.SetMergeConfig(cfg)
	// 配置后新建桶应继承
	req := request.NewHttpRequest("http://a.com/api/x", nil, "GET", nil)
	req.Host = "a.com"
	r := rs.RouterFor(req)
	if r.GetMergeConfig().SiblingMergeThreshold != 2 {
		t.Errorf("E11 新建桶未继承配置: %d", r.GetMergeConfig().SiblingMergeThreshold)
	}
}

func TestAssetE12_ConfigPropagationToExistingBucket(t *testing.T) {
	rs := NewRouterSet()
	req := request.NewHttpRequest("http://a.com/api/x", nil, "GET", nil)
	req.Host = "a.com"
	r := rs.RouterFor(req)
	cfg := DefaultMergeConfig
	cfg.SiblingMergeThreshold = 2
	rs.SetMergeConfig(cfg)
	if r.GetMergeConfig().SiblingMergeThreshold != 2 {
		t.Errorf("E12 已有桶未收到配置传播: %d", r.GetMergeConfig().SiblingMergeThreshold)
	}
}

func TestAssetE13_CurlBatchBadSamples(t *testing.T) {
	rs := NewRouterSet()
	curls := []string{
		"curl http://a.com/api/x",
		"curl -X POST http://a.com/api/y -d 'a=1'",
		"not a curl at all",
		"curl http://a.com",
	}
	res := rs.ReverseCurls(curls)
	if res.Processed < 1 {
		t.Errorf("E13 至少应处理合法 curl，Processed=%d", res.Processed)
	}
	if res.Failed == 0 {
		t.Error("E13 应有坏样本被跳过")
	}
}

func TestAssetE14_StatsStableAcrossHosts(t *testing.T) {
	rs := NewRouterSet()
	for _, h := range []string{"a.com", "b.com", "c.com"} {
		for _, id := range []string{"1", "2", "3"} {
			req := request.NewHttpRequest("http://"+h+"/api/items/"+id, nil, "GET", nil)
			req.Host = h
			rs.ReverseHttpRequest(req)
		}
	}
	stats := rs.Stats()
	if len(stats) != 3 {
		t.Errorf("E14 Stats 应含 3 个 host，got %d", len(stats))
	}
	for _, s := range stats {
		if s.RequestsProcessed != 3 {
			t.Errorf("E14 每 host 处理 3 请求，got %d", s.RequestsProcessed)
		}
	}
}

func TestAssetE15_HostsStableSorted(t *testing.T) {
	rs := NewRouterSet()
	for _, h := range []string{"c.com", "a.com", "b.com"} {
		req := request.NewHttpRequest("http://"+h+"/api/x", nil, "GET", nil)
		req.Host = h
		rs.ReverseHttpRequest(req)
	}
	hosts := rs.Hosts()
	if hosts[0] != "a.com" || hosts[1] != "b.com" || hosts[2] != "c.com" {
		t.Errorf("E15 Hosts 应稳定排序: %v", hosts)
	}
}

func TestAssetE16_RouterSetNilConfigMethods(t *testing.T) {
	var rs *RouterSet
	rs.SetMergeConfig(DefaultMergeConfig) // 不应 panic
	rs.SetMergeRule(nil)
	rs.SetLogLevel(LogLevelDebug)
	if rs.Hosts() != nil {
		t.Error("E16 nil RouterSet Hosts 应为 nil")
	}
}

// ===========================================================================
// 第二批：继续补齐 B~E 矩阵目标（B30/C30/D25/E20）
// ===========================================================================

// B17 根层变量：路径直接位于根下也能合并。
func TestAssetB17_RootLevelVariableMerge(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/1", "/2", "/3")
	if r.Tree.Root.GetChildByType("request_path_variable") == nil {
		t.Error("B17 根下数字 1/2/3 应合并为变量")
	}
	if _, ok := r.NormalizeURL(request.NewHttpRequest("/5", nil, "GET", nil)); !ok {
		t.Error("B17 根层变量应能归一化未知数字")
	}
}

// B18 路径大小写敏感：/Api 与 /api 是两个不同固定段。
func TestAssetB18_PathCaseSensitive(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/api/x", "/Api/x")
	root := r.Tree.Root
	if root.FindChildByKey("api") == nil || root.FindChildByKey("Api") == nil {
		t.Error("B18 大小写路径应各自保留为不同固定段")
	}
}

// B19 百分号编码的空格段解码为固定段。
func TestAssetB19_EncodedSpaceSegment(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/api/a%20b/1", "/api/a%20b/2", "/api/a%20b/3")
	if r.Tree.Root.FindChildByKey("api").FindChildByKey("a b") == nil {
		t.Error("B19 编码空格应解码为固定段 'a b'")
	}
}

// B20 保留字符段（路径中的 + 保持字面量）。
func TestAssetB20_ReservedCharSegment(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/api/a+b")
	if r.Tree.Root.FindChildByKey("api").FindChildByKey("a+b") == nil {
		t.Error("B20 路径中的 + 应按字面量保留段 a+b")
	}
}

// B21 纯 query URL：根方法节点下建参数。
func TestAssetB21_QueryOnlyURL(t *testing.T) {
	r := newSilentRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/?a=1", nil, "GET", nil))
	m := r.Tree.Root.FindChildByKey("GET")
	if m == nil || m.FindChildByKey("a") == nil {
		t.Error("B21 纯 query 请求应在根方法节点下建参数 a")
	}
}

// B22 同路径 GET 与 POST 共存。
func TestAssetB22_GetPostCoexist(t *testing.T) {
	r := newSilentRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/resource", nil, "GET", nil))
	r.ReverseHttpRequest(request.NewHttpRequest("/api/resource", nil, "POST", nil))
	res := r.Tree.Root.FindChildByKey("api").FindChildByKey("resource")
	if res.FindChildByKey("GET") == nil || res.FindChildByKey("POST") == nil {
		t.Error("B22 同路径 GET/POST 应共存")
	}
}

// B23 方法节点请求计数随喂入累加。
func TestAssetB23_MethodRequestCountAccumulates(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/api/x", "/api/x", "/api/x")
	m := r.Tree.Root.FindChildByKey("api").FindChildByKey("x").FindChildByKey("GET")
	if m.GetRequestCount() != 3 {
		t.Errorf("B23 方法请求数=%d want 3", m.GetRequestCount())
	}
}

// B24 变量合并发生在正确的父层，且兄弟固定段不被吞并。
func TestAssetB24_VarMergeDoesNotSwallowFixedSibling(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/api/users/1", "/api/users/2", "/api/users/3", "/api/users/me")
	users := r.Tree.Root.FindChildByKey("api").FindChildByKey("users")
	if users.GetChildByType("request_path_variable") == nil {
		t.Error("B24 数字 1/2/3 应合并为变量")
	}
	if users.FindChildByKey("me") == nil {
		t.Error("B24 固定段 me 不应被合并吞并")
	}
}

// ---------------------------------------------------------------------------
// C 类补足
// ---------------------------------------------------------------------------

// C15 负数整数：integer 模式含可选负号，3 样本即合并。
func TestAssetC15_NegativeIntegerMerge(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/api/coord/-1", "/api/coord/-2", "/api/coord/-3")
	pv := assetPathVar(t, r, "api", "coord")
	if pv == nil {
		t.Fatal("C15 负数整数应合并为变量")
	}
	if _, ok := r.NormalizeURL(request.NewHttpRequest("/api/coord/-99", nil, "GET", nil)); !ok {
		t.Error("C15 变量应能归一化负数值")
	}
}

// C16 semver 形态：公共前缀 + 整数后缀在 3 样本即按前缀模式合并。
func TestAssetC16_SemverMergeAtThree(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/api/app/v1.2.3", "/api/app/v1.2.4", "/api/app/v1.2.5")
	app := r.Tree.Root.FindChildByKey("api").FindChildByKey("app")
	if app.GetChildByType("request_path_variable") == nil {
		t.Error("C16 semver 形态应按公共前缀合并")
	}
}

// C17 字母+数字混合（匹配 alphanumeric 正则 ^[a-zA-Z]+[0-9]+$）在 3 样本合并，变量名带 _code 后缀。
func TestAssetC17_AlphaNumericMerge(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/api/codes/abc123", "/api/codes/def456", "/api/codes/ghi789")
	pv := assetPathVar(t, r, "api", "codes")
	if !strings.HasSuffix(pv.GetKey(), "_code") {
		t.Errorf("C17 alphanumeric 变量名=%s，应带 _code 后缀", pv.GetKey())
	}
}

// C18 手机号合并，变量名带 _phone 后缀。
func TestAssetC18_PhoneMerge(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/api/phones/13800000001", "/api/phones/13900000002", "/api/phones/13700000003")
	pv := assetPathVar(t, r, "api", "phones")
	if !strings.HasSuffix(pv.GetKey(), "_phone") {
		t.Errorf("C18 phone 变量名=%s，应带 _phone 后缀", pv.GetKey())
	}
}

// C19 前缀模式合并，变量名含公共前缀 user。
func TestAssetC19_PrefixPatternMergeName(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/api/items/user_1", "/api/items/user_2", "/api/items/user_3")
	pv := assetPathVar(t, r, "api", "items")
	if !strings.HasPrefix(pv.GetKey(), "user") {
		t.Errorf("C19 前缀模式变量名=%s，应含公共前缀 user", pv.GetKey())
	}
}

// C20 变量 Pattern 判定：合并后数字命中特定值、拒绝非数字。
func TestAssetC20_VarPatternEnforced(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/api/users/101", "/api/users/202", "/api/users/303")
	pv := assetPathVar(t, r, "api", "users")
	if pv.GetPattern() == nil || !pv.GetPattern().MatchString("404") {
		t.Error("C20 变量模式应命中数字 404")
	}
	if pv.GetPattern() != nil && pv.GetPattern().MatchString("abc") {
		t.Error("C20 变量模式不应命中非数字 abc")
	}
}

// C21 不同位置的变量：第二段与第三段各自独立合并。
func TestAssetC21_IndependentVarsAtDifferentDepth(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/api/a/1/x/2", "/api/a/3/x/4", "/api/a/5/x/6")
	a := r.Tree.Root.FindChildByKey("api").FindChildByKey("a")
	aVar := a.GetChildByType("request_path_variable")
	if aVar == nil {
		t.Fatal("C21 第二段 1/3/5 应合并")
	}
	x := aVar.FindChildByKey("x")
	if x == nil {
		t.Fatal("C21 变量下应有固定段 x")
	}
	if x.GetChildByType("request_path_variable") == nil {
		t.Error("C21 第四段 2/4/6 应独立合并")
	}
}

// ---------------------------------------------------------------------------
// D 类补足
// ---------------------------------------------------------------------------

// D17 路径内嵌参数与 query 同名：只建一个参数节点，且多值。
func TestAssetD17_PathParamAndQuerySameName(t *testing.T) {
	r := newSilentRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/filter=1?filter=2", nil, "GET", nil))
	m := r.Tree.Root.FindChildByKey("api").FindChildByKey("filter").FindChildByKey("GET")
	p := m.FindChildByKey("filter").(*node.RequestParamNode)
	if !p.IsMultiValue() {
		t.Error("D17 路径参数与 query 同名出现两个值应为多值")
	}
}

// D18 multipart 表单体：不 panic，且能解析出字段。
func TestAssetD18_MultipartBody(t *testing.T) {
	r := newSilentRouter()
	h := request.Headers{}
	h.Set("Content-Type", "multipart/form-data; boundary=----testboundary")
	body := []byte("------testboundary\r\nContent-Disposition: form-data; name=\"file\"\r\n\r\nhello\r\n------testboundary--\r\n")
	r.ReverseHttpRequest(request.NewHttpRequest("/api/upload", h, "POST", body))
	m := r.Tree.Root.FindChildByKey("api").FindChildByKey("upload").FindChildByKey("POST")
	if m.FindChildByKey("file") == nil {
		t.Error("D18 multipart 应解析出 file 字段")
	}
}

// D19 Accept 头规范化为值节点。
func TestAssetD19_AcceptHeaderValue(t *testing.T) {
	r := newSilentRouter()
	h := request.Headers{}
	h.Set("Accept", "application/json")
	r.ReverseHttpRequest(request.NewHttpRequest("/api/x", h, "GET", nil))
	m := r.Tree.Root.FindChildByKey("api").FindChildByKey("x").FindChildByKey("GET")
	acc := m.FindChildByKey("Accept")
	if acc == nil || acc.FindChildByKey("application/json") == nil {
		t.Error("D19 Accept 头缺失或未建值节点")
	}
}

// D20 Authorization Basic：规范化为 Basic 值节点。
func TestAssetD20_AuthorizationBasic(t *testing.T) {
	r := newSilentRouter()
	h := request.Headers{}
	h.Set("Authorization", "Basic dXNlcjpwYXNz")
	r.ReverseHttpRequest(request.NewHttpRequest("/api/sec", h, "GET", nil))
	m := r.Tree.Root.FindChildByKey("api").FindChildByKey("sec").FindChildByKey("GET")
	auth := m.FindChildByKey("Authorization")
	if auth == nil || auth.FindChildByKey("Basic") == nil {
		t.Error("D20 Authorization 应规范化为 Basic 值节点")
	}
}

// D21 Cookie 多键。
func TestAssetD21_CookieMultiKey(t *testing.T) {
	r := newSilentRouter()
	h := request.Headers{}
	h.Set("Cookie", "a=1; b=2")
	r.ReverseHttpRequest(request.NewHttpRequest("/api/x", h, "GET", nil))
	m := r.Tree.Root.FindChildByKey("api").FindChildByKey("x").FindChildByKey("GET")
	if m.FindChildByKey("a") == nil || m.FindChildByKey("b") == nil {
		t.Error("D21 Cookie 多键 a/b 缺失")
	}
}

// D22 同方法不同 Content-Type 共存。
func TestAssetD22_SameMethodDiffCTCoexist(t *testing.T) {
	r := newSilentRouter()
	for _, ct := range []string{"application/json", "application/xml"} {
		h := request.Headers{}
		h.Set("Content-Type", ct)
		var body []byte
		if ct == "application/json" {
			body = []byte(`{}`)
		} else {
			body = []byte(`<a/>`)
		}
		r.ReverseHttpRequest(request.NewHttpRequest("/api/data", h, "POST", body))
	}
	post := r.Tree.Root.FindChildByKey("api").FindChildByKey("data").FindChildByKey("POST")
	if post.FindChildByKey("application/json") == nil || post.FindChildByKey("application/xml") == nil {
		t.Error("D22 两种 Content-Type 应各自建节点")
	}
}

// D23 参数值进 ValueMetric：多次出现计数累加。
func TestAssetD23_ParamValueCount(t *testing.T) {
	r := newSilentRouter()
	assetFeed(t, r, "/api/x?status=active", "/api/x?status=active", "/api/x?status=disabled")
	m := r.Tree.Root.FindChildByKey("api").FindChildByKey("x").FindChildByKey("GET")
	p := m.FindChildByKey("status").(*node.RequestParamNode)
	if p.GetValueMetric().GetValueCount("active") != 2 {
		t.Errorf("D23 active 计数=%d want 2", p.GetValueMetric().GetValueCount("active"))
	}
}

// ---------------------------------------------------------------------------
// E 类补足
// ---------------------------------------------------------------------------

// E17 NormalizeURL 的 Host 大小写归一路由。
func TestAssetE17_NormalizeHostCase(t *testing.T) {
	rs := NewRouterSet()
	for _, id := range []string{"1", "2", "3"} {
		rs.ReverseHttpRequest(request.NewHttpRequest("http://example.com/api/users/"+id, nil, "GET", nil))
	}
	probe := request.NewHttpRequest("http://Example.com/api/users/99", nil, "GET", nil)
	probe.Host = "Example.COM"
	if _, ok := rs.NormalizeURL(probe); !ok {
		t.Error("E17 NormalizeURL 的 Host 大小写应归一化到已建桶")
	}
}

// E18 NormalizeURLs 返回兼容键（method + template，无 host）。
func TestAssetE18_NormalizeURLsCompatKey(t *testing.T) {
	rs := NewRouterSet()
	for _, id := range []string{"1", "2", "3"} {
		req := request.NewHttpRequest("http://a.com/api/users/"+id, nil, "GET", nil)
		req.Host = "a.com"
		rs.ReverseHttpRequest(req)
	}
	in := []*request.HttpRequest{
		{Url: "http://a.com/api/users/10", Host: "a.com", Method: "GET"},
	}
	norm := rs.NormalizeURLs(in)
	found := false
	for k := range norm {
		if strings.HasPrefix(k, "GET /api/users/{") {
			found = true
		}
	}
	if !found {
		t.Errorf("E18 NormalizeURLs 兼容键=%v", norm)
	}
}

// E19 ReverseHttpRequest 批量混合：坏样本不影响好样本。
func TestAssetE19_BatchMixedValidAndInvalid(t *testing.T) {
	rs := NewRouterSet()
	reqs := []*request.HttpRequest{
		request.NewHttpRequest("http://a.com/api/x", nil, "GET", nil),
		nil,
		request.NewHttpRequest("http://a.com/api/y", nil, "GET", nil),
	}
	processed, failed := 0, 0
	for _, req := range reqs {
		if err := rs.ReverseHttpRequest(req); err != nil {
			failed++
		} else {
			processed++
		}
	}
	if processed != 2 || failed != 1 {
		t.Errorf("E19 批量 processed=%d failed=%d want 2/1", processed, failed)
	}
	// 好样本已建桶
	if len(rs.Hosts()) != 1 {
		t.Errorf("E19 桶数=%v want [a.com]", rs.Hosts())
	}
}

// E20 不同 host 各自请求计数独立。
func TestAssetE20_PerHostRequestCountsIndependent(t *testing.T) {
	rs := NewRouterSet()
	for _, h := range []string{"a.com", "b.com"} {
		req := request.NewHttpRequest("http://"+h+"/api/x", nil, "GET", nil)
		req.Host = h
		rs.ReverseHttpRequest(req)
	}
	req := request.NewHttpRequest("http://a.com/api/x", nil, "GET", nil)
	req.Host = "a.com"
	rs.ReverseHttpRequest(req)
	stats := rs.Stats()
	if stats["a.com"].RequestsProcessed != 2 {
		t.Errorf("E20 a.com 请求数=%d want 2（独立于 b.com）", stats["a.com"].RequestsProcessed)
	}
}
