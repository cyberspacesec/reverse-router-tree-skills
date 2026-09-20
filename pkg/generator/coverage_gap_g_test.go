package generator

// coverage_gap_g_test.go — 补齐 pkg/generator 零覆盖块（assertion/derive/generator/spec）。
// 测试前缀统一 TestGapG_，注释简体中文。随机性全部固定种子或断言语义，不依赖概率。
// 只新增测试，不改业务代码。

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

// gapGTree 构造一棵最小路由树：root/api/users/[var]/GET，供断言导航复用。
func gapGTree(t *testing.T, withVar bool, varKey string) node.Node[node.NodeContext] {
	t.Helper()
	root := node.NewBaseNode[node.NodeContext]("root", "root", "", node.NewBaseNodeContext())
	api := node.NewRequestPathNode("api")
	users := node.NewRequestPathNode("users")
	mustG := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("挂载失败: %v", err)
		}
	}
	mustG(root.AddChild(api))
	mustG(api.AddChild(users))
	parent := node.Node[node.NodeContext](users)
	if withVar {
		vn := node.NewRequestPathVariableNode(varKey, "[0-9]+")
		mustG(users.AddChild(vn))
		parent = vn
	}
	get := node.NewRequestMethodNode("GET")
	mustG(parent.AddChild(get))
	return root
}

// --- navigate / navigateToMethod 失败分支（隔离 fake-T 触发，不污染主 T） ---

// TestGapG_NavigateVarMissing {var} 下无变量节点时Fatalf分支。
func TestGapG_NavigateVarMissing(t *testing.T) {
	root := gapGTree(t, false, "")
	ft := &testing.T{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		navigate(ft, root, []string{"api", "users", "{var}"})
	}()
	<-done
	if !ft.Failed() {
		t.Error("{var} 缺失时期望 Fatalf 标记失败")
	}
}

// TestGapG_NavigateKeyMissing 普通段缺失时Fatalf分支。
func TestGapG_NavigateKeyMissing(t *testing.T) {
	root := gapGTree(t, false, "")
	ft := &testing.T{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		navigate(ft, root, []string{"api", "users", "不存在"})
	}()
	<-done
	if !ft.Failed() {
		t.Error("缺失段时期望 Fatalf 标记失败")
	}
}

// TestGapG_NavigateToMethodMissing 方法节点缺失时Fatalf分支。
func TestGapG_NavigateToMethodMissing(t *testing.T) {
	root := gapGTree(t, false, "")
	ft := &testing.T{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		navigateToMethod(ft, root, []string{"api", "users"}, "POST")
	}()
	<-done
	if !ft.Failed() {
		t.Error("缺失方法时期望 Fatalf 标记失败")
	}
}

// --- PathVarAssertion.Check 失败分支（隔离 fake-T 断言语义） ---

// TestGapG_PathVarMergeMiss 期望合并但无变量节点。
func TestGapG_PathVarMergeMiss(t *testing.T) {
	root := gapGTree(t, false, "")
	a := &PathVarAssertion{PathSegments: []string{"api", "users"}, ExpectMerge: true, ExpectVarName: "users_id"}
	ft := &testing.T{}
	a.Check(ft, root)
	if !ft.Failed() {
		t.Error("期望合并但缺失时应报错")
	}
}

// TestGapG_PathVarWrongType 变量节点类型断言失败（挂普通路径节点冒充）。
func TestGapG_PathVarWrongType(t *testing.T) {
	root := gapGTree(t, false, "")
	users := root.FindChildByKey("api").FindChildByKey("users")
	// 用 request_path 类型节点占据 GetChildByType("request_path_variable") 无法命中，
	// 故改用自定义思路：直接挂一个 key 相同的变量位占位符不可行；
	// 此处改为覆盖 ExpectMerge=false 但出现变量的反向分支。
	vn := node.NewRequestPathVariableNode("users_id", "[0-9]+")
	if err := users.AddChild(vn); err != nil {
		t.Fatal(err)
	}
	a := &PathVarAssertion{PathSegments: []string{"api", "users"}, ExpectMerge: false}
	ft := &testing.T{}
	a.Check(ft, root)
	if !ft.Failed() {
		t.Error("期望不合并但出现变量时应报错")
	}
	_ = users
}

