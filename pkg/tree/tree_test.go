package tree

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

func TestNewTree(t *testing.T) {
	tree := NewTree()
	if tree.Root == nil {
		t.Fatal("新创建的树应该有根节点")
	}
	if tree.Root.GetType() != "root" {
		t.Errorf("根节点类型错误，期望 'root'，得到 '%s'", tree.Root.GetType())
	}
}

func TestTree_AddNode(t *testing.T) {
	tree := NewTree()

	childNode := node.NewRequestMethodNode("GET")
	err := tree.AddNode("", childNode)
	if err != nil {
		t.Fatalf("添加节点到空路径失败: %v", err)
	}

	getNode := tree.Root.FindChildByKey("GET")
	if getNode == nil {
		t.Fatal("应该找到 GET 方法节点")
	}
}

func TestTree_AddNodeWithPath(t *testing.T) {
	tree := NewTree()

	methodNode := node.NewRequestMethodNode("GET")
	err := tree.AddNode("api/users", methodNode)
	if err != nil {
		t.Fatalf("添加节点到路径失败: %v", err)
	}

	apiNode := tree.Root.FindChildByKey("api")
	if apiNode == nil {
		t.Fatal("应该创建 'api' 路径节点")
	}

	usersNode := apiNode.FindChildByKey("users")
	if usersNode == nil {
		t.Fatal("应该创建 'users' 路径节点")
	}

	getNode := usersNode.FindChildByKey("GET")
	if getNode == nil {
		t.Fatal("应该找到 GET 方法节点")
	}
}

func TestTree_AddNodeDuplicatePath(t *testing.T) {
	tree := NewTree()

	err := tree.AddNode("api/users", node.NewRequestMethodNode("GET"))
	if err != nil {
		t.Fatalf("第一次添加失败: %v", err)
	}

	err = tree.AddNode("api/users", node.NewRequestMethodNode("POST"))
	if err != nil {
		t.Fatalf("第二次添加失败: %v", err)
	}

	usersNode := tree.Root.FindChildByKey("api").FindChildByKey("users")
	if usersNode.GetChildCount() != 2 {
		t.Errorf("users下应该有2个子节点，实际: %d", usersNode.GetChildCount())
	}
}

func TestTree_FindNodeByPath(t *testing.T) {
	tree := NewTree()

	tree.AddNode("api/users", node.NewRequestMethodNode("GET"))

	found := tree.FindNodeByPath("api/users")
	if found == nil {
		t.Fatal("应该找到 'api/users' 路径")
	}

	notFound := tree.FindNodeByPath("api/posts")
	if notFound != nil {
		t.Error("不应该找到 'api/posts' 路径")
	}

	root := tree.FindNodeByPath("")
	if root == nil || root.GetType() != "root" {
		t.Error("空路径应该返回根节点")
	}
}

func TestTree_AddNodeNil(t *testing.T) {
	tree := NewTree()
	err := tree.AddNode("api/users", nil)
	if err == nil {
		t.Error("添加nil节点应该返回错误")
	}
}

// 测试路由树可视化
func TestTree_String(t *testing.T) {
	tree := NewTree()
	tree.AddNode("api/users", node.NewRequestMethodNode("GET"))
	tree.AddNode("api/posts", node.NewRequestMethodNode("POST"))

	// 添加路径变量节点
	usersNode := tree.Root.FindChildByKey("api").FindChildByKey("users")
	usersNode.AddChild(node.NewRequestPathVariableNode("id", "[0-9]+"))

	output := tree.String()
	t.Logf("路由树可视化:\n%s", output)

	if !strings.Contains(output, "api") {
		t.Error("输出应该包含 'api'")
	}
	if !strings.Contains(output, "users") {
		t.Error("输出应该包含 'users'")
	}
	if !strings.Contains(output, "GET") {
		t.Error("输出应该包含 'GET'")
	}
	if !strings.Contains(output, "POST") {
		t.Error("输出应该包含 'POST'")
	}
	if !strings.Contains(output, "Var") {
		t.Error("输出应该包含 'Var'（路径变量）")
	}
}

