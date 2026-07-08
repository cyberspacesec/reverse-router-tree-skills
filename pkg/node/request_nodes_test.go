package node

import (
	"testing"
)

// 测试请求路径节点
func TestRequestPathNode(t *testing.T) {
	pathNode := NewRequestPathNode("/users")

	// 测试路径匹配
	if !pathNode.IsMatch("/users") {
		t.Error("路径节点应该匹配 '/users'")
	}

	if pathNode.IsMatch("/posts") {
		t.Error("路径节点不应该匹配 '/posts'")
	}

	// 测试节点类型
	if pathNode.GetType() != "request_path" {
		t.Errorf("节点类型错误，期望 'request_path'，得到 %s", pathNode.GetType())
	}
}

// 测试HTTP方法节点
func TestRequestMethodNode(t *testing.T) {
	methodNode := NewRequestMethodNode(MethodGET)

	// 测试方法匹配
	if !methodNode.IsMatch("GET") {
		t.Error("方法节点应该匹配 'GET'")
	}

	if methodNode.IsMatch("POST") {
		t.Error("方法节点不应该匹配 'POST'")
	}

	// 测试节点类型
	if methodNode.GetType() != "request_method" {
		t.Errorf("节点类型错误，期望 'request_method'，得到 %s", methodNode.GetType())
	}
}

// 测试内容类型节点
func TestRequestContentTypeNode(t *testing.T) {
	ctNode := NewRequestContentTypeNode("application/json")

	// 测试内容类型匹配
	if !ctNode.IsMatch("application/json") {
		t.Error("内容类型节点应该匹配 'application/json'")
	}

	if ctNode.IsMatch("text/plain") {
		t.Error("内容类型节点不应该匹配 'text/plain'")
	}

	// 测试节点类型
	if ctNode.GetType() != "request_content_type" {
		t.Errorf("节点类型错误，期望 'request_content_type'，得到 %s", ctNode.GetType())
	}
}

// 测试路径变量节点
func TestRequestPathVariableNode(t *testing.T) {
	// 创建一个ID变量节点，只匹配数字
	idNode := NewRequestPathVariableNode("id", "[0-9]+")

	// 测试变量匹配
	if !idNode.IsMatch("123") {
		t.Error("ID变量节点应该匹配 '123'")
	}

	if idNode.IsMatch("abc") {
		t.Error("ID变量节点不应该匹配 'abc'")
	}

	// 测试变量提取
	if !idNode.ExtractValue("123") {
		t.Error("应该能够提取值 '123'")
	}

	// 验证提取的值
	value, exists := idNode.GetContext().GetKey("id")
	if !exists {
		t.Error("上下文中应该存在'id'键")
	}
	if value != "123" {
		t.Errorf("提取的值错误，期望 '123'，得到 %v", value)
	}

	// 测试字符串表示
	strRep := idNode.String()
	expected := "{id:[0-9]+}"
	if strRep != expected {
		t.Errorf("字符串表示错误，期望 '%s'，得到 '%s'", expected, strRep)
	}

	// 测试没有模式的变量节点
	anyNode := NewRequestPathVariableNode("any", "")
	if !anyNode.IsMatch("anything") {
		t.Error("无模式变量节点应该匹配任何非空路径段")
	}

	// 测试字符串表示（无模式，显示默认推断类型string）
	strRep = anyNode.String()
	expected = "{any:string}"
	if strRep != expected {
		t.Errorf("字符串表示错误，期望 '%s'，得到 '%s'", expected, strRep)
	}
}

