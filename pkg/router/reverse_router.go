package router

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/inference"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/tree"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

// MergeConfig 合并策略配置
type MergeConfig struct {
	// SiblingMergeThreshold 同层兄弟路径节点数量超过此阈值时尝试合并
	SiblingMergeThreshold int

	// PatternSimilarityThreshold 模式相似度阈值
	// 值中符合某种模式的比例超过此阈值时才合并（0.0-1.0）
	// 例如：5个值中有4个是数字，相似度=0.8，超过阈值0.6则合并
	PatternSimilarityThreshold float64

	// SimilarLengthBreakThreshold 相似长度字符串的合并突破阈值
	// 当同层兄弟节点数量 >= 此值，且都匹配 similar_length_strings 模式（长度相似
	// 但无结构化模式）时，仍允许合并为变量。
	// 理由：大量长度相似的字符串兄弟节点强烈暗示是变量值集合（如城市名、人名、
	// 商品名等），而非固定路由名（固定路由名很少超过5个同层）。
	// 设为 0 表示禁用此突破规则（与旧行为一致：similar_length_strings 永不合并）。
	SimilarLengthBreakThreshold int

	// RequiredParamThreshold 必需参数推断的出现率阈值
	// 参数出现率 = 参数出现次数 / 路由总请求次数。
	// 出现率 >= 此阈值时判定为必需参数（0.0-1.0）。
	// 默认 0.9：允许少量请求遗漏（如 10 次请求中出现 9 次即判定必需）。
	RequiredParamThreshold float64
}

// DefaultMergeConfig 默认合并配置
var DefaultMergeConfig = MergeConfig{
	SiblingMergeThreshold:       3,
	PatternSimilarityThreshold:  0.6,
	SimilarLengthBreakThreshold: 6,
	RequiredParamThreshold:      0.9,
}

// ReverseRouter 用于对请求进行逆向工程，反推出web应用的路由树
type ReverseRouter struct {
	Tree          *tree.Tree
	inferenceRule inference.TypeInferenceRule
	chainRule     *inference.ChainTypeInferenceRule
	mergeConfig   MergeConfig
	// limits/redact 用 atomic.Pointer 持有：命中快路径（无锁）可直接 Load，
	// Set 系方法 Store 新指针。无锁、-race 安全，配置变更低频。
	limits ResourceLimitsHolder
	redact RedactHolder
	// mergeRule 可选的自定义合并规则。nil 时走内置 findMergeableSiblings 逻辑。
	// 读写均由 mergeMu 保护（与合并临界区同锁），SetMergeRule/GetMergeRule
	// 持 mergeMu，findMergeableSiblings 在 checkAndMergeSiblings 临界区内直接读字段。
	mergeRule  MergeRule
	bodyParser *request.BodyParser
	logger     *RouterLogger
	stats      *RouterStats
	// pathRouter/paramRouter/ctRouter 是查询侧复用的内部路由组件，无状态，
	// 构造后只读。locateMethodNode/IsNeedRequest/FindRouteNode 通过它们查找
	// 路径末端/参数/Content-Type 子节点，消除重复的 FindChildByKey 内联逻辑。
	pathRouter  *RequestPathRouter
	paramRouter *RequestParamRouter
	ctRouter    *RequestContentTypeRouter
	// mergeMu 保护 checkAndMergeSiblings/mergeSiblings 的整个临界区。
	// 合并涉及"读兄弟数→决策→删旧节点→建变量节点→迁移孙节点"多步，
	// BaseNode.childMu 只保护单次子节点操作，无法保证整体原子——
	// 两个 goroutine 并发合并同一 parent 会读到中间态导致 double-move/丢节点。
	// 合并是低频操作（每 N 请求触发一次），router 级串行化代价可接受。
	mergeMu sync.Mutex
}

// NewReverseRouter 创建一个新的逆向路由器（使用默认配置）
func NewReverseRouter() *ReverseRouter {
	chainRule := inference.NewChainTypeInferenceRule()
	r := &ReverseRouter{
		Tree:          tree.NewTree(),
		inferenceRule: chainRule,
		chainRule:     chainRule,
		mergeConfig:   DefaultMergeConfig,
		bodyParser:    request.NewBodyParser(),
		logger:        NewRouterLogger(),
		stats:         NewRouterStats(),
		pathRouter:    &RequestPathRouter{},
		paramRouter:   &RequestParamRouter{},
		ctRouter:      &RequestContentTypeRouter{},
	}
	r.limits.StoreLimits(DefaultResourceLimits)
	r.redact.StoreRedact(DefaultRedactConfig())
	return r
}

// NewReverseRouterWithTree 使用已有的路由树创建逆向路由器
func NewReverseRouterWithTree(t *tree.Tree) *ReverseRouter {
	chainRule := inference.NewChainTypeInferenceRule()
	r := &ReverseRouter{
		Tree:          t,
		inferenceRule: chainRule,
		chainRule:     chainRule,
		mergeConfig:   DefaultMergeConfig,
		bodyParser:    request.NewBodyParser(),
		logger:        NewRouterLogger(),
		stats:         NewRouterStats(),
		pathRouter:    &RequestPathRouter{},
		paramRouter:   &RequestParamRouter{},
		ctRouter:      &RequestContentTypeRouter{},
	}
	r.limits.StoreLimits(DefaultResourceLimits)
	r.redact.StoreRedact(DefaultRedactConfig())
	return r
}

// SetInferenceRule 设置类型推断规则
func (x *ReverseRouter) SetInferenceRule(rule inference.TypeInferenceRule) {
	if rule != nil {
		x.inferenceRule = rule
	}
}

// SetMergeConfig 设置合并策略配置
func (x *ReverseRouter) SetMergeConfig(config MergeConfig) {
	if config.SiblingMergeThreshold < 2 {
		config.SiblingMergeThreshold = 2
	}
	if config.PatternSimilarityThreshold < 0.0 {
		config.PatternSimilarityThreshold = 0.0
	}
	if config.PatternSimilarityThreshold > 1.0 {
		config.PatternSimilarityThreshold = 1.0
	}
	// mergeConfig 的读写均用 mergeMu 串行化：合并临界区（checkAndMergeSiblingsLocked
	// 及内部）已在 mergeMu 下读 mergeConfig；此处写与 GetMergeConfig/InferRequiredParams
	// 的读也加锁，避免 RouterSet 配置传播与并发喂入产生数据竞争（-race）。
	x.mergeMu.Lock()
	x.mergeConfig = config
	x.mergeMu.Unlock()
}

// GetMergeConfig 获取合并策略配置
func (x *ReverseRouter) GetMergeConfig() MergeConfig {
	x.mergeMu.Lock()
	defer x.mergeMu.Unlock()
	return x.mergeConfig
}

// SetResourceLimits 设置资源上限（0 表示对应项不限制）。
// atomic 无锁写入，并发喂入侧直接 Load，无 -race 风险。
func (x *ReverseRouter) SetResourceLimits(limits ResourceLimits) {
	x.limits.StoreLimits(limits)
}

// GetResourceLimits 获取当前资源上限。
func (x *ReverseRouter) GetResourceLimits() ResourceLimits {
	return x.limits.LoadLimits()
}

// SetRedactConfig 设置敏感数据脱敏名单（参数/cookie 名大小写不敏感）。
// 命中名单的值只建占位节点，不存原值；nil/空名单表示关闭对应维度脱敏。
func (x *ReverseRouter) SetRedactConfig(cfg RedactConfig) {
	x.redact.StoreRedact(cfg)
}

// applyMetricCap 给值节点的 ValueMetric 设置上限（node 层统一入口）。
// maxUnique <= 0 时不设（保持无限制）。
func (x *ReverseRouter) applyMetricCap(m interface {
	SetMaxUnique(int, string)
}) {
	if m == nil {
		return
	}
	if maxUnique := x.limits.LoadLimits().MaxValuesPerMetric; maxUnique > 0 {
		m.SetMaxUnique(maxUnique, CappedMetricKey)
	}
}

// childrenGuard 检查父节点子节点数是否已达上限。
// 命中已存在节点不走此检查（纯查询无增长）；仅新建子节点前调用。
// 返回 nil 表示允许创建，非 nil 为 fail-soft 错误（调用方计 Errors 并返回）。
func (x *ReverseRouter) childrenGuard(parent node.Node[node.NodeContext]) error {
	maxChildren := x.limits.LoadLimits().MaxChildrenPerNode
	if maxChildren <= 0 {
		return nil
	}
	if parent.GetChildCount() >= maxChildren {
		x.stats.Warnings.Add(1)
		x.stats.RejectedChildren.Add(1)
		return fmt.Errorf("父节点 '%s' 子节点数达上限 %d，拒绝新建", parent.GetKey(), maxChildren)
	}
	return nil
}