// 测试JSON导出
func TestTree_ToJSON(t *testing.T) {
	tree := NewTree()
	tree.AddNode("api/users", node.NewRequestMethodNode("GET"))

	jsonData, err := tree.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON 失败: %v", err)
	}

	t.Logf("JSON 输出:\n%s", string(jsonData))

	var root RouteNodeJSON
	if err := json.Unmarshal(jsonData, &root); err != nil {
		t.Fatalf("JSON反序列化失败: %v", err)
	}

	if root.Type != "root" {
		t.Errorf("根节点类型应该是 'root'，得到 '%s'", root.Type)
	}

	apiNode := findChildByKey(root.Children, "api")
	if apiNode == nil {
		t.Fatal("应该找到 'api' 节点")
	}
	if apiNode.Type != "request_path" {
		t.Errorf("api节点类型应该是 'request_path'，得到 '%s'", apiNode.Type)
	}
}

// 测试JSON导入导出往返
func TestTree_JSONRoundTrip(t *testing.T) {
	tree := NewTree()
	tree.AddNode("api/users", node.NewRequestMethodNode("GET"))
	tree.AddNode("api/posts", node.NewRequestMethodNode("POST"))
	tree.Root.FindChildByKey("api").FindChildByKey("users").AddChild(node.NewRequestParamNode("page", "1", false))

	jsonData, err := tree.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON 失败: %v", err)
	}

	tree2 := NewTree()
	err = tree2.FromJSON(jsonData)
	if err != nil {
		t.Fatalf("FromJSON 失败: %v", err)
	}

	apiNode := tree2.Root.FindChildByKey("api")
	if apiNode == nil {
		t.Fatal("导入后应该找到 api 节点")
	}
	usersNode := apiNode.FindChildByKey("users")
	if usersNode == nil {
		t.Fatal("导入后应该找到 users 节点")
	}
	getNode := usersNode.FindChildByKey("GET")
	if getNode == nil {
		t.Fatal("导入后应该找到 GET 方法节点")
	}
}

// 测试 JSON 往返保留类型信息（物理类型、逻辑类型、必需性、出现计数）
func TestTree_JSONRoundTrip_WithTypeInfo(t *testing.T) {
	tree := NewTree()
	tree.AddNode("api/users", node.NewRequestMethodNode("GET"))

	usersNode := tree.Root.FindChildByKey("api").FindChildByKey("users")
	getNode := usersNode.FindChildByKey("GET")

	// 创建带完整类型信息的参数节点
	pageNode := node.NewRequestParamNode("page", "1", false)
	pageNode.SetValueType(value.Type(value.PhysicalTypeInteger))
	pageNode.SetLogicalType(value.LogicalTypeInteger)
	pageNode.SetRequired(true)
	pageNode.IncrementPresenceCount()
	pageNode.IncrementPresenceCount()
	pageNode.IncrementPresenceCount()
	getNode.AddChild(pageNode)

	// 创建带类型信息的路径变量节点
	varNode := node.NewRequestPathVariableNode("users_id", "[0-9]+")
	varNode.SetType(value.Type(value.PhysicalTypeInteger))
	varNode.SetLogicalType(value.LogicalTypePhoneNumber)
	usersNode.AddChild(varNode)

	jsonData, err := tree.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON 失败: %v", err)
	}

	tree2 := NewTree()
	if err := tree2.FromJSON(jsonData); err != nil {
		t.Fatalf("FromJSON 失败: %v", err)
	}

	// 验证参数节点类型信息往返一致
	usersNode2 := tree2.Root.FindChildByKey("api").FindChildByKey("users")
	getNode2 := usersNode2.FindChildByKey("GET")
	pageNode2 := getNode2.FindChildByKey("page").(*node.RequestParamNode)

	if !pageNode2.IsRequired() {
		t.Error("往返后 page 必需性应保留为 true")
	}
	if pageNode2.GetValueType() != value.Type(value.PhysicalTypeInteger) {
		t.Errorf("往返后 page 物理类型应为 integer，实际: %s", pageNode2.GetValueType())
	}
	if pageNode2.GetLogicalType() != value.LogicalTypeInteger {
		t.Errorf("往返后 page 逻辑类型应为 integer，实际: %s", pageNode2.GetLogicalType())
	}
	if pageNode2.GetPresenceCount() != 3 {
		t.Errorf("往返后 page 出现计数应为3，实际: %d", pageNode2.GetPresenceCount())
	}

	// 验证路径变量节点类型信息往返一致
	varNode2 := usersNode2.GetChildByType("request_path_variable").(*node.RequestPathVariableNode)
	if varNode2.GetValueType() != value.Type(value.PhysicalTypeInteger) {
		t.Errorf("往返后变量物理类型应为 integer，实际: %s", varNode2.GetValueType())
	}
	if varNode2.GetLogicalType() != value.LogicalTypePhoneNumber {
		t.Errorf("往返后变量逻辑类型应为 phone，实际: %s", varNode2.GetLogicalType())
	}
	// 验证正则模式往返一致（无模式则反序列化后仍无模式）
	p2 := varNode2.GetPattern()
	if p2 == nil || p2.String() != "[0-9]+" {
		t.Errorf("往返后变量正则模式应为 [0-9]+，实际: %v", p2)
	}
}