// 测试查询参数节点
func TestRequestParamNode(t *testing.T) {
	// 创建一个必需的参数节点
	requiredNode := NewRequestParamNode("page", "1", true)

	// 测试参数匹配
	if !requiredNode.IsMatch("page") {
		t.Error("参数节点应该匹配参数名 'page'")
	}

	if !requiredNode.IsMatch("page=2") {
		t.Error("参数节点应该匹配 'page=2'")
	}

	if !requiredNode.IsMatch("sort=name&page=2") {
		t.Error("参数节点应该匹配包含 'page=2' 的查询字符串")
	}

	// 测试参数提取
	if !requiredNode.ExtractValue("page=2") {
		t.Error("应该能够从 'page=2' 提取值")
	}

	// 验证提取的值
	value, exists := requiredNode.GetContext().GetKey("page")
	if !exists {
		t.Error("上下文中应该存在'page'键")
	}
	if value != "2" {
		t.Errorf("提取的值错误，期望 '2'，得到 %v", value)
	}

	// 测试多参数查询字符串
	if !requiredNode.ExtractValue("sort=name&page=3&limit=10") {
		t.Error("应该能够从多参数查询字符串中提取值")
	}

	value, _ = requiredNode.GetContext().GetKey("page")
	if value != "3" {
		t.Errorf("从多参数字符串中提取的值错误，期望 '3'，得到 %v", value)
	}

	// 测试默认值
	if !requiredNode.ExtractValue("sort=name") {
		t.Error("对于必需参数，应该在参数缺失时使用默认值")
	}

	value, _ = requiredNode.GetContext().GetKey("page")
	if value != "1" {
		t.Errorf("默认值错误，期望 '1'，得到 %v", value)
	}

	// 测试可选参数
	optionalNode := NewRequestParamNode("limit", "10", false)

	// 可选参数只在参数名出现时才匹配，不应匹配不包含该参数的查询字符串
	if optionalNode.IsMatch("sort=name") {
		t.Error("可选参数节点不应该匹配不包含该参数名的查询字符串")
	}

	// 可选参数在参数名出现时应该匹配
	if !optionalNode.IsMatch("limit") {
		t.Error("可选参数节点应该匹配参数名 'limit'")
	}

	if !optionalNode.IsMatch("limit=20") {
		t.Error("可选参数节点应该匹配 'limit=20'")
	}

	if !optionalNode.IsMatch("sort=name&limit=20") {
		t.Error("可选参数节点应该匹配包含 'limit=20' 的查询字符串")
	}

	// 注意：ExtractValue 即使参数不存在也能处理（使用默认值）
	// 这和 IsMatch 是不同的语义
	if !optionalNode.ExtractValue("sort=name") {
		t.Error("可选参数应该在参数缺失时使用默认值")
	}

	value, _ = optionalNode.GetContext().GetKey("limit")
	if value != "10" {
		t.Errorf("可选参数默认值错误，期望 '10'，得到 %v", value)
	}

	// 测试参数值不误匹配（page_size 不应匹配 page 参数）
	if requiredNode.IsMatch("page_size=100") {
		t.Error("参数 'page' 不应该匹配 'page_size=100'")
	}
}

// 测试节点组合构建路由树
func TestNodeCombination(t *testing.T) {
	// 创建根节点
	root := NewBaseNode[NodeContext]("root", "/", "", NewBaseNodeContext())

	// 添加HTTP方法节点
	getNode := NewRequestMethodNode(MethodGET)
	root.AddChild(getNode)

	// 添加路径节点
	usersNode := NewRequestPathNode("/users")
	getNode.AddChild(usersNode)

	// 添加变量节点（用户ID）
	userIdNode := NewRequestPathVariableNode("id", "[0-9]+")
	usersNode.AddChild(userIdNode)

	// 添加内容类型节点
	jsonNode := NewRequestContentTypeNode("application/json")
	userIdNode.AddChild(jsonNode)

	// 测试路径匹配
	// 找到用户节点
	foundUser := root.FindNode(func(n Node[NodeContext]) bool {
		return n.GetType() == "request_path" && n.GetKey() == "/users"
	})

	if foundUser == nil {
		t.Error("应该能找到用户路径节点")
	} else {
		t.Logf("找到用户节点: %s", foundUser.GetKey())
	}

	// 找到变量节点 - FindNode返回的是接口类型
	foundVar := root.FindNode(func(n Node[NodeContext]) bool {
		return n.GetType() == "request_path_variable"
	})

	if foundVar == nil {
		t.Error("应该能找到路径变量节点")
	} else {
		// 我们可以使用节点的特性而不是依赖类型断言
		t.Logf("找到变量节点: 类型=%s, 键=%s", foundVar.GetType(), foundVar.GetKey())

		// 检查节点属性是否符合预期
		if foundVar.GetType() != "request_path_variable" || foundVar.GetKey() != "id" {
			t.Errorf("变量节点属性错误，期望类型=request_path_variable, 键=id，得到类型=%s, 键=%s",
				foundVar.GetType(), foundVar.GetKey())
		}
	}

	// 模拟请求匹配 - 使用GetChildByType和递归查找
	// GET /users/123 Content-Type: application/json
	var matchId Node[NodeContext] = nil

	// 首先找到GET方法节点
	for _, child := range root.GetChildren() {
		if child.GetType() == "request_method" && child.IsMatch(MethodGET) {
			// 然后找到/users路径节点
			for _, pathNode := range child.GetChildren() {
				if pathNode.GetType() == "request_path" && pathNode.IsMatch("/users") {
					// 然后找到变量节点
					for _, varNode := range pathNode.GetChildren() {
						if varNode.GetType() == "request_path_variable" {
							// 对于变量节点，我们需要检查是否匹配"123"
							// 使用IsMatch而不是类型断言
							if varNode.IsMatch("123") {
								matchId = varNode
								break
							}
						}
					}
				}
			}
		}
	}

	if matchId == nil {
		t.Error("应该匹配ID变量")
		return
	}

	// 检查变量节点是否能匹配"123"
	if !matchId.IsMatch("123") {
		t.Error("变量节点应该匹配'123'")
		return
	}

	// 验证变量节点的上下文处理 - 直接在上下文中设置值
	ctx := matchId.GetContext()
	ctx.SetKey("id", "123")

	// 检查设置的值
	value, exists := ctx.GetKey("id")
	if !exists || value != "123" {
		t.Errorf("设置的ID值错误，期望 '123'，得到 %v", value)
	}

	// 匹配内容类型 - 直接从子节点中查找
	matchJson := false
	for _, ct := range matchId.GetChildren() {
		if ct.GetType() == "request_content_type" && ct.IsMatch("application/json") {
			matchJson = true
			break
		}
	}

	if !matchJson {
		t.Error("应该匹配application/json内容类型")
	} else {
		t.Log("成功匹配完整路由路径")
	}
}

