// Package exporter 提供将逆向路由树导出为标准 API 规范格式的能力。
//
// 当前支持 OpenAPI 3.0.3 规范导出，把黑盒流量逆向出的路由结构转化为
// 可被 Swagger UI、Redoc 等工具直接渲染的 API 文档。
package exporter

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/tree"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

// OpenAPIExporter 将路由树导出为 OpenAPI 3.0.3 规范。
type OpenAPIExporter struct {
	// Title 文档标题，默认 "Reverse Engineered API"
	Title string
	// Version 文档版本，默认 "1.0.0"
	Version string
	// Description 文档描述
	Description string
	// ServerURL API 服务地址，如 "https://api.example.com"。空则不输出 servers
	ServerURL string
	// IncludeOptionalParameters 是否包含可选参数（默认包含）
	IncludeOptionalParameters bool
}

// NewOpenAPIExporter 创建默认配置的导出器。
func NewOpenAPIExporter() *OpenAPIExporter {
	return &OpenAPIExporter{
		Title:                     "Reverse Engineered API",
		Version:                   "1.0.0",
		Description:               "由 reverse-router-tree 从黑盒流量逆向工程生成的 API 规范。",
		IncludeOptionalParameters: true,
	}
}

// === OpenAPI 3.0.3 结构定义 ===
//
// 仅定义导出所需的字段。securitySchemes 从 Authorization header 推断并填充。

// openAPIDoc OpenAPI 文档根结构
type openAPIDoc struct {
	OpenAPI    string                     `json:"openapi"`
	Info       openAPIInfo                `json:"info"`
	Servers    []openAPIServer            `json:"servers,omitempty"`
	Paths      map[string]*openAPIPathItem `json:"paths"`
	Components openAPIComponents          `json:"components,omitempty"`
}

type openAPIInfo struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version"`
}

type openAPIServer struct {
	URL string `json:"url"`
}

// openAPIPathItem 一个路径下的所有方法。
// 字段名用小写，json tag 用小写方法名（get/post/...）。
type openAPIPathItem struct {
	Get     *openAPIOperation `json:"get,omitempty"`
	Post    *openAPIOperation `json:"post,omitempty"`
	Put     *openAPIOperation `json:"put,omitempty"`
	Patch   *openAPIOperation `json:"patch,omitempty"`
	Delete  *openAPIOperation `json:"delete,omitempty"`
	Head    *openAPIOperation `json:"head,omitempty"`
	Options *openAPIOperation `json:"options,omitempty"`
}

type openAPIOperation struct {
	Summary     string           `json:"summary,omitempty"`
	Description string           `json:"description,omitempty"`
	OperationID string           `json:"operationId,omitempty"`
	Parameters  []openAPIParam   `json:"parameters,omitempty"`
	RequestBody *openAPIRequestBody `json:"requestBody,omitempty"`
	Responses   map[string]openAPIResponse `json:"responses"`
	// Security 声明该 operation 适用的安全方案（引用 components.securitySchemes 的 key）。
	// 为空时省略。OpenAPI 规范要求 security 是数组 of 对象，值的数组为空表示该方案无需 scope。
	Security []map[string][]string `json:"security,omitempty"`
}

type openAPIParam struct {
	Name        string          `json:"name"`
	In          string          `json:"in"` // query/path/header/cookie
	Description string          `json:"description,omitempty"`
	Required    bool            `json:"required"`
	Schema      openAPISchema   `json:"schema"`
}

type openAPIRequestBody struct {
	Description string                       `json:"description,omitempty"`
	Required    bool                         `json:"required"`
	Content     map[string]openAPIMediaType `json:"content"`
}

type openAPIMediaType struct {
	Schema openAPISchema `json:"schema"`
}

type openAPIResponse struct {
	Description string `json:"description"`
}

