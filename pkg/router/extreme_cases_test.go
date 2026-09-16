package router

// extreme_cases_test.go — 50 个极端 case，全面评估从流量中还原 Web 路由树的能力。
//
// 分组：
//   A: 基础还原能力（基线验证）
//   B: 模式识别极端 case（部分当前失败，hex_id 修复后全通）
//   C: 不应合并的 case（误合并保护）
//   D: 完整路由树还原端到端场景
//   E: 边界条件与鲁棒性

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

// ─────────────────────────────────────────────────────────────────────────────
// Group A: 基础还原能力
// ─────────────────────────────────────────────────────────────────────────────

// Case A01: 多级嵌套 REST — users/{user_id}/posts/{post_id}/comments/{comment_id}
func TestExtremeA01_DeeplyNestedRestPathVariables(t *testing.T) {
	r := newSilentRouter()
	for i := 1; i <= 3; i++ {
		url := fmt.Sprintf("/api/users/%d/posts/%d/comments/%d", i, i*10, i*100)
		r.ReverseHttpRequest(request.NewHttpRequest(url, nil, "GET", nil))
	}
	usersNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("users")
	if usersNode == nil {
		t.Fatal("应有 users 路径节点")
	}
	userVar := usersNode.GetChildByType("request_path_variable")
	if userVar == nil {
		t.Fatal("users 下应有路径变量节点（{user_id}）")
	}
	postsNode := userVar.FindChildByKey("posts")
	if postsNode == nil {
		t.Fatal("路径变量下应保留 posts 子节点")
	}
	postVar := postsNode.GetChildByType("request_path_variable")
	if postVar == nil {
		t.Fatal("posts 下应有路径变量节点（{post_id}）")
	}
	commentsNode := postVar.FindChildByKey("comments")
	if commentsNode == nil {
		t.Fatal("第二层路径变量下应保留 comments 子节点")
	}
	commentVar := commentsNode.GetChildByType("request_path_variable")
	if commentVar == nil {
		t.Fatal("comments 下应有路径变量节点（{comment_id}）")
	}
}

// Case A02: 多租户路径 tenants/{id}/resources
func TestExtremeA02_MultiTenantPathIsolation(t *testing.T) {
	r := newSilentRouter()
	for _, tid := range []string{"tenant-001", "tenant-002", "tenant-003"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/tenants/"+tid+"/resources", nil, "GET", nil))
	}
	tenantsNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("tenants")
	varNode := tenantsNode.GetChildByType("request_path_variable")
	if varNode == nil {
		t.Fatal("tenant_001/tenant_002/tenant_003 应被合并为路径变量")
	}
	if varNode.FindChildByKey("resources") == nil {
		t.Fatal("路径变量下应保留 resources 子节点")
	}
}

// Case A03: 10 层静态路径 — /a/b/c/d/e/f/g/h/i/j
func TestExtremeA03_TenLevelStaticPath(t *testing.T) {
	r := newSilentRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/a/b/c/d/e/f/g/h/i/j", nil, "GET", nil))
	cur := r.Tree.Root
	for _, seg := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"} {
		cur = cur.FindChildByKey(seg)
		if cur == nil {
			t.Fatalf("应找到段 %q", seg)
		}
	}
}

// Case A04: API 版本号路径 v1/v2/v3 合并
func TestExtremeA04_APIVersionMerge(t *testing.T) {
	r := newSilentRouter()
	for _, v := range []string{"v1", "v2", "v3"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/"+v+"/users", nil, "GET", nil))
	}
	apiNode := r.Tree.Root.FindChildByKey("api")
	versionVar := apiNode.GetChildByType("request_path_variable")
	if versionVar == nil {
		t.Fatal("v1/v2/v3 应被合并为路径变量（版本号模式）")
	}
	if versionVar.FindChildByKey("users") == nil {
		t.Fatal("版本变量下应保留 users 子节点")
	}
}

// Case A05: 固定路径 + 数字路径混合，数字合并、固定保留
func TestExtremeA05_MixedFixedAndNumericMerge(t *testing.T) {
	r := newSilentRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/users/list", nil, "GET", nil))
	r.ReverseHttpRequest(request.NewHttpRequest("/api/users/create", nil, "POST", nil))
	for _, id := range []string{"101", "102", "103"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/users/"+id, nil, "GET", nil))
	}
	usersNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("users")
	if usersNode.FindChildByKey("list") == nil {
		t.Error("list 固定路径应保留")
	}
	if usersNode.FindChildByKey("create") == nil {
		t.Error("create 固定路径应保留")
	}
	if usersNode.GetChildByType("request_path_variable") == nil {
		t.Error("101/102/103 应合并为路径变量")
	}
}

// Case A06: 中国手机号路径
func TestExtremeA06_ChinesePhonePathMerge(t *testing.T) {
	r := newSilentRouter()
	for _, phone := range []string{"13812345678", "13987654321", "18566667777"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/users/"+phone, nil, "GET", nil))
	}
	usersNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("users")
	varNode := usersNode.GetChildByType("request_path_variable")
	if varNode == nil {
		t.Fatal("手机号应被合并为路径变量")
	}
	if varNode.GetKey() != "users_phone" {
		t.Errorf("变量名应为 users_phone，实际: %s", varNode.GetKey())
	}
}

// Case A07: 银行卡号路径
func TestExtremeA07_BankCardPathMerge(t *testing.T) {
	r := newSilentRouter()
	for _, card := range []string{"4111111111111111", "5500005555555559", "3714496353984311"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/payments/"+card, nil, "GET", nil))
	}
	paymentsNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("payments")
	varNode := paymentsNode.GetChildByType("request_path_variable")
	if varNode == nil {
		t.Fatal("银行卡号应被合并为路径变量")
	}
	if varNode.GetKey() != "payments_bankcard" {
		t.Errorf("变量名应为 payments_bankcard，实际: %s", varNode.GetKey())
	}
}

// Case A08: 中国车牌号路径
func TestExtremeA08_LicensePlatePathMerge(t *testing.T) {
	r := newSilentRouter()
	for _, plate := range []string{"京A12345", "沪B23456", "粤C34567"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/vehicles/"+plate, nil, "GET", nil))
	}
	vehiclesNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("vehicles")
	varNode := vehiclesNode.GetChildByType("request_path_variable")
	if varNode == nil {
		t.Fatal("车牌号应被合并为路径变量")
	}
	if varNode.GetKey() != "vehicles_plate" {
		t.Errorf("变量名应为 vehicles_plate，实际: %s", varNode.GetKey())
	}
}

// Case A09: IP 地址路径
func TestExtremeA09_IPAddressPathMerge(t *testing.T) {
	r := newSilentRouter()
	for _, ip := range []string{"192.168.1.1", "10.0.0.1", "172.16.0.1"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/hosts/"+ip, nil, "GET", nil))
	}
	hostsNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("hosts")
	varNode := hostsNode.GetChildByType("request_path_variable")
	if varNode == nil {
		t.Fatal("IP 地址应被合并为路径变量")
	}
	if varNode.GetKey() != "hosts_ip" {
		t.Errorf("变量名应为 hosts_ip，实际: %s", varNode.GetKey())
	}
}