// 测试 RequestHeaderNode
func TestRequestHeaderNode(t *testing.T) {
	// 创建 Accept header 分组节点
	headerNode := NewRequestHeaderNode("Accept")

	if headerNode.GetHeaderName() != "Accept" {
		t.Errorf("Header名称应该是 'Accept'，实际: '%s'", headerNode.GetHeaderName())
	}
	if headerNode.GetType() != "request_header" {
		t.Errorf("节点类型应该是 'request_header'，实际: '%s'", headerNode.GetType())
	}
	if headerNode.GetKey() != "Accept" {
		t.Errorf("节点Key应该是 'Accept'，实际: '%s'", headerNode.GetKey())
	}

	// 测试 FindOrCreateValueNode
	jsonValNode := headerNode.FindOrCreateValueNode("application/json")
	if jsonValNode == nil {
		t.Fatal("应该创建 application/json 值节点")
	}
	if jsonValNode.GetHeaderName() != "Accept" {
		t.Errorf("值节点的 headerName 应该是 'Accept'，实际: '%s'", jsonValNode.GetHeaderName())
	}
	if jsonValNode.GetHeaderValue() != "application/json" {
		t.Errorf("值节点的 headerValue 应该是 'application/json'，实际: '%s'", jsonValNode.GetHeaderValue())
	}
	if jsonValNode.GetType() != "request_header_value" {
		t.Errorf("值节点类型应该是 'request_header_value'，实际: '%s'", jsonValNode.GetType())
	}

	// 再次查找相同的值节点，应该返回已有节点
	jsonValNode2 := headerNode.FindOrCreateValueNode("application/json")
	if jsonValNode2 == nil {
		t.Fatal("应该找到已有的 application/json 值节点")
	}

	// 创建第二个值节点
	htmlValNode := headerNode.FindOrCreateValueNode("text/html")
	if htmlValNode == nil {
		t.Fatal("应该创建 text/html 值节点")
	}

	// 验证子节点数量
	if headerNode.GetChildCount() != 2 {
		t.Errorf("应该有2个子节点，实际: %d", headerNode.GetChildCount())
	}
}

// 测试 RequestHeaderValueNode
func TestRequestHeaderValueNode(t *testing.T) {
	valNode := NewRequestHeaderValueNode("Authorization", "Bearer")

	if valNode.GetHeaderName() != "Authorization" {
		t.Errorf("Header名称应该是 'Authorization'，实际: '%s'", valNode.GetHeaderName())
	}
	if valNode.GetHeaderValue() != "Bearer" {
		t.Errorf("HeaderValue 应该是 'Bearer'，实际: '%s'", valNode.GetHeaderValue())
	}

	// 测试 IsMatch
	if !valNode.IsMatch("Bearer") {
		t.Error("应该匹配 'Bearer'")
	}
	if valNode.IsMatch("Basic") {
		t.Error("不应该匹配 'Basic'")
	}

	// 测试 ObserveValue
	valNode.ObserveValue("Bearer")
	metric := valNode.GetValueMetric()
	if metric.GetUniqueValueCount() != 1 {
		t.Errorf("唯一值数量应该是1，实际: %d", metric.GetUniqueValueCount())
	}
}