type openAPISchema struct {
	Type       string                  `json:"type,omitempty"`
	Format     string                  `json:"format,omitempty"`
	Pattern    string                  `json:"pattern,omitempty"`
	Default    string                  `json:"default,omitempty"`
	Properties map[string]openAPISchema `json:"properties,omitempty"`
	Required   []string                `json:"required,omitempty"`
}

type openAPIComponents struct {
	Schemas         map[string]openAPISchema         `json:"schemas,omitempty"`
	SecuritySchemes map[string]openAPISecurityScheme `json:"securitySchemes,omitempty"`
}

type openAPISecurityScheme struct {
	Type        string `json:"type"`
	Scheme      string `json:"scheme,omitempty"`
	Description string `json:"description,omitempty"`
}

// === 导出逻辑 ===

// Export 将路由树导出为 OpenAPI 3.0.3 JSON 文档。
func (e *OpenAPIExporter) Export(t *tree.Tree) ([]byte, error) {
	if t == nil || t.Root == nil {
		return nil, fmt.Errorf("路由树不能为空")
	}

	doc := &openAPIDoc{
		OpenAPI: "3.0.3",
		Info: openAPIInfo{
			Title:       e.Title,
			Description: e.Description,
			Version:     e.Version,
		},
		Paths: make(map[string]*openAPIPathItem),
	}
	if e.ServerURL != "" {
		doc.Servers = []openAPIServer{{URL: e.ServerURL}}
	}

	// 收集所有端点
	endpoints := e.collectEndpoints(t.Root, nil)
	// 按路径+方法排序，保证输出稳定
	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].path != endpoints[j].path {
			return endpoints[i].path < endpoints[j].path
		}
		return endpoints[i].method < endpoints[j].method
	})

	// securitySchemes 在 buildOperation 期间累积（从 Authorization header 推断），
	// 保持 exporter 本身无状态、可重入。
	securitySchemes := make(map[string]openAPISecurityScheme)
	for _, ep := range endpoints {
		item := doc.Paths[ep.path]
		if item == nil {
			item = &openAPIPathItem{}
			doc.Paths[ep.path] = item
		}
		op := e.buildOperation(ep, securitySchemes)
		e.setOperation(item, ep.method, op)
	}

	if len(securitySchemes) > 0 {
		doc.Components.SecuritySchemes = securitySchemes
	}

	return json.MarshalIndent(doc, "", "  ")
}

// endpoint 收集到的一个端点（路径+方法+参数信息）
type endpoint struct {
	path          string
	method        string // 小写
	pathVariables []*node.RequestPathVariableNode
	queryParams   []*node.RequestParamNode
	bodyParams    []*node.RequestParamNode
	headerParams  []headerParam
	cookieParams  []cookieParam
	contentType   string
	operationID   string
}

type headerParam struct {
	name  string
	value string
}

type cookieParam struct {
	name  string
	value string
}

// pathSegment 记录路径段信息，用于构造 OpenAPI path 和路径变量 schema。
type pathSegment struct {
	key     string // 拼接用：request_path 用 key，request_path_variable 用 {key}
	varNode *node.RequestPathVariableNode // 路径变量节点（nil 表示固定路径段）
}

// collectEndpoints 递归遍历路由树，收集所有端点。
// segs 维护从 root 到当前节点的路径段（用于拼接 path 和提取路径变量）。
func (e *OpenAPIExporter) collectEndpoints(n node.Node[node.NodeContext], segs []pathSegment) []*endpoint {
	if n == nil {
		return nil
	}

	nodeType := n.GetType()

	// 跳过 root 节点本身
	if nodeType != "root" {
		switch nodeType {
		case "request_path":
			segs = append(segs, pathSegment{key: n.GetKey()})
		case "request_path_variable":
			pv, _ := n.(*node.RequestPathVariableNode)
			segs = append(segs, pathSegment{key: "{" + n.GetKey() + "}", varNode: pv})
		case "request_method":
			// 到达方法节点，构造端点
			return e.collectFromMethodNode(n, segs)
		default:
			// 其他类型（param/content_type/header/cookie）不应出现在路径栈中
		}
	}

	var endpoints []*endpoint
	for _, child := range n.GetChildren() {
		endpoints = append(endpoints, e.collectEndpoints(child, segs)...)
	}
	return endpoints
}

