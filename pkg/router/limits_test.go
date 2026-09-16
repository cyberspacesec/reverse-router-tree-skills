package router

// limits_test.go — P0 生产护栏单测：资源上限、脱敏、RouterSet 容量治理。

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
)

// feedOne 喂单条请求，错误直接 fatal。
func feedOne(t *testing.T, r *ReverseRouter, req *request.HttpRequest) {
	t.Helper()
	if err := r.ReverseHttpRequest(req); err != nil {
		t.Fatalf("喂入失败: %v", err)
	}
}

// --- MaxChildrenPerNode ---

func TestLimits_MaxChildrenPerNodeBlocksNew(t *testing.T) {
	r := newSilentRouter()
	r.SetResourceLimits(ResourceLimits{MaxChildrenPerNode: 3})
	feedOne(t, r, request.NewHttpRequest("/a", nil, "GET", nil))
	feedOne(t, r, request.NewHttpRequest("/b", nil, "GET", nil))
	feedOne(t, r, request.NewHttpRequest("/c", nil, "GET", nil))
	// 第 4 个新段应被拒绝（fail-soft 返回错误）
	if err := r.ReverseHttpRequest(request.NewHttpRequest("/d", nil, "GET", nil)); err == nil {
		t.Fatal("超限新建子节点应返回错误")
	}
	if r.Tree.Root.FindChildByKey("d") != nil {
		t.Error("超限节点 d 不应被创建")
	}
	// 已存在节点仍可正常命中
	feedOne(t, r, request.NewHttpRequest("/a", nil, "GET", nil))
	if s := r.GetStats(); s.Warnings == 0 {
		t.Error("超限拒绝应计 Warnings")
	}
}

func TestLimits_MaxChildrenZeroMeansUnlimited(t *testing.T) {
	r := newSilentRouter()
	r.SetResourceLimits(ResourceLimits{})
	// 关合并：seg0/seg1… 会触发前缀合并，50 段会被收成 1 个变量节点
	cfg := DefaultMergeConfig
	cfg.SiblingMergeThreshold = 100000
	r.SetMergeConfig(cfg)
	for i := 0; i < 50; i++ {
		feedOne(t, r, request.NewHttpRequest(fmt.Sprintf("/seg%d", i), nil, "GET", nil))
	}
	if r.Tree.Root.GetChildCount() != 50 {
		t.Errorf("0=不限制，应建 50 节点，实际 %d", r.Tree.Root.GetChildCount())
	}
}

// --- MaxValuesPerMetric ---

func TestLimits_MaxValuesPerMetricCapped(t *testing.T) {
	r := newSilentRouter()
	r.SetResourceLimits(ResourceLimits{MaxValuesPerMetric: 5})
	for i := 0; i < 10; i++ {
		feedOne(t, r, request.NewHttpRequest(fmt.Sprintf("/api/x?v=%d", i), nil, "GET", nil))
	}
	m := r.Tree.Root.FindChildByKey("api").FindChildByKey("x").FindChildByKey("GET")
	p := m.FindChildByKey("v").(*node.RequestParamNode)
	metric := p.GetValueMetric()
	if got := metric.GetUniqueValueCount(); got > 6 { // 5 原值 + [capped] 占位
		t.Errorf("unique 值数=%d，应 ≤6（含占位桶）", got)
	}
	if metric.CappedDropped() == 0 {
		t.Error("超限新值应计入 CappedDropped")
	}
	if metric.GetValueCount(CappedMetricKey) == 0 {
		t.Errorf("占位桶 %q 应有计数", CappedMetricKey)
	}
}

// --- MaxSegmentLen ---

func TestLimits_MaxSegmentLenTruncates(t *testing.T) {
	r := newSilentRouter()
	r.SetResourceLimits(ResourceLimits{MaxSegmentLen: 16})
	longSeg := strings.Repeat("a", 100)
	feedOne(t, r, request.NewHttpRequest("/api/"+longSeg, nil, "GET", nil))
	api := r.Tree.Root.FindChildByKey("api")
	if api == nil {
		t.Fatal("api 节点缺失")
	}
	found := false
	for _, c := range api.GetChildren() {
		if len(c.GetKey()) == 16 {
			found = true
		}
		if len(c.GetKey()) > 16 {
			t.Errorf("截断后段长=%d，不应超过 16", len(c.GetKey()))
		}
	}
	if !found {
		t.Error("应存在截断到 16 字节的段节点")
	}
}