// TestGapG_PathVarNameMismatch 变量名不符分支。
func TestGapG_PathVarNameMismatch(t *testing.T) {
	root := gapGTree(t, true, "users_id")
	a := &PathVarAssertion{PathSegments: []string{"api", "users"}, ExpectMerge: true, ExpectVarName: "other_id"}
	ft := &testing.T{}
	a.Check(ft, root)
	if !ft.Failed() {
		t.Error("变量名不符时应报错")
	}
}

// TestGapG_PathVarPhysicalMismatch 物理类型不符分支。
func TestGapG_PathVarPhysicalMismatch(t *testing.T) {
	root := gapGTree(t, true, "users_id")
	a := &PathVarAssertion{PathSegments: []string{"api", "users"}, ExpectMerge: true,
		ExpectVarName: "users_id", ExpectPhysical: value.PhysicalTypeInteger}
	ft := &testing.T{}
	a.Check(ft, root)
	if !ft.Failed() {
		t.Error("物理类型不符时应报错")
	}
}

// TestGapG_PathVarLogicalMismatch 逻辑类型不符分支。
func TestGapG_PathVarLogicalMismatch(t *testing.T) {
	root := gapGTree(t, true, "users_id")
	a := &PathVarAssertion{PathSegments: []string{"api", "users"}, ExpectMerge: true,
		ExpectVarName: "users_id", ExpectLogical: value.LogicalTypePhoneNumber}
	ft := &testing.T{}
	a.Check(ft, root)
	if !ft.Failed() {
		t.Error("逻辑类型不符时应报错")
	}
}

// TestGapG_PathVarPatternMissing 期望有模式但实际nil分支。
func TestGapG_PathVarPatternMissing(t *testing.T) {
	root := node.NewBaseNode[node.NodeContext]("root", "root", "", node.NewBaseNodeContext())
	api := node.NewRequestPathNode("api")
	users := node.NewRequestPathNode("users")
	if err := root.AddChild(api); err != nil {
		t.Fatal(err)
	}
	if err := api.AddChild(users); err != nil {
		t.Fatal(err)
	}
	vn := node.NewRequestPathVariableNode("users_id", "")
	if err := users.AddChild(vn); err != nil {
		t.Fatal(err)
	}
	a := &PathVarAssertion{PathSegments: []string{"api", "users"}, ExpectMerge: true,
		ExpectVarName: "users_id", ExpectPatternSet: true}
	ft := &testing.T{}
	a.Check(ft, root)
	if !ft.Failed() {
		t.Error("期望模式但为nil时应报错")
	}
}

// TestGapG_PathVarFixedMissing 期望保留固定路径但缺失分支。
func TestGapG_PathVarFixedMissing(t *testing.T) {
	root := gapGTree(t, true, "users_id")
	a := &PathVarAssertion{PathSegments: []string{"api", "users"}, ExpectMerge: true,
		ExpectVarName: "users_id", ExpectFixedKept: []string{"list"}}
	ft := &testing.T{}
	a.Check(ft, root)
	if !ft.Failed() {
		t.Error("固定路径缺失时应报错")
	}
}

// --- ParamAssertion.Check 失败分支 ---

// TestGapG_ParamMissing 参数缺失分支。
func TestGapG_ParamMissing(t *testing.T) {
	root := gapGTree(t, false, "")
	a := &ParamAssertion{PathSegments: []string{"api", "users"}, Method: "GET", ParamName: "page"}
	ft := &testing.T{}
	a.Check(ft, root)
	if !ft.Failed() {
		t.Error("参数缺失时应报错")
	}
}

// TestGapG_ParamTypeMismatch 参数节点类型断言失败（挂方法节点冒充同名参数）。
func TestGapG_ParamTypeMismatch(t *testing.T) {
	root := gapGTree(t, false, "")
	get := root.FindChildByKey("api").FindChildByKey("users").FindChildByKey("GET")
	if err := get.AddChild(node.NewRequestMethodNode("page")); err != nil {
		t.Fatal(err)
	}
	a := &ParamAssertion{PathSegments: []string{"api", "users"}, Method: "GET", ParamName: "page"}
	ft := &testing.T{}
	a.Check(ft, root)
	if !ft.Failed() {
		t.Error("参数类型断言失败时应报错")
	}
}