// collectFromMethodNode 从方法节点提取端点信息。
func (e *OpenAPIExporter) collectFromMethodNode(methodNode node.Node[node.NodeContext], segs []pathSegment) []*endpoint {
	// 拼接 path
	parts := make([]string, 0, len(segs))
	var pathVars []*node.RequestPathVariableNode
	for _, s := range segs {
		parts = append(parts, s.key)
		if s.varNode != nil {
			pathVars = append(pathVars, s.varNode)
		}
	}
	path := "/" + strings.Join(parts, "/")

	ep := &endpoint{
		path:          path,
		method:        strings.ToLower(methodNode.GetKey()),
		pathVariables: pathVars,
	}

	// 遍历方法节点的子节点
	for _, child := range methodNode.GetChildren() {
		switch child.GetType() {
		case "request_param":
			paramNode := child.(*node.RequestParamNode)
			if ep.contentType != "" {
				ep.bodyParams = append(ep.bodyParams, paramNode)
			} else {
				ep.queryParams = append(ep.queryParams, paramNode)
			}
		case "request_content_type":
			ep.contentType = child.GetKey()
		case "request_header":
			// Header 分组节点，其子节点是 header 值
			for _, hv := range child.GetChildren() {
				ep.headerParams = append(ep.headerParams, headerParam{
					name:  child.GetKey(),
					value: hv.GetKey(),
				})
			}
		case "request_cookie":
			for _, cv := range child.GetChildren() {
				ep.cookieParams = append(ep.cookieParams, cookieParam{
					name:  child.GetKey(),
					value: cv.GetKey(),
				})
			}
		}
	}

	// 如果有 contentType 但参数已被归入 query（因为 contentType 子节点在 param 之后遍历），
	// 重新分类：有 contentType 时所有 param 视为 body 参数
	if ep.contentType != "" && len(ep.bodyParams) == 0 {
		ep.bodyParams = ep.queryParams
		ep.queryParams = nil
	}

	ep.operationID = fmt.Sprintf("%s_%s", ep.method, sanitizeOperationID(ep.path))
	if len(ep.operationID) > 100 {
		ep.operationID = ep.operationID[:100]
	}

	return []*endpoint{ep}
}

// buildOperation 从端点信息构建 OpenAPI operation。
func (e *OpenAPIExporter) buildOperation(ep *endpoint, securitySchemes map[string]openAPISecurityScheme) *openAPIOperation {
	op := &openAPIOperation{
		Summary:     fmt.Sprintf("%s %s", strings.ToUpper(ep.method), ep.path),
		OperationID: ep.operationID,
		Responses: map[string]openAPIResponse{
			"200": {Description: "成功响应（逆向推断，未经实际验证）"},
		},
	}

	// 路径变量参数（required=true）
	for _, pv := range ep.pathVariables {
		op.Parameters = append(op.Parameters, openAPIParam{
			Name:        pv.GetKey(),
			In:          "path",
			Required:    true,
			Description: describePathVariable(pv),
			Schema:      schemaForPathVariable(pv),
		})
	}

	// Query 参数
	for _, p := range ep.queryParams {
		if !e.IncludeOptionalParameters && !p.IsRequired() {
			continue
		}
		op.Parameters = append(op.Parameters, openAPIParam{
			Name:     p.GetKey(),
			In:       "query",
			Required: p.IsRequired(),
			Schema:   schemaForParam(p),
		})
	}

	// Header 参数（同名去重）。Authorization 若能识别为已知 HTTP 认证方案
	// （Bearer/Basic/Digest），则转为 operation 的 security 声明并注册到
	// components.securitySchemes，不作为普通 header 参数输出（避免与 security 重复）。
	seenHeader := make(map[string]bool)
	for _, h := range ep.headerParams {
		if seenHeader[h.name] {
			continue
		}
		seenHeader[h.name] = true
		if scheme, ok := securitySchemeFromAuth(h.name, h.value); ok {
			securitySchemes[scheme.name] = scheme.def
			op.Security = append(op.Security, map[string][]string{scheme.name: {}})
			continue
		}
		op.Parameters = append(op.Parameters, openAPIParam{
			Name:     h.name,
			In:       "header",
			Required: false,
			Schema:   openAPISchema{Type: "string"},
		})
	}

	// Cookie 参数（同名去重）
	seenCookie := make(map[string]bool)
	for _, c := range ep.cookieParams {
		if seenCookie[c.name] {
			continue
		}
		seenCookie[c.name] = true
		op.Parameters = append(op.Parameters, openAPIParam{
			Name:     c.name,
			In:       "cookie",
			Required: false,
			Schema:   openAPISchema{Type: "string"},
		})
	}

	// Request Body（有 Content-Type 时）
	if ep.contentType != "" {
		op.RequestBody = e.buildRequestBody(ep)
	}

	// 按参数名排序，保证输出稳定
	sort.Slice(op.Parameters, func(i, j int) bool {
		if op.Parameters[i].In != op.Parameters[j].In {
			return op.Parameters[i].In < op.Parameters[j].In
		}
		return op.Parameters[i].Name < op.Parameters[j].Name
	})

	return op
}

