package tree

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

// Tree 路由树容器，存储和管理路由节点
type Tree struct {
	Root node.Node[node.NodeContext]
}

// NewTree 创建一棵空的路由树
func NewTree() *Tree {
	rootContext := node.NewBaseNodeContext()
	root := node.NewBaseNode[node.NodeContext]("root", "root", "", rootContext)

	return &Tree{
		Root: root,
	}
}

// AddNode 根据路径将节点添加到路由树中
func (x *Tree) AddNode(path string, n node.Node[node.NodeContext]) error {
	if n == nil {
		return fmt.Errorf("不能添加nil节点")
	}

	normalizedPath := normalizePath(path)
	if normalizedPath == "" {
		return x.Root.AddChild(n)
	}

	segments := strings.Split(normalizedPath, "/")
	currentNode := x.Root

	for _, segment := range segments {
		if segment == "" {
			continue
		}

		child := currentNode.FindChildByKey(segment)
		if child != nil {
			currentNode = child
			continue
		}

		newPathNode := node.NewRequestPathNode(segment)
		if err := currentNode.AddChild(newPathNode); err != nil {
			return fmt.Errorf("添加路径节点 '%s' 失败: %w", segment, err)
		}
		currentNode = newPathNode
	}

	return currentNode.AddChild(n)
}

// FindNodeByPath 根据路径在路由树中查找节点
func (x *Tree) FindNodeByPath(path string) node.Node[node.NodeContext] {
	normalizedPath := normalizePath(path)
	if normalizedPath == "" {
		return x.Root
	}

	segments := strings.Split(normalizedPath, "/")
	currentNode := x.Root

	for _, segment := range segments {
		if segment == "" {
			continue
		}

		child := currentNode.FindChildByKey(segment)
		if child == nil {
			return nil
		}
		currentNode = child
	}

	return currentNode
}

// String 返回路由树的文本表示（树形结构）
func (x *Tree) String() string {
	var sb strings.Builder
	x.printNode(x.Root, "", true, &sb)
	return sb.String()
}

// Print 打印路由树到标准输出
func (x *Tree) Print() {
	fmt.Print(x.String())
}

// printNode 递归打印节点及其子节点
func (x *Tree) printNode(n node.Node[node.NodeContext], prefix string, isLast bool, sb *strings.Builder) {
	if n == nil {
		return
	}

	nodeType := n.GetType()
	key := n.GetKey()

	// 格式化节点显示
	display := formatNodeDisplay(nodeType, key, n)

	if nodeType == "root" {
		sb.WriteString(display + "\n")
	} else {
		connector := "├── "
		if isLast {
			connector = "└── "
		}
		sb.WriteString(prefix + connector + display + "\n")
	}

	// 递归打印子节点
	children := n.GetChildren()
	for i, child := range children {
		childIsLast := i == len(children)-1
		childPrefix := prefix
		if nodeType != "root" {
			if isLast {
				childPrefix += "    "
			} else {
				childPrefix += "│   "
			}
		}
		x.printNode(child, childPrefix, childIsLast, sb)
	}
}

// formatNodeDisplay 格式化单个节点的显示文本
func formatNodeDisplay(nodeType, key string, n node.Node[node.NodeContext]) string {
	switch nodeType {
	case "root":
		return "root"
	case "request_path":
		return key + " [Path]"
	case "request_path_variable":
		// 尝试获取变量节点的类型信息
		if pathVarNode, ok := n.(*node.RequestPathVariableNode); ok {
			typeStr := string(pathVarNode.GetValueType())
			return fmt.Sprintf("{%s} [Var, %s]", key, typeStr)
		}
		return fmt.Sprintf("{%s} [Var]", key)
	case "request_method":
		return key + " [Method]"
	case "request_content_type":
		return key + " [ContentType]"
	case "request_param":
		// 标注必需参数（*）和类型信息
		if paramNode, ok := n.(*node.RequestParamNode); ok {
			requiredMark := ""
			if paramNode.IsRequired() {
				requiredMark = "*" // 必需参数标记
			}
			logicalType := paramNode.GetLogicalType()
			if logicalType != "" && logicalType != "string" {
				return fmt.Sprintf("%s%s [Param, %s]", key, requiredMark, logicalType)
			}
			return fmt.Sprintf("%s%s [Param]", key, requiredMark)
		}
		return key + " [Param]"
	case "request_header":
		if headerNode, ok := n.(*node.RequestHeaderNode); ok {
			return fmt.Sprintf("%s [Header]", headerNode.GetHeaderName())
		}
		return key + " [Header]"
	case "request_header_value":
		if headerValNode, ok := n.(*node.RequestHeaderValueNode); ok {
			return fmt.Sprintf("%s: %s [HeaderValue]", headerValNode.GetHeaderName(), headerValNode.GetHeaderValue())
		}
		return key + " [HeaderValue]"
	case "request_cookie":
		if cookieNode, ok := n.(*node.RequestCookieNode); ok {
			return fmt.Sprintf("%s [Cookie]", cookieNode.GetCookieName())
		}
		return key + " [Cookie]"
	case "request_cookie_value":
		if cookieValNode, ok := n.(*node.RequestCookieValueNode); ok {
			return fmt.Sprintf("%s=%s [CookieValue]", cookieValNode.GetCookieName(), cookieValNode.GetCookieValue())
		}
		return key + " [CookieValue]"
	default:
		if key != "" {
			return fmt.Sprintf("%s [%s]", key, nodeType)
		}
		return fmt.Sprintf("[%s]", nodeType)
	}
}