// TestGapG_ParamPhysicalMismatch 参数物理类型不符。
func TestGapG_ParamPhysicalMismatch(t *testing.T) {
	root := gapGTree(t, false, "")
	get := root.FindChildByKey("api").FindChildByKey("users").FindChildByKey("GET")
	p := node.NewRequestParamNode("page", "1", false)
	p.SetValueType(value.Type(value.PhysicalTypeString))
	p.SetLogicalType(value.LogicalTypeString)
	if err := get.AddChild(p); err != nil {
		t.Fatal(err)
	}
	a := &ParamAssertion{PathSegments: []string{"api", "users"}, Method: "GET", ParamName: "page",
		ExpectPhysical: value.PhysicalTypeInteger}
	ft := &testing.T{}
	a.Check(ft, root)
	if !ft.Failed() {
		t.Error("参数物理类型不符时应报错")
	}
}

// TestGapG_ParamLogicalMismatch 参数逻辑类型不符。
func TestGapG_ParamLogicalMismatch(t *testing.T) {
	root := gapGTree(t, false, "")
	get := root.FindChildByKey("api").FindChildByKey("users").FindChildByKey("GET")
	p := node.NewRequestParamNode("page", "1", false)
	p.SetValueType(value.Type(value.PhysicalTypeInteger))
	p.SetLogicalType(value.LogicalTypeString)
	if err := get.AddChild(p); err != nil {
		t.Fatal(err)
	}
	a := &ParamAssertion{PathSegments: []string{"api", "users"}, Method: "GET", ParamName: "page",
		ExpectPhysical: value.PhysicalTypeInteger, ExpectLogical: value.LogicalTypeInteger}
	ft := &testing.T{}
	a.Check(ft, root)
	if !ft.Failed() {
		t.Error("参数逻辑类型不符时应报错")
	}
}

// TestGapG_ParamRequiredMismatch 必需性不符分支。
func TestGapG_ParamRequiredMismatch(t *testing.T) {
	root := gapGTree(t, false, "")
	get := root.FindChildByKey("api").FindChildByKey("users").FindChildByKey("GET")
	if err := get.AddChild(node.NewRequestParamNode("page", "1", false)); err != nil {
		t.Fatal(err)
	}
	a := &ParamAssertion{PathSegments: []string{"api", "users"}, Method: "GET", ParamName: "page",
		ExpectRequired: true}
	ft := &testing.T{}
	a.Check(ft, root)
	if !ft.Failed() {
		t.Error("必需性不符时应报错")
	}
}

// --- MethodAssertion / ContentTypeAssertion / HeaderAssertion / CookieAssertion / StatsAssertion ---

// TestGapG_MethodExistMiss 期望存在但缺失。
func TestGapG_MethodExistMiss(t *testing.T) {
	root := gapGTree(t, false, "")
	a := &MethodAssertion{PathSegments: []string{"api", "users"}, Method: "POST", ExpectExists: true}
	ft := &testing.T{}
	a.Check(ft, root)
	if !ft.Failed() {
		t.Error("期望方法存在但缺失时应报错")
	}
}

// TestGapG_MethodNotExistHit 期望不存在但出现。
func TestGapG_MethodNotExistHit(t *testing.T) {
	root := gapGTree(t, false, "")
	a := &MethodAssertion{PathSegments: []string{"api", "users"}, Method: "GET", ExpectExists: false}
	ft := &testing.T{}
	a.Check(ft, root)
	if !ft.Failed() {
		t.Error("期望方法不存在但出现时应报错")
	}
}

// TestGapG_CTMissing Content-Type缺失分支。
func TestGapG_CTMissing(t *testing.T) {
	root := gapGTree(t, false, "")
	a := &ContentTypeAssertion{PathSegments: []string{"api", "users"}, Method: "GET", CT: "application/json"}
	ft := &testing.T{}
	a.Check(ft, root)
	if !ft.Failed() {
		t.Error("CT缺失时应报错")
	}
}

// TestGapG_CTWrongType CT节点类型错误分支。
func TestGapG_CTWrongType(t *testing.T) {
	root := gapGTree(t, false, "")
	get := root.FindChildByKey("api").FindChildByKey("users").FindChildByKey("GET")
	if err := get.AddChild(node.NewRequestMethodNode("application/json")); err != nil {
		t.Fatal(err)
	}
	a := &ContentTypeAssertion{PathSegments: []string{"api", "users"}, Method: "GET", CT: "application/json"}
	ft := &testing.T{}
	a.Check(ft, root)
	if !ft.Failed() {
		t.Error("CT类型错误时应报错")
	}
}

