package router

import (
	"sort"
	"strings"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
)

// 本文件收敛 URL 归一化的对外 API：失败原因、批量明细、资产清单、便捷入口。
//
// 背景：归一化是本项目的核心输出（测绘 URL 资产去重/聚合/检索），调用方最关心
// 三件事——"这条流量归到哪条资产"、"归不上时到底卡在哪一步"、"树里现在有哪些资产"。
// 为此在兼容旧签名的基础上提供：
//   - Detailed 变体：失败时返回机器可读的 NormalizeReason（unknown_path /
//     unknown_method / unknown_host / unknown_project / invalid_request），
//     便于监控分类告警与排查"采集缺口"；
//   - 批量明细 NormalizeReport：Matched 按资产键分桶 + Unmatched 逐条原因，
//     替代静默丢弃；
//   - ListAssets 资产清单：遍历树枚举全部已知资产，无需逐条请求试探；
//   - NormalizeCurl / NormalizeURLString：手里只有 curl 或裸 URL 时直达归一化。
//
// 命名约定：单条 Detailed 返回 (NormalizedRoute, NormalizeReason)；批量 Detailed
// 返回 NormalizeReport；清单返回排序后的切片/map（稳定输出，便于 diff 与快照测试）。

// NormalizeReason 归一化失败的机器可读原因。NormalizeOK 表示成功。
type NormalizeReason string

const (
	// NormalizeOK 归一化成功。
	NormalizeOK NormalizeReason = "ok"
	// NormalizeReasonInvalidRequest 请求非法：nil 请求、URL 解析失败等。
	NormalizeReasonInvalidRequest NormalizeReason = "invalid_request"
	// NormalizeReasonUnknownHost 目标 host 在 RouterSet 中尚无建桶（无采集数据）。
	NormalizeReasonUnknownHost NormalizeReason = "unknown_host"
	// NormalizeReasonUnknownProject 项目在 ProjectManager 中不存在。
	NormalizeReasonUnknownProject NormalizeReason = "unknown_project"
	// NormalizeReasonUnknownPath 路径未命中：某段既无固定节点也无变量节点可回退。
	NormalizeReasonUnknownPath NormalizeReason = "unknown_path"
	// NormalizeReasonUnknownMethod 路径命中但该 HTTP 方法从未采集过。
	NormalizeReasonUnknownMethod NormalizeReason = "unknown_method"
)

// NormalizeMiss 批量归一化中未命中的单条样本。
type NormalizeMiss struct {
	// Index 样本在输入切片中的下标（从 0 起）
	Index int
	// URL 样本 URL（nil 请求记为 "<nil>"）
	URL string
	// Reason 未命中原因
	Reason NormalizeReason
}

// NormalizeReport 批量归一化的明细报告：成功分桶 + 失败逐条原因。
type NormalizeReport struct {
	// Matched 资产键到原始 URL 的分桶（键为 AssetKey 或 HostAssetKey，见各方法注释）
	Matched map[string][]string
	// Unmatched 未命中样本明细（含下标、URL、原因）
	Unmatched []NormalizeMiss
}

// MatchedCount 成功归入资产的样本数。
func (r NormalizeReport) MatchedCount() int {
	n := 0
	for _, urls := range r.Matched {
		n += len(urls)
	}
	return n
}

// UnmatchedCount 未命中的样本数。
func (r NormalizeReport) UnmatchedCount() int { return len(r.Unmatched) }

// --- ReverseRouter ---