// Case A10: 日期路径 (归档路由)
func TestExtremeA10_DatePathMerge(t *testing.T) {
	r := newSilentRouter()
	for _, d := range []string{"2024-01-15", "2024-02-20", "2024-03-25"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/reports/"+d, nil, "GET", nil))
	}
	reportsNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("reports")
	varNode := reportsNode.GetChildByType("request_path_variable")
	if varNode == nil {
		t.Fatal("日期应被合并为路径变量")
	}
	if varNode.GetKey() != "reports_date" {
		t.Errorf("变量名应为 reports_date，实际: %s", varNode.GetKey())
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Group B: 模式识别极端 case
// ─────────────────────────────────────────────────────────────────────────────

// Case B01: MongoDB ObjectId (24 位十六进制字符) — hex_id 修复后应合并
func TestExtremeB01_MongoDBObjectIdPathMerge(t *testing.T) {
	r := newSilentRouter()
	oids := []string{
		"507f1f77bcf86cd799439011",
		"6ba7b810bcf86cd799439000",
		"ffffffff00112233aabbccdd",
	}
	for _, oid := range oids {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/docs/"+oid, nil, "GET", nil))
	}
	docsNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("docs")
	varNode := docsNode.GetChildByType("request_path_variable")
	if varNode == nil {
		t.Fatal("MongoDB ObjectId（24位 hex）应被合并为路径变量（当前失败说明缺少 hex_id 模式，需修复）")
	}
	t.Logf("ObjectId 变量名: %s, pattern: %v", varNode.GetKey(), varNode.(*node.RequestPathVariableNode).GetPattern())
}

// Case B02: MD5 哈希路径 (32 位十六进制) — hex_id 修复后应合并
func TestExtremeB02_MD5HashPathMerge(t *testing.T) {
	r := newSilentRouter()
	hashes := []string{
		"5d41402abc4b2a76b9719d911017c592",
		"7215ee9c7d9dc229d2921a40e899ec5f",
		"b14a7b8059d9c055954c92674ce60032",
	}
	for _, h := range hashes {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/files/"+h, nil, "GET", nil))
	}
	filesNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("files")
	varNode := filesNode.GetChildByType("request_path_variable")
	if varNode == nil {
		t.Fatal("MD5 哈希路径（32位 hex）应被合并为路径变量（当前失败说明缺少 hex_id 模式，需修复）")
	}
}

// Case B03: SHA1 哈希路径 (40 位十六进制)
func TestExtremeB03_SHA1HashPathMerge(t *testing.T) {
	r := newSilentRouter()
	hashes := []string{
		"da39a3ee5e6b4b0d3255bfef95601890afd80709",
		"adc83b19e793491b1c6ea0fd8b46cd9f32e592fc",
		"e3b0c44298fc1c149afbf4c8996fb92427ae41e4",
	}
	for _, h := range hashes {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/commits/"+h, nil, "GET", nil))
	}
	commitsNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("commits")
	varNode := commitsNode.GetChildByType("request_path_variable")
	if varNode == nil {
		t.Fatal("SHA1 哈希路径（40位 hex）应被合并为路径变量（当前失败说明缺少 hex_id 模式，需修复）")
	}
}

// Case B04: 16 位以上纯数字 ID (物理类型应为 string)
func TestExtremeB04_LongNumericIDMerge(t *testing.T) {
	r := newSilentRouter()
	for _, id := range []string{"1234567890123456", "9876543210987654", "1111222233334444"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/orders/"+id, nil, "GET", nil))
	}
	ordersNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("orders")
	varNode := ordersNode.GetChildByType("request_path_variable")
	if varNode == nil {
		t.Fatal("16位纯数字 ID 应被合并为路径变量")
	}
	// 16 位以上应降级为 string 物理类型
	vn := varNode.(*node.RequestPathVariableNode)
	if vn.GetValueType() == value.Type(value.PhysicalTypeInteger) {
		t.Errorf("16位以上数字 ID 物理类型不应为 integer（应为 string），实际: %s", vn.GetValueType())
	}
}

// Case B05: 字母数字产品码 (ABC123 风格)
func TestExtremeB05_AlphanumericProductCode(t *testing.T) {
	r := newSilentRouter()
	for _, code := range []string{"ABC123", "DEF456", "GHI789"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/products/"+code, nil, "GET", nil))
	}
	productsNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("products")
	varNode := productsNode.GetChildByType("request_path_variable")
	if varNode == nil {
		t.Fatal("字母数字产品码应被合并为路径变量（alphanumeric 模式）")
	}
	if varNode.GetKey() != "products_code" {
		t.Errorf("变量名应为 products_code，实际: %s", varNode.GetKey())
	}
}

// Case B06: 前缀模式 order_001/order_002/order_003
func TestExtremeB06_PrefixPatternMerge(t *testing.T) {
	r := newSilentRouter()
	for _, id := range []string{"order_001", "order_002", "order_003"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/orders/"+id, nil, "GET", nil))
	}
	ordersNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("orders")
	varNode := ordersNode.GetChildByType("request_path_variable")
	if varNode == nil {
		t.Fatal("前缀模式 order_001/002/003 应被合并为路径变量")
	}
	t.Logf("前缀变量名: %s", varNode.GetKey())
}

// Case B07: 后缀模式 001_item/002_item/003_item
func TestExtremeB07_SuffixPatternMerge(t *testing.T) {
	r := newSilentRouter()
	for _, id := range []string{"001_item", "002_item", "003_item"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/items/"+id, nil, "GET", nil))
	}
	itemsNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("items")
	varNode := itemsNode.GetChildByType("request_path_variable")
	if varNode == nil {
		t.Fatal("后缀模式 001_item/002_item/003_item 应被合并为路径变量")
	}
	t.Logf("后缀变量名: %s", varNode.GetKey())
}

// Case B08: 浮点数路径 (价格/坐标)
func TestExtremeB08_FloatPathMerge(t *testing.T) {
	r := newSilentRouter()
	for _, v := range []string{"39.9", "99.9", "129.9"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/prices/"+v, nil, "GET", nil))
	}
	pricesNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("prices")
	varNode := pricesNode.GetChildByType("request_path_variable")
	if varNode == nil {
		t.Fatal("浮点数应被合并为路径变量")
	}
	if varNode.GetKey() != "prices_value" {
		t.Errorf("变量名应为 prices_value，实际: %s", varNode.GetKey())
	}
}

// Case B09: 负数路径 — 当前不支持（文档化的局限性）
func TestExtremeB09_NegativeIntegerPath_KnownLimitation(t *testing.T) {
	r := newSilentRouter()
	for _, v := range []string{"-1", "-2", "-3"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/adjustments/"+v, nil, "GET", nil))
	}
	adjNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("adjustments")
	varNode := adjNode.GetChildByType("request_path_variable")
	// 负数不匹配现有整数/浮点模式，需要 >=6 个才能通过 similar_length 突破
	// 这是已知局限：3 个负数不会自动合并
	if varNode != nil {
		t.Logf("负数路径被合并为变量 %s（意外通过）", varNode.GetKey())
	} else {
		t.Logf("已知局限：3 个负数路径（-1/-2/-3）不会自动合并（无负数整数模式）")
	}
}

// Case B10: UUID v4 路径合并
func TestExtremeB10_UUIDv4PathMerge(t *testing.T) {
	r := newSilentRouter()
	uuids := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		"6ba7b811-9dad-11d1-80b4-00c04fd430c8",
	}
	for _, uuid := range uuids {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/resources/"+uuid, nil, "GET", nil))
	}
	resNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("resources")
	varNode := resNode.GetChildByType("request_path_variable")
	if varNode == nil {
		t.Fatal("UUID 应被合并为路径变量")
	}
	vn := varNode.(*node.RequestPathVariableNode)
	if vn.GetLogicalType() != value.LogicalTypeUUID {
		t.Errorf("UUID 变量逻辑类型应为 uuid，实际: %s", vn.GetLogicalType())
	}
}

// Case B11: 身份证号路径
func TestExtremeB11_ChineseIDCardPathMerge(t *testing.T) {
	r := newSilentRouter()
	for _, idc := range []string{"110101199001011234", "320101198005015678", "440301197511221234"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/verify/"+idc, nil, "GET", nil))
	}
	verifyNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("verify")
	varNode := verifyNode.GetChildByType("request_path_variable")
	if varNode == nil {
		t.Fatal("身份证号应被合并为路径变量")
	}
	if varNode.GetKey() != "verify_idcard" {
		t.Errorf("变量名应为 verify_idcard，实际: %s", varNode.GetKey())
	}
}

// Case B12: 零填充数字 ID (0001, 0002, 0003)
func TestExtremeB12_ZeroPaddedIntegerIDs(t *testing.T) {
	r := newSilentRouter()
	for _, id := range []string{"0001", "0002", "0003"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/tickets/"+id, nil, "GET", nil))
	}
	ticketsNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("tickets")
	varNode := ticketsNode.GetChildByType("request_path_variable")
	if varNode == nil {
		t.Fatal("零填充整数应被合并为路径变量")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Group C: 不应合并 (误合并保护)
// ─────────────────────────────────────────────────────────────────────────────

// Case C01: 少量英文单词路径不合并 (admin/manager/guest)
func TestExtremeC01_FewWordPathsNoMerge(t *testing.T) {
	r := newSilentRouter()
	for _, role := range []string{"admin", "manager", "guest"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/roles/"+role, nil, "GET", nil))
	}
	rolesNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("roles")
	if rolesNode.GetChildByType("request_path_variable") != nil {
		t.Error("admin/manager/guest 不应被合并为路径变量")
	}
	for _, role := range []string{"admin", "manager", "guest"} {
		if rolesNode.FindChildByKey(role) == nil {
			t.Errorf("固定路径 %q 应保留", role)
		}
	}
}

// Case C02: 文件扩展名路径不合并 (data.json/data.xml/data.html)
func TestExtremeC02_FileExtensionPathNoMerge(t *testing.T) {
	r := newSilentRouter()
	for _, f := range []string{"data.json", "data.xml", "data.html"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/export/"+f, nil, "GET", nil))
	}
	exportNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("export")
	if exportNode.GetChildByType("request_path_variable") != nil {
		t.Error("有文件扩展名的路径不应合并为变量")
	}
}

// Case C03: 仅2个数字 ID 兄弟节点（低于阈值，不应合并）
func TestExtremeC03_OnlyTwoNumericSiblings(t *testing.T) {
	r := newSilentRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/items/123", nil, "GET", nil))
	r.ReverseHttpRequest(request.NewHttpRequest("/api/items/456", nil, "GET", nil))
	itemsNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("items")
	if itemsNode.GetChildByType("request_path_variable") != nil {
		t.Error("仅2个数字兄弟（< 阈值3）不应合并")
	}
}

// Case C04: 重复相同 ID 不触发合并
func TestExtremeC04_RepeatedSameIDNoMerge(t *testing.T) {
	r := newSilentRouter()
	for i := 0; i < 10; i++ {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/users/42", nil, "GET", nil))
	}
	usersNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("users")
	if usersNode.GetChildByType("request_path_variable") != nil {
		t.Error("同一个 ID 重复请求不应触发合并")
	}
}

// Case C05: 高阈值配置下数字 ID 不合并
func TestExtremeC05_HighThresholdPreventsMerge(t *testing.T) {
	r := newSilentRouter()
	r.SetMergeConfig(MergeConfig{SiblingMergeThreshold: 6, PatternSimilarityThreshold: 0.6})
	for _, id := range []string{"1", "2", "3"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/orders/"+id, nil, "GET", nil))
	}
	ordersNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("orders")
	if ordersNode.GetChildByType("request_path_variable") != nil {
		t.Error("阈值6时3个节点不应合并")
	}
}

// Case C06: 小数量固定管理路由（< 6 个）不应合并
func TestExtremeC06_AdminRoutesFewSiblingsNoMerge(t *testing.T) {
	r := newSilentRouter()
	for _, path := range []string{"users", "roles", "settings", "logs"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/admin/"+path, nil, "GET", nil))
	}
	adminNode := r.Tree.Root.FindChildByKey("admin")
	if adminNode.GetChildByType("request_path_variable") != nil {
		t.Error("4 个管理路由不应合并（< similar_length 突破阈值 6）")
	}
	for _, path := range []string{"users", "roles", "settings", "logs"} {
		if adminNode.FindChildByKey(path) == nil {
			t.Errorf("固定管理路由 %q 应保留", path)
		}
	}
}

// Case C07: HTTP 状态码路径（记录行为：整数模式会合并，这是已知设计取舍）
func TestExtremeC07_HTTPStatusCodePaths_DesignTradeoff(t *testing.T) {
	r := newSilentRouter()
	for _, code := range []string{"400", "404", "500"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/errors/"+code, nil, "GET", nil))
	}
	errorsNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("errors")
	varNode := errorsNode.GetChildByType("request_path_variable")
	// 设计取舍：从黑盒流量视角，400/404/500 是整数，会被合并为 {errors_id}。
	// 无法区分"状态码路由"与"数值 ID 路由"，属于已知局限。
	if varNode != nil {
		t.Logf("已知设计取舍：HTTP 状态码 400/404/500 被合并为路径变量 %s（纯整数无法区分）", varNode.GetKey())
	} else {
		t.Logf("状态码未合并（可能阈值未达到）")
	}
}

// Case C08: 单一唯一 ID 不合并
func TestExtremeC08_SingleUniqueIDNoMerge(t *testing.T) {
	r := newSilentRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/singleton/unique-id-123", nil, "GET", nil))
	singletonNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("singleton")
	if singletonNode.GetChildByType("request_path_variable") != nil {
		t.Error("单个路径段不应触发路径变量合并")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Group D: 完整路由树还原端到端场景
// ─────────────────────────────────────────────────────────────────────────────

// Case D01: 完整 CRUD 操作还原
func TestExtremeD01_FullCRUDRestoration(t *testing.T) {
	r := newSilentRouter()
	// GET 列表
	r.ReverseHttpRequest(request.NewHttpRequest("/api/articles", nil, "GET", nil))
	// POST 创建
	r.ReverseHttpRequest(request.NewHttpRequest("/api/articles",
		request.Headers{"Content-Type": "application/json"}, "POST", []byte(`{"title":"test","content":"body"}`)))
	// GET 单条
	for _, id := range []string{"1001", "1002", "1003"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/articles/"+id, nil, "GET", nil))
	}
	// PUT 更新
	r.ReverseHttpRequest(request.NewHttpRequest("/api/articles/1001",
		request.Headers{"Content-Type": "application/json"}, "PUT", []byte(`{"title":"updated"}`)))
	// DELETE
	r.ReverseHttpRequest(request.NewHttpRequest("/api/articles/1001", nil, "DELETE", nil))

	articlesNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("articles")
	if articlesNode == nil {
		t.Fatal("应有 articles 节点")
	}
	if articlesNode.FindChildByKey("GET") == nil {
		t.Error("应有 GET 方法节点（列表）")
	}
	if articlesNode.FindChildByKey("POST") == nil {
		t.Error("应有 POST 方法节点（创建）")
	}
	varNode := articlesNode.GetChildByType("request_path_variable")
	if varNode == nil {
		t.Error("1001/1002/1003 应合并为路径变量")
	} else {
		if varNode.FindChildByKey("GET") == nil {
			t.Error("路径变量下应有 GET 方法（查询单条）")
		}
		if varNode.FindChildByKey("PUT") == nil {
			t.Error("路径变量下应有 PUT 方法（更新）")
		}
		if varNode.FindChildByKey("DELETE") == nil {
			t.Error("路径变量下应有 DELETE 方法（删除）")
		}
	}
	s := r.GetStats()
	if s.RequestsProcessed < 7 {
		t.Errorf("应处理 >=7 个请求，实际: %d", s.RequestsProcessed)
	}
}

// Case D02: 电商商品目录完整路由
func TestExtremeD02_EcommerceProductCatalog(t *testing.T) {
	r := newSilentRouter()
	// 商品列表 + 搜索
	r.ReverseHttpRequest(request.NewHttpRequest("/api/products?category=electronics&page=1", nil, "GET", nil))
	r.ReverseHttpRequest(request.NewHttpRequest("/api/products?category=clothing&page=2", nil, "GET", nil))
	r.ReverseHttpRequest(request.NewHttpRequest("/api/products/search?q=laptop&sort=price", nil, "GET", nil))
	// 商品详情
	for _, id := range []string{"SKU001", "SKU002", "SKU003"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/products/"+id, nil, "GET", nil))
	}
	// 购物车
	r.ReverseHttpRequest(request.NewHttpRequest("/api/cart",
		request.Headers{"Content-Type": "application/json"}, "POST",
		[]byte(`{"product_id":"SKU001","quantity":2}`)))
	r.ReverseHttpRequest(request.NewHttpRequest("/api/cart", nil, "GET", nil))
	// 订单
	r.ReverseHttpRequest(request.NewHttpRequest("/api/orders",
		request.Headers{"Content-Type": "application/json"}, "POST",
		[]byte(`{"cart_id":"cart123","address":"北京市朝阳区"}`)))
	for _, oid := range []string{"ORD001", "ORD002", "ORD003"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/orders/"+oid, nil, "GET", nil))
	}

	productNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("products")
	if productNode == nil {
		t.Fatal("应有 products 节点")
	}
	// 搜索子路径
	if productNode.FindChildByKey("search") == nil {
		t.Error("应有 search 固定子路径")
	}
	// SKU001/SKU002/SKU003 应合并（alphanumeric 模式）
	if productNode.GetChildByType("request_path_variable") == nil {
		t.Error("SKU 商品代码应合并为路径变量")
	}
	// 参数推断
	getNode := productNode.FindChildByKey("GET")
	if getNode == nil {
		t.Fatal("应有 products GET 节点")
	}
	if getNode.FindChildByKey("category") == nil {
		t.Error("应有 category 参数节点")
	}
	if getNode.FindChildByKey("page") == nil {
		t.Error("应有 page 参数节点")
	}
}

// Case D03: 认证流程还原（固定路径 + token 参数）
func TestExtremeD03_AuthenticationFlowRestoration(t *testing.T) {
	r := newSilentRouter()
	// login/refresh/logout 都是固定路径，不应合并
	r.ReverseHttpRequest(request.NewHttpRequest("/api/auth/login",
		request.Headers{"Content-Type": "application/json"}, "POST",
		[]byte(`{"username":"admin","password":"secret"}`)))
	r.ReverseHttpRequest(request.NewHttpRequest("/api/auth/refresh",
		request.Headers{"Content-Type": "application/json", "Authorization": "Bearer oldtoken"}, "POST", nil))
	r.ReverseHttpRequest(request.NewHttpRequest("/api/auth/logout",
		request.Headers{"Authorization": "Bearer validtoken"}, "POST", nil))

	authNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("auth")
	if authNode == nil {
		t.Fatal("应有 auth 节点")
	}
	// login/refresh/logout 3个固定路径不应合并
	if authNode.GetChildByType("request_path_variable") != nil {
		t.Error("login/refresh/logout 是固定路径，不应合并为变量")
	}
	for _, path := range []string{"login", "refresh", "logout"} {
		if authNode.FindChildByKey(path) == nil {
			t.Errorf("固定认证路径 %q 应保留", path)
		}
	}

	// 验证 body 参数解析
	loginNode := authNode.FindChildByKey("login").FindChildByKey("POST")
	if loginNode == nil {
		t.Fatal("应有 login POST 节点")
	}
	if loginNode.FindChildByKey("username") == nil {
		t.Error("应解析出 username body 参数")
	}
}

// Case D04: 博客 CMS 路由树
func TestExtremeD04_BlogCMSRouteTree(t *testing.T) {
	r := newSilentRouter()
	// 分类路由
	r.ReverseHttpRequest(request.NewHttpRequest("/blog/tech", nil, "GET", nil))
	r.ReverseHttpRequest(request.NewHttpRequest("/blog/life", nil, "GET", nil))
	r.ReverseHttpRequest(request.NewHttpRequest("/blog/travel", nil, "GET", nil))
	// 文章列表（分类 + 日期）
	r.ReverseHttpRequest(request.NewHttpRequest("/blog/2024/01/articles", nil, "GET", nil))
	r.ReverseHttpRequest(request.NewHttpRequest("/blog/2024/02/articles", nil, "GET", nil))
	r.ReverseHttpRequest(request.NewHttpRequest("/blog/2024/03/articles", nil, "GET", nil))
	// 具体文章（按数字 ID）
	for _, id := range []string{"1001", "1002", "1003"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/blog/posts/"+id, nil, "GET", nil))
	}

	blogNode := r.Tree.Root.FindChildByKey("blog")
	if blogNode == nil {
		t.Fatal("应有 blog 节点")
	}
	// 年份路径（2024 × 3 → 不合并，因为只有1个唯一年份值重复）
	// 月份 01/02/03 应合并（3个整数）
	yearNode := blogNode.FindChildByKey("2024")
	if yearNode == nil {
		t.Error("应有 2024 年份路径节点")
	} else {
		monthVar := yearNode.GetChildByType("request_path_variable")
		if monthVar == nil {
			t.Error("01/02/03 月份应合并为路径变量")
		}
	}
	// 文章 ID 应合并
	postsNode := blogNode.FindChildByKey("posts")
	if postsNode == nil {
		t.Fatal("应有 posts 节点")
	}
	if postsNode.GetChildByType("request_path_variable") == nil {
		t.Error("文章 ID 1001/1002/1003 应合并为路径变量")
	}
}

// Case D05: 微服务代理路径
func TestExtremeD05_MicroserviceProxyPaths(t *testing.T) {
	r := newSilentRouter()
	services := []string{"user-service", "order-service", "payment-service"}
	for _, svc := range services {
		r.ReverseHttpRequest(request.NewHttpRequest("/gateway/"+svc+"/api/v1/health", nil, "GET", nil))
	}
	gatewayNode := r.Tree.Root.FindChildByKey("gateway")
	if gatewayNode == nil {
		t.Fatal("应有 gateway 节点")
	}
	// user-service/order-service/payment-service 都含 -service 后缀
	// 后缀模式或相似长度字符串，3 个可能合并也可能不合并（长度差异）
	t.Logf("微服务路径子节点数: %d", gatewayNode.GetChildCount())
	for _, child := range gatewayNode.GetChildren() {
		t.Logf("  [%s] %s", child.GetType(), child.GetKey())
	}
}

// Case D06: GitHub 风格 API 路径
func TestExtremeD06_GitHubStyleAPI(t *testing.T) {
	r := newSilentRouter()
	// /repos/{owner}/{repo}/issues/{issue_number}
	// 6 个相似长度(6字符)的 owner → similar_length_strings 达到突破阈值 → 合并
	// repo 名均为 6 字符，owner 合并后通过级联合并将 repo 也合并
	repos := [][]string{
		{"golang", "gobase", "1"},
		{"docker", "dcbase", "2"},
		{"python", "pybase", "3"},
		{"django", "djbase", "4"},
		{"nodejs", "njbase", "5"},
		{"meteor", "mtbase", "6"},
	}
	for _, parts := range repos {
		url := fmt.Sprintf("/repos/%s/%s/issues/%s", parts[0], parts[1], parts[2])
		r.ReverseHttpRequest(request.NewHttpRequest(url, nil, "GET", nil))
	}
	reposNode := r.Tree.Root.FindChildByKey("repos")
	if reposNode == nil {
		t.Fatal("应有 repos 节点")
	}
	// 6 个同长 owner 通过 similar_length_strings 合并为路径变量
	ownerVar := reposNode.GetChildByType("request_path_variable")
	if ownerVar == nil {
		t.Fatal("repos 下的 6 个同长 owner 应合并为路径变量")
	}
	// 级联合并：owner 合并后各 repo 汇聚，6 个同长 repo 也应合并
	repoVar := ownerVar.GetChildByType("request_path_variable")
	if repoVar == nil {
		t.Fatal("owner 合并后级联合并：各 repo 也应合并为路径变量")
	}
	issuesNode := repoVar.FindChildByKey("issues")
	if issuesNode == nil {
		t.Fatal("应有 issues 节点")
	}
	issueVar := issuesNode.GetChildByType("request_path_variable")
	if issueVar == nil {
		t.Fatal("issue 编号应合并为路径变量")
	}
}

// Case D07: 必需参数推断
func TestExtremeD07_RequiredParamInference(t *testing.T) {
	r := newSilentRouter()
	// page 每次都有，size 有时缺失，format 很少出现
	for i := 0; i < 10; i++ {
		url := fmt.Sprintf("/api/list?page=%d", i+1)
		if i < 8 {
			url += "&size=20"
		}
		if i < 2 {
			url += "&format=json"
		}
		r.ReverseHttpRequest(request.NewHttpRequest(url, nil, "GET", nil))
	}
	count := r.InferRequiredParams()
	if count != 1 {
		t.Errorf("应推断出 1 个必需参数(page)，实际: %d", count)
	}
	listNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("list").FindChildByKey("GET")
	pageNode := listNode.FindChildByKey("page").(*node.RequestParamNode)
	if !pageNode.IsRequired() {
		t.Error("page 出现率 10/10 应为必需参数")
	}
	sizeNode := listNode.FindChildByKey("size").(*node.RequestParamNode)
	if sizeNode.IsRequired() {
		t.Error("size 出现率 8/10 < 0.9，应为可选")
	}
	formatNode := listNode.FindChildByKey("format").(*node.RequestParamNode)
	if formatNode.IsRequired() {
		t.Error("format 出现率 2/10 << 0.9，应为可选")
	}
}

// Case D08: Content-Type 多样路由
func TestExtremeD08_ContentTypeRouting(t *testing.T) {
	r := newSilentRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/data",
		request.Headers{"Content-Type": "application/json"}, "POST", []byte(`{"key":"val"}`)))
	r.ReverseHttpRequest(request.NewHttpRequest("/api/data",
		request.Headers{"Content-Type": "application/xml"}, "POST", []byte(`<key>val</key>`)))
	r.ReverseHttpRequest(request.NewHttpRequest("/api/data",
		request.Headers{"Content-Type": "application/x-www-form-urlencoded"}, "POST", []byte("key=val")))

	dataNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("data")
	postNode := dataNode.FindChildByKey("POST")
	if postNode == nil {
		t.Fatal("应有 POST 节点")
	}
	ctTypes := 0
	for _, child := range postNode.GetChildren() {
		if child.GetType() == "request_content_type" {
			ctTypes++
		}
	}
	if ctTypes < 2 {
		t.Errorf("应有至少 2 个 Content-Type 节点，实际: %d", ctTypes)
	}
}

// Case D09: 用户权限管理多层嵌套
func TestExtremeD09_UserPermissionManagement(t *testing.T) {
	r := newSilentRouter()
	// /api/orgs/{org}/members/{user}/permissions
	// 成员 ID 用 user001/user002/user003 格式（前缀模式），3 个即可合并
	orgs := []string{"001", "002", "003"}
	users := []string{"user001", "user002", "user003"}
	for _, org := range orgs {
		for _, user := range users {
			r.ReverseHttpRequest(request.NewHttpRequest(
				fmt.Sprintf("/api/orgs/%s/members/%s/permissions", org, user),
				nil, "GET", nil))
		}
	}
	orgsNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("orgs")
	orgVar := orgsNode.GetChildByType("request_path_variable")
	if orgVar == nil {
		t.Fatal("org ID 应合并为路径变量")
	}
	membersNode := orgVar.FindChildByKey("members")
	if membersNode == nil {
		t.Fatal("应有 members 节点")
	}
	memberVar := membersNode.GetChildByType("request_path_variable")
	if memberVar == nil {
		t.Fatal("member 名应合并为路径变量")
	}
	if memberVar.FindChildByKey("permissions") == nil {
		t.Fatal("应有 permissions 子节点")
	}
}

// Case D10: 完整 Accept Header + Cookie 路由树
func TestExtremeD10_CompleteHeaderCookieRouting(t *testing.T) {
	r := newSilentRouter()
	headers := []request.Headers{
		{"Accept": "application/json", "Accept-Language": "zh-CN", "Cookie": "theme=dark"},
		{"Accept": "text/html", "Accept-Language": "en-US", "Cookie": "theme=light"},
		{"Accept": "application/xml", "Accept-Language": "ja", "Cookie": "theme=auto"},
	}
	for _, h := range headers {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/content", h, "GET", nil))
	}
	getNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("content").FindChildByKey("GET")
	if getNode == nil {
		t.Fatal("应有 GET 节点")
	}
	acceptNode := getNode.FindChildByKey("Accept")
	if acceptNode == nil {
		t.Fatal("应有 Accept header 路由节点")
	}
	if acceptNode.GetChildCount() < 3 {
		t.Errorf("Accept 分组应有 3 个值节点，实际: %d", acceptNode.GetChildCount())
	}
	langNode := getNode.FindChildByKey("Accept-Language")
	if langNode == nil {
		t.Fatal("应有 Accept-Language header 路由节点")
	}
	themeNode := getNode.FindChildByKey("theme")
	if themeNode == nil {
		t.Fatal("应有 theme Cookie 路由节点")
	}
	if themeNode.GetChildCount() < 3 {
		t.Errorf("theme cookie 分组应有 3 个值，实际: %d", themeNode.GetChildCount())
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Group E: 边界条件与鲁棒性
// ─────────────────────────────────────────────────────────────────────────────

// Case E01: 根路径 /
func TestExtremeE01_RootPath(t *testing.T) {
	r := newSilentRouter()
	if err := r.ReverseHttpRequest(request.NewHttpRequest("/", nil, "GET", nil)); err != nil {
		t.Fatalf("根路径请求不应报错: %v", err)
	}
}

// Case E02: 空参数值 (?flag&debug)
func TestExtremeE02_EmptyParamValues(t *testing.T) {
	r := newSilentRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/data?flag&debug&verbose", nil, "GET", nil))
	getNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("data").FindChildByKey("GET")
	for _, param := range []string{"flag", "debug", "verbose"} {
		if getNode.FindChildByKey(param) == nil {
			t.Errorf("空值参数 %q 应被识别", param)
		}
	}
}

// Case E03: 多值参数 (?tag=go&tag=web&tag=api)
func TestExtremeE03_MultiValueParams(t *testing.T) {
	r := newSilentRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/articles?tag=go&tag=web&tag=api", nil, "GET", nil))
	getNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("articles").FindChildByKey("GET")
	tagNode := getNode.FindChildByKey("tag")
	if tagNode == nil {
		t.Fatal("应有 tag 参数节点")
	}
	pn := tagNode.(*node.RequestParamNode)
	if pn.GetValueMetric().GetUniqueValueCount() != 3 {
		t.Errorf("tag 应有 3 个唯一值，实际: %d", pn.GetValueMetric().GetUniqueValueCount())
	}
}

// Case E04: URL 编码路径段
func TestExtremeE04_URLEncodedPath(t *testing.T) {
	r := newSilentRouter()
	// %E7%94%A8%E6%88%B7 = 用户
	r.ReverseHttpRequest(request.NewHttpRequest("/api/%E7%94%A8%E6%88%B7/list", nil, "GET", nil))
	apiNode := r.Tree.Root.FindChildByKey("api")
	if apiNode == nil {
		t.Fatal("应有 api 节点")
	}
	// 解码后应为 "用户"
	userNode := apiNode.FindChildByKey("用户")
	if userNode == nil {
		t.Error("URL 编码路径应解码为 '用户'")
	}
}

// Case E05: 路径遍历过滤 (./.. 应被清理)
func TestExtremeE05_PathTraversalFiltering(t *testing.T) {
	r := newSilentRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/./users/../admin/list", nil, "GET", nil))
	apiNode := r.Tree.Root.FindChildByKey("api")
	if apiNode == nil {
		t.Fatal("应有 api 节点")
	}
	// "." 应被过滤
	if apiNode.FindChildByKey(".") != nil {
		t.Error("'.' 路径段应被过滤")
	}
	t.Logf("路径遍历处理后的树:\n%s", r.Tree.String())
}

// Case E06: 超长路径变量链（连续多个变量段）
func TestExtremeE06_ConsecutivePathVariables(t *testing.T) {
	r := newSilentRouter()
	// /grid/{row}/{col}/{depth}
	for row := 1; row <= 3; row++ {
		for col := 1; col <= 3; col++ {
			for depth := 1; depth <= 3; depth++ {
				url := fmt.Sprintf("/grid/%d/%d/%d", row, col, depth)
				r.ReverseHttpRequest(request.NewHttpRequest(url, nil, "GET", nil))
			}
		}
	}
	gridNode := r.Tree.Root.FindChildByKey("grid")
	if gridNode == nil {
		t.Fatal("应有 grid 节点")
	}
	rowVar := gridNode.GetChildByType("request_path_variable")
	if rowVar == nil {
		t.Fatal("行变量应合并")
	}
	colVar := rowVar.GetChildByType("request_path_variable")
	if colVar == nil {
		t.Fatal("列变量应合并")
	}
	depthVar := colVar.GetChildByType("request_path_variable")
	if depthVar == nil {
		t.Fatal("深度变量应合并")
	}
	t.Logf("连续变量链 grid/{row}/{col}/{depth} 还原成功")
}

// Case E07: 参数大小写不敏感
func TestExtremeE07_ParamNameCaseInsensitive(t *testing.T) {
	r := newSilentRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/data?UserID=1", nil, "GET", nil))
	r.ReverseHttpRequest(request.NewHttpRequest("/api/data?userid=2", nil, "GET", nil))
	r.ReverseHttpRequest(request.NewHttpRequest("/api/data?USERID=3", nil, "GET", nil))
	getNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("data").FindChildByKey("GET")
	if getNode.FindChildByKey("userid") == nil {
		t.Fatal("应有小写 userid 参数节点")
	}
	if getNode.FindChildByKey("UserID") != nil {
		t.Error("不应有大写 UserID 参数节点（应归一化为小写）")
	}
	pn := getNode.FindChildByKey("userid").(*node.RequestParamNode)
	if pn.GetValueMetric().GetUniqueValueCount() != 3 {
		t.Errorf("userid 应有 3 个唯一值，实际: %d", pn.GetValueMetric().GetUniqueValueCount())
	}
}

// Case E08: JSON body 嵌套参数扁平化
func TestExtremeE08_JSONBodyNestedParams(t *testing.T) {
	r := newSilentRouter()
	body := []byte(`{"user":{"name":"alice","profile":{"age":25,"city":"北京"}},"action":"create"}`)
	r.ReverseHttpRequest(request.NewHttpRequest("/api/users",
		request.Headers{"Content-Type": "application/json"}, "POST", body))

	postNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("users").FindChildByKey("POST")
	if postNode == nil {
		t.Fatal("应有 POST 节点")
	}
	// JSON 嵌套应扁平化为 user.name, user.profile.age 等
	params := make(map[string]bool)
	for _, child := range postNode.GetChildren() {
		if child.GetType() == "request_param" {
			params[child.GetKey()] = true
		}
	}
	if !params["user.name"] {
		t.Error("嵌套 JSON 参数 user.name 应被扁平化")
	}
	if !params["action"] {
		t.Error("顶层参数 action 应被识别")
	}
	t.Logf("扁平化参数: %v", params)
}

// Case E09: 表单 body 参数解析
func TestExtremeE09_FormBodyParsing(t *testing.T) {
	r := newSilentRouter()
	body := []byte("username=admin&password=secret&remember_me=true")
	r.ReverseHttpRequest(request.NewHttpRequest("/api/auth/login",
		request.Headers{"Content-Type": "application/x-www-form-urlencoded"}, "POST", body))

	postNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("auth").FindChildByKey("login").FindChildByKey("POST")
	if postNode == nil {
		t.Fatal("应有 POST 节点")
	}
	for _, param := range []string{"username", "password", "remember_me"} {
		if postNode.FindChildByKey(param) == nil {
			t.Errorf("表单参数 %q 应被识别", param)
		}
	}
}

// Case E10: 并发处理 + 合并一致性
func TestExtremeE10_ConcurrentMergeConsistency(t *testing.T) {
	r := newSilentRouter()
	done := make(chan bool, 200)
	// 并发 100 个不同数字 ID
	for i := 0; i < 100; i++ {
		go func(id int) {
			r.ReverseHttpRequest(request.NewHttpRequest(
				fmt.Sprintf("/api/concurrent/items/%d", id), nil, "GET", nil))
			done <- true
		}(i)
	}
	for i := 0; i < 100; i++ {
		<-done
	}
	// 补充确保合并触发
	for i := 1000; i < 1003; i++ {
		r.ReverseHttpRequest(request.NewHttpRequest(
			fmt.Sprintf("/api/concurrent/items/%d", i), nil, "GET", nil))
	}
	itemsNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("concurrent").FindChildByKey("items")
	if itemsNode == nil {
		t.Fatal("并发后应有 items 节点")
	}
	varNode := itemsNode.GetChildByType("request_path_variable")
	if varNode == nil {
		t.Error("并发100+个数字ID后应合并为路径变量")
	}
	stats := r.GetStats()
	t.Logf("并发统计: processed=%d variables=%d merges=%d",
		stats.RequestsProcessed, stats.PathVariablesIdentified, stats.MergeAttempts)
}

// Case E11: 相似长度字符串突破阈值合并（>=6 同层兄弟）
func TestExtremeE11_SimilarLengthBreakThreshold(t *testing.T) {
	r := newSilentRouter()
	// 城市名（相似长度），超过6个应触发突破合并
	cities := []string{"beijing", "tianjin", "nanjing", "wuhan", "xian", "suzhou", "dalian"}
	for _, city := range cities {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/cities/"+city, nil, "GET", nil))
	}
	citiesNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("cities")
	varNode := citiesNode.GetChildByType("request_path_variable")
	if varNode == nil {
		t.Error("7个相似长度城市名（>= SimilarLengthBreakThreshold=6）应合并为路径变量")
	}
}

// Case E12: 相似长度字符串低于突破阈值不合并（<=5 个）
func TestExtremeE12_SimilarLengthBelowThresholdNoMerge(t *testing.T) {
	r := newSilentRouter()
	// 5 个城市名，低于突破阈值 6，不应合并
	for _, city := range []string{"beijing", "tianjin", "nanjing", "wuhan", "suzhou"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/locations/"+city, nil, "GET", nil))
	}
	locNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("locations")
	if locNode.GetChildByType("request_path_variable") != nil {
		t.Error("5 个相似长度词（< 突破阈值 6）不应合并为路径变量")
	}
}

// Case E13: 路由树 JSON 序列化/反序列化往返
func TestExtremeE13_TreeSerializationRoundtrip(t *testing.T) {
	r := newSilentRouter()
	urls := []string{
		"/api/users/123",
		"/api/users/456",
		"/api/users/789",
		"/api/products/ABC123",
		"/api/products/DEF456",
		"/api/products/GHI789",
	}
	for _, url := range urls {
		r.ReverseHttpRequest(request.NewHttpRequest(url, nil, "GET", nil))
	}
	jsonData, err := r.Tree.ToJSON()
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}
	if len(jsonData) == 0 {
		t.Fatal("序列化结果不应为空")
	}
	// 验证 JSON 包含预期的路径变量
	jsonStr := string(jsonData)
	if !strings.Contains(jsonStr, "request_path_variable") {
		t.Error("序列化 JSON 应包含路径变量节点")
	}
	t.Logf("序列化 JSON 大小: %d bytes", len(jsonData))
}

// Case E14: 混合 UUID 和整数 ID（同一资源两批不同请求）
func TestExtremeE14_MixedUUIDAndIntegerIDs(t *testing.T) {
	r := newSilentRouter()
	// UUID 批次
	for _, uuid := range []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		"6ba7b811-9dad-11d1-80b4-00c04fd430c8",
	} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/items/"+uuid, nil, "GET", nil))
	}
	// 现在有 UUID 路径变量了，再发数字 ID
	for _, id := range []string{"100", "200", "300"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/items/"+id, nil, "GET", nil))
	}
	itemsNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("items")
	varNode := itemsNode.GetChildByType("request_path_variable")
	if varNode == nil {
		t.Fatal("应有路径变量节点")
	}
	t.Logf("混合 UUID+整数路由变量: %s pattern=%v", varNode.GetKey(),
		varNode.(*node.RequestPathVariableNode).GetPattern())
}

// Case E15: 大量请求后统计指标正确
func TestExtremeE15_StatsAccuracyUnderLoad(t *testing.T) {
	r := newSilentRouter()
	total := 0
	// 100 个不同 UUID 路径
	for i := 0; i < 100; i++ {
		uuid := fmt.Sprintf("550e8400-e29b-41d4-%04d-446655440000", i)
		r.ReverseHttpRequest(request.NewHttpRequest("/bench/items/"+uuid, nil, "GET", nil))
		total++
	}
	// 50 个查询参数
	for i := 0; i < 50; i++ {
		r.ReverseHttpRequest(request.NewHttpRequest(
			fmt.Sprintf("/bench/search?q=keyword&page=%d", i+1), nil, "GET", nil))
		total++
	}
	stats := r.GetStats()
	if stats.RequestsProcessed != int64(total) {
		t.Errorf("处理请求数应为 %d，实际: %d", total, stats.RequestsProcessed)
	}
	if stats.PathVariablesIdentified == 0 {
		t.Error("应识别至少 1 个路径变量")
	}
	t.Logf("统计: %+v", stats)
}

// ─────────────────────────────────────────────────────────────────────────────
// Group F: 深度场景与遗漏分支覆盖
// ─────────────────────────────────────────────────────────────────────────────

// Case F01: 四层级联合并——每层均为整数 ID，验证 cascadeMergeLocked 递归深度
func TestExtremeF01_FourLevelCascadeMerge(t *testing.T) {
	r := newSilentRouter()
	for i := 1; i <= 3; i++ {
		url := fmt.Sprintf("/wh/%d/zone/%d/shelf/%d/item/%d", i, i*10, i*100, i*1000)
		r.ReverseHttpRequest(request.NewHttpRequest(url, nil, "GET", nil))
	}
	// wh → {wh_id}
	whNode := r.Tree.Root.FindChildByKey("wh")
	if whNode == nil {
		t.Fatal("应有 wh 路径节点")
	}
	whVar := whNode.GetChildByType("request_path_variable")
	if whVar == nil {
		t.Fatal("wh 下应合并出路径变量 {wh_id}")
	}
	// {wh_id} → zone → {zone_id}
	zoneNode := whVar.FindChildByKey("zone")
	if zoneNode == nil {
		t.Fatal("{wh_id} 下应有 zone 子节点（cascade 后保留）")
	}
	zoneVar := zoneNode.GetChildByType("request_path_variable")
	if zoneVar == nil {
		t.Fatal("zone 下应级联合并出路径变量 {zone_id}")
	}
	// {zone_id} → shelf → {shelf_id}
	shelfNode := zoneVar.FindChildByKey("shelf")
	if shelfNode == nil {
		t.Fatal("{zone_id} 下应有 shelf 子节点")
	}
	shelfVar := shelfNode.GetChildByType("request_path_variable")
	if shelfVar == nil {
		t.Fatal("shelf 下应级联合并出路径变量 {shelf_id}")
	}
	// {shelf_id} → item → {item_id}
	itemNode := shelfVar.FindChildByKey("item")
	if itemNode == nil {
		t.Fatal("{shelf_id} 下应有 item 子节点")
	}
	itemVar := itemNode.GetChildByType("request_path_variable")
	if itemVar == nil {
		t.Fatal("item 下应级联合并出路径变量 {item_id}（四层级联）")
	}
	t.Logf("四层级联成功: wh/%s/zone/%s/shelf/%s/item/%s",
		whVar.GetKey(), zoneVar.GetKey(), shelfVar.GetKey(), itemVar.GetKey())
}

// Case F02: 选择性部分合并——整数节点合并，单词固定节点保留
// 场景：先加入 "admin"，后续 3 个整数 ID → 整体相似度 3/4=0.75 → 部分合并
func TestExtremeF02_SelectivePartialMerge(t *testing.T) {
	r := newSilentRouter()
	// admin 先到，使兄弟节点集合混合
	r.ReverseHttpRequest(request.NewHttpRequest("/api/resources/admin", nil, "GET", nil))
	r.ReverseHttpRequest(request.NewHttpRequest("/api/resources/1", nil, "GET", nil))
	r.ReverseHttpRequest(request.NewHttpRequest("/api/resources/2", nil, "GET", nil))
	r.ReverseHttpRequest(request.NewHttpRequest("/api/resources/3", nil, "GET", nil))

	resNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("resources")
	if resNode == nil {
		t.Fatal("应有 resources 路径节点")
	}
	// admin 应作为固定路径保留
	if resNode.FindChildByKey("admin") == nil {
		t.Error("admin 固定路径应保留，不应被合并")
	}
	// 整数 1/2/3 应被合并为路径变量
	if resNode.GetChildByType("request_path_variable") == nil {
		t.Error("整数 ID 1/2/3 应被合并为路径变量")
	}
}

// Case F03: 后缀模式路径变量命名——001_audit/002_audit/003_audit → audit_id
func TestExtremeF03_SuffixPatternVariableNaming(t *testing.T) {
	r := newSilentRouter()
	for _, seg := range []string{"001_audit", "002_audit", "003_audit"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/logs/"+seg, nil, "GET", nil))
	}
	logsNode := r.Tree.Root.FindChildByKey("logs")
	if logsNode == nil {
		t.Fatal("应有 logs 路径节点")
	}
	varNode := logsNode.GetChildByType("request_path_variable")
	if varNode == nil {
		t.Fatal("应合并为路径变量（后缀模式：001_audit/002_audit/003_audit）")
	}
	// 后缀 _audit → 去除前导下划线 → audit → audit_id
	if varNode.GetKey() != "audit_id" {
		t.Errorf("变量名应为 audit_id（从后缀 _audit 推断），实际: %s", varNode.GetKey())
	}
	t.Logf("后缀模式变量名: %s", varNode.GetKey())
}

// Case F04: IsNeedRequest 对已完整探索路由返回 false
func TestExtremeF04_IsNeedRequestReturnsFalseForKnownRoute(t *testing.T) {
	r := newSilentRouter()
	req := request.NewHttpRequest("/api/health", nil, "GET", nil)
	// 连续处理同一路由 3 次
	for i := 0; i < 3; i++ {
		r.ReverseHttpRequest(req)
	}
	// 已见过此路由，不需要再请求
	if r.IsNeedRequest(req) {
		t.Error("已处理 3 次的路由，IsNeedRequest 应返回 false")
	}
}

// Case F05: IsNeedRequest 对新增查询参数返回 true
func TestExtremeF05_IsNeedRequestReturnsTrueForNewParam(t *testing.T) {
	r := newSilentRouter()
	// 先喂一个只有 q 参数的请求
	r.ReverseHttpRequest(request.NewHttpRequest("/api/search?q=test", nil, "GET", nil))

	// 新的请求多了 page 参数 → 需要额外请求
	newReq := request.NewHttpRequest("/api/search?q=test&page=2", nil, "GET", nil)
	if !r.IsNeedRequest(newReq) {
		t.Error("新增 page 参数的请求，IsNeedRequest 应返回 true")
	}
}

// Case F06: 路径中的 key=value 格式参数段（PathParam）
// URL 如 /api/v1/action=delete → "action" 作为路径节点，"delete" 作为参数值
func TestExtremeF06_PathSegmentKeyValueFormat(t *testing.T) {
	r := newSilentRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/v1/action=delete", nil, "POST", nil))

	apiNode := r.Tree.Root.FindChildByKey("api")
	if apiNode == nil {
		t.Fatal("应有 api 路径节点")
	}
	v1Node := apiNode.FindChildByKey("v1")
	if v1Node == nil {
		t.Fatal("应有 v1 路径节点")
	}
	// key=value 路径段以 key（"action"）为节点键
	actionNode := v1Node.FindChildByKey("action")
	if actionNode == nil {
		t.Fatal("路径参数段 action=delete 应以 'action' 为节点键创建路径节点")
	}
	// method 节点下应有 action 参数
	postNode := actionNode.FindChildByKey("POST")
	if postNode == nil {
		t.Fatal("应有 POST 方法节点")
	}
	actionParam := postNode.FindChildByKey("action")
	if actionParam == nil {
		t.Error("路径参数 action=delete 应在方法节点下创建 action 参数节点")
	}
	t.Logf("路径参数节点已正确创建：action 路径 + action 参数")
}

// Case F07: 非 Bearer 认证头规范化（Basic、Digest 方案）
func TestExtremeF07_NonBearerAuthorizationNormalization(t *testing.T) {
	r := newSilentRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/basic", request.Headers{
		"Authorization": "Basic dXNlcjpwYXNz",
	}, "GET", nil))
	r.ReverseHttpRequest(request.NewHttpRequest("/api/digest", request.Headers{
		"Authorization": "Digest username=alice, realm=example.com, nonce=abc123",
	}, "GET", nil))

	// Basic 方案：应规范化为 "Basic"
	basicNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("basic").FindChildByKey("GET")
	if basicNode == nil {
		t.Fatal("应有 /api/basic GET 方法节点")
	}
	authGroup := basicNode.FindChildByKey("Authorization")
	if authGroup == nil {
		t.Fatal("应有 Authorization header 分组节点")
	}
	if authGroup.FindChildByKey("Basic") == nil {
		t.Error("Authorization 值应规范化为方案名 'Basic'，不含 token 部分")
	}

	// Digest 方案：应规范化为 "Digest"
	digestNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("digest").FindChildByKey("GET")
	if digestNode == nil {
		t.Fatal("应有 /api/digest GET 方法节点")
	}
	authGroup2 := digestNode.FindChildByKey("Authorization")
	if authGroup2 == nil {
		t.Fatal("应有 Authorization header 分组节点")
	}
	if authGroup2.FindChildByKey("Digest") == nil {
		t.Error("Authorization 值应规范化为方案名 'Digest'，不含凭证部分")
	}
}