// TestGapG_HeaderMissing Header名称缺失分支。
func TestGapG_HeaderMissing(t *testing.T) {
	root := gapGTree(t, false, "")
	a := &HeaderAssertion{PathSegments: []string{"api", "users"}, Method: "GET", HeaderName: "Accept"}
	ft := &testing.T{}
	a.Check(ft, root)
	if !ft.Failed() {
		t.Error("Header缺失时应报错")
	}
}

// TestGapG_HeaderWrongType Header类型错误分支。
func TestGapG_HeaderWrongType(t *testing.T) {
	root := gapGTree(t, false, "")
	get := root.FindChildByKey("api").FindChildByKey("users").FindChildByKey("GET")
	if err := get.AddChild(node.NewRequestMethodNode("Accept")); err != nil {
		t.Fatal(err)
	}
	a := &HeaderAssertion{PathSegments: []string{"api", "users"}, Method: "GET", HeaderName: "Accept"}
	ft := &testing.T{}
	a.Check(ft, root)
	if !ft.Failed() {
		t.Error("Header类型错误时应报错")
	}
}

// TestGapG_HeaderValueMissing Header值缺失分支。
func TestGapG_HeaderValueMissing(t *testing.T) {
	root := gapGTree(t, false, "")
	get := root.FindChildByKey("api").FindChildByKey("users").FindChildByKey("GET")
	if err := get.AddChild(node.NewRequestHeaderNode("Accept")); err != nil {
		t.Fatal(err)
	}
	a := &HeaderAssertion{PathSegments: []string{"api", "users"}, Method: "GET",
		HeaderName: "Accept", ExpectNormValues: []string{"application/json"}}
	ft := &testing.T{}
	a.Check(ft, root)
	if !ft.Failed() {
		t.Error("Header值缺失时应报错")
	}
}

// TestGapG_CookieMissing Cookie名称缺失分支。
func TestGapG_CookieMissing(t *testing.T) {
	root := gapGTree(t, false, "")
	a := &CookieAssertion{PathSegments: []string{"api", "users"}, Method: "GET", CookieName: "lang"}
	ft := &testing.T{}
	a.Check(ft, root)
	if !ft.Failed() {
		t.Error("Cookie缺失时应报错")
	}
}

// TestGapG_CookieWrongType Cookie类型错误分支。
func TestGapG_CookieWrongType(t *testing.T) {
	root := gapGTree(t, false, "")
	get := root.FindChildByKey("api").FindChildByKey("users").FindChildByKey("GET")
	if err := get.AddChild(node.NewRequestMethodNode("lang")); err != nil {
		t.Fatal(err)
	}
	a := &CookieAssertion{PathSegments: []string{"api", "users"}, Method: "GET", CookieName: "lang"}
	ft := &testing.T{}
	a.Check(ft, root)
	if !ft.Failed() {
		t.Error("Cookie类型错误时应报错")
	}
}

// TestGapG_StatsAllMiss 统计五项下界分支一次触发。
func TestGapG_StatsAllMiss(t *testing.T) {
	root := node.NewBaseNode[node.NodeContext]("root", "root", "", node.NewBaseNodeContext())
	a := &StatsAssertion{MinMethods: 1, MinParams: 1, MinPathVars: 1, MinTotalReqs: 100, MinTotalNodes: 100}
	ft := &testing.T{}
	a.Check(ft, root)
	if !ft.Failed() {
		t.Error("统计下界全缺失时应报错")
	}
}

// TestGapG_SpecAssertionsNoMerge 不合并路径变量走Values[0]固定路径分支。
func TestGapG_SpecAssertionsNoMerge(t *testing.T) {
	pv := &PathVarSpec{Pattern: patternFixedWords, Values: []string{"admin", "manager", "guest"}}
	deriveExpectations(pv, "roles", 3)
	if pv.ExpectMerge {
		t.Fatal("fixed_words 不应合并")
	}
	s := &Spec{Seed: 1, Resources: []*Resource{{
		Name: "roles", Prefix: []string{"api"},
		Operations: []*Operation{{Method: "GET", Kind: OpGetOne, PathVar: pv, Repeat: 3}},
	}}}
	asserts := s.Assertions()
	found := false
	for _, a := range asserts {
		if ma, ok := a.(*MethodAssertion); ok {
			found = true
			if len(ma.PathSegments) != 3 || ma.PathSegments[2] != "admin" {
				t.Errorf("不合并时方法父段应为固定首值，实际 %v", ma.PathSegments)
			}
		}
	}
	if !found {
		t.Error("应产出方法断言")
	}
}

// --- derive.go 零覆盖分支 ---