// 测试 RequestCookieNode
func TestRequestCookieNode(t *testing.T) {
	// 创建 lang cookie 分组节点
	cookieNode := NewRequestCookieNode("lang")

	if cookieNode.GetCookieName() != "lang" {
		t.Errorf("Cookie名称应该是 'lang'，实际: '%s'", cookieNode.GetCookieName())
	}
	if cookieNode.GetType() != "request_cookie" {
		t.Errorf("节点类型应该是 'request_cookie'，实际: '%s'", cookieNode.GetType())
	}
	if cookieNode.GetKey() != "lang" {
		t.Errorf("节点Key应该是 'lang'，实际: '%s'", cookieNode.GetKey())
	}

	// 测试 FindOrCreateValueNode
	zhValNode := cookieNode.FindOrCreateValueNode("zh-CN")
	if zhValNode == nil {
		t.Fatal("应该创建 zh-CN 值节点")
	}
	if zhValNode.GetCookieName() != "lang" {
		t.Errorf("值节点的 cookieName 应该是 'lang'，实际: '%s'", zhValNode.GetCookieName())
	}
	if zhValNode.GetCookieValue() != "zh-CN" {
		t.Errorf("值节点的 cookieValue 应该是 'zh-CN'，实际: '%s'", zhValNode.GetCookieValue())
	}
	if zhValNode.GetType() != "request_cookie_value" {
		t.Errorf("值节点类型应该是 'request_cookie_value'，实际: '%s'", zhValNode.GetType())
	}

	// 创建第二个值节点
	enValNode := cookieNode.FindOrCreateValueNode("en-US")
	if enValNode == nil {
		t.Fatal("应该创建 en-US 值节点")
	}

	// 验证子节点数量
	if cookieNode.GetChildCount() != 2 {
		t.Errorf("应该有2个子节点，实际: %d", cookieNode.GetChildCount())
	}
}

// 测试 RequestCookieValueNode
func TestRequestCookieValueNode(t *testing.T) {
	valNode := NewRequestCookieValueNode("theme", "dark")

	if valNode.GetCookieName() != "theme" {
		t.Errorf("Cookie名称应该是 'theme'，实际: '%s'", valNode.GetCookieName())
	}
	if valNode.GetCookieValue() != "dark" {
		t.Errorf("CookieValue 应该是 'dark'，实际: '%s'", valNode.GetCookieValue())
	}

	// 测试 IsMatch
	if !valNode.IsMatch("dark") {
		t.Error("应该匹配 'dark'")
	}
	if valNode.IsMatch("light") {
		t.Error("不应该匹配 'light'")
	}
}

// 测试 RequestParamNode 大小写不敏感
func TestRequestParamNode_CaseInsensitive(t *testing.T) {
	// 参数名统一小写
	paramNode := NewRequestParamNode("Page", "1", false)

	// 内部 key 应该是小写
	if paramNode.GetParamName() != "page" {
		t.Errorf("参数名应该是 'page'（小写），实际: '%s'", paramNode.GetParamName())
	}

	// IsMatch 应该大小写不敏感
	if !paramNode.IsMatch("Page") {
		t.Error("应该匹配 'Page'（大小写不敏感）")
	}
	if !paramNode.IsMatch("PAGE") {
		t.Error("应该匹配 'PAGE'（大小写不敏感）")
	}
	if !paramNode.IsMatch("page") {
		t.Error("应该匹配 'page'")
	}
}

// 测试 RequestParamNode 多值参数
func TestRequestParamNode_MultiValue(t *testing.T) {
	paramNode := NewRequestParamNode("tag", "go", false)

	// 提取多值参数
	if !paramNode.ExtractValue("tag=go&tag=web&tag=api") {
		t.Error("应该能够提取多值参数")
	}

	// 上下文中的值应该是逗号分隔
	value, exists := paramNode.GetContext().GetKey("tag")
	if !exists {
		t.Error("上下文中应该存在 'tag' 键")
	}
	if value != "go,web,api" {
		t.Errorf("多值参数应该是逗号分隔，实际: '%s'", value)
	}

	// 应该标记为多值参数
	if !paramNode.IsMultiValue() {
		t.Error("应该是多值参数")
	}
}

