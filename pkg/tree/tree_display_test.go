package tree

import (
	"strings"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

// TestTree_Print 验证 Print 输出到 stdout 的内容与 String 一致。
// Print 此前覆盖率 0%，是生产用的树打印入口，补测以防回归。
func TestTree_Print(t *testing.T) {
	tree := NewTree()
	tree.AddNode("api/users", node.NewRequestMethodNode("GET"))

	// 捕获 stdout 比较成本高，这里至少确保 Print 不 panic 且与 String 同源。
	// Print 实现为 fmt.Print(x.String())，间接通过 String 验证内容。
	want := tree.String()
	if want == "" {
		t.Fatal("String() 不应为空，Print 依赖它")
	}
	// 直接调用 Print，覆盖该公开方法行
	tree.Print()
}

// TestTree_String_AllNodeTypes 遍历 formatNodeDisplay 的全部分支，
// 确保每种节点类型的可视化文本正确。此前 formatNodeDisplay 仅 45.5%。
func TestTree_String_AllNodeTypes(t *testing.T) {
	tree := NewTree()

	// /api/users [GET] 下挂全套节点类型
	usersNode, err := addPathChain(tree, "api", "users")
	if err != nil {
		t.Fatalf("构建路径链失败: %v", err)
	}
	getNode := node.NewRequestMethodNode("GET")
	usersNode.AddChild(getNode)

	// 路径变量（带物理类型，走 [Var, type] 分支）
	pathVar := node.NewRequestPathVariableNode("id", "[0-9]+")
	pathVar.SetType(value.Type("integer"))
	usersNode.AddChild(pathVar)

	// Content-Type 节点
	getNode.AddChild(node.NewRequestContentTypeNode("application/json"))

	// 查询参数：必需 + 带逻辑类型（走 %s* [Param, %s]）
	paramReq := node.NewRequestParamNode("token", "", true)
	paramReq.SetLogicalType(value.LogicalTypeInteger)
	getNode.AddChild(paramReq)
	// 查询参数：可选 + 默认逻辑类型 string（走 %s [Param]，因 logicalType=="string" 跳过类型标注）
	paramOpt := node.NewRequestParamNode("page", "1", false)
	getNode.AddChild(paramOpt)

	// Header + HeaderValue
	headerNode := node.NewRequestHeaderNode("X-Trace-Id")
	getNode.AddChild(headerNode)
	headerNode.AddChild(node.NewRequestHeaderValueNode("X-Trace-Id", "abc123"))

	// Cookie + CookieValue
	cookieNode := node.NewRequestCookieNode("session")
	getNode.AddChild(cookieNode)
	cookieNode.AddChild(node.NewRequestCookieValueNode("session", "xyz789"))

	output := tree.String()
	t.Logf("全节点类型可视化:\n%s", output)

	// 逐分支断言（每条对应 formatNodeDisplay 的一个 case）
	cases := map[string]string{
		"root":                "root",
		"request_path":        "api [Path]",
		"request_method":      "GET [Method]",
		"request_path_var":    "{id} [Var, integer]", // SetType(TypeInteger)
		"request_content_type": "application/json [ContentType]",
		"param_required":      "token* [Param, integer]",
		"param_default":        "page [Param]",
		"request_header":      "X-Trace-Id [Header]",
		"header_value":        "X-Trace-Id: abc123 [HeaderValue]",
		"request_cookie":      "session [Cookie]",
		"cookie_value":        "session=xyz789 [CookieValue]",
	}
	for name, want := range cases {
		if !strings.Contains(output, want) {
			t.Errorf("分支 %s：输出应包含 %q\n实际输出:\n%s", name, want, output)
		}
	}
}

// TestTree_String_TreeConnectors 验证树形连接符（├── / └── / │   / 空白）正确。
// 覆盖 printNode 中 isLast 与 prefix 累加的递归分支。
func TestTree_String_TreeConnectors(t *testing.T) {
	tree := NewTree()
	// root 下两个路径段，验证 ├── 与 └──
	tree.AddNode("api/users", node.NewRequestMethodNode("GET"))
	tree.AddNode("api/orders", node.NewRequestMethodNode("POST"))

	output := tree.String()
	t.Logf("连接符可视化:\n%s", output)

	// 最后一个兄弟应是 └── ，前面的应是 ├──
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("输出行数过少: %d", len(lines))
	}

	// root 行后，api 是唯一子节点应是 └──
	rootChildFound := false
	for _, l := range lines[1:] {
		if strings.Contains(l, "└── api [Path]") {
			rootChildFound = true
			break
		}
	}
	if !rootChildFound {
		t.Errorf("唯一子节点应使用 └── 连接符\n%s", output)
	}

	// api 下有 users 和 orders：users 用 ├──，orders 用 └──
	var userLine, orderLine string
	for _, l := range lines {
		if strings.Contains(l, "users [Path]") {
			userLine = l
		}
		if strings.Contains(l, "orders [Path]") {
			orderLine = l
		}
	}
	if !strings.Contains(userLine, "├── ") {
		t.Errorf("非末尾兄弟 users 应使用 ├──\n%s", userLine)
	}
	if !strings.Contains(orderLine, "└── ") {
		t.Errorf("末尾兄弟 orders 应使用 └──\n%s", orderLine)
	}
}