// TestGapG_PrefixNoCommon 无公共前缀时回退parentKey分支。
func TestGapG_PrefixNoCommon(t *testing.T) {
	pv := &PathVarSpec{Pattern: patternPrefix, Values: []string{"abc", "def", "ghi"}}
	deriveExpectations(pv, "users", 3)
	if !pv.ExpectMerge {
		t.Fatal("无公共前缀仍应合并")
	}
	if pv.ExpectVarName != "users_id" {
		t.Errorf("无公共前缀应回退parentKey，实际 %q", pv.ExpectVarName)
	}
}

// TestGapG_SuffixNoCommon 无公共后缀时回退parentKey分支。
func TestGapG_SuffixNoCommon(t *testing.T) {
	pv := &PathVarSpec{Pattern: patternSuffix, Values: []string{"abc", "def", "ghi"}}
	deriveExpectations(pv, "users", 3)
	if !pv.ExpectMerge {
		t.Fatal("无公共后缀仍应合并")
	}
	if pv.ExpectVarName != "users_id" {
		t.Errorf("无公共后缀应回退parentKey，实际 %q", pv.ExpectVarName)
	}
}

// TestGapG_CommonPrefixEmpty 空切片与最短串分支。
func TestGapG_CommonPrefixEmpty(t *testing.T) {
	if got := commonPrefixBase(nil); got != "" {
		t.Errorf("空切片应返回空，实际 %q", got)
	}
	if got := longestCommonPrefix(nil); got != "" {
		t.Errorf("空切片最长前缀应为空，实际 %q", got)
	}
	// 最短串在后：min切换分支
	if got := longestCommonPrefix([]string{"abcdef", "abc"}); got != "abc" {
		t.Errorf("最短串分支期望 abc，实际 %q", got)
	}
	// 首字符即分歧：返回空前缀后trim分支
	pv := &PathVarSpec{Pattern: patternPrefix, Values: []string{"x1", "y2", "z3"}}
	deriveExpectations(pv, "users", 3)
	if pv.ExpectVarName != "users_id" {
		t.Errorf("零公共前缀应回退，实际 %q", pv.ExpectVarName)
	}
}

// TestGapG_LongestPrefixMismatch 中途分歧返回分支。
func TestGapG_LongestPrefixMismatch(t *testing.T) {
	got := longestCommonPrefix([]string{"user_001", "user_002", "user_x03"})
	if got != "user_" {
		t.Errorf("期望公共前缀 user_，实际 %q", got)
	}
	// 第二个最短：min取values[1]分支
	got2 := longestCommonPrefix([]string{"user_00123", "user_00", "user_00456"})
	if got2 != "user_00" {
		t.Errorf("期望 user_00，实际 %q", got2)
	}
}

// TestGapG_CommonSuffixEmpty 空切片与无公共后缀分支。
func TestGapG_CommonSuffixEmpty(t *testing.T) {
	if got := commonSuffixBase(nil); got != "" {
		t.Errorf("空切片应返回空，实际 %q", got)
	}
	if got := longestCommonSuffix(nil); got != "" {
		t.Errorf("空切片最长后缀应为空，实际 %q", got)
	}
	if got := longestCommonSuffix([]string{"abc", "def", "ghi"}); got != "" {
		t.Errorf("无公共后缀应为空，实际 %q", got)
	}
}

// TestGapG_LongestSuffixMinSwitch 最短串切换与长度不足分支。
func TestGapG_LongestSuffixMinSwitch(t *testing.T) {
	// min取后位元素分支
	got := longestCommonSuffix([]string{"aaa_user", "02_user", "3_user"})
	if got != "_user" {
		t.Errorf("期望 _user，实际 %q", got)
	}
	// v长度不足i分支（len(v)<i）
	got2 := longestCommonSuffix([]string{"ab", "xxab", "ab"})
	if got2 != "ab" {
		t.Errorf("期望 ab，实际 %q", got2)
	}
}

// TestGapG_TrimLeadingDigitsNoDigits 无前导数字直接返回分支。
func TestGapG_TrimLeadingDigitsNoDigits(t *testing.T) {
	if got := trimLeadingDigits("_user"); got != "_user" {
		t.Errorf("无数字应原样返回，实际 %q", got)
	}
	if got := trimLeadingDigits(""); got != "" {
		t.Errorf("空串应返回空，实际 %q", got)
	}
}