// truncateSegment 按 MaxSegmentLen 截断超长路径段（0 表示不限制）。
func (x *ReverseRouter) truncateSegment(seg string) string {
	maxLen := x.limits.LoadLimits().MaxSegmentLen
	if maxLen > 0 && len(seg) > maxLen {
		x.stats.Warnings.Add(1)
		x.stats.TruncatedSegments.Add(1)
		return seg[:maxLen]
	}
	return seg
}

// SetLogger 设置自定义日志器。传入 nil 关闭日志。
func (x *ReverseRouter) SetLogger(l *RouterLogger) {
	if l == nil {
		x.logger = &RouterLogger{enabled: false}
	} else {
		x.logger = l
	}
}

// SetLogLevel 调整日志级别。
func (x *ReverseRouter) SetLogLevel(level LogLevel) {
	if x.logger != nil {
		x.logger.SetLevel(level)
	}
}

// GetStats 返回统计指标的快照。
func (x *ReverseRouter) GetStats() StatsSnapshot {
	return x.stats.Snapshot()
}

// HealthReport 是面向用户的一目了然的运行健康报告：性能 + 规模 + 护栏状态。
// 性能：请求量、平均/最大延迟；规模：Tree.RouteStats 节点分布；护栏：拒绝/截断/脱敏细分。
type HealthReport struct {
	Stats StatsSnapshot   `json:"stats"`
	Tree  tree.RouteStats `json:"tree"`
	// AvgLatencyHuman 平均延迟人类可读串（如 "725ns"、"1.2ms"）。
	AvgLatencyHuman string `json:"avg_latency_human"`
	// MaxLatencyHuman 最大延迟人类可读串。
	MaxLatencyHuman string `json:"max_latency_human"`
	// GuardRejectRate 护栏拒绝率 = rejected_children / requests（0~1，无请求时 0）。
	GuardRejectRate float64 `json:"guard_reject_rate"`
	// ErrorRate 错误率 = errors / requests（0~1，无请求时 0）。
	ErrorRate float64 `json:"error_rate"`
}

// Health 返回当前路由器的健康报告（性能+规模+护栏，一次调用拿全）。
func (x *ReverseRouter) Health() HealthReport {
	snap := x.stats.Snapshot()
	var treeStats tree.RouteStats
	if x.Tree != nil {
		treeStats = x.Tree.Stats()
	}
	rep := HealthReport{Stats: snap, Tree: treeStats}
	rep.AvgLatencyHuman = formatNanos(snap.AvgProcessingNanos())
	rep.MaxLatencyHuman = formatNanos(snap.MaxProcessingNanos)
	if snap.RequestsProcessed > 0 {
		rep.GuardRejectRate = float64(snap.RejectedChildren) / float64(snap.RequestsProcessed)
		rep.ErrorRate = float64(snap.Errors) / float64(snap.RequestsProcessed)
	}
	return rep
}

// formatNanos 将纳秒转人类可读串（ns/us/ms/s 自适应单位）。
func formatNanos(ns int64) string {
	if ns <= 0 {
		return "0ns"
	}
	const (
		us = 1000
		ms = 1000 * us
		s  = 1000 * ms
	)
	switch {
	case ns < us:
		return fmt.Sprintf("%dns", ns)
	case ns < ms:
		return fmt.Sprintf("%.1fus", float64(ns)/us)
	case ns < s:
		return fmt.Sprintf("%.1fms", float64(ns)/ms)
	default:
		return fmt.Sprintf("%.2fs", float64(ns)/s)
	}
}

// ResetStats 清零统计计数器。
func (x *ReverseRouter) ResetStats() {
	x.stats.Reset()
}

// ReverseHttpRequest 核心方法：将一个HTTP请求逆向工程到路由树中
func (x *ReverseRouter) ReverseHttpRequest(req *request.HttpRequest) error {
	if req == nil {
		x.stats.Errors.Add(1)
		return fmt.Errorf("请求不能为nil")
	}
	// 性能可观测：记录单请求处理耗时（成功路径），累加总量并更新最大值。
	start := time.Now()

	x.logger.Debug("开始处理请求", "url", req.Url, "method", req.Method)

	// 第1步：解析URL
	urlParser := request.NewUrlParser(req.Url)
	paths, params, err := urlParser.Parse()
	if err != nil {
		x.stats.Errors.Add(1)
		x.logger.Error("解析URL失败", "url", req.Url, "error", err)
		return fmt.Errorf("解析URL失败: %w", err)
	}
	// paths 中每个 *HttpRequestPath 来自 sync.Pool，函数结束归还（Pool 化消除每段堆分配）
	defer request.ReleasePaths(paths)

	// 第2步：路径匹配/创建
	currentNode := x.Tree.Root
	// 预分配 path 参数容量（路径中 key=value 段数 ≤ len(paths)），避免 append 扩容
	pathParams := make([]*request.HttpParam, 0, len(paths))
	for _, pathSegment := range paths {
		// 检查路径段是否为 key=value 格式的路径参数
		if pathSegment.IsPathParam() {
			// 将路径参数收集起来，后续和查询参数一起处理
			pathParams = append(pathParams, request.NewHttpParam(
				pathSegment.GetPathParamKey(),
				pathSegment.GetPathParamValue(),
			))
			// 路径参数段仍然作为路径节点存在，但键名使用参数键名
			currentNode, err = x.findOrCreatePathNode(currentNode, pathSegment.GetPathParamKey())
			if err != nil {
				x.stats.Errors.Add(1)
				return fmt.Errorf("处理路径参数段 '%s' 失败: %w", pathSegment.Path, err)
			}
		} else {
			currentNode, err = x.findOrCreatePathNode(currentNode, pathSegment.Path)
			if err != nil {
				x.stats.Errors.Add(1)
				return fmt.Errorf("处理路径段 '%s' 失败: %w", pathSegment.Path, err)
			}
		}
	}

	// 第4步：创建HTTP方法节点
	method := strings.ToUpper(req.Method)
	if method == "" {
		method = "GET"
	}
	methodNode, err := x.findOrCreateMethodNode(currentNode, method)
	if err != nil {
		x.stats.Errors.Add(1)
		return fmt.Errorf("处理HTTP方法 '%s' 失败: %w", method, err)
	}

	// 第5步：创建查询参数节点（包括路径中嵌入的参数和请求体参数）
	// 预分配总容量 = path 参数 + query 参数（body 参数后续 append），
	// 避免逐次 append 扩容。注意用新 slice 而非 append(pathParams,...)，
	// 以免复用 pathParams 底层数组造成别名问题。
	allParams := make([]*request.HttpParam, len(pathParams)+len(params))
	copy(allParams, pathParams)
	copy(allParams[len(pathParams):], params)

	// 解析请求体参数（表单/JSON/multipart），按 Content-Type 分发
	contentType := req.Headers.GetContentType()
	bodyParams, err := x.bodyParser.Parse(contentType, req.GetBody())
	if err != nil {
		x.stats.Errors.Add(1)
		x.stats.Warnings.Add(1)
		x.logger.Warn("解析请求体参数失败", "url", req.Url, "content_type", contentType, "error", err)
		return fmt.Errorf("解析请求体参数失败: %w", err)
	}
	if len(bodyParams) > 0 {
		x.stats.BodyParamsParsed.Add(int64(len(bodyParams)))
		x.logger.Debug("解析请求体参数", "count", len(bodyParams), "content_type", contentType)
	}
	allParams = append(allParams, bodyParams...)

	if len(allParams) > 0 {
		err = x.processParams(methodNode, allParams)
		if err != nil {
			x.stats.Errors.Add(1)
			return fmt.Errorf("处理查询参数失败: %w", err)
		}
	}

	// 第6步：创建Content-Type节点
	if contentType != "" && (method == "POST" || method == "PUT" || method == "PATCH") {
		_, err = x.findOrCreateContentTypeNode(methodNode, contentType)
		if err != nil {
			x.stats.Errors.Add(1)
			return fmt.Errorf("处理Content-Type '%s' 失败: %w", contentType, err)
		}
	}

	// 第7步：处理路由相关的Header节点
	err = x.processRoutingHeaders(methodNode, req.Headers)
	if err != nil {
		x.stats.Errors.Add(1)
		return fmt.Errorf("处理路由Header失败: %w", err)
	}

	// 第8步：处理Cookie路由节点
	err = x.processCookies(methodNode, req.Headers)
	if err != nil {
		x.stats.Errors.Add(1)
		return fmt.Errorf("处理Cookie路由失败: %w", err)
	}

	// 第9步：增加请求计数
	currentNode.IncrementRequestCount()
	methodNode.IncrementRequestCount()

	x.stats.RequestsProcessed.Add(1)
	x.stats.recordLatency(time.Since(start).Nanoseconds())
	x.logger.Debug("请求处理完成", "url", req.Url, "method", method, "params", len(allParams), "body_params", len(bodyParams))

	return nil
}