// 测试路由统计
func TestTree_Stats(t *testing.T) {
	tree := NewTree()
	tree.AddNode("api/users", node.NewRequestMethodNode("GET"))
	tree.AddNode("api/posts", node.NewRequestMethodNode("POST"))
	usersNode := tree.Root.FindChildByKey("api").FindChildByKey("users")
	usersNode.FindChildByKey("GET").AddChild(node.NewRequestParamNode("page", "1", false))

	stats := tree.Stats()
	t.Logf("路由统计: %+v", stats)

	if stats.TotalNodes < 5 {
		t.Errorf("总节点数应该>=5，实际: %d", stats.TotalNodes)
	}
	if stats.PathNodes < 2 {
		t.Errorf("路径节点数应该>=2，实际: %d", stats.PathNodes)
	}
	if stats.MethodNodes < 2 {
		t.Errorf("方法节点数应该>=2，实际: %d", stats.MethodNodes)
	}
	if stats.ParamNodes < 1 {
		t.Errorf("参数节点数应该>=1，实际: %d", stats.ParamNodes)
	}
}

// 辅助函数
func findChildByKey(children []*RouteNodeJSON, key string) *RouteNodeJSON {
	for _, child := range children {
		if child.Key == key {
			return child
		}
	}
	return nil
}

// 测试 Header/Cookie 节点的可视化
func TestTree_HeaderCookieVisualization(t *testing.T) {
	tree := NewTree()
	tree.AddNode("api/data", node.NewRequestMethodNode("GET"))

	getNode := tree.Root.FindChildByKey("api").FindChildByKey("data").FindChildByKey("GET")

	// 添加 Header 节点
	acceptHeader := node.NewRequestHeaderNode("Accept")
	acceptHeader.FindOrCreateValueNode("application/json")
	acceptHeader.FindOrCreateValueNode("text/html")
	getNode.AddChild(acceptHeader)

	// 添加 Cookie 节点
	langCookie := node.NewRequestCookieNode("lang")
	langCookie.FindOrCreateValueNode("zh-CN")
	langCookie.FindOrCreateValueNode("en-US")
	getNode.AddChild(langCookie)

	output := tree.String()
	t.Logf("Header/Cookie 可视化:\n%s", output)

	if !strings.Contains(output, "Header") {
		t.Error("输出应该包含 'Header'")
	}
	if !strings.Contains(output, "Cookie") {
		t.Error("输出应该包含 'Cookie'")
	}
}