// NormalizeURLDetailed 将单条请求归一化，失败时返回具体原因。
// 成功时 Reason 为 NormalizeOK。本方法是 NormalizeURL 的超集，
// NormalizeURL 内部即调本方法（旧签名保持兼容）。
func (x *ReverseRouter) NormalizeURLDetailed(req *request.HttpRequest) (NormalizedRoute, NormalizeReason) {
	if req == nil || x == nil || x.Tree == nil || x.Tree.Root == nil {
		return NormalizedRoute{}, NormalizeReasonInvalidRequest
	}
	paths, _, err := request.NewUrlParser(req.Url).Parse()
	if err != nil {
		return NormalizedRoute{}, NormalizeReasonInvalidRequest
	}
	defer request.ReleasePaths(paths)

	method := strings.ToUpper(req.Method)
	if method == "" {
		method = "GET"
	}
	pathEnd, parts, pathParams, ok := x.normalizePathSegments(paths)
	if !ok {
		return NormalizedRoute{}, NormalizeReasonUnknownPath
	}
	methodNode := pathEnd.FindChildByKey(method)
	if methodNode == nil || methodNode.GetType() != "request_method" {
		return NormalizedRoute{}, NormalizeReasonUnknownMethod
	}

	result := NormalizedRoute{
		Host:           req.Host,
		Method:         method,
		Template:       "/" + strings.Join(parts, "/"),
		PathParams:     pathParams,
		QueryParams:    make([]string, 0),
		RequiredParams: make([]string, 0),
	}
	if strings.TrimSpace(result.Host) == "" {
		result.Host = request.ExtractHost(req.Url)
	}
	if result.Template == "/" && len(parts) == 0 {
		result.Template = "/"
	}
	for _, child := range methodNode.GetChildren() {
		if child.GetType() != "request_param" {
			continue
		}
		param, ok := child.(*node.RequestParamNode)
		if !ok {
			continue
		}
		name := param.GetParamName()
		result.QueryParams = append(result.QueryParams, name)
		if param.IsRequired() {
			result.RequiredParams = append(result.RequiredParams, name)
		}
	}
	return result, NormalizeOK
}

// NormalizeURLsDetailed 批量归一化，返回资产明细报告（键为 AssetKey）。
// 失败样本不再静默丢弃，逐条记入 Unmatched（含原因），便于排查采集缺口。
func (x *ReverseRouter) NormalizeURLsDetailed(reqs []*request.HttpRequest) NormalizeReport {
	report := NormalizeReport{Matched: make(map[string][]string)}
	for i, req := range reqs {
		normalized, reason := x.NormalizeURLDetailed(req)
		url := "<nil>"
		if req != nil {
			url = req.Url
		}
		if reason != NormalizeOK {
			report.Unmatched = append(report.Unmatched, NormalizeMiss{Index: i, URL: url, Reason: reason})
			continue
		}
		key := normalized.AssetKey()
		report.Matched[key] = append(report.Matched[key], url)
	}
	return report
}

// NormalizeCurl 直接归一化一条 curl 命令：解析失败返回 invalid_request。
func (x *ReverseRouter) NormalizeCurl(raw string) (NormalizedRoute, bool) {
	route, reason := x.NormalizeCurlDetailed(raw)
	return route, reason == NormalizeOK
}

// NormalizeCurlDetailed 直接归一化一条 curl 命令，失败时返回具体原因。
func (x *ReverseRouter) NormalizeCurlDetailed(raw string) (NormalizedRoute, NormalizeReason) {
	req, err := request.ParseCurl(raw)
	if err != nil {
		return NormalizedRoute{}, NormalizeReasonInvalidRequest
	}
	return x.NormalizeURLDetailed(req)
}

// NormalizeURLString 直接归一化裸 URL + 方法，无需手动构造 HttpRequest。
// host 从 URL 自动提取。method 为空时视为 GET（与 NormalizeURL 一致）。
func (x *ReverseRouter) NormalizeURLString(url, method string) (NormalizedRoute, bool) {
	route, reason := x.NormalizeURLDetailed(request.NewHttpRequest(url, nil, method, nil))
	return route, reason == NormalizeOK
}