// Case F08: PUT/PATCH 请求的 Content-Type 路由节点创建
func TestExtremeF08_PutPatchContentTypeRouting(t *testing.T) {
	r := newSilentRouter()
	jsonHeaders := request.Headers{"Content-Type": "application/json"}
	formHeaders := request.Headers{"Content-Type": "application/x-www-form-urlencoded"}
	r.ReverseHttpRequest(request.NewHttpRequest("/api/users/1", jsonHeaders, "PUT", nil))
	r.ReverseHttpRequest(request.NewHttpRequest("/api/orders/1", formHeaders, "PATCH", nil))

	// PUT 路由下应有 Content-Type 节点
	usersNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("users")
	if usersNode == nil {
		t.Fatal("应有 users 路径节点")
	}
	// 注意：PUT 请求只有1个 user/1，尚未触发路径变量合并
	putNode := usersNode.FindChildByKey("1").FindChildByKey("PUT")
	if putNode == nil {
		t.Fatal("应有 PUT 方法节点")
	}
	ctNode := putNode.FindChildByKey("application/json")
	if ctNode == nil {
		t.Error("PUT 请求下应有 Content-Type: application/json 节点")
	}

	// PATCH 路由下应有 Content-Type 节点
	ordersNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("orders")
	patchNode := ordersNode.FindChildByKey("1").FindChildByKey("PATCH")
	if patchNode == nil {
		t.Fatal("应有 PATCH 方法节点")
	}
	ctNode2 := patchNode.FindChildByKey("application/x-www-form-urlencoded")
	if ctNode2 == nil {
		t.Error("PATCH 请求下应有 Content-Type: application/x-www-form-urlencoded 节点")
	}
}