// 测试 Header/Cookie 节点的 JSON 导出
func TestTree_HeaderCookieJSON(t *testing.T) {
	tree := NewTree()
	tree.AddNode("api/data", node.NewRequestMethodNode("GET"))

	getNode := tree.Root.FindChildByKey("api").FindChildByKey("data").FindChildByKey("GET")

	acceptHeader := node.NewRequestHeaderNode("Accept")
	acceptHeader.FindOrCreateValueNode("application/json")
	getNode.AddChild(acceptHeader)

	jsonData, err := tree.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON 失败: %v", err)
	}

	t.Logf("JSON:\n%s", string(jsonData))

	var root RouteNodeJSON
	if err := json.Unmarshal(jsonData, &root); err != nil {
		t.Fatalf("JSON反序列化失败: %v", err)
	}

	// 找到 header 节点
	apiNode := findChildByKey(root.Children, "api")
	dataNode := findChildByKey(apiNode.Children, "data")
	getNodeJSON := findChildByKey(dataNode.Children, "GET")
	acceptNode := findChildByKey(getNodeJSON.Children, "Accept")

	if acceptNode == nil {
		t.Fatal("JSON 中应该找到 Accept 节点")
	}
	if acceptNode.Type != "request_header" {
		t.Errorf("Accept 节点类型应该是 'request_header'，实际: '%s'", acceptNode.Type)
	}

	// 值子节点
	jsonValNode := findChildByKey(acceptNode.Children, "application/json")
	if jsonValNode == nil {
		t.Fatal("JSON 中应该找到 application/json 值节点")
	}
	if jsonValNode.Type != "request_header_value" {
		t.Errorf("值节点类型应该是 'request_header_value'，实际: '%s'", jsonValNode.Type)
	}
}

// 测试 Header/Cookie JSON 往返
func TestTree_HeaderCookieJSONRoundTrip(t *testing.T) {
	tree := NewTree()
	tree.AddNode("api/data", node.NewRequestMethodNode("GET"))

	getNode := tree.Root.FindChildByKey("api").FindChildByKey("data").FindChildByKey("GET")

	acceptHeader := node.NewRequestHeaderNode("Accept")
	acceptHeader.FindOrCreateValueNode("application/json")
	getNode.AddChild(acceptHeader)

	langCookie := node.NewRequestCookieNode("lang")
	langCookie.FindOrCreateValueNode("zh-CN")
	getNode.AddChild(langCookie)

	jsonData, _ := tree.ToJSON()

	tree2 := NewTree()
	tree2.FromJSON(jsonData)

	// 验证 Header 节点
	dataNode2 := tree2.Root.FindChildByKey("api").FindChildByKey("data")
	getNode2 := dataNode2.FindChildByKey("GET")
	acceptNode2 := getNode2.FindChildByKey("Accept")
	if acceptNode2 == nil {
		t.Fatal("导入后应该找到 Accept Header 节点")
	}
	if acceptNode2.GetType() != "request_header" {
		t.Errorf("导入后 Accept 节点类型错误: '%s'", acceptNode2.GetType())
	}

	// 验证 Cookie 节点
	langNode2 := getNode2.FindChildByKey("lang")
	if langNode2 == nil {
		t.Fatal("导入后应该找到 lang Cookie 节点")
	}
	if langNode2.GetType() != "request_cookie" {
		t.Errorf("导入后 lang 节点类型错误: '%s'", langNode2.GetType())
	}
}