// ListAssets 枚举树中全部已知资产（方法+路径模板+参数），按 AssetKey 稳定排序。
// 变量段输出为 {变量名}，RequiredParams 依赖 InferRequiredParams 是否已执行。
// 空树返回空切片（非 nil 判断可用 len==0）。
func (x *ReverseRouter) ListAssets() []NormalizedRoute {
	if x == nil || x.Tree == nil || x.Tree.Root == nil {
		return []NormalizedRoute{}
	}
	var assets []NormalizedRoute
	var walk func(n node.Node[node.NodeContext], segments, pathParams []string)
	walk = func(n node.Node[node.NodeContext], segments, pathParams []string) {
		for _, child := range n.GetChildren() {
			switch child.GetType() {
			case "request_path":
				walk(child, append(segments, child.GetKey()), pathParams)
			case "request_path_variable":
				name := child.GetKey()
				walk(child, append(segments, "{"+name+"}"), append(pathParams, name))
			case "request_method":
				assets = append(assets, buildAsset(child, segments, pathParams))
			}
		}
	}
	walk(x.Tree.Root, nil, nil)
	if assets == nil {
		return []NormalizedRoute{}
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i].AssetKey() < assets[j].AssetKey() })
	return assets
}

// buildAsset 由方法节点与其祖先路径段拼装一条资产。
func buildAsset(methodNode node.Node[node.NodeContext], segments, pathParams []string) NormalizedRoute {
	template := "/" + strings.Join(segments, "/")
	if len(segments) == 0 {
		template = "/"
	}
	asset := NormalizedRoute{
		Method:         methodNode.GetKey(),
		Template:       template,
		PathParams:     append([]string{}, pathParams...),
		QueryParams:    make([]string, 0),
		RequiredParams: make([]string, 0),
	}
	for _, child := range methodNode.GetChildren() {
		if child.GetType() != "request_param" {
			continue
		}
		param, ok := child.(*node.RequestParamNode)
		if !ok {
			continue
		}
		name := param.GetParamName()
		asset.QueryParams = append(asset.QueryParams, name)
		if param.IsRequired() {
			asset.RequiredParams = append(asset.RequiredParams, name)
		}
	}
	return asset
}

// --- RouterSet ---

// lookupRouter 只读查找 host 桶（不懒建）。归一化是读操作，不应因一次
// 试探就建出空桶——否则 Hosts()/Stats 会被"查过但从未采集"的 host 污染，
// 且 MaxHosts 语义会被读流量意外消耗。
func (s *RouterSet) lookupRouter(host string) *ReverseRouter {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.routers[host]
}

// NormalizeURLDetailed 在请求所属 host 的路由树中归一化，失败返回具体原因。
// 未知 host 返回 unknown_host（只读查找，不建桶）。
func (s *RouterSet) NormalizeURLDetailed(req *request.HttpRequest) (NormalizedRoute, NormalizeReason) {
	if req == nil || s == nil {
		return NormalizedRoute{}, NormalizeReasonInvalidRequest
	}
	r := s.lookupRouter(routerHost(req))
	if r == nil {
		return NormalizedRoute{}, NormalizeReasonUnknownHost
	}
	result, reason := r.NormalizeURLDetailed(req)
	if reason == NormalizeOK {
		result.Host = routerHost(req)
	}
	return result, reason
}

// NormalizeURLsDetailed 批量归一化，返回明细报告（键为 AssetKey）。
func (s *RouterSet) NormalizeURLsDetailed(reqs []*request.HttpRequest) NormalizeReport {
	report := NormalizeReport{Matched: make(map[string][]string)}
	for i, req := range reqs {
		normalized, reason := s.NormalizeURLDetailed(req)
		url := "<nil>"
		if req != nil {
			url = req.Url
		}
		if reason != NormalizeOK {
			report.Unmatched = append(report.Unmatched, NormalizeMiss{Index: i, URL: url, Reason: reason})
			continue
		}
		key := normalized.AssetKey()
		report.Matched[key] = append(report.Matched[key], url)
	}
	return report
}