// inferredSecurityScheme 是从 Authorization header 推断出的安全方案。
type inferredSecurityScheme struct {
	name string // securitySchemes map 的 key，如 "bearerAuth"
	def  openAPISecurityScheme
}

// securitySchemeFromAuth 从 header 名+规范化值推断 OpenAPI 安全方案。
//
// router 的 normalizeAuthorization 把 "Bearer xxx" 规范化为 "Bearer"，
// 故 header 值子节点的 key 就是 HTTP 认证方案名（首字母大写）。
// 仅识别 OpenAPI 3.0.3 标准的 http 方案：Bearer/Basic/Digest。
// 其余（含无法识别的 Authorization 值）返回 ok=false，由调用方当普通 header 输出。
//
// 注意：从黑盒流量只能看到 Authorization 头的存在与方案名，无法知道是否为
// OAuth2/JWT 等更具体机制，故统一映射为 http 方案。
func securitySchemeFromAuth(headerName, normalizedValue string) (inferredSecurityScheme, bool) {
	if !strings.EqualFold(headerName, "Authorization") {
		return inferredSecurityScheme{}, false
	}
	var scheme string
	switch strings.ToLower(normalizedValue) {
	case "bearer":
		scheme = "bearer"
	case "basic":
		scheme = "basic"
	case "digest":
		scheme = "digest"
	default:
		return inferredSecurityScheme{}, false
	}
	return inferredSecurityScheme{
		name: scheme + "Auth",
		def: openAPISecurityScheme{
			Type:        "http",
			Scheme:      scheme,
			Description: "由抓包流量中的 Authorization 头推断",
		},
	}, true
}

// buildRequestBody 构建请求体。
// 有 body 参数时生成 object schema，否则用空 object。
func (e *OpenAPIExporter) buildRequestBody(ep *endpoint) *openAPIRequestBody {
	mediaType := ep.contentType
	// 规范化 Content-Type（去 charset）
	if idx := strings.Index(mediaType, ";"); idx >= 0 {
		mediaType = strings.TrimSpace(mediaType[:idx])
	}

	properties := make(map[string]openAPISchema)
	required := make([]string, 0)
	for _, p := range ep.bodyParams {
		if !e.IncludeOptionalParameters && !p.IsRequired() {
			continue
		}
		properties[p.GetKey()] = schemaForParam(p)
		if p.IsRequired() {
			required = append(required, p.GetKey())
		}
	}

	body := &openAPIRequestBody{
		Description: "请求体（由黑盒流量推断）",
		Content: map[string]openAPIMediaType{
			mediaType: {
				Schema: openAPISchema{
					Type:       "object",
					Properties: properties,
					Required:   required,
				},
			},
		},
	}
	return body
}

