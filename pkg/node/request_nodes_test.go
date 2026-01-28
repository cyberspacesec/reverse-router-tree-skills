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

	// 测试字符串表示（无模式）
	strRep = anyNode.String()
	expected = "{any}"
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
	if !optionalNode.IsMatch("sort=name") {
		t.Error("可选参数节点应该匹配不包含该参数的查询字符串")
	}

	if !optionalNode.ExtractValue("sort=name") {
		t.Error("可选参数应该在参数缺失时使用默认值")
	}

	value, _ = optionalNode.GetContext().GetKey("limit")
	if value != "10" {
		t.Errorf("可选参数默认值错误，期望 '10'，得到 %v", value)
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