// NormalizeAssetsDetailed 批量归一化，返回明细报告（键为 HostAssetKey，多目标场景）。
func (s *RouterSet) NormalizeAssetsDetailed(reqs []*request.HttpRequest) NormalizeReport {
	report := NormalizeReport{Matched: make(map[string][]string)}
	for i, req := range reqs {
		normalized, reason := s.NormalizeURLDetailed(req)
		url := "<nil>"
		if req != nil {
			url = req.Url
		}
		if reason != NormalizeOK {
			report.Unmatched = append(report.Unmatched, NormalizeMiss{Index: i, URL: url, Reason: reason})
			continue
		}
		key := normalized.HostAssetKey()
		report.Matched[key] = append(report.Matched[key], url)
	}
	return report
}

// NormalizeCurl 直接归一化一条 curl 命令（路由到其 host 桶）。
func (s *RouterSet) NormalizeCurl(raw string) (NormalizedRoute, bool) {
	route, reason := s.NormalizeCurlDetailed(raw)
	return route, reason == NormalizeOK
}

// NormalizeCurlDetailed 直接归一化一条 curl 命令，失败返回具体原因。
func (s *RouterSet) NormalizeCurlDetailed(raw string) (NormalizedRoute, NormalizeReason) {
	req, err := request.ParseCurl(raw)
	if err != nil {
		return NormalizedRoute{}, NormalizeReasonInvalidRequest
	}
	return s.NormalizeURLDetailed(req)
}

// NormalizeURLString 直接归一化裸 URL + 方法（host 从 URL 提取）。
func (s *RouterSet) NormalizeURLString(url, method string) (NormalizedRoute, bool) {
	route, reason := s.NormalizeURLDetailed(request.NewHttpRequest(url, nil, method, nil))
	return route, reason == NormalizeOK
}

// ListAssets 枚举各 host 的全部已知资产：host → 资产列表（Host 字段已填充）。
// 不存在 host 的试探不再建桶（只读快照）。
func (s *RouterSet) ListAssets() map[string][]NormalizedRoute {
	result := make(map[string][]NormalizedRoute)
	if s == nil {
		return result
	}
	s.mu.RLock()
	snapshot := make(map[string]*ReverseRouter, len(s.routers))
	for h, r := range s.routers {
		snapshot[h] = r
	}
	s.mu.RUnlock()
	for h, r := range snapshot {
		assets := r.ListAssets()
		for i := range assets {
			assets[i].Host = h
		}
		result[h] = assets
	}
	return result
}

// --- ProjectManager ---

// NormalizeURLDetailed 在指定项目内归一化，失败返回具体原因。
// 项目不存在返回 unknown_project（不自动建项目：读操作无副作用）。
func (m *ProjectManager) NormalizeURLDetailed(projectID string, req *request.HttpRequest) (NormalizedRoute, NormalizeReason) {
	if m == nil || req == nil {
		return NormalizedRoute{}, NormalizeReasonInvalidRequest
	}
	m.mu.RLock()
	rs := m.projects[normalizeProjectID(projectID)]
	m.mu.RUnlock()
	if rs == nil {
		return NormalizedRoute{}, NormalizeReasonUnknownProject
	}
	return rs.NormalizeURLDetailed(req)
}

// NormalizeURLs 在指定项目内批量归一化（键为 AssetKey），失败静默跳过。
// 需要失败明细时用 NormalizeURLsDetailed。
func (m *ProjectManager) NormalizeURLs(projectID string, reqs []*request.HttpRequest) map[string][]string {
	report := m.NormalizeURLsDetailed(projectID, reqs)
	return report.Matched
}