// Case F09: 合并后新值命中已有路径变量（不触发二次合并）
func TestExtremeF09_NewValuesMatchExistingPathVariable(t *testing.T) {
	r := newSilentRouter()
	// 先触发合并
	for i := 1; i <= 3; i++ {
		r.ReverseHttpRequest(request.NewHttpRequest(fmt.Sprintf("/api/users/%d", i), nil, "GET", nil))
	}
	// 验证合并已发生
	usersNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("users")
	if usersNode.GetChildByType("request_path_variable") == nil {
		t.Fatal("前 3 个请求应触发合并")
	}
	pathVarCountBefore := 0
	for _, child := range usersNode.GetChildren() {
		if child.GetType() == "request_path_variable" {
			pathVarCountBefore++
		}
	}

	// 新增更多整数 ID → 应命中已有路径变量，不应产生新的固定路径节点或额外路径变量
	for i := 4; i <= 10; i++ {
		r.ReverseHttpRequest(request.NewHttpRequest(fmt.Sprintf("/api/users/%d", i), nil, "GET", nil))
	}
	pathVarCountAfter := 0
	fixedPathCount := 0
	for _, child := range usersNode.GetChildren() {
		if child.GetType() == "request_path_variable" {
			pathVarCountAfter++
		} else if child.GetType() == "request_path" {
			fixedPathCount++
		}
	}
	if pathVarCountAfter != pathVarCountBefore {
		t.Errorf("新整数 ID 不应产生新路径变量，合并前 %d 个路径变量，合并后 %d 个",
			pathVarCountBefore, pathVarCountAfter)
	}
	if fixedPathCount != 0 {
		t.Errorf("新整数 ID 应命中已有路径变量，不应创建 %d 个固定路径节点", fixedPathCount)
	}
}