// findOrCreatePathNode 在当前节点下查找或创建路径节点
// 处理的边界条件：
//   - 尾部斜杠：/api/users/ 和 /api/users 视为相同
//   - 路径变量匹配：使用 IsMatch 进行严格匹配
//   - 编码路径：URL解码已在 UrlParser 中处理
func (x *ReverseRouter) findOrCreatePathNode(parent node.Node[node.NodeContext], pathSegment string) (node.Node[node.NodeContext], error) {
	if pathSegment == "" {
		return parent, nil
	}

	// 生产护栏：超长路径段截断（防恶意超长 URL 撑爆内存/日志）
	pathSegment = x.truncateSegment(pathSegment)

	// 首先尝试精确匹配路径节点
	child := parent.FindChildByKey(pathSegment)
	if child != nil && child.GetType() == "request_path" {
		return child, nil
	}

	// 尝试匹配已有的路径变量节点
	pathVarChild := parent.GetChildByType("request_path_variable")
	if pathVarChild != nil {
		pathVarNode := pathVarChild.(*node.RequestPathVariableNode)
		if pathVarNode.IsMatch(pathSegment) {
			pathVarNode.ObserveValue(pathSegment)
			// 增量推断：仅当 unique 值数自上次推断后变化时才重算。
			// 合并后的变量节点可能累积上千个值，每次命中都全量推断是 O(N²)；
			// 绝大多数命中是重复值（uniqueCount 不变），跳过重算。
			// 必须用 InferPhysicalAndLogical 分别回填，不能用单一 Infer——
			// 后者会把逻辑类型串（如 "uuid"）覆盖到物理类型字段。
			if x.chainRule != nil {
				uniqueCount := pathVarNode.GetValueMetric().GetUniqueValueCount()
				if uniqueCount != pathVarNode.GetLastInferredUniqueCount() {
					physicalType, logicalType, err := x.chainRule.InferPhysicalAndLogical(pathVarNode)
					x.stats.TypeInferences.Add(1)
					if err == nil {
						pathVarNode.SetType(value.Type(physicalType))
						pathVarNode.SetLogicalType(logicalType)
					}
					pathVarNode.SetLastInferredUniqueCount(uniqueCount)
				}
			}
			return pathVarChild, nil
		}
	}

	// 没有找到匹配的节点，创建新的路径节点。
	// 并发安全：多个 goroutine 可能同时通过上面的查找（无锁乐观读），都判定 child==nil
	// 后进入创建分支。用 mergeMu 串行化 create+AddChild+merge 临界区，并在锁内 double-check
	// （其他 goroutine 可能已抢先创建了同 key 节点），避免重复 AddChild 导致 children slice
	// 出现重复 key 节点 + 合并逻辑误判。命中已存在节点（前两个分支）不进此锁，吞吐不受影响。
	x.mergeMu.Lock()
	defer x.mergeMu.Unlock()

	// 锁内 double-check：精确匹配可能已被并发创建
	if child := parent.FindChildByKey(pathSegment); child != nil && child.GetType() == "request_path" {
		return child, nil
	}
	// 锁内 double-check：路径变量节点可能已被并发创建
	if pathVarChild := parent.GetChildByType("request_path_variable"); pathVarChild != nil {
		pathVarNode := pathVarChild.(*node.RequestPathVariableNode)
		if pathVarNode.IsMatch(pathSegment) {
			pathVarNode.ObserveValue(pathSegment)
			if x.chainRule != nil {
				uniqueCount := pathVarNode.GetValueMetric().GetUniqueValueCount()
				if uniqueCount != pathVarNode.GetLastInferredUniqueCount() {
					physicalType, logicalType, err := x.chainRule.InferPhysicalAndLogical(pathVarNode)
					x.stats.TypeInferences.Add(1)
					if err == nil {
						pathVarNode.SetType(value.Type(physicalType))
						pathVarNode.SetLogicalType(logicalType)
					}
					pathVarNode.SetLastInferredUniqueCount(uniqueCount)
				}
			}
			return pathVarChild, nil
		}
	}

	// 生产护栏：子节点数上限（防高基数路径段撑爆单父节点）
	if err := x.childrenGuard(parent); err != nil {
		return nil, err
	}
	newPathNode := node.NewRequestPathNode(pathSegment)
	if err := parent.AddChild(newPathNode); err != nil {
		return nil, fmt.Errorf("添加路径节点失败: %w", err)
	}

	// 检查是否需要合并兄弟节点为路径变量（已在 mergeMu 临界区内）
	x.checkAndMergeSiblingsLocked(parent)

	// 合并可能已将 newPathNode 从 parent 中移除，并生成了路径变量节点。
	// 若如此，继续路径处理应在路径变量节点下进行，否则后续节点会被挂到游离子树。
	if pv := parent.GetChildByType("request_path_variable"); pv != nil {
		pathVarNode := pv.(*node.RequestPathVariableNode)
		if pathVarNode.IsMatch(pathSegment) {
			return pathVarNode, nil
		}
	}
	return newPathNode, nil
}

// checkAndMergeSiblingsLocked 检查同一父节点下的兄弟路径节点数量（无锁版本）。
// 整个合并临界区由调用方持 mergeMu 串行化，避免并发合并同一 parent 导致中间态竞争。
// 见 ReverseRouter.mergeMu 注释。调用方必须确保已持 mergeMu。
func (x *ReverseRouter) checkAndMergeSiblingsLocked(parent node.Node[node.NodeContext]) {

	pathChildren := make([]node.Node[node.NodeContext], 0)
	for _, child := range parent.GetChildren() {
		if child.GetType() == "request_path" {
			pathChildren = append(pathChildren, child)
		}
	}

	if len(pathChildren) < x.mergeConfig.SiblingMergeThreshold {
		return
	}

	x.stats.MergeAttempts.Add(1)
	x.logger.Debug("尝试合并兄弟节点", "parent", parent.GetKey(), "children", len(pathChildren))

	// 尝试找到一组匹配模式的兄弟节点进行合并
	// 使用选择策略：只合并匹配模式的节点，保留不匹配的固定路径节点
	mergeable := x.findMergeableSiblings(parent, pathChildren)
	if len(mergeable) < x.mergeConfig.SiblingMergeThreshold {
		x.stats.MergeSkipped.Add(1)
		x.logger.Debug("合并跳过：可合并节点不足", "mergeable", len(mergeable), "threshold", x.mergeConfig.SiblingMergeThreshold)
		return
	}

	x.mergeSiblings(parent, mergeable)
}

// findMergeableSiblings 从兄弟节点中找到可以被合并的子集
// 策略：使用模式检测，只返回匹配模式的节点
// 如果整体匹配率很高（>=0.8），合并全部；否则只合并匹配模式的子集
func (x *ReverseRouter) findMergeableSiblings(parent node.Node[node.NodeContext], children []node.Node[node.NodeContext]) []node.Node[node.NodeContext] {
	if len(children) == 0 {
		return children
	}

	values := make([]string, len(children))
	for i, child := range children {
		values[i] = child.GetKey()
	}

	// 若注入了自定义合并规则，先交由它决策（直接读字段：本函数在
	// checkAndMergeSiblings 的 mergeMu 临界区内被调用，无需再加锁）。
	if x.mergeRule != nil {
		detector := NewPatternDetector()
		patternName, similarity := detector.DetectPattern(values)
		action, mergeable := x.mergeRule.Decide(MergeContext{
			Parent:     parent,
			Siblings:   children,
			Values:     values,
			Pattern:    patternName,
			Similarity: similarity,
			Config:     x.mergeConfig,
		})
		switch action {
		case MergeActionMerge:
			if len(mergeable) == 0 {
				return nil
			}
			// 过滤出确属 children 的节点，防止规则返回外部节点。
			return intersectNodes(children, mergeable)
		case MergeActionSkip:
			return nil
		case MergeActionDefault:
			// 落到下方内置逻辑
		}
	}

	detector := NewPatternDetector()
	patternName, similarity := detector.DetectPattern(values)

	// similar_length_strings 默认不合并（避免固定路径名被误合并）
	// 但当兄弟节点数量足够多时突破此规则（见 SimilarLengthBreakThreshold）
	if patternName == "similar_length_strings" {
		if x.mergeConfig.SimilarLengthBreakThreshold > 0 &&
			len(children) >= x.mergeConfig.SimilarLengthBreakThreshold {
			// 大量相似长度字符串 → 视为变量值集合，全部合并
			return children
		}
		return nil
	}

	// 前缀/后缀模式（如 user_001/user_002）：similarity 已 >= 0.6，全部合并
	// 这类模式基于值集合的公共前缀/后缀检测，无法逐值用正则匹配，直接合并全部
	if patternName == "prefix" || patternName == "suffix" {
		return children
	}

	// 如果整体模式匹配率非常高（>=0.8），全部可合并
	if similarity >= 0.8 {
		return children
	}

	// 如果整体匹配率中等（0.4-0.8），只合并匹配模式的子集
	// 逐个检测每个值是否匹配主导模式
	mergeable := make([]node.Node[node.NodeContext], 0)
	for i, child := range children {
		if x.valueMatchesPattern(detector, values[i], patternName) {
			mergeable = append(mergeable, child)
		}
	}

	return mergeable
}