// 测试统计信息包含 Header/Cookie
func TestTree_StatsWithHeaderCookie(t *testing.T) {
	tree := NewTree()
	tree.AddNode("api/data", node.NewRequestMethodNode("GET"))

	getNode := tree.Root.FindChildByKey("api").FindChildByKey("data").FindChildByKey("GET")

	acceptHeader := node.NewRequestHeaderNode("Accept")
	acceptHeader.FindOrCreateValueNode("application/json")
	acceptHeader.FindOrCreateValueNode("text/html")
	getNode.AddChild(acceptHeader)

	langCookie := node.NewRequestCookieNode("lang")
	langCookie.FindOrCreateValueNode("zh-CN")
	getNode.AddChild(langCookie)

	stats := tree.Stats()

	if stats.HeaderNodes != 1 {
		t.Errorf("HeaderNodes 应该是1，实际: %d", stats.HeaderNodes)
	}
	if stats.HeaderValueNodes != 2 {
		t.Errorf("HeaderValueNodes 应该是2，实际: %d", stats.HeaderValueNodes)
	}
	if stats.CookieNodes != 1 {
		t.Errorf("CookieNodes 应该是1，实际: %d", stats.CookieNodes)
	}
	if stats.CookieValueNodes != 1 {
		t.Errorf("CookieValueNodes 应该是1，实际: %d", stats.CookieValueNodes)
	}

	t.Logf("完整统计: %+v", stats)
}

// TestNormalizePath_ViaAddNode 覆盖 normalizePath 的 "//" 压缩循环与首尾斜杠裁剪分支。
// 通过 AddNode 间接调用，路径含多重斜杠应被归一为单段。
func TestNormalizePath_ViaAddNode(t *testing.T) {
	tree := NewTree()
	// 路径含首尾斜杠 + 连续斜杠，应归一为 "api/users"
	err := tree.AddNode("//api//users//", node.NewRequestMethodNode("GET"))
	if err != nil {
		t.Fatalf("AddNode 归一化路径失败: %v", err)
	}
	apiNode := tree.Root.FindChildByKey("api")
	if apiNode == nil {
		t.Fatal("归一化后应找到 api 节点")
	}
	usersNode := apiNode.FindChildByKey("users")
	if usersNode == nil {
		t.Fatal("归一化后应找到 users 节点")
	}
	// 不应出现空 key 的中间节点（连续斜杠压缩的副作用）
	for _, child := range tree.Root.GetChildren() {
		if child.GetKey() == "" {
			t.Error("归一化后不应存在空 key 节点")
		}
	}
}

// TestTree_FromJSON_InvalidJSON 覆盖 FromJSON 的 json.Unmarshal 失败分支（80%→更高）。
func TestTree_FromJSON_InvalidJSON(t *testing.T) {
	tree := NewTree()
	err := tree.FromJSON([]byte(`{invalid json`))
	if err == nil {
		t.Fatal("非法 JSON 应返回错误")
	}
	if !strings.Contains(err.Error(), "JSON反序列化失败") {
		t.Errorf("错误信息应含 'JSON反序列化失败'，实际: %v", err)
	}
}