// Case F10: 路径变量与非匹配固定节点共存（不同模式的 ID 不强制合并）
func TestExtremeF10_PathVariableCoexistsWithNonMatchingFixed(t *testing.T) {
	r := newSilentRouter()
	// 先合并整数 ID → {order_id}（模式：[0-9]+）
	for i := 1; i <= 3; i++ {
		r.ReverseHttpRequest(request.NewHttpRequest(fmt.Sprintf("/api/orders/%d", i), nil, "GET", nil))
	}
	// 再访问不匹配整数模式的特殊端点
	r.ReverseHttpRequest(request.NewHttpRequest("/api/orders/pending", nil, "GET", nil))
	r.ReverseHttpRequest(request.NewHttpRequest("/api/orders/history", nil, "GET", nil))

	ordersNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("orders")
	if ordersNode == nil {
		t.Fatal("应有 orders 路径节点")
	}
	// 整数路径变量应存在
	if ordersNode.GetChildByType("request_path_variable") == nil {
		t.Error("整数 ID 应已合并为路径变量")
	}
	// pending 和 history 不匹配整数模式，应作为固定路径保留
	if ordersNode.FindChildByKey("pending") == nil {
		t.Error("pending 不匹配整数模式，应保留为固定路径节点")
	}
	if ordersNode.FindChildByKey("history") == nil {
		t.Error("history 不匹配整数模式，应保留为固定路径节点")
	}
	t.Logf("路径变量与固定节点共存: orders/{order_id} + orders/pending + orders/history")
}