// valueMatchesPattern 检查单个值是否匹配指定的模式
func (x *ReverseRouter) valueMatchesPattern(detector *PatternDetector, val string, patternName string) bool {
	// 使用 PatternDetector 的正则来匹配
	for i, pattern := range detector.patterns {
		if detector.names[i] == patternName {
			return pattern.MatchString(val)
		}
	}
	return false
}

// PatternDetector 值模式检测器
// 用于判断一组字符串值是否符合某种模式（如纯数字、UUID等）
type PatternDetector struct {
	patterns []*regexp.Regexp
	names    []string
}

// NewPatternDetector 创建模式检测器
// 模式顺序很重要：更具体的模式放在前面，通用模式（如纯数字）放在后面
// 当多个模式匹配率相同时，优先选择靠前的（更具体的）模式
// 例如：身份证号同时匹配 integer 和 idcard，应优先识别为 idcard
func NewPatternDetector() *PatternDetector {
	return &PatternDetector{
		patterns: []*regexp.Regexp{
			// 高度具体的格式（优先匹配）
			regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`), // UUID
			regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`),                              // 邮箱
			regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`),                                          // IP地址
			regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`),                                                            // 日期格式
			regexp.MustCompile(`^(?:\+?86)?1[3-9]\d{9}$`),                                                       // 中国手机号
			regexp.MustCompile(`^[1-9]\d{5}(?:19|20)\d{2}(?:0[1-9]|1[0-2])(?:0[1-9]|[12]\d|3[01])\d{3}[\dXx]$|^[1-9]\d{5}\d{2}(?:0[1-9]|1[0-2])(?:0[1-9]|[12]\d|3[01])\d{3}$`), // 身份证号
			regexp.MustCompile(`^[3-6]\d{15,18}$`),              // 银行卡号
			regexp.MustCompile(`^[\p{Han}][A-Z][A-Z0-9]{5,6}$`), // 车牌号
			regexp.MustCompile(`^v[0-9]+$`),                     // 版本号格式，如 v1, v2, v3
			regexp.MustCompile(`^[a-zA-Z]+[0-9]+$`),             // 字母+数字格式，如 abc123
			// 注意：不包含邮政编码(postalcode)模式
			// 6位纯数字无法与普通数字ID、验证码、订单号等可靠区分，
			// 误判率太高。6位数字会回退到 integer 模式，生成更合理的 {xxx_id} 变量名。
			// 通用格式（最后匹配）
			regexp.MustCompile(`^[0-9]+\.[0-9]+$`),  // 浮点数
			regexp.MustCompile(`^[0-9]+$`),          // 纯数字
			regexp.MustCompile(`^[0-9a-fA-F]{8,}$`), // hex 字符串 ID（MongoDB ObjectId/MD5/SHA 等）
		},
		names: []string{
			"uuid",
			"email",
			"ip",
			"date",
			"phone",
			"idcard",
			"bankcard",
			"plate",
			"version",
			"alphanumeric",
			"float",
			"integer",
			"hex_id",
		},
	}
}

// DetectPattern 检测一组值中最匹配的模式
// 返回匹配的模式名称和匹配比例
func (d *PatternDetector) DetectPattern(values []string) (string, float64) {
	if len(values) == 0 {
		return "", 0.0
	}

	bestPattern := ""
	bestRatio := 0.0

	for i, pattern := range d.patterns {
		matches := 0
		for _, val := range values {
			if pattern.MatchString(val) {
				matches++
			}
		}
		ratio := float64(matches) / float64(len(values))
		if ratio > bestRatio {
			bestRatio = ratio
			bestPattern = d.names[i]
		}
	}

	// 如果没有模式匹配足够多的值，检查前缀/后缀模式
	// 例如 user_001/user_002/user_003 → 公共前缀 user_ + 变量后缀
	if bestRatio < 0.5 {
		if ratio := detectPrefixPattern(values); ratio >= 0.6 {
			return "prefix", ratio
		}
		if ratio := detectSuffixPattern(values); ratio >= 0.6 {
			return "suffix", ratio
		}
	}

	// 如果没有模式匹配足够多的值，检查是否都是"类似长度"的字符串
	if bestRatio < 0.5 {
		// 检查长度一致性
		lengths := make([]int, len(values))
		for i, v := range values {
			lengths[i] = len(v)
		}
		avgLen := average(lengths)
		similarLenCount := 0
		for _, l := range lengths {
			if abs(l-int(avgLen)) <= 2 {
				similarLenCount++
			}
		}
		lenSimilarity := float64(similarLenCount) / float64(len(values))
		if lenSimilarity >= 0.8 {
			return "similar_length_strings", lenSimilarity
		}
	}

	return bestPattern, bestRatio
}

// mergeSiblings 将兄弟节点合并为一个路径变量节点
func (x *ReverseRouter) mergeSiblings(parent node.Node[node.NodeContext], children []node.Node[node.NodeContext]) {
	if len(children) == 0 {
		return
	}

	// 推断变量名和模式
	values := make([]string, len(children))
	for i, child := range children {
		values[i] = child.GetKey()
	}

	detector := NewPatternDetector()
	patternName, similarity := detector.DetectPattern(values)
	x.stats.PatternDetections.Add(1)
	x.logger.Debug("模式检测", "parent", parent.GetKey(), "pattern", patternName, "similarity", similarity, "values", len(values))

	// 根据推断的模式决定变量名
	// 对 prefix/suffix 模式，需要 values 来提取前缀/后缀生成变量名
	varName := inferVariableNameWithContext(parent.GetKey(), patternName, values)

	// 根据模式生成正则
	patternStr := inferPatternRegexWithContext(patternName, values)

	// 创建路径变量节点
	varNode := node.NewRequestPathVariableNode(varName, patternStr)
	// 生产护栏：ValueMetric 上限（防高基数字段撑爆内存）
	x.applyMetricCap(varNode.GetValueMetric())

	// 收集所有观察到的值并合并子树
	for _, child := range children {
		varNode.ObserveValue(child.GetKey())

		for _, grandchild := range child.GetChildren() {
			existing := varNode.FindChildByKey(grandchild.GetKey())
			if existing == nil {
				child.RemoveChild(grandchild)
				varNode.AddChild(grandchild)
			} else {
				existing.MergeWith(grandchild)
			}
		}

		parent.RemoveChild(child)
	}

	// 如果有链式推断规则，在所有值观察完之后推断物理类型和逻辑类型。
	// 合并产生的新变量节点首次推断后，记录 uniqueCount 缓存，使后续
	// findOrCreatePathNode 命中走增量逻辑（仅 uniqueCount 变化才重算）。
	if x.chainRule != nil {
		physicalType, logicalType, err := x.chainRule.InferPhysicalAndLogical(varNode)
		x.stats.TypeInferences.Add(1)
		if err == nil {
			varNode.SetType(value.Type(physicalType))
			varNode.SetLogicalType(logicalType)
		}
		varNode.SetLastInferredUniqueCount(varNode.GetValueMetric().GetUniqueValueCount())
	}

	parent.AddChild(varNode)
	x.stats.PathVariablesIdentified.Add(1)
	x.logger.Info("识别路径变量", "parent", parent.GetKey(), "var_name", varName, "pattern", patternName, "physical_type", varNode.GetValueType(), "logical_type", varNode.GetLogicalType(), "merged_count", len(children))

	// 级联合并：varNode 汇聚了多个兄弟节点的孙子节点，可能形成新的可合并组。
	// 例如 users/1/2/3 合并后，posts 下的 10/20/30 已就绪但尚未触发合并检查。
	x.cascadeMergeLocked(varNode)
}