// TestGapG_NormalizeEmpty 空值直接返回分支。
func TestGapG_NormalizeEmpty(t *testing.T) {
	if got := normalizeHeaderValue("Accept", ""); got != "" {
		t.Errorf("空值应返回空，实际 %q", got)
	}
}

// TestGapG_AcceptSemicolon 无逗号但有分号分支。
func TestGapG_AcceptSemicolon(t *testing.T) {
	got := normalizeHeaderValue("Accept", "application/json;q=0.9")
	if got != "application/json" {
		t.Errorf("分号去因子期望 application/json，实际 %q", got)
	}
}

// TestGapG_AuthorizationNoSpace 无空格单token分支（len(parts)>0恒真但需覆盖）。
func TestGapG_AuthorizationNoSpace(t *testing.T) {
	got := normalizeHeaderValue("Authorization", "TokenOnly")
	if got != "TokenOnly" {
		t.Errorf("无空格应原样返回，实际 %q", got)
	}
}

// TestGapG_LangSemicolon 无逗号但有分号分支。
func TestGapG_LangSemicolon(t *testing.T) {
	got := normalizeHeaderValue("Accept-Language", "zh-CN;q=0.9")
	if got != "zh-CN" {
		t.Errorf("分号去因子期望 zh-CN，实际 %q", got)
	}
}

// TestGapG_NormalizePassthrough 透传分支（X-Api-Version/X-Requested-With/default）。
func TestGapG_NormalizePassthrough(t *testing.T) {
	if got := normalizeHeaderValue("X-Api-Version", "v9"); got != "v9" {
		t.Errorf("版本应透传，实际 %q", got)
	}
	if got := normalizeHeaderValue("X-Requested-With", "XMLHttpRequest"); got != "XMLHttpRequest" {
		t.Errorf("XHR应透传，实际 %q", got)
	}
	if got := normalizeHeaderValue("X-Custom", "zzz"); got != "zzz" {
		t.Errorf("default应透传，实际 %q", got)
	}
}

// TestGapG_MarshalNil nil body分支。
func TestGapG_MarshalNil(t *testing.T) {
	data, err := marshalBody(nil)
	if err != nil || data != nil {
		t.Errorf("nil body应返回nil,nil，实际 %v %v", data, err)
	}
}

// TestGapG_MarshalUnsupported 非法Content-Type分支。
func TestGapG_MarshalUnsupported(t *testing.T) {
	if _, err := marshalBody(&BodySpec{ContentType: "text/plain"}); err == nil {
		t.Error("非法CT应报错")
	}
}

// TestGapG_SpecRequestsRepeatZero Repeat<1回退1分支。
func TestGapG_SpecRequestsRepeatZero(t *testing.T) {
	s := &Spec{Seed: 7, Resources: []*Resource{{
		Name: "users", Prefix: []string{"api"},
		Operations: []*Operation{{Method: "GET", Kind: OpList, Repeat: 0}},
	}}}
	reqs := s.Requests(rand.New(rand.NewSource(7)))
	if len(reqs) != 1 {
		t.Errorf("Repeat=0应回退1个请求，实际 %d", len(reqs))
	}
}

// TestGapG_BuildRequestCookieMerge 多Cookie拼接existing分支。
func TestGapG_BuildRequestCookieMerge(t *testing.T) {
	res := &Resource{Name: "users", Prefix: []string{"api"}}
	op := &Operation{Method: "GET", Kind: OpList, Repeat: 2,
		Cookies: []*CookieSpec{{Name: "a", Values: []string{"1"}}, {Name: "b", Values: []string{"2"}}}}
	req := buildRequest(res, op, 0, rand.New(rand.NewSource(1)))
	cookie := req.Headers.Get("Cookie")
	if !strings.Contains(cookie, "a=1") || !strings.Contains(cookie, "b=2") {
		t.Errorf("多Cookie应拼接，实际 %q", cookie)
	}
}

// TestGapG_BuildRequestMethodDefault 空方法回退GET分支。
func TestGapG_BuildRequestMethodDefault(t *testing.T) {
	res := &Resource{Name: "users", Prefix: []string{"api"}}
	op := &Operation{Kind: OpList, Repeat: 1}
	req := buildRequest(res, op, 0, rand.New(rand.NewSource(1)))
	if req.Method != "GET" {
		t.Errorf("空方法应回退GET，实际 %q", req.Method)
	}
	if !strings.HasPrefix(req.Url, "/api/users") {
		t.Errorf("路径异常: %q", req.Url)
	}
}