// Case F11: 자식 레벨 먼저 병합 후 상위 레벨 병합——깊이 우선 병합 구조 무결성
// 시나리오: users/1/posts/[10,20,30] → {post_id} 先行合并，再触发 users 合并
func TestExtremeF11_PreMergedSubPathSurvivesParentMerge(t *testing.T) {
	r := newSilentRouter()
	// users/1 下先形成 3 个 post → {post_id} 触发合并
	for p := 1; p <= 3; p++ {
		r.ReverseHttpRequest(request.NewHttpRequest(fmt.Sprintf("/api/users/1/posts/%d", p*10), nil, "GET", nil))
	}
	// users/2 下类似
	for p := 1; p <= 3; p++ {
		r.ReverseHttpRequest(request.NewHttpRequest(fmt.Sprintf("/api/users/2/posts/%d", p*100), nil, "GET", nil))
	}
	// users/3 触发 users 层级的合并
	r.ReverseHttpRequest(request.NewHttpRequest("/api/users/3/posts/1000", nil, "GET", nil))

	usersNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("users")
	// users/[1,2,3] 应合并为 {user_id}
	userVar := usersNode.GetChildByType("request_path_variable")
	if userVar == nil {
		t.Fatal("users/[1,2,3] 应合并为路径变量 {user_id}")
	}
	// {user_id} 下应有 posts 子节点
	postsNode := userVar.FindChildByKey("posts")
	if postsNode == nil {
		t.Fatal("{user_id} 下应有 posts 子节点（上层合并后保留）")
	}
	// posts 下应有路径变量（子层级已合并 OR 级联合并新触发）
	postVar := postsNode.GetChildByType("request_path_variable")
	if postVar == nil {
		t.Fatal("posts 下应有路径变量 {post_id}（子层先合并或级联合并）")
	}
	t.Logf("结构完整：%s → posts → %s", userVar.GetKey(), postVar.GetKey())
}