// cascadeMergeLocked 对新生成的路径变量节点做向下级联合并。
// 将 varNode 的每个 request_path 子节点递归执行合并检查，
// 确保因父层合并而聚合的孙节点也能被正确合并为路径变量。
// 调用方必须持有 mergeMu 锁。
func (x *ReverseRouter) cascadeMergeLocked(varNode node.Node[node.NodeContext]) {
	for _, child := range varNode.GetChildren() {
		switch child.GetType() {
		case "request_path":
			x.checkAndMergeSiblingsLocked(child)
			x.cascadeMergeLocked(child)
		case "request_path_variable":
			x.cascadeMergeLocked(child)
		}
	}
}

// inferVariableName 根据父节点和模式推断变量名
func inferVariableName(parentKey string, patternName string) string {
	switch patternName {
	case "integer", "hex_id":
		return parentKey + "_id"
	case "uuid":
		return parentKey + "_uuid"
	case "float":
		return parentKey + "_value"
	case "date":
		return parentKey + "_date"
	case "ip":
		return parentKey + "_ip"
	case "version":
		return parentKey + "_version"
	case "alphanumeric":
		return parentKey + "_code"
	case "phone":
		return parentKey + "_phone"
	case "idcard":
		return parentKey + "_idcard"
	case "bankcard":
		return parentKey + "_bankcard"
	case "plate":
		return parentKey + "_plate"
	default:
		return "var_" + parentKey
	}
}

// inferVariableNameWithContext 带值集合上下文的变量名推断
// 对 prefix/suffix 模式，从值集合中提取前缀/后缀生成更有意义的变量名
func inferVariableNameWithContext(parentKey, patternName string, values []string) string {
	switch patternName {
	case "prefix":
		// user_001/user_002 → 提取前缀 user_ → 变量名 user_id
		prefix := trimTrailingDigits(longestCommonPrefix(values))
		base := trimTrailingSeparator(prefix)
		if base == "" {
			base = parentKey
		}
		return base + "_id"
	case "suffix":
		// 001_user/002_user → 提取后缀 _user → 变量名 user_id
		suffix := trimLeadingDigits(longestCommonSuffix(values))
		base := trimLeadingSeparator(suffix)
		if base == "" {
			base = parentKey
		}
		return base + "_id"
	default:
		return inferVariableName(parentKey, patternName)
	}
}

// inferPatternRegex 根据模式名推断正则表达式
func inferPatternRegex(patternName string) string {
	switch patternName {
	case "integer":
		return "[0-9]+"
	case "hex_id":
		return "[0-9a-fA-F]+"
	case "uuid":
		return "[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}"
	case "float":
		return "[0-9]+\\.[0-9]+"
	case "date":
		return "\\d{4}-\\d{2}-\\d{2}"
	case "ip":
		return "\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}"
	case "version":
		return "v[0-9]+"
	case "alphanumeric":
		return "[a-zA-Z]+[0-9]+"
	case "phone":
		return "(?:\\+?86)?1[3-9][0-9]{9}"
	case "idcard":
		return "[1-9][0-9]{5}(?:19|20)[0-9]{2}(?:0[1-9]|1[0-2])(?:0[1-9]|[12][0-9]|3[01])[0-9]{3}[0-9Xx]"
	case "bankcard":
		return "[3-6][0-9]{15,18}"
	case "plate":
		return "[\\p{Han}][A-Z][A-Z0-9]{5,6}"
	default:
		return "" // 无模式，匹配任何值
	}
}

// inferPatternRegexWithContext 带值集合上下文的正则推断
// 对 prefix/suffix 模式，生成"前缀/后缀+变量部分"的精确正则
func inferPatternRegexWithContext(patternName string, values []string) string {
	switch patternName {
	case "prefix":
		// user_001/user_002 → user_[0-9]+
		prefix := trimTrailingDigits(longestCommonPrefix(values))
		if prefix == "" {
			return ""
		}
		return regexp.QuoteMeta(prefix) + "[0-9]+"
	case "suffix":
		// 001_user/002_user → [0-9]+_user
		suffix := trimLeadingDigits(longestCommonSuffix(values))
		if suffix == "" {
			return ""
		}
		return "[0-9]+" + regexp.QuoteMeta(suffix)
	default:
		return inferPatternRegex(patternName)
	}
}

// trimTrailingSeparator 去掉末尾的分隔符（_、-等）
// user_ → user
func trimTrailingSeparator(s string) string {
	for len(s) > 0 {
		c := s[len(s)-1]
		if c == '_' || c == '-' || c == '.' {
			s = s[:len(s)-1]
			continue
		}
		break
	}
	return s
}

// trimLeadingSeparator 去掉开头的分隔符（_、-等）
// _user → user
func trimLeadingSeparator(s string) string {
	for len(s) > 0 {
		c := s[0]
		if c == '_' || c == '-' || c == '.' {
			s = s[1:]
			continue
		}
		break
	}
	return s
}

// findOrCreateMethodNode 在路径节点下查找或创建HTTP方法节点
func (x *ReverseRouter) findOrCreateMethodNode(parent node.Node[node.NodeContext], method string) (node.Node[node.NodeContext], error) {
	methodChild := parent.FindChildByKey(method)
	if methodChild != nil && methodChild.GetType() == "request_method" {
		return methodChild, nil
	}

	// 生产护栏：子节点数上限
	if err := x.childrenGuard(parent); err != nil {
		return nil, err
	}
	newMethodNode := node.NewRequestMethodNode(method)
	if err := parent.AddChild(newMethodNode); err != nil {
		return nil, fmt.Errorf("添加方法节点失败: %w", err)
	}

	return newMethodNode, nil
}

// processParams 处理查询参数
func (x *ReverseRouter) processParams(methodNode node.Node[node.NodeContext], params []*request.HttpParam) error {
	// 同一请求内同名参数出现多次（如 ?tag=a&tag=b）→ 多值参数。
	// 先统计，再逐条下发，避免多值信息在单个 HttpParam 内丢失。
	multi := make(map[string]bool, len(params))
	seen := make(map[string]int, len(params))
	for _, p := range params {
		name := strings.ToLower(p.Name)
		seen[name]++
		if seen[name] > 1 {
			multi[name] = true
		}
	}
	for _, param := range params {
		if err := x.findOrCreateParamNode(methodNode, param, multi[strings.ToLower(param.Name)]); err != nil {
			return err
		}
	}
	return nil
}

// findOrCreateParamNode 在方法节点下查找或创建参数节点
// 支持多值参数检测和类型推断
func (x *ReverseRouter) findOrCreateParamNode(methodNode node.Node[node.NodeContext], param *request.HttpParam, multiValue bool) error {
	// 参数名统一小写
	paramName := strings.ToLower(param.Name)

	paramChild := methodNode.FindChildByKey(paramName)
	if paramChild != nil && paramChild.GetType() == "request_param" {
		paramNode := paramChild.(*node.RequestParamNode)
		if multiValue {
			paramNode.SetMultiValue(true)
		}
		// 累加参数出现次数（用于必需性推断）
		paramNode.IncrementPresenceCount()
		// 脱敏参数只记结构：不存原值（Metric/类型推断/日志都跳过）。
		redacted := x.redact.IsParamRedacted(paramName)
		storedValue := param.Value
		if redacted {
			storedValue = ""
			x.stats.RedactedValues.Add(1)
		}
		// 观察参数值用于类型推断
		if storedValue != "" {
			paramNode.ObserveValue(storedValue)
		}
		paramNode.GetContext().SetKey(paramName, storedValue)

		// 增量推断：仅当 unique 值数自上次推断后变化时才重算。
		// 参数值大量重复（如 page=1 反复出现），每次命中全量推断是 O(N²)。
		if storedValue != "" && x.chainRule != nil {
			uniqueCount := paramNode.GetValueMetric().GetUniqueValueCount()
			if uniqueCount != paramNode.GetLastInferredUniqueCount() {
				physicalType, logicalType, err := x.chainRule.InferPhysicalAndLogical(paramNode)
				x.stats.TypeInferences.Add(1)
				if err == nil {
					paramNode.SetValueType(value.Type(physicalType))
					paramNode.SetLogicalType(logicalType)
				}
				paramNode.SetLastInferredUniqueCount(uniqueCount)
			}
		}
		return nil
	}

	// 脱敏参数：默认值传空，不存原值（NewRequestParamNode 会观察默认值）。
	storedDefault := param.Value
	logValue := param.Value
	if x.redact.IsParamRedacted(paramName) {
		storedDefault = ""
		logValue = "[redacted]"
		x.stats.RedactedValues.Add(1)
	}
	newParamNode := node.NewRequestParamNode(paramName, storedDefault, false)
	// 生产护栏：ValueMetric 上限
	x.applyMetricCap(newParamNode.GetValueMetric())
	if multiValue {
		newParamNode.SetMultiValue(true)
	}
	// 新参数首次出现，presenceCount 设为 1
	newParamNode.IncrementPresenceCount()

	// 如果参数值不为空，尝试类型推断（首次推断后记录 uniqueCount 缓存）
	if storedDefault != "" && x.chainRule != nil {
		physicalType, logicalType, err := x.chainRule.InferPhysicalAndLogical(newParamNode)
		x.stats.TypeInferences.Add(1)
		if err == nil {
			newParamNode.SetValueType(value.Type(physicalType))
			newParamNode.SetLogicalType(logicalType)
		}
		newParamNode.SetLastInferredUniqueCount(newParamNode.GetValueMetric().GetUniqueValueCount())
	}

	// 生产护栏：子节点数上限
	if err := x.childrenGuard(methodNode); err != nil {
		return err
	}
	if err := methodNode.AddChild(newParamNode); err != nil {
		return fmt.Errorf("添加参数节点 '%s' 失败: %w", paramName, err)
	}
	x.stats.ParamsCreated.Add(1)
	x.logger.Debug("创建参数节点", "method", methodNode.GetKey(), "param", paramName, "value", logValue, "physical_type", newParamNode.GetValueType(), "logical_type", newParamNode.GetLogicalType())

	return nil
}