// TestTree_JSONRoundTrip_AllNodeTypes 覆盖 jsonToNode/nodeToJSON 的全部节点类型分支，
// 特别是 request_header_value / request_cookie_value（用 Value 字段构造）与 default 分支。
func TestTree_JSONRoundTrip_AllNodeTypes(t *testing.T) {
	tree := NewTree()
	tree.AddNode("api/data", node.NewRequestMethodNode("POST"))

	postNode := tree.Root.FindChildByKey("api").FindChildByKey("data").FindChildByKey("POST")
	// Content-Type 节点
	postNode.AddChild(node.NewRequestContentTypeNode("application/json"))
	// 参数节点带完整字段（multiValue + presenceCount + defaultValue 走各分支）
	p := node.NewRequestParamNode("tags", "default", true)
	p.SetValueType(value.Type(value.PhysicalTypeInteger))
	p.SetLogicalType(value.LogicalTypeInteger)
	p.SetMultiValue(true)
	p.SetPresenceCount(5)
	postNode.AddChild(p)
	// Header + 值
	h := node.NewRequestHeaderNode("X-Trace-Id")
	h.AddChild(node.NewRequestHeaderValueNode("X-Trace-Id", "abc-123"))
	postNode.AddChild(h)
	// Cookie + 值
	c := node.NewRequestCookieNode("session")
	c.AddChild(node.NewRequestCookieValueNode("session", "xyz-789"))
	postNode.AddChild(c)

	jsonData, err := tree.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON 失败: %v", err)
	}

	tree2 := NewTree()
	if err := tree2.FromJSON(jsonData); err != nil {
		t.Fatalf("FromJSON 失败: %v", err)
	}

	post2 := tree2.Root.FindChildByKey("api").FindChildByKey("data").FindChildByKey("POST")
	if post2 == nil {
		t.Fatal("往返后应找到 POST 节点")
	}
	// Content-Type
	if ct := post2.FindChildByKey("application/json"); ct == nil || ct.GetType() != "request_content_type" {
		t.Error("往返后 Content-Type 节点缺失或类型错误")
	}
	// 参数 multiValue/presenceCount 往返
	if p2, ok := post2.FindChildByKey("tags").(*node.RequestParamNode); ok {
		if !p2.IsMultiValue() {
			t.Error("往返后 multiValue 应保留 true")
		}
		if p2.GetPresenceCount() != 5 {
			t.Errorf("往返后 presenceCount 应为 5，实际 %d", p2.GetPresenceCount())
		}
		if p2.GetDefaultValue() != "default" {
			t.Errorf("往返后 defaultValue 应为 'default'，实际 %q", p2.GetDefaultValue())
		}
	} else {
		t.Error("往返后 tags 参数节点应可断言为 RequestParamNode")
	}
	// Header 值节点（走 request_header_value 的 jn.Value/jn.Key 构造分支）
	h2 := post2.GetChildByType("request_header")
	if h2 == nil {
		t.Fatal("往返后应找到 header 节点")
	}
	hv2 := h2.GetChildByType("request_header_value")
	if hv2 == nil {
		t.Fatal("往返后应找到 header 值节点")
	}
	// Cookie 值节点
	c2 := post2.GetChildByType("request_cookie")
	if c2 == nil {
		t.Fatal("往返后应找到 cookie 节点")
	}
	cv2 := c2.GetChildByType("request_cookie_value")
	if cv2 == nil {
		t.Fatal("往返后应找到 cookie 值节点")
	}
}

// TestTree_JSONRoundTrip_DefaultType 覆盖 jsonToNode 的 default 分支：
// 未知节点类型经 JSON 往返后应重建为 BaseNode，保留 type/key。
func TestTree_JSONRoundTrip_DefaultType(t *testing.T) {
	// 直接构造 JSON 含未知类型节点，覆盖 jsonToNode 的 default 分支
	raw := []byte(`{
		"type": "root",
		"key": "root",
		"children": [{
			"type": "custom_unknown",
			"key": "mystery",
			"value": "v1",
			"children": [{"type": "request_path", "key": "leaf"}]
		}]
	}`)
	tree2 := NewTree()
	if err := tree2.FromJSON(raw); err != nil {
		t.Fatalf("FromJSON 失败: %v", err)
	}
	unknown := tree2.Root.FindChildByKey("mystery")
	if unknown == nil {
		t.Fatal("default 分支应重建 mystery 节点")
	}
	if unknown.GetType() != "custom_unknown" {
		t.Errorf("default 分支应保留 type 'custom_unknown'，实际 %q", unknown.GetType())
	}
	if unknown.GetValue() != "v1" {
		t.Errorf("default 分支应保留 value 'v1'，实际 %q", unknown.GetValue())
	}
	if unknown.FindChildByKey("leaf") == nil {
		t.Error("default 节点的子节点应被递归重建")
	}
}