// Case F12: Accept-Language 规范化（剥离质量因子，只保留首选语言）
func TestExtremeF12_AcceptLanguageNormalization(t *testing.T) {
	r := newSilentRouter()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/content", request.Headers{
		"Accept-Language": "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7",
	}, "GET", nil))

	getNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("content").FindChildByKey("GET")
	if getNode == nil {
		t.Fatal("应有 GET 方法节点")
	}
	langGroup := getNode.FindChildByKey("Accept-Language")
	if langGroup == nil {
		t.Fatal("应有 Accept-Language header 分组节点")
	}
	// Accept-Language 规范化应只保留首选语言 "zh-CN"，去掉质量因子部分
	if langGroup.FindChildByKey("zh-CN") == nil {
		t.Errorf("Accept-Language 应规范化为首选语言 'zh-CN'，树:\n%s", r.Tree.String())
	}
}

// Case F13: ReverseHttpRequest 对 nil 请求返回错误并统计
func TestExtremeF13_NilRequestReturnsError(t *testing.T) {
	r := newSilentRouter()
	err := r.ReverseHttpRequest(nil)
	if err == nil {
		t.Error("nil 请求应返回错误")
	}
	stats := r.GetStats()
	if stats.Errors == 0 {
		t.Error("nil 请求应增加错误计数")
	}
}