// findOrCreateContentTypeNode 在方法节点下查找或创建Content-Type节点
func (x *ReverseRouter) findOrCreateContentTypeNode(methodNode node.Node[node.NodeContext], contentType string) (node.Node[node.NodeContext], error) {
	ctChild := methodNode.FindChildByKey(contentType)
	if ctChild != nil && ctChild.GetType() == "request_content_type" {
		return ctChild, nil
	}

	newCTNode := node.NewRequestContentTypeNode(contentType)
	// 生产护栏：子节点数上限
	if err := x.childrenGuard(methodNode); err != nil {
		return nil, err
	}
	if err := methodNode.AddChild(newCTNode); err != nil {
		return nil, fmt.Errorf("添加Content-Type节点失败: %w", err)
	}

	return newCTNode, nil
}

// InferRequiredParams 推断所有参数节点的必需性
//
// 遍历路由树中所有方法节点，对其下的每个参数节点基于出现频率推断是否为必需参数。
// 出现率 = 参数出现次数 / 方法节点请求次数；出现率 >= RequiredParamThreshold 判定为必需。
//
// 建议在路由树构建完成、导出/序列化之前调用此方法，以获得最准确的推断结果。
// 样本不足（方法节点请求次数 <= 1）的参数保持当前 required 值不变。
//
// 返回推断的必需参数数量。
func (x *ReverseRouter) InferRequiredParams() int {
	if x.Tree == nil || x.Tree.Root == nil {
		return 0
	}

	x.mergeMu.Lock()
	threshold := x.mergeConfig.RequiredParamThreshold
	x.mergeMu.Unlock()
	if threshold <= 0 {
		threshold = 0.9
	}

	requiredCount := 0
	x.visitMethodNodes(x.Tree.Root, func(methodNode node.Node[node.NodeContext]) {
		totalRequests := methodNode.GetRequestCount()
		for _, child := range methodNode.GetChildren() {
			if child.GetType() != "request_param" {
				continue
			}
			paramNode, ok := child.(*node.RequestParamNode)
			if !ok {
				continue
			}
			if paramNode.InferRequired(totalRequests, threshold) {
				requiredCount++
				x.logger.Debug("参数标记为必需", "param", paramNode.GetKey(), "presence", paramNode.GetPresenceCount(), "total", totalRequests)
			}
		}
	})

	x.stats.RequiredParamsInferred.Add(int64(requiredCount))
	x.logger.Info("必需参数推断完成", "required_count", requiredCount, "threshold", threshold)

	return requiredCount
}

// visitMethodNodes 递归遍历节点树，对所有 request_method 类型节点执行回调
func (x *ReverseRouter) visitMethodNodes(n node.Node[node.NodeContext], callback func(node.Node[node.NodeContext])) {
	if n == nil {
		return
	}
	if n.GetType() == "request_method" {
		callback(n)
	}
	for _, child := range n.GetChildren() {
		x.visitMethodNodes(child, callback)
	}
}

// IsNeedRequest 判断某个URL是否还需要请求
// 检查路径、方法、参数、Content-Type、Header路由、Cookie路由
func (x *ReverseRouter) IsNeedRequest(req *request.HttpRequest) bool {
	if req == nil {
		return false
	}

	urlParser := request.NewUrlParser(req.Url)
	paths, params, err := urlParser.Parse()
	if err != nil {
		return true
	}
	defer request.ReleasePaths(paths)

	method := strings.ToUpper(req.Method)
	if method == "" {
		method = "GET"
	}

	// 复用 locateMethodNode 定位方法节点（与 FindRouteNode 一致）
	methodChild := x.locateMethodNode(paths, method)
	if methodChild == nil {
		return true
	}

	// 检查参数（复用 paramRouter，内部做 ToLower）
	if len(params) > 0 {
		for _, param := range params {
			paramChild, _ := x.paramRouter.FindNode(methodChild, param.Name)
			if paramChild == nil {
				return true
			}
		}
	}

	// 检查Content-Type（复用 ctRouter）
	contentType := req.Headers.GetContentType()
	if contentType != "" && (method == "POST" || method == "PUT" || method == "PATCH") {
		ctChild, _ := x.ctRouter.FindNode(methodChild, contentType)
		if ctChild == nil {
			return true
		}
	}

	// 检查路由Header（两层查找，仍内联：无对应 HeaderRouter 组件）
	for headerName, normalize := range routingHeaders {
		val := req.Headers.Get(headerName)
		if val == "" {
			continue
		}
		normalizedValue := normalize(val)
		if normalizedValue == "" {
			continue
		}
		headerGroupChild := methodChild.FindChildByKey(headerName)
		if headerGroupChild == nil || headerGroupChild.GetType() != "request_header" {
			return true
		}
		// 检查值是否已存在
		valueChild := headerGroupChild.FindChildByKey(normalizedValue)
		if valueChild == nil {
			return true
		}
	}

	// 检查Cookie路由（两层查找，仍内联：无对应 CookieRouter 组件）
	cookieHeader := req.Headers.Get("Cookie")
	if cookieHeader != "" {
		cookies := request.ParseCookies(cookieHeader)
		for cookieName, cookieValue := range cookies {
			if cookieValue == "" {
				continue
			}
			cookieGroupChild := methodChild.FindChildByKey(cookieName)
			if cookieGroupChild == nil || cookieGroupChild.GetType() != "request_cookie" {
				return true
			}
			valueChild := cookieGroupChild.FindChildByKey(cookieValue)
			if valueChild == nil {
				return true
			}
		}
	}

	if methodChild.GetRequestCount() > 0 {
		return false
	}

	return true
}

// NormalizedRoute 是一条流量归一化后的路由资产。
// AssetKey 由 HTTP 方法和路径模板组成，保证不同方法不会被误合并。
type NormalizedRoute struct {
	Host           string
	Method         string
	Template       string
	PathParams     []string
	QueryParams    []string
	RequiredParams []string
}

// AssetKey 返回适合作为测绘资产唯一键的稳定标识。
func (r NormalizedRoute) AssetKey() string {
	if r.Method == "" {
		return r.Template
	}
	return r.Method + " " + r.Template
}

// HostAssetKey 返回包含目标 Host 的多目标资产唯一键。
// 空 Host 使用 "<unknown>"，避免与方法字段产生歧义并保持键格式稳定。
func (r NormalizedRoute) HostAssetKey() string {
	host := r.Host
	if host == "" {
		host = "<unknown>"
	}
	key := r.AssetKey()
	if key == "" {
		return host
	}
	return host + " " + key
}

// normalizePathSegments 沿已构建的路由树逐段定位，并同时生成路径模板。
// 匹配语义与 RequestPathRouter.FindNode 保持一致：固定路径优先，未命中时回退路径变量。
func (x *ReverseRouter) normalizePathSegments(paths []*request.HttpRequestPath) (node.Node[node.NodeContext], []string, []string, bool) {
	current := node.Node[node.NodeContext](x.Tree.Root)
	segments := make([]string, 0, len(paths))
	pathParams := make([]string, 0)
	for _, path := range paths {
		if path == nil || path.Path == "" {
			continue
		}
		child := current.FindChildByKey(path.Path)
		if child != nil {
			current = child
			segments = append(segments, path.Path)
			continue
		}
		pathVar := current.GetChildByType("request_path_variable")
		if pathVar == nil {
			return nil, nil, nil, false
		}
		current = pathVar
		name := pathVar.GetKey()
		segments = append(segments, "{"+name+"}")
		pathParams = append(pathParams, name)
	}
	return current, segments, pathParams, true
}