// normalizePath 标准化路径
func normalizePath(path string) string {
	normalizedPath := strings.Trim(path, "/")
	for strings.Contains(normalizedPath, "//") {
		normalizedPath = strings.ReplaceAll(normalizedPath, "//", "/")
	}
	return normalizedPath
}

// === JSON 序列化 ===

// RouteNodeJSON 路由节点的JSON表示
type RouteNodeJSON struct {
	Type     string           `json:"type"`
	Key      string           `json:"key"`
	Value    string           `json:"value,omitempty"`
	Dynamic  bool             `json:"dynamic,omitempty"`
	Children []*RouteNodeJSON `json:"children,omitempty"`
	Requests int64            `json:"requests,omitempty"`
	// 路径变量特有字段
	InferredType string `json:"inferred_type,omitempty"`
	Pattern      string `json:"pattern,omitempty"`
	// 参数特有字段
	Required       *bool  `json:"required,omitempty"`
	PhysicalType   string `json:"physical_type,omitempty"`
	LogicalType    string `json:"logical_type,omitempty"`
	PresenceCount  int64  `json:"presence_count,omitempty"`
	DefaultValue   string `json:"default_value,omitempty"`
	MultiValue     bool   `json:"multi_value,omitempty"`
}

// ToJSON 将路由树导出为JSON格式
func (x *Tree) ToJSON() ([]byte, error) {
	root := x.nodeToJSON(x.Root)
	return json.MarshalIndent(root, "", "  ")
}

// nodeToJSON 将节点递归转换为JSON结构
func (x *Tree) nodeToJSON(n node.Node[node.NodeContext]) *RouteNodeJSON {
	if n == nil {
		return nil
	}

	result := &RouteNodeJSON{
		Type:     n.GetType(),
		Key:      n.GetKey(),
		Value:    n.GetValue(),
		Dynamic:  n.IsDynamic(),
		Requests: n.GetRequestCount(),
	}

	// 路径变量节点的额外信息
	if pathVarNode, ok := n.(*node.RequestPathVariableNode); ok {
		result.InferredType = string(pathVarNode.GetValueType())
		result.LogicalType = string(pathVarNode.GetLogicalType())
		// 导出正则模式源串，使反序列化后 IsMatch 仍能严格匹配（否则退化为启发式）
		if p := pathVarNode.GetPattern(); p != nil {
			result.Pattern = p.String()
		}
	}

	// 参数节点的额外信息
	if paramNode, ok := n.(*node.RequestParamNode); ok {
		req := paramNode.IsRequired()
		result.Required = &req
		result.PhysicalType = string(paramNode.GetValueType())
		result.LogicalType = string(paramNode.GetLogicalType())
		result.PresenceCount = paramNode.GetPresenceCount()
		result.DefaultValue = paramNode.GetDefaultValue()
		result.MultiValue = paramNode.IsMultiValue()
	}

	// 递归处理子节点
	children := n.GetChildren()
	if len(children) > 0 {
		result.Children = make([]*RouteNodeJSON, 0, len(children))
		for _, child := range children {
			result.Children = append(result.Children, x.nodeToJSON(child))
		}
	}

	return result
}

// FromJSON 从JSON格式导入路由树
func (x *Tree) FromJSON(data []byte) error {
	var root RouteNodeJSON
	if err := json.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("JSON反序列化失败: %w", err)
	}

	x.Root = x.jsonToNode(&root)
	return nil
}