// Case F14: 根路径 "/" 之后有路径变量合并——测试最浅路径层的合并
func TestExtremeF14_RootLevelPathVariableMerge(t *testing.T) {
	r := newSilentRouter()
	// 根路径下直接有 ID 段（无前缀）
	for i := 1; i <= 3; i++ {
		r.ReverseHttpRequest(request.NewHttpRequest(fmt.Sprintf("/%d/info", i), nil, "GET", nil))
	}
	// root 下 [1, 2, 3] 应合并为路径变量
	rootVar := r.Tree.Root.GetChildByType("request_path_variable")
	if rootVar == nil {
		t.Fatal("根路径下的整数段 1/2/3 应合并为路径变量")
	}
	if rootVar.FindChildByKey("info") == nil {
		t.Fatal("路径变量下应保留 info 子节点")
	}
}

// Case F15: FindRouteNode 对合并后路径变量的定位
func TestExtremeF15_FindRouteNodeAfterMerge(t *testing.T) {
	r := newSilentRouter()
	for i := 1; i <= 3; i++ {
		r.ReverseHttpRequest(request.NewHttpRequest(fmt.Sprintf("/api/products/%d", i), nil, "GET", nil))
	}
	// 验证 FindRouteNode 能正确定位合并后的路由
	req := request.NewHttpRequest("/api/products/999", nil, "GET", nil)
	methodNode, _, err := r.FindRouteNode(req)
	if err != nil {
		t.Fatalf("FindRouteNode 不应返回错误: %v", err)
	}
	if methodNode == nil {
		t.Fatal("FindRouteNode 应定位到 /api/products/{product_id}/GET 方法节点")
	}
	if methodNode.GetKey() != "GET" {
		t.Errorf("方法节点 key 应为 GET，实际: %s", methodNode.GetKey())
	}
}