// NormalizeURL 将一条已采集请求归一化为方法+路径模板资产。
// 请求必须已经被 ReverseHttpRequest 收录；未命中已知路由时返回 false。
// 需要失败原因时用 NormalizeURLDetailed（本方法内部即调它，保持行为一致）。
func (x *ReverseRouter) NormalizeURL(req *request.HttpRequest) (NormalizedRoute, bool) {
	route, reason := x.NormalizeURLDetailed(req)
	return route, reason == NormalizeOK
}

// NormalizeURLs 批量将请求归入方法+模板资产键。
// 失败样本静默跳过；需要失败明细时用 NormalizeURLsDetailed。
func (x *ReverseRouter) NormalizeURLs(reqs []*request.HttpRequest) map[string][]string {
	return x.NormalizeURLsDetailed(reqs).Matched
}

// FindRouteNode 在已构建的路由树中查找给定请求会命中的"方法节点"
// （request_method 类型），返回该节点及其下匹配的 Content-Type 子节点（若有）。
//
// 与 IsNeedRequest 的区别：
//   - IsNeedRequest 返回 bool（请求是否还需发出去采集）
//   - FindRouteNode 返回命中的具体节点（供上层查询"这个请求落到哪条路由"）
//
// 命中规则与 ReverseHttpRequest 的路由构建一致：路径段精确匹配，未命中固定
// 路径时回退到路径变量节点；方法节点按大写匹配。
//
// 返回：
//   - methodNode: 命中的方法节点；路径或方法未命中时返回 nil
//   - contentTypeNode: methodNode 下匹配请求 Content-Type 的子节点（仅 POST/PUT/PATCH
//     且请求带 Content-Type 时查找）；无则为 nil
//   - err: URL 解析失败等错误
func (x *ReverseRouter) FindRouteNode(req *request.HttpRequest) (methodNode, contentTypeNode node.Node[node.NodeContext], err error) {
	if req == nil {
		return nil, nil, fmt.Errorf("请求不能为nil")
	}

	urlParser := request.NewUrlParser(req.Url)
	paths, _, err := urlParser.Parse()
	if err != nil {
		return nil, nil, fmt.Errorf("解析URL失败: %w", err)
	}
	defer request.ReleasePaths(paths)

	method := strings.ToUpper(req.Method)
	if method == "" {
		method = "GET"
	}
	methodNode = x.locateMethodNode(paths, method)
	if methodNode == nil {
		return nil, nil, nil // 路径或方法未命中
	}

	contentType := req.Headers.GetContentType()
	if contentType != "" && (method == "POST" || method == "PUT" || method == "PATCH") {
		contentTypeNode, _ = x.ctRouter.FindNode(methodNode, contentType)
	}

	return methodNode, contentTypeNode, nil
}

// locateMethodNode 沿路径段逐级匹配定位方法节点。
// 复用 pathRouter.FindNode 完成路径段下钻（含路径变量无条件回退），
// 再按 key（大写方法名）匹配方法子节点。路径或方法未命中返回 nil。
//
// IsNeedRequest 与 FindRouteNode 共用本方法，保证两者路由定位逻辑一致。
func (x *ReverseRouter) locateMethodNode(paths []*request.HttpRequestPath, method string) node.Node[node.NodeContext] {
	pathEnd, _ := x.pathRouter.FindNode(node.Node[node.NodeContext](x.Tree.Root), paths)
	if pathEnd == nil {
		return nil
	}
	return pathEnd.FindChildByKey(method)
}

// 数学辅助函数
func average(nums []int) float64 {
	if len(nums) == 0 {
		return 0
	}
	sum := 0
	for _, n := range nums {
		sum += n
	}
	return float64(sum) / float64(len(nums))
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// detectPrefixPattern 检测一组值是否具有"公共前缀+变量后缀"模式
// 例如：user_001/user_002/user_003 → 公共前缀 "user_" + 变量部分 001/002/003
//
// 返回：变量部分匹配率（0.0-1.0）。匹配率 = 变量部分符合"数字或字母数字"的值占比。
// 要求：
//   - 至少3个值才能可靠识别公共前缀
//   - 公共前缀长度 >= 1（避免空前缀）
//   - 公共前缀不以数字结尾（前缀应切到字符类别边界，如 user_ 而非 user_00）
//   - 变量部分长度 >= 1
func detectPrefixPattern(values []string) float64 {
	if len(values) < 3 {
		return 0
	}

	// 计算最长公共前缀
	commonPrefix := longestCommonPrefix(values)
	if len(commonPrefix) == 0 {
		return 0
	}

	// 将公共前缀回退到字符类别边界
	// 例如 user_00 → user_（去掉末尾数字，因为数字属于变量部分）
	commonPrefix = trimTrailingDigits(commonPrefix)
	if len(commonPrefix) == 0 {
		return 0
	}

	prefixLen := len(commonPrefix)
	variableMatches := 0
	for _, v := range values {
		if len(v) <= prefixLen {
			// 值长度 <= 前缀长度，没有变量部分
			continue
		}
		variable := v[prefixLen:]
		if isInteger(variable) || isAlphanumeric(variable) {
			variableMatches++
		}
	}

	return float64(variableMatches) / float64(len(values))
}

// detectSuffixPattern 检测一组值是否具有"变量前缀+公共后缀"模式
// 例如：001_user/002_user/003_user → 变量部分 001/002/003 + 公共后缀 "_user"
//
// 返回：变量部分匹配率（0.0-1.0）。
func detectSuffixPattern(values []string) float64 {
	if len(values) < 3 {
		return 0
	}

	// 计算最长公共后缀
	commonSuffix := longestCommonSuffix(values)
	if len(commonSuffix) == 0 {
		return 0
	}

	// 将公共后缀回退到字符类别边界
	// 例如 _user00 → _user（去掉开头数字）
	commonSuffix = trimLeadingDigits(commonSuffix)
	if len(commonSuffix) == 0 {
		return 0
	}

	suffixLen := len(commonSuffix)
	variableMatches := 0
	for _, v := range values {
		if len(v) <= suffixLen {
			continue
		}
		variable := v[:len(v)-suffixLen]
		if isInteger(variable) || isAlphanumeric(variable) {
			variableMatches++
		}
	}

	return float64(variableMatches) / float64(len(values))
}

// longestCommonPrefix 计算一组字符串的最长公共前缀
func longestCommonPrefix(values []string) string {
	if len(values) == 0 {
		return ""
	}
	prefix := values[0]
	for _, v := range values[1:] {
		j := 0
		for j < len(prefix) && j < len(v) && prefix[j] == v[j] {
			j++
		}
		prefix = prefix[:j]
		if prefix == "" {
			return ""
		}
	}
	return prefix
}

// longestCommonSuffix 计算一组字符串的最长公共后缀
func longestCommonSuffix(values []string) string {
	if len(values) == 0 {
		return ""
	}
	suffix := values[0]
	for _, v := range values[1:] {
		j := 0
		for j < len(suffix) && j < len(v) &&
			suffix[len(suffix)-1-j] == v[len(v)-1-j] {
			j++
		}
		suffix = suffix[len(suffix)-j:]
		if suffix == "" {
			return ""
		}
	}
	return suffix
}

// trimTrailingDigits 去掉字符串末尾的数字部分
// 用于将前缀切到字符类别边界：user_00 → user_
func trimTrailingDigits(s string) string {
	i := len(s)
	for i > 0 && s[i-1] >= '0' && s[i-1] <= '9' {
		i--
	}
	return s[:i]
}

// trimLeadingDigits 去掉字符串开头的数字部分
// 用于将后缀切到字符类别边界：00_user → _user
func trimLeadingDigits(s string) string {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	return s[i:]
}

// isAlphanumeric 判断字符串是否为字母数字组合（至少含一个字母和一个数字）
func isAlphanumeric(s string) bool {
	if len(s) == 0 {
		return false
	}
	hasLetter := false
	hasDigit := false
	for _, c := range s {
		if c >= '0' && c <= '9' {
			hasDigit = true
		} else if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			hasLetter = true
		} else {
			return false
		}
	}
	return hasLetter && hasDigit
}

