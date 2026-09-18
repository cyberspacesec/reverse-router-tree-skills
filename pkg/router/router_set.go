package router

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/inference"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
)

// RouterSet 按目标 host 分桶管理独立的 ReverseRouter，供网络空间测绘多目标采集使用。
type RouterSet struct {
	mu               sync.RWMutex
	routers          map[string]*ReverseRouter
	config           MergeConfig
	limits           ResourceLimits
	redact           RedactConfig
	maxHosts         int
	rule             inference.TypeInferenceRule
	mergeRule        MergeRule
	logger           *RouterLogger
	loggerConfigured bool
	logLevel         *LogLevel
}

// NewRouterSet 创建一个按 host 隔离的路由器集合。
func NewRouterSet() *RouterSet {
	return &RouterSet{
		routers:  make(map[string]*ReverseRouter),
		config:   DefaultMergeConfig,
		limits:   DefaultResourceLimits,
		redact:   DefaultRedactConfig(),
		maxHosts: DefaultMaxHosts,
	}
}

func routerHost(req *request.HttpRequest) string {
	if req == nil {
		return ""
	}
	if h := strings.TrimSpace(req.Host); h != "" {
		return strings.ToLower(h)
	}
	return strings.ToLower(request.ExtractHost(req.Url))
}

// RouterFor 返回请求对应的 host 路由器，并在首次访问时懒创建。
// host 桶数达 MaxHosts 上限时返回 nil（调用方 fail-soft）；0 表示不限制。
func (s *RouterSet) RouterFor(req *request.HttpRequest) *ReverseRouter {
	if s == nil {
		return nil
	}
	host := routerHost(req)
	s.mu.RLock()
	r := s.routers[host]
	s.mu.RUnlock()
	if r != nil {
		return r
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if r = s.routers[host]; r == nil {
		if s.maxHosts > 0 && len(s.routers) >= s.maxHosts {
			return nil
		}
		r = NewReverseRouter()
		r.SetMergeConfig(s.config)
		r.SetResourceLimits(s.limits)
		r.SetRedactConfig(s.redact)
		if s.rule != nil {
			r.SetInferenceRule(s.rule)
		}
		if s.mergeRule != nil {
			r.SetMergeRule(s.mergeRule)
		}
		if s.loggerConfigured {
			r.SetLogger(s.logger)
		}
		if s.logLevel != nil {
			r.SetLogLevel(*s.logLevel)
		}
		s.routers[host] = r
	}
	return r
}

// SetMaxHosts 设置最大 host 桶数（0 表示不限制）。超限后新 host 建桶失败。
func (s *RouterSet) SetMaxHosts(n int) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.maxHosts = n
	s.mu.Unlock()
}

// Delete 删除指定 host 的路由树，释放内存。host 不存在时无操作，返回 false。
func (s *RouterSet) Delete(host string) bool {
	if s == nil {
		return false
	}
	host = strings.ToLower(strings.TrimSpace(host))
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.routers[host]; !ok {
		return false
	}
	delete(s.routers, host)
	return true
}

// SetResourceLimits 设置默认资源上限，并同步已有 Host 路由器。
func (s *RouterSet) SetResourceLimits(limits ResourceLimits) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.limits = limits
	routers := make([]*ReverseRouter, 0, len(s.routers))
	for _, r := range s.routers {
		routers = append(routers, r)
	}
	s.mu.Unlock()
	for _, r := range routers {
		r.SetResourceLimits(limits)
	}
}

// SetRedactConfig 设置默认脱敏名单，并同步已有 Host 路由器。
func (s *RouterSet) SetRedactConfig(cfg RedactConfig) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.redact = cfg
	routers := make([]*ReverseRouter, 0, len(s.routers))
	for _, r := range s.routers {
		routers = append(routers, r)
	}
	s.mu.Unlock()
	for _, r := range routers {
		r.SetRedactConfig(cfg)
	}
}

// ReverseHttpRequest 将请求分发到对应 host 的独立路由树。
func (s *RouterSet) ReverseHttpRequest(req *request.HttpRequest) error {
	if req == nil {
		return fmt.Errorf("请求不能为nil")
	}
	if s == nil {
		return fmt.Errorf("RouterSet不能为nil")
	}
	r := s.RouterFor(req)
	if r == nil {
		return fmt.Errorf("无法为请求获取路由树")
	}
	return r.ReverseHttpRequest(req)
}

// ReverseCurls 解析并分发一批 curl 请求。
func (s *RouterSet) ReverseCurls(curls []string) BatchResult {
	result := BatchResult{}
	for i, raw := range curls {
		req, err := request.ParseCurl(raw)
		if err != nil {
			result.Failed++
			appendBatchError(&result, i, raw, err)
			continue
		}
		if err = s.ReverseHttpRequest(req); err != nil {
			result.Failed++
			appendBatchError(&result, i, raw, err)
			continue
		}
		result.Processed++
	}
	return result
}