// NormalizeURLsDetailed 在指定项目内批量归一化，返回明细报告（键为 AssetKey）。
func (m *ProjectManager) NormalizeURLsDetailed(projectID string, reqs []*request.HttpRequest) NormalizeReport {
	report := NormalizeReport{Matched: make(map[string][]string)}
	for i, req := range reqs {
		normalized, reason := m.NormalizeURLDetailed(projectID, req)
		url := "<nil>"
		if req != nil {
			url = req.Url
		}
		if reason != NormalizeOK {
			report.Unmatched = append(report.Unmatched, NormalizeMiss{Index: i, URL: url, Reason: reason})
			continue
		}
		key := normalized.AssetKey()
		report.Matched[key] = append(report.Matched[key], url)
	}
	return report
}

// NormalizeAssetsDetailed 在指定项目内批量归一化，返回明细报告（键为 HostAssetKey）。
func (m *ProjectManager) NormalizeAssetsDetailed(projectID string, reqs []*request.HttpRequest) NormalizeReport {
	report := NormalizeReport{Matched: make(map[string][]string)}
	for i, req := range reqs {
		normalized, reason := m.NormalizeURLDetailed(projectID, req)
		url := "<nil>"
		if req != nil {
			url = req.Url
		}
		if reason != NormalizeOK {
			report.Unmatched = append(report.Unmatched, NormalizeMiss{Index: i, URL: url, Reason: reason})
			continue
		}
		key := normalized.HostAssetKey()
		report.Matched[key] = append(report.Matched[key], url)
	}
	return report
}

// NormalizeCurl 在指定项目内直接归一化一条 curl 命令。
func (m *ProjectManager) NormalizeCurl(projectID, raw string) (NormalizedRoute, bool) {
	route, reason := m.NormalizeCurlDetailed(projectID, raw)
	return route, reason == NormalizeOK
}

// NormalizeCurlDetailed 在指定项目内直接归一化一条 curl 命令，失败返回具体原因。
func (m *ProjectManager) NormalizeCurlDetailed(projectID, raw string) (NormalizedRoute, NormalizeReason) {
	req, err := request.ParseCurl(raw)
	if err != nil {
		return NormalizedRoute{}, NormalizeReasonInvalidRequest
	}
	return m.NormalizeURLDetailed(projectID, req)
}

// ProjectAssets 返回单个项目的资产清单：host → 资产列表。项目不存在返回空 map。
func (m *ProjectManager) ProjectAssets(projectID string) map[string][]NormalizedRoute {
	if m == nil {
		return map[string][]NormalizedRoute{}
	}
	m.mu.RLock()
	rs := m.projects[normalizeProjectID(projectID)]
	m.mu.RUnlock()
	if rs == nil {
		return map[string][]NormalizedRoute{}
	}
	return rs.ListAssets()
}

// ListAssets 返回全项目的资产清单：项目 → host → 资产列表。
func (m *ProjectManager) ListAssets() map[string]map[string][]NormalizedRoute {
	result := make(map[string]map[string][]NormalizedRoute)
	if m == nil {
		return result
	}
	m.mu.RLock()
	snapshot := make(map[string]*RouterSet, len(m.projects))
	for id, rs := range m.projects {
		snapshot[id] = rs
	}
	m.mu.RUnlock()
	for id, rs := range snapshot {
		result[id] = rs.ListAssets()
	}
	return result
}

// String 返回失败原因的人类可读串，便于日志与排查。
func (r NormalizeReason) String() string {
	switch r {
	case NormalizeOK:
		return "ok"
	case NormalizeReasonInvalidRequest:
		return "invalid_request（请求为nil或URL无法解析）"
	case NormalizeReasonUnknownHost:
		return "unknown_host（该host尚无采集数据）"
	case NormalizeReasonUnknownProject:
		return "unknown_project（项目不存在）"
	case NormalizeReasonUnknownPath:
		return "unknown_path（路径段未命中已知路由）"
	case NormalizeReasonUnknownMethod:
		return "unknown_method（路径命中但方法未采集过）"
	default:
		return string(r)
	}
}