// schemaForParam 根据参数节点的物理/逻辑类型生成 schema。
func schemaForParam(p *node.RequestParamNode) openAPISchema {
	return buildSchema(string(p.GetValueType()), string(p.GetLogicalType()), p.GetDefaultValue(), "")
}

// schemaForPathVariable 根据路径变量节点生成 schema。
func schemaForPathVariable(pv *node.RequestPathVariableNode) openAPISchema {
	patternStr := ""
	if pv.GetPattern() != nil {
		patternStr = pv.GetPattern().String()
	}
	return buildSchema(string(pv.GetValueType()), string(pv.GetLogicalType()), "", patternStr)
}

// describePathVariable 生成路径变量的描述。
func describePathVariable(pv *node.RequestPathVariableNode) string {
	parts := []string{"路径变量"}
	if lt := pv.GetLogicalType(); lt != "" && string(lt) != string(value.LogicalTypeString) {
		parts = append(parts, fmt.Sprintf("逻辑类型: %s", lt))
	}
	if pt := pv.GetValueType(); pt != "" && string(pt) != string(value.PhysicalTypeString) {
		parts = append(parts, fmt.Sprintf("物理类型: %s", pt))
	}
	return strings.Join(parts, ", ")
}

// buildSchema 把物理/逻辑类型映射为 OpenAPI schema type+format。
func buildSchema(physicalType, logicalType, defaultVal, pattern string) openAPISchema {
	schema := openAPISchema{}

	// 逻辑类型优先决定 format（更具体）
	switch logicalType {
	case "integer", "int":
		schema.Type = "integer"
	case "float", "decimal", "currency", "percentage":
		schema.Type = "number"
	case "boolean":
		schema.Type = "boolean"
	case "date":
		schema.Type = "string"
		schema.Format = "date"
	case "datetime":
		schema.Type = "string"
		schema.Format = "date-time"
	case "time":
		schema.Type = "string"
		schema.Format = "time"
	case "email":
		schema.Type = "string"
		schema.Format = "email"
	case "url":
		schema.Type = "string"
		schema.Format = "uri"
	case "uuid":
		schema.Type = "string"
		schema.Format = "uuid"
	case "ipaddress":
		schema.Type = "string"
		schema.Format = "ipv4"
	case "phone", "idcard", "bankcard", "plate":
		schema.Type = "string"
	default:
		// 回退到物理类型
		switch physicalType {
		case "integer":
			schema.Type = "integer"
		case "float":
			schema.Type = "number"
		case "boolean":
			schema.Type = "boolean"
		case "array":
			schema.Type = "array"
		case "object":
			schema.Type = "object"
		default:
			schema.Type = "string"
		}
	}

	if pattern != "" {
		schema.Pattern = pattern
	}
	if defaultVal != "" {
		schema.Default = defaultVal
	}
	return schema
}

// setOperation 把 operation 设置到 path item 的对应方法字段。
func (e *OpenAPIExporter) setOperation(item *openAPIPathItem, method string, op *openAPIOperation) {
	switch method {
	case "get":
		item.Get = op
	case "post":
		item.Post = op
	case "put":
		item.Put = op
	case "patch":
		item.Patch = op
	case "delete":
		item.Delete = op
	case "head":
		item.Head = op
	case "options":
		item.Options = op
	}
}

// sanitizeOperationID 把路径清理为合法的 operationId（只保留字母数字下划线）。
func sanitizeOperationID(path string) string {
	var b strings.Builder
	for _, c := range path {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			b.WriteRune(c)
		} else if c == '{' || c == '}' {
			// 路径变量占位符去掉花括号
			continue
		} else {
			b.WriteRune('_')
		}
	}
	s := b.String()
	s = strings.Trim(s, "_")
	// 合并连续下划线
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}
	if s == "" {
		s = "root"
	}
	return s
}