// --- generator.go 零覆盖分支 ---

// TestGapG_GenPathVarSimilarNoBreak genPathVar走similar不突破分支（固定种子扫描）。
func TestGapG_GenPathVarSimilarNoBreak(t *testing.T) {
	hitBreak := false
	hitNoBreak := false
	for seed := int64(0); seed < 200 && !(hitBreak && hitNoBreak); seed++ {
		g := NewGenerator(seed)
		pv := g.genPathVar("users")
		if pv.Pattern != patternSimilarLength {
			continue
		}
		if len(pv.Values) >= 6 {
			hitBreak = true
		} else {
			hitNoBreak = true
		}
	}
	if !hitNoBreak {
		t.Error("200种子内应命中similar不突破分支")
	}
	if !hitBreak {
		t.Error("200种子内应命中similar突破分支")
	}
}

// TestGapG_GenPathVarFixedWords genPathVar走fixed分支。
func TestGapG_GenPathVarFixedWords(t *testing.T) {
	found := false
	for seed := int64(0); seed < 500 && !found; seed++ {
		g := NewGenerator(seed)
		if pv := g.genPathVar("users"); pv.Pattern == patternFixedWords {
			found = true
			if len(pv.Values) != 3 {
				t.Errorf("fixed应3个值，实际 %v", pv.Values)
			}
		}
	}
	if !found {
		t.Error("500种子内应命中fixed分支")
	}
}

// TestGapG_PickAtLeastCap k>len防御分支（min==len时Intn(1)=0，k==len不触发；直接大min验证）。
func TestGapG_PickAtLeastCap(t *testing.T) {
	defer func() {
		// Intn panic属于非法调用，不应发生；此处仅防意外
		if r := recover(); r != nil {
			t.Fatalf("不应panic: %v", r)
		}
	}()
	rnd := rand.New(rand.NewSource(3))
	got := pickAtLeast(rnd, allCRUD, 5)
	if len(got) != 5 {
		t.Errorf("min==len应全取，实际 %d", len(got))
	}
}

// TestGapG_PickUniqueTruncate n>len截断分支。
func TestGapG_PickUniqueTruncate(t *testing.T) {
	rnd := rand.New(rand.NewSource(3))
	got := pickUniqueBodyFieldNames(rnd, 100)
	if len(got) != 6 {
		t.Errorf("超限应截断为池长6，实际 %d", len(got))
	}
	seen := map[string]bool{}
	for _, n := range got {
		if seen[n] {
			t.Errorf("截断结果不应重复: %v", got)
		}
		seen[n] = true
	}
}

// TestGapG_HeaderRawDefault 未知Header走default分支。
func TestGapG_HeaderRawDefault(t *testing.T) {
	got := genHeaderRawValues(rand.New(rand.NewSource(1)), "X-Unknown")
	if len(got) != 1 || got[0] != "x" {
		t.Errorf("未知Header应返回[x]，实际 %v", got)
	}
}

// --- spec.go String 未覆盖行（57-58：pathvar为nil时的query/body/header/cookie明细行） ---

// TestGapG_SpecStringDetails 无pathvar但有query/body/header/cookie的dump分支。
func TestGapG_SpecStringDetails(t *testing.T) {
	s := &Spec{Seed: 9, Resources: []*Resource{{
		Name: "users", Prefix: []string{"api"},
		Operations: []*Operation{{
			Method: "GET", Kind: OpList, Repeat: 2,
			QueryParams: []*QueryParamSpec{{Name: "page", Values: []string{"1"}, Presence: 1.0,
				ExpectType: value.PhysicalTypeInteger, ExpectLogic: value.LogicalTypeString, ExpectRequired: true}},
			Body:    &BodySpec{ContentType: "application/json", Fields: []*BodyFieldSpec{{Name: "name", Values: []string{"a"}}}},
			Headers: []*HeaderSpec{{Name: "Accept", Values: []string{"application/json"}, ExpectNormValues: []string{"application/json"}}},
			Cookies: []*CookieSpec{{Name: "lang", Values: []string{"zh"}}},
		}},
	}}}
	dump := s.String()
	for _, want := range []string{"query page", "body ct=", "header Accept", "cookie lang"} {
		if !strings.Contains(dump, want) {
			t.Errorf("dump应含 %q，实际:\n%s", want, dump)
		}
	}
}