// jsonToNode 将JSON结构递归转换为节点
func (x *Tree) jsonToNode(jn *RouteNodeJSON) node.Node[node.NodeContext] {
	if jn == nil {
		return nil
	}

	var n node.Node[node.NodeContext]
	context := node.NewBaseNodeContext()

	switch jn.Type {
	case "root":
		n = node.NewBaseNode[node.NodeContext]("root", jn.Key, jn.Value, context)
	case "request_path":
		n = node.NewRequestPathNode(jn.Key)
	case "request_path_variable":
		varNode := node.NewRequestPathVariableNode(jn.Key, jn.Pattern)
		if jn.InferredType != "" {
			varNode.SetType(value.Type(jn.InferredType))
		}
		if jn.LogicalType != "" {
			varNode.SetLogicalType(value.LogicalType(jn.LogicalType))
		}
		n = varNode
	case "request_method":
		n = node.NewRequestMethodNode(jn.Key)
	case "request_content_type":
		n = node.NewRequestContentTypeNode(jn.Key)
	case "request_param":
		required := false
		if jn.Required != nil {
			required = *jn.Required
		}
		// 使用 JSON 中的 default_value（如有），否则回退到 value
		defaultValue := jn.DefaultValue
		if defaultValue == "" {
			defaultValue = jn.Value
		}
		paramNode := node.NewRequestParamNode(jn.Key, defaultValue, required)
		if jn.PhysicalType != "" {
			paramNode.SetValueType(value.Type(jn.PhysicalType))
		}
		if jn.LogicalType != "" {
			paramNode.SetLogicalType(value.LogicalType(jn.LogicalType))
		}
		if jn.MultiValue {
			paramNode.SetMultiValue(true)
		}
		if jn.PresenceCount > 0 {
			paramNode.SetPresenceCount(jn.PresenceCount)
		}
		n = paramNode
	case "request_header":
		n = node.NewRequestHeaderNode(jn.Key)
	case "request_header_value":
		n = node.NewRequestHeaderValueNode(jn.Value, jn.Key)
	case "request_cookie":
		n = node.NewRequestCookieNode(jn.Key)
	case "request_cookie_value":
		n = node.NewRequestCookieValueNode(jn.Value, jn.Key)
	default:
		n = node.NewBaseNode[node.NodeContext](jn.Type, jn.Key, jn.Value, context)
	}

	// 递归处理子节点
	if jn.Children != nil {
		for _, childJSON := range jn.Children {
			child := x.jsonToNode(childJSON)
			if child != nil {
				n.AddChild(child)
			}
		}
	}

	return n
}

// === 路由统计 ===

// RouteStats 路由树统计信息
type RouteStats struct {
	TotalNodes          int `json:"total_nodes"`
	PathNodes           int `json:"path_nodes"`
	PathVariableNodes   int `json:"path_variable_nodes"`
	MethodNodes         int `json:"method_nodes"`
	ContentTypeNodes    int `json:"content_type_nodes"`
	ParamNodes          int `json:"param_nodes"`
	MaxDepth            int `json:"max_depth"`
	LeafNodes           int `json:"leaf_nodes"`
	HeaderNodes         int   `json:"header_nodes"`
	HeaderValueNodes    int   `json:"header_value_nodes"`
	CookieNodes         int   `json:"cookie_nodes"`
	CookieValueNodes    int   `json:"cookie_value_nodes"`
	TotalRequestCount   int64 `json:"total_request_count"`
}

// Stats 获取路由树的统计信息
func (x *Tree) Stats() RouteStats {
	stats := RouteStats{}
	x.collectStats(x.Root, 0, &stats)
	return stats
}

// collectStats 递归收集统计信息
func (x *Tree) collectStats(n node.Node[node.NodeContext], depth int, stats *RouteStats) {
	if n == nil {
		return
	}

	stats.TotalNodes++
	stats.TotalRequestCount += n.GetRequestCount()

	if depth > stats.MaxDepth {
		stats.MaxDepth = depth
	}

	switch n.GetType() {
	case "request_path":
		stats.PathNodes++
	case "request_path_variable":
		stats.PathVariableNodes++
	case "request_method":
		stats.MethodNodes++
	case "request_content_type":
		stats.ContentTypeNodes++
	case "request_param":
		stats.ParamNodes++
	case "request_header":
		stats.HeaderNodes++
	case "request_header_value":
		stats.HeaderValueNodes++
	case "request_cookie":
		stats.CookieNodes++
	case "request_cookie_value":
		stats.CookieValueNodes++
	}

	if n.IsLeaf() && n.GetType() != "root" {
		stats.LeafNodes++
	}

	for _, child := range n.GetChildren() {
		x.collectStats(child, depth+1, stats)
	}
}