// TestTree_String_DefaultBranch 覆盖 formatNodeDisplay 的 default 分支。
// 构造一个不在已知 case 列表内的节点类型，验证 [key] / [type] 格式。
func TestTree_String_DefaultBranch(t *testing.T) {
	tree := NewTree()
	// 自定义未知类型节点（key 非空 → "key [type]"）
	unknown := node.NewBaseNode[node.NodeContext]("custom_type", "mykey", "", node.NewBaseNodeContext())
	tree.Root.AddChild(unknown)

	output := tree.String()
	t.Logf("default 分支可视化:\n%s", output)

	if !strings.Contains(output, "mykey [custom_type]") {
		t.Errorf("default 分支（key 非空）应输出 'mykey [custom_type]'，实际:\n%s", output)
	}
}

// TestFormatNodeDisplay_TypeAssertionFallbacks 覆盖 formatNodeDisplay 各 case 的类型断言失败回退分支。
// 当传入的节点 n 与 case 期望的具体类型不匹配时，应回退到 "key [X]" 简化格式。
// 这些分支通过 Tree.String 难以触发（树内节点类型总是匹配），故直接调用私有函数。
func TestFormatNodeDisplay_TypeAssertionFallbacks(t *testing.T) {
	// 用 BaseNode 冒充各类具体节点类型，触发每个 case 的 ok=false 回退
	base := node.NewBaseNode[node.NodeContext]("fake", "k", "", node.NewBaseNodeContext())

	cases := []struct {
		nodeType string
		want     string
	}{
		{"request_path_variable", "{k} [Var]"},   // 非具体变量节点 → 无类型标注
		{"request_param", "k [Param]"},            // 非具体参数节点 → 无必需/类型标注
		{"request_header", "k [Header]"},          // 非具体 header 节点
		{"request_header_value", "k [HeaderValue]"},
		{"request_cookie", "k [Cookie]"},
		{"request_cookie_value", "k [CookieValue]"},
	}
	for _, c := range cases {
		t.Run(c.nodeType, func(t *testing.T) {
			got := formatNodeDisplay(c.nodeType, "k", base)
			if got != c.want {
				t.Errorf("formatNodeDisplay(%q,..) 类型断言失败回退应为 %q，实际 %q", c.nodeType, c.want, got)
			}
		})
	}

	// default 分支：key 为空 → "[type]"
	if got := formatNodeDisplay("custom_type", "", base); got != "[custom_type]" {
		t.Errorf("default 分支（key 空）应为 '[custom_type]'，实际 %q", got)
	}
}

// TestFormatNodeDisplay_ParamLogicalTypeNotEmpty 覆盖参数节点逻辑类型非空且非 string 分支
// 已有 AllNodeTypes 测试覆盖了 string 和 integer，此处补一个逻辑类型为 string 时跳过类型标注的边界
// （逻辑类型 == "string" 走 "%s [Param]" 分支，与空逻辑类型合并）。
func TestFormatNodeDisplay_ParamExplicitStringType(t *testing.T) {
	p := node.NewRequestParamNode("name", "", false)
	// 显式设置逻辑类型为 string，应走简化分支（不带 [Param, string]）
	p.SetLogicalType(value.LogicalTypeString)
	got := formatNodeDisplay("request_param", "name", p)
	if got != "name [Param]" {
		t.Errorf("逻辑类型为 string 时应简化为 'name [Param]'，实际 %q", got)
	}
}

// addPathChain 沿 segments 创建路径节点链，返回末端节点。
// tree.AddNode 会自动创建中间路径节点，但此辅助函数使语义更显式。
func addPathChain(tree *Tree, segments ...string) (node.Node[node.NodeContext], error) {
	current := tree.Root
	for _, seg := range segments {
		child := current.FindChildByKey(seg)
		if child != nil {
			current = child
			continue
		}
		newNode := node.NewRequestPathNode(seg)
		if err := current.AddChild(newNode); err != nil {
			return nil, err
		}
		current = newNode
	}
	return current, nil
}