// isInteger 判断字符串是否为整数
func isInteger(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i, c := range s {
		if i == 0 && (c == '+' || c == '-') {
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// routingHeaderEntry 路由 header 条目（有序，优先级从前到后）。
type routingHeaderEntry struct {
	name      string
	normalize func(string) string
}

// routingHeaderList 路由 header 有序列表。
// processRoutingHeaders 单次遍历请求 headers 时，对每个 header 名按此列表
// 大小写不敏感匹配，命中则规范化处理。替代原 map 遍历的 5×O(n) Get
// （原实现对每个 header 调 headers.Get，每次 Get 遍历全部 headers + EqualFold，
// 5 个 header 共 25 次比较；现单次遍历 headers，每个 header 名最多 5 次比较）。
var routingHeaderList = []routingHeaderEntry{
	{"Accept", normalizeAccept},
	{"Authorization", normalizeAuthorization},
	{"X-Api-Version", identityNormalize},
	{"Accept-Language", normalizeAcceptLanguage},
	{"X-Requested-With", identityNormalize},
}

// routingHeaders 定义影响路由决策的Header及其规范化方式
// key: header名称（大小写不敏感）
// normalize: 值规范化函数（如提取 Bearer 方案、截取第一个MIME类型等）
// 保留供外部可能的查询（向后兼容），内部优先用 routingHeaderList。
var routingHeaders = map[string]func(string) string{
	"Accept":           normalizeAccept,
	"Authorization":    normalizeAuthorization,
	"X-Api-Version":    identityNormalize,
	"Accept-Language":  normalizeAcceptLanguage,
	"X-Requested-With": identityNormalize,
}

// matchRoutingHeader 在 routingHeaderList 中按大小写不敏感匹配 header 名，
// 返回命中条目（含规范名 canonicalName 与 normalize 函数）；未命中返回 nil。
// 单次遍历列表（≤5 项），避免重复扫描。
func matchRoutingHeader(headerName string) *routingHeaderEntry {
	for i := range routingHeaderList {
		if strings.EqualFold(routingHeaderList[i].name, headerName) {
			return &routingHeaderList[i]
		}
	}
	return nil
}

// normalizeAccept 规范化 Accept header
// 只取第一个MIME类型，忽略质量因子
// 如 "application/json, text/html;q=0.9" → "application/json"
func normalizeAccept(val string) string {
	if val == "" {
		return ""
	}
	// 取第一个MIME类型（逗号前），去掉分号后的质量因子
	parts := strings.SplitN(val, ",", 2)
	mime := strings.TrimSpace(parts[0])
	// 去掉 ;q=xxx 部分
	if semiIdx := strings.Index(mime, ";"); semiIdx >= 0 {
		mime = strings.TrimSpace(mime[:semiIdx])
	}
	return mime
}

// normalizeAuthorization 规范化 Authorization header
// 只取认证方案（如 Bearer、Basic、Token）
func normalizeAuthorization(val string) string {
	if val == "" {
		return ""
	}
	parts := strings.SplitN(val, " ", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	// 覆盖说明：此分支为不可达死代码——val 非空时 strings.SplitN 恒返回至少 1 个 part，
	// len(parts) > 0 恒为真。为保持语句覆盖率天花板透明，保留原样并文档化（见 TestGapRC_AuthorizationReturn）。
	return ""
}

// normalizeAcceptLanguage 规范化 Accept-Language header
// 只取第一个语言标签，忽略质量因子
// 如 "zh-CN,zh;q=0.9,en;q=0.8" → "zh-CN"
func normalizeAcceptLanguage(val string) string {
	if val == "" {
		return ""
	}
	parts := strings.SplitN(val, ",", 2)
	lang := strings.TrimSpace(parts[0])
	if semiIdx := strings.Index(lang, ";"); semiIdx >= 0 {
		lang = strings.TrimSpace(lang[:semiIdx])
	}
	return lang
}

// identityNormalize 不做规范化，直接返回原值
func identityNormalize(val string) string {
	return val
}

// processRoutingHeaders 处理路由相关的Header
// 使用两层结构：Header名称节点 → Header值节点
// 这样同一个Header的不同值作为兄弟节点，便于后续变量合并
//
// 单次遍历请求 headers，对每个 header 名按 routingHeaderList 固定优先级
// 大小写不敏感匹配，命中则规范化并建节点。替代原 map 遍历对每个 header
// 调 headers.Get（每次 Get 遍历全部 headers + EqualFold，5 个 header 共 25 次比较）。
//
// 节点 key 用规范名（routingHeaderList 中的固定大小写，如 "Authorization"），
// 而非请求中传入的 header 名（可能大小写不一）。这样不同请求用不同大小写
// 写同一 header 时不再建多个节点，行为更一致。
func (x *ReverseRouter) processRoutingHeaders(methodNode node.Node[node.NodeContext], headers request.Headers) error {
	for headerName, val := range headers {
		if val == "" {
			continue
		}
		// 在路由 header 列表中大小写不敏感匹配，一次拿到 normalize 与规范名
		entry := matchRoutingHeader(headerName)
		if entry == nil {
			continue
		}

		normalizedValue := entry.normalize(val)
		if normalizedValue == "" {
			continue
		}

		// 用规范名（routingHeaderList 中的固定大小写）作节点 key
		canonicalName := entry.name

		// 查找或创建Header名称分组节点
		headerGroupChild := methodNode.FindChildByKey(canonicalName)
		var headerGroupNode *node.RequestHeaderNode

		if headerGroupChild != nil && headerGroupChild.GetType() == "request_header" {
			headerGroupNode = headerGroupChild.(*node.RequestHeaderNode)
		} else {
			// 创建新的Header名称分组节点
			headerGroupNode = node.NewRequestHeaderNode(canonicalName)
			// 生产护栏：子节点数上限
			if err := x.childrenGuard(methodNode); err != nil {
				return err
			}
			if err := methodNode.AddChild(headerGroupNode); err != nil {
				return fmt.Errorf("添加Header路由节点 '%s' 失败: %w", canonicalName, err)
			}
		}

		// 生产护栏：值节点数上限（防高基数 header 值撑爆分组）。
		// 已存在值只做计数累加不增长，无需检查；仅新建前检查。
		if headerGroupNode.FindChildByKey(normalizedValue) == nil {
			if err := x.childrenGuard(headerGroupNode); err != nil {
				continue
			}
		}
		// 在分组节点下查找或创建Header值节点
		if vn := headerGroupNode.FindOrCreateValueNode(normalizedValue); vn != nil {
			// 生产护栏：ValueMetric 上限
			x.applyMetricCap(vn.GetValueMetric())
		}
	}

	return nil
}

// processCookies 处理Cookie路由
// 使用两层结构：Cookie名称节点 → Cookie值节点
// 这样同一个Cookie的不同值作为兄弟节点，便于后续变量合并
func (x *ReverseRouter) processCookies(methodNode node.Node[node.NodeContext], headers request.Headers) error {
	cookieHeader := headers.Get("Cookie")
	if cookieHeader == "" {
		return nil
	}

	cookies := request.ParseCookies(cookieHeader)

	for cookieName, cookieValue := range cookies {
		if cookieValue == "" {
			continue
		}

		// 查找或创建Cookie名称分组节点
		cookieChild := methodNode.FindChildByKey(cookieName)
		var cookieGroupNode *node.RequestCookieNode

		if cookieChild != nil && cookieChild.GetType() == "request_cookie" {
			cookieGroupNode = cookieChild.(*node.RequestCookieNode)
		} else {
			// 创建新的Cookie名称分组节点
			cookieGroupNode = node.NewRequestCookieNode(cookieName)
			// 生产护栏：子节点数上限
			if err := x.childrenGuard(methodNode); err != nil {
				return err
			}
			if err := methodNode.AddChild(cookieGroupNode); err != nil {
				return fmt.Errorf("添加Cookie路由节点 '%s' 失败: %w", cookieName, err)
			}
		}

		// 脱敏 cookie 只记结构：值替换为占位键，不存原值。
		storedCookieValue := cookieValue
		if x.redact.IsCookieRedacted(cookieName) {
			storedCookieValue = RedactedCookieValue
			x.stats.RedactedValues.Add(1)
		}
		// 生产护栏：值节点数上限（超限跳过该值，fail-soft）
		if cookieGroupNode.FindChildByKey(storedCookieValue) == nil {
			if err := x.childrenGuard(cookieGroupNode); err != nil {
				continue
			}
		}
		// 在分组节点下查找或创建Cookie值节点
		if vn := cookieGroupNode.FindOrCreateValueNode(storedCookieValue); vn != nil {
			// 生产护栏：ValueMetric 上限
			x.applyMetricCap(vn.GetValueMetric())
		}
	}

	return nil
}