// 测试 RequestParamNode 默认值自动观察
func TestRequestParamNode_DefaultValueObserved(t *testing.T) {
	paramNode := NewRequestParamNode("page", "1", false)

	// 默认值应该被自动观察
	metric := paramNode.GetValueMetric()
	if metric.GetUniqueValueCount() != 1 {
		t.Errorf("默认值应该被观察，唯一值数量应该是1，实际: %d", metric.GetUniqueValueCount())
	}
	if metric.GetValueCount("1") != 1 {
		t.Errorf("默认值 '1' 的计数应该是1，实际: %d", metric.GetValueCount("1"))
	}
}

// 测试 RequestParamNode 必需性推断
func TestRequestParamNode_InferRequired(t *testing.T) {
	// 场景1：出现率高 → 必需
	pn1 := NewRequestParamNode("page", "", false)
	for i := 0; i < 10; i++ {
		pn1.IncrementPresenceCount()
	}
	// 10/10 = 1.0 >= 0.9 → 必需
	if !pn1.InferRequired(10, 0.9) {
		t.Error("出现率 10/10 应判定为必需")
	}
	if !pn1.IsRequired() {
		t.Error("InferRequired 后 IsRequired 应为 true")
	}

	// 场景2：出现率中等 → 可选
	pn2 := NewRequestParamNode("size", "", false)
	for i := 0; i < 6; i++ {
		pn2.IncrementPresenceCount()
	}
	// 6/10 = 0.6 < 0.9 → 可选
	if pn2.InferRequired(10, 0.9) {
		t.Error("出现率 6/10 应判定为可选")
	}
	if pn2.IsRequired() {
		t.Error("InferRequired 后 IsRequired 应为 false")
	}

	// 场景3：样本不足 → 保持默认
	pn3 := NewRequestParamNode("foo", "", false)
	pn3.IncrementPresenceCount()
	// 单次请求，样本不足，保持默认 false
	if pn3.InferRequired(1, 0.9) {
		t.Error("样本不足（1次请求）不应判定为必需，应保持默认")
	}

	// 场景4：阈值边界
	pn4 := NewRequestParamNode("bar", "", false)
	for i := 0; i < 9; i++ {
		pn4.IncrementPresenceCount()
	}
	// 9/10 = 0.9 == 阈值0.9 → 必需（>=）
	if !pn4.InferRequired(10, 0.9) {
		t.Error("出现率 9/10 = 0.9 达到阈值应判定为必需")
	}
}

// 测试 RequestParamNode 出现计数
func TestRequestParamNode_PresenceCount(t *testing.T) {
	pn := NewRequestParamNode("tag", "", false)

	if pn.GetPresenceCount() != 0 {
		t.Errorf("初始出现次数应为0，实际: %d", pn.GetPresenceCount())
	}

	pn.IncrementPresenceCount()
	pn.IncrementPresenceCount()
	pn.IncrementPresenceCount()

	if pn.GetPresenceCount() != 3 {
		t.Errorf("3次累加后出现次数应为3，实际: %d", pn.GetPresenceCount())
	}
}

// 测试 RequestPathVariableNode 文件扩展名排除
func TestRequestPathVariableNode_FileExtensionExclusion(t *testing.T) {
	// 无模式的变量节点应该排除有文件扩展名的路径段
	varNode := NewRequestPathVariableNode("resource", "")

	// 没有文件扩展名应该匹配
	if !varNode.IsMatch("123") {
		t.Error("无扩展名路径 '123' 应该匹配")
	}

	// 有文件扩展名不应该匹配
	if varNode.IsMatch("data.json") {
		t.Error("有扩展名路径 'data.json' 不应该匹配")
	}
	if varNode.IsMatch("style.css") {
		t.Error("有扩展名路径 'style.css' 不应该匹配")
	}
	if varNode.IsMatch("page.html") {
		t.Error("有扩展名路径 'page.html' 不应该匹配")
	}

	// 有正则模式的变量节点应该按模式匹配（不检查扩展名）
	intVarNode := NewRequestPathVariableNode("id", "[0-9]+")
	if !intVarNode.IsMatch("123") {
		t.Error("数字模式应该匹配 '123'")
	}
}