// --- 参数脱敏 ---

func TestRedact_ParamPasswordNotStored(t *testing.T) {
	r := newSilentRouter() // 默认名单含 password
	feedOne(t, r, request.NewHttpRequest("/api/login?password=secret123&user=alice", nil, "GET", nil))
	m := r.Tree.Root.FindChildByKey("api").FindChildByKey("login").FindChildByKey("GET")
	pw := m.FindChildByKey("password").(*node.RequestParamNode)
	metric := pw.GetValueMetric()
	if metric.GetValueCount("secret123") != 0 {
		t.Error("脱敏参数原值不应存入 ValueMetric")
	}
	// 非脱敏参数不受影响
	u := m.FindChildByKey("user").(*node.RequestParamNode)
	if u.GetValueMetric().GetValueCount("alice") != 1 {
		t.Error("非脱敏参数 user=alice 应正常记录")
	}
}

func TestRedact_ParamCaseInsensitive(t *testing.T) {
	r := newSilentRouter()
	feedOne(t, r, request.NewHttpRequest("/api/login?Password=S3cret", nil, "GET", nil))
	m := r.Tree.Root.FindChildByKey("api").FindChildByKey("login").FindChildByKey("GET")
	pw := m.FindChildByKey("password").(*node.RequestParamNode)
	if pw.GetValueMetric().GetValueCount("S3cret") != 0 {
		t.Error("脱敏名单大小写不敏感：Password 原值不应存储")
	}
}

func TestRedact_CustomListAndDisable(t *testing.T) {
	r := newSilentRouter()
	r.SetRedactConfig(RedactConfig{Params: []string{"token"}})
	feedOne(t, r, request.NewHttpRequest("/api/x?token=abc&password=plain", nil, "GET", nil))
	m := r.Tree.Root.FindChildByKey("api").FindChildByKey("x").FindChildByKey("GET")
	if m.FindChildByKey("token").(*node.RequestParamNode).GetValueMetric().GetValueCount("abc") != 0 {
		t.Error("自定义名单 token 应脱敏")
	}
	if m.FindChildByKey("password").(*node.RequestParamNode).GetValueMetric().GetValueCount("plain") != 1 {
		t.Error("自定义名单替换默认名单后 password 不再脱敏，应正常记录")
	}
}

// --- Cookie 脱敏 ---

func TestRedact_CookieSessionIDPlaceholder(t *testing.T) {
	r := newSilentRouter() // 默认名单含 sessionid
	h := request.Headers{}
	h.Set("Cookie", "sessionid=deadbeef; theme=dark")
	feedOne(t, r, request.NewHttpRequest("/api/home", h, "GET", nil))
	m := r.Tree.Root.FindChildByKey("api").FindChildByKey("home").FindChildByKey("GET")
	sess := m.FindChildByKey("sessionid")
	if sess == nil {
		t.Fatal("sessionid 分组节点应保留（只记结构）")
	}
	if sess.FindChildByKey("deadbeef") != nil {
		t.Error("脱敏 cookie 原值不应建值节点")
	}
	if sess.FindChildByKey(RedactedCookieValue) == nil {
		t.Errorf("脱敏 cookie 应建占位值节点 %q", RedactedCookieValue)
	}
	if m.FindChildByKey("theme").FindChildByKey("dark") == nil {
		t.Error("非脱敏 cookie theme=dark 应正常记录")
	}
}

// --- 脱敏不影响归一化 ---

func TestRedact_NormalizeUnaffected(t *testing.T) {
	r := newSilentRouter()
	feedOne(t, r, request.NewHttpRequest("/api/login?password=x&user=alice", nil, "GET", nil))
	n, ok := r.NormalizeURL(request.NewHttpRequest("/api/login?password=y&user=bob", nil, "GET", nil))
	if !ok {
		t.Fatal("脱敏不应影响路由归一化命中")
	}
	if n.Template != "/api/login" {
		t.Errorf("Template=%s want /api/login", n.Template)
	}
}

// --- RouterSet 容量治理 ---

func hostReq(host, url string) *request.HttpRequest {
	req := request.NewHttpRequest(url, nil, "GET", nil)
	req.Host = host
	return req
}