// NormalizeURL 在请求所属 host 的路由树中执行归一化。
func (s *RouterSet) NormalizeURL(req *request.HttpRequest) (NormalizedRoute, bool) {
	r := s.RouterFor(req)
	if r == nil {
		return NormalizedRoute{}, false
	}
	result, ok := r.NormalizeURL(req)
	if ok {
		result.Host = routerHost(req)
	}
	return result, ok
}

// NormalizeURLs 批量归一化请求，返回 AssetKey 到原始 URL 的分桶。
func (s *RouterSet) NormalizeURLs(reqs []*request.HttpRequest) map[string][]string {
	result := make(map[string][]string)
	for _, req := range reqs {
		n, ok := s.NormalizeURL(req)
		if !ok {
			continue
		}
		result[n.AssetKey()] = append(result[n.AssetKey()], req.Url)
	}
	return result
}

// NormalizeAssets 批量归一化请求，返回包含 Host 的多目标资产键到原始 URL 的分桶。
func (s *RouterSet) NormalizeAssets(reqs []*request.HttpRequest) map[string][]string {
	result := make(map[string][]string)
	for _, req := range reqs {
		n, ok := s.NormalizeURL(req)
		if !ok {
			continue
		}
		result[n.HostAssetKey()] = append(result[n.HostAssetKey()], req.Url)
	}
	return result
}

// snapshotRouters 返回当前子路由器快照，避免配置传播时持有集合锁。
func (s *RouterSet) snapshotRouters() []*ReverseRouter {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*ReverseRouter, 0, len(s.routers))
	for _, r := range s.routers {
		result = append(result, r)
	}
	return result
}

// SetMergeConfig 设置默认合并配置，并同步已有 Host 路由器。
func (s *RouterSet) SetMergeConfig(config MergeConfig) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.config = config
	routers := make([]*ReverseRouter, 0, len(s.routers))
	for _, r := range s.routers {
		routers = append(routers, r)
	}
	s.mu.Unlock()
	for _, r := range routers {
		r.SetMergeConfig(config)
	}
}

// SetInferenceRule 设置默认类型推断规则，并同步已有 Host 路由器。
func (s *RouterSet) SetInferenceRule(rule inference.TypeInferenceRule) {
	if s == nil || rule == nil {
		return
	}
	s.mu.Lock()
	s.rule = rule
	routers := make([]*ReverseRouter, 0, len(s.routers))
	for _, r := range s.routers {
		routers = append(routers, r)
	}
	s.mu.Unlock()
	for _, r := range routers {
		r.SetInferenceRule(rule)
	}
}

// SetMergeRule 设置自定义合并规则，并同步已有 Host 路由器。
func (s *RouterSet) SetMergeRule(rule MergeRule) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.mergeRule = rule
	routers := make([]*ReverseRouter, 0, len(s.routers))
	for _, r := range s.routers {
		routers = append(routers, r)
	}
	s.mu.Unlock()
	for _, r := range routers {
		r.SetMergeRule(rule)
	}
}

// SetLogger 设置日志器，并同步已有 Host 路由器；nil 会关闭日志。
func (s *RouterSet) SetLogger(logger *RouterLogger) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.logger = logger
	s.loggerConfigured = true
	routers := make([]*ReverseRouter, 0, len(s.routers))
	for _, r := range s.routers {
		routers = append(routers, r)
	}
	s.mu.Unlock()
	for _, r := range routers {
		r.SetLogger(logger)
	}
}

// SetLogLevel 设置日志级别，并同步已有 Host 路由器。
func (s *RouterSet) SetLogLevel(level LogLevel) {
	if s == nil {
		return
	}
	s.mu.Lock()
	pl := level
	s.logLevel = &pl
	routers := make([]*ReverseRouter, 0, len(s.routers))
	for _, r := range s.routers {
		routers = append(routers, r)
	}
	s.mu.Unlock()
	for _, r := range routers {
		r.SetLogLevel(level)
	}
}

// Hosts 返回当前所有 host，结果稳定排序。
func (s *RouterSet) Hosts() []string {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	hosts := make([]string, 0, len(s.routers))
	for h := range s.routers {
		hosts = append(hosts, h)
	}
	s.mu.RUnlock()
	sort.Strings(hosts)
	return hosts
}

// Stats 返回各 host 的统计快照。
func (s *RouterSet) Stats() map[string]StatsSnapshot {
	result := make(map[string]StatsSnapshot)
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
		result[h] = r.GetStats()
	}
	return result
}

// Health 返回各 host 的健康报告（性能+规模+护栏，key 为 host）。
func (s *RouterSet) Health() map[string]HealthReport {
	result := make(map[string]HealthReport)
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
		result[h] = r.Health()
	}
	return result
}

// appendBatchError 追加批量处理失败详情，沿用 ReverseRouter 的数量和长度限制。
func appendBatchError(result *BatchResult, index int, raw string, err error) {
	if len(result.Errors) >= maxBatchErrors {
		return
	}
	if len(raw) > maxBatchErrorRawLen {
		raw = raw[:maxBatchErrorRawLen] + "...(truncated)"
	}
	result.Errors = append(result.Errors, BatchError{Index: index, Raw: raw, Err: err})
}