// gapGVarImpostor 冒充路径变量类型：GetType 返回 request_path_variable，
// 但具体类型不是 *node.RequestPathVariableNode，用于覆盖 PathVarAssertion
// 中类型断言失败的防御分支（GetChildByType 按类型索引命中，断言失败）。
type gapGVarImpostor struct {
	node.Node[node.NodeContext]
}

// GetType 覆写为路径变量类型。
func (s *gapGVarImpostor) GetType() string { return "request_path_variable" }

// TestGapG_PathVarTypeAssertFail 变量节点类型断言失败分支。
func TestGapG_PathVarTypeAssertFail(t *testing.T) {
	root := gapGTree(t, false, "")
	users := root.FindChildByKey("api").FindChildByKey("users")
	imp := &gapGVarImpostor{Node: node.NewBaseNode[node.NodeContext]("other", "impostor", "", node.NewBaseNodeContext())}
	if err := users.AddChild(imp); err != nil {
		t.Fatal(err)
	}
	a := &PathVarAssertion{PathSegments: []string{"api", "users"}, ExpectMerge: true, ExpectVarName: "impostor"}
	ft := &testing.T{}
	a.Check(ft, root)
	if !ft.Failed() {
		t.Error("类型断言失败时应报错")
	}
}

// TestGapG_SpecAssertionsEmptyValues PathVar 不合并且 Values 为空时回退 resSegs 分支。
func TestGapG_SpecAssertionsEmptyValues(t *testing.T) {
	pv := &PathVarSpec{Pattern: "unknown_pattern"}
	deriveExpectations(pv, "roles", 0)
	if pv.ExpectMerge {
		t.Fatal("未知模式不应合并")
	}
	pv.Values = nil // 显式清空，触发 else 回退分支
	s := &Spec{Seed: 2, Resources: []*Resource{{
		Name: "roles", Prefix: []string{"api"},
		Operations: []*Operation{{Method: "GET", Kind: OpGetOne, PathVar: pv, Repeat: 1}},
	}}}
	asserts := s.Assertions()
	found := false
	for _, a := range asserts {
		if ma, ok := a.(*MethodAssertion); ok {
			found = true
			if len(ma.PathSegments) != 2 || ma.PathSegments[0] != "api" || ma.PathSegments[1] != "roles" {
				t.Errorf("空值时方法父段应回退 resSegs，实际 %v", ma.PathSegments)
			}
		}
	}
	if !found {
		t.Error("应产出方法断言")
	}
}

// TestGapG_TrimLeadingDigitsWithDigits 前导数字循环体分支。
func TestGapG_TrimLeadingDigitsWithDigits(t *testing.T) {
	if got := trimLeadingDigits("123_user"); got != "_user" {
		t.Errorf("期望 _user，实际 %q", got)
	}
	if got := trimLeadingDigits("007"); got != "" {
		t.Errorf("全数字应 strip 为空，实际 %q", got)
	}
}

// TestGapG_OpKindUnknown 未知 OpKind 走 default 分支。
func TestGapG_OpKindUnknown(t *testing.T) {
	if got := OpKind(99).String(); got != "Unknown" {
		t.Errorf("未知 OpKind 应返回 Unknown，实际 %q", got)
	}
	// 回归：已知值不受影响
	if got := OpDelete.String(); got != "Delete" {
		t.Errorf("OpDelete 应为 Delete，实际 %q", got)
	}
}
func TestGapG_FullGenerateDeterministic(t *testing.T) {
	g := NewGenerator(DefaultSeed)
	spec := g.Generate()
	rnd := rand.New(rand.NewSource(spec.Seed))
	var _ []*request.HttpRequest = spec.Requests(rnd)
	asserts := spec.Assertions()
	if len(asserts) == 0 {
		t.Fatal("断言不应为空")
	}
	// 仅验证断言构造不panic且类型齐全（真实Check由e2e覆盖）
	seen := map[string]bool{}
	for _, a := range asserts {
		switch a.(type) {
		case *MethodAssertion:
			seen["method"] = true
		case *PathVarAssertion:
			seen["pathvar"] = true
		case *ParamAssertion:
			seen["param"] = true
		case *ContentTypeAssertion:
			seen["ct"] = true
		case *HeaderAssertion:
			seen["header"] = true
		case *CookieAssertion:
			seen["cookie"] = true
		case *StatsAssertion:
			seen["stats"] = true
		}
	}
	if !seen["method"] || !seen["stats"] {
		t.Errorf("断言类型缺失: %v", seen)
	}
}