func TestRouterSet_MaxHostsRejectsNewBucket(t *testing.T) {
	rs := NewRouterSet()
	rs.SetMaxHosts(2)
	if err := rs.ReverseHttpRequest(hostReq("a.com", "http://a.com/x")); err != nil {
		t.Fatal(err)
	}
	if err := rs.ReverseHttpRequest(hostReq("b.com", "http://b.com/x")); err != nil {
		t.Fatal(err)
	}
	if err := rs.ReverseHttpRequest(hostReq("c.com", "http://c.com/x")); err == nil {
		t.Fatal("超 MaxHosts 的新 host 建桶应失败")
	}
	if len(rs.Hosts()) != 2 {
		t.Errorf("桶数=%d want 2", len(rs.Hosts()))
	}
	// 已有桶仍可喂入
	if err := rs.ReverseHttpRequest(hostReq("a.com", "http://a.com/y")); err != nil {
		t.Errorf("已有桶喂入不应受影响: %v", err)
	}
}

func TestRouterSet_MaxHostsZeroUnlimited(t *testing.T) {
	rs := NewRouterSet()
	rs.SetMaxHosts(0)
	for i := 0; i < 10; i++ {
		h := fmt.Sprintf("h%d.com", i)
		if err := rs.ReverseHttpRequest(hostReq(h, "http://"+h+"/x")); err != nil {
			t.Fatalf("0=不限制，%s 建桶失败: %v", h, err)
		}
	}
	if len(rs.Hosts()) != 10 {
		t.Errorf("桶数=%d want 10", len(rs.Hosts()))
	}
}

func TestRouterSet_Delete(t *testing.T) {
	rs := NewRouterSet()
	rs.ReverseHttpRequest(hostReq("a.com", "http://a.com/x"))
	if !rs.Delete("a.com") {
		t.Fatal("删除已存在 host 应返回 true")
	}
	if len(rs.Hosts()) != 0 {
		t.Errorf("删除后桶应为空，Hosts=%v", rs.Hosts())
	}
	if rs.Delete("a.com") {
		t.Error("重复删除应返回 false")
	}
	// 删除后可重建
	if err := rs.ReverseHttpRequest(hostReq("a.com", "http://a.com/x")); err != nil {
		t.Errorf("删除后重建桶失败: %v", err)
	}
}

func TestRouterSet_LimitsRedactPropagate(t *testing.T) {
	rs := NewRouterSet()
	rs.ReverseHttpRequest(hostReq("a.com", "http://a.com/x"))
	rs.SetResourceLimits(ResourceLimits{MaxChildrenPerNode: 7})
	rs.SetRedactConfig(RedactConfig{Params: []string{"secret"}})
	r := rs.RouterFor(hostReq("a.com", "http://a.com/x"))
	if r.GetResourceLimits().MaxChildrenPerNode != 7 {
		t.Error("已有桶未收到资源上限传播")
	}
	// 新建桶继承
	r2 := rs.RouterFor(hostReq("b.com", "http://b.com/x"))
	if r2 == nil {
		t.Fatal("新建桶不应为 nil")
	}
	if r2.GetResourceLimits().MaxChildrenPerNode != 7 {
		t.Error("新建桶未继承资源上限")
	}
	// 脱敏传播验证：secret 参数原值不应存储
	rs.ReverseHttpRequest(request.NewHttpRequest("http://a.com/y?secret=zzz", nil, "GET", nil))
	m := r.Tree.Root.FindChildByKey("y").FindChildByKey("GET")
	if m.FindChildByKey("secret").(*node.RequestParamNode).GetValueMetric().GetValueCount("zzz") != 0 {
		t.Error("传播后 secret 应脱敏")
	}
}

// --- 默认值回归：开启护栏不破坏既有极端用例量级 ---

func TestLimits_DefaultsCoverExtremeUsage(t *testing.T) {
	r := newSilentRouter()
	if got := r.GetResourceLimits(); got != DefaultResourceLimits {
		t.Errorf("默认上限=%+v want %+v", got, DefaultResourceLimits)
	}
	// 500 参数 / 500 层深在默认上限下应全部成功
	var b strings.Builder
	b.WriteString("/deep")
	for i := 0; i < 500; i++ {
		fmt.Fprintf(&b, "/s%d", i)
	}
	b.WriteString("?")
	for i := 0; i < 500; i++ {
		if i > 0 {
			b.WriteString("&")
		}
		fmt.Fprintf(&b, "p%d=v%d", i, i)
	}
	feedOne(t, r, request.NewHttpRequest(b.String(), nil, "GET", nil))
}
