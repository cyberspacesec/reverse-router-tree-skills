package router

import (
	"fmt"
	"sort"
	"sync"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/inference"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
)

// ProjectManager 按安全测试项目隔离管理多个 RouterSet。
//
// 背景：用户可能同时跑很多个安全测试项目，每个项目采集不同的目标资产。
// 若共用一个 RouterSet（按 host 分桶），不同项目的同 host 流量会互相污染
// （host 桶是全局的）；若调用方自己维护 map[string]*RouterSet，则容量治理、
// 配置传播、统计聚合都要重复造轮子。本管理器提供项目级隔离 + 统一治理：
//
//   - 隔离：每个项目独立 RouterSet，host 桶、合并状态、统计完全独立。
//   - 治理：MaxProjects 防项目爆炸；默认配置（合并/上限/脱敏/host上限/日志）
//     向已有项目传播，新建项目自动继承。
//   - 可观测：Stats/Health 按项目聚合，一次调用看全所有项目的性能与规模。
//
// 并发安全：内部 RWMutex 保护项目表；各项目 RouterSet 自带锁。nil 安全。
type ProjectManager struct {
	mu          sync.RWMutex
	projects    map[string]*RouterSet
	config      MergeConfig
	limits      ResourceLimits
	redact      RedactConfig
	maxHosts    int
	maxProjects int
	rule        inference.TypeInferenceRule
	mergeRule   MergeRule
	logger      *RouterLogger
	logged      bool
	logLevel    *LogLevel
}

// NewProjectManager 创建项目管理器（默认配置，默认项目上限）。
func NewProjectManager() *ProjectManager {
	return &ProjectManager{
		projects:    make(map[string]*RouterSet),
		config:      DefaultMergeConfig,
		limits:      DefaultResourceLimits,
		redact:      DefaultRedactConfig(),
		maxHosts:    DefaultMaxHosts,
		maxProjects: DefaultMaxProjects,
	}
}

// normalizeProjectID 项目 ID 规范化：去首尾空白；空 ID 视为无效。
func normalizeProjectID(id string) string {
	// 项目 ID 保持原样大小写（与 host 不同，项目名是用户自定义标识），
	// 仅去空白，避免 "proj" 与 " proj " 建出两个项目。
	if len(id) == 0 {
		return ""
	}
	trimmed := id
	for len(trimmed) > 0 && (trimmed[0] == ' ' || trimmed[0] == '\t' || trimmed[0] == '\n' || trimmed[0] == '\r') {
		trimmed = trimmed[1:]
	}
	for len(trimmed) > 0 {
		c := trimmed[len(trimmed)-1]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			trimmed = trimmed[:len(trimmed)-1]
			continue
		}
		break
	}
	return trimmed
}

// Project 返回指定项目的 RouterSet，首次访问时懒创建并继承管理器默认配置。
// 项目数达 MaxProjects 上限时返回错误（调用方 fail-soft）；空 ID 返回错误。
// 0 上限表示不限制。已存在项目直接返回，不受上限影响。
func (m *ProjectManager) Project(id string) (*RouterSet, error) {
	if m == nil {
		return nil, fmt.Errorf("ProjectManager不能为nil")
	}
	id = normalizeProjectID(id)
	if id == "" {
		return nil, fmt.Errorf("项目ID不能为空")
	}
	m.mu.RLock()
	rs := m.projects[id]
	m.mu.RUnlock()
	if rs != nil {
		return rs, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if rs = m.projects[id]; rs != nil {
		return rs, nil
	}
	if m.maxProjects > 0 && len(m.projects) >= m.maxProjects {
		return nil, fmt.Errorf("项目数达上限 %d，拒绝新建项目 '%s'", m.maxProjects, id)
	}
	rs = NewRouterSet()
	rs.SetMergeConfig(m.config)
	rs.SetResourceLimits(m.limits)
	rs.SetRedactConfig(m.redact)
	rs.SetMaxHosts(m.maxHosts)
	if m.rule != nil {
		rs.SetInferenceRule(m.rule)
	}
	if m.mergeRule != nil {
		rs.SetMergeRule(m.mergeRule)
	}
	if m.logged {
		rs.SetLogger(m.logger)
	}
	if m.logLevel != nil {
		rs.SetLogLevel(*m.logLevel)
	}
	m.projects[id] = rs
	return rs, nil
}

// Delete 删除指定项目及其全部路由树，释放内存。项目不存在返回 false。
func (m *ProjectManager) Delete(id string) bool {
	if m == nil {
		return false
	}
	id = normalizeProjectID(id)
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.projects[id]; !ok {
		return false
	}
	delete(m.projects, id)
	return true
}

// Projects 返回当前所有项目 ID，稳定排序。
func (m *ProjectManager) Projects() []string {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	ids := make([]string, 0, len(m.projects))
	for id := range m.projects {
		ids = append(ids, id)
	}
	m.mu.RUnlock()
	sort.Strings(ids)
	return ids
}

// SetMaxProjects 设置最大项目数（0 表示不限制）。超限后新项目建桶失败。
func (m *ProjectManager) SetMaxProjects(n int) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.maxProjects = n
	m.mu.Unlock()
}

// SetMaxHosts 设置各项目的默认 host 上限，并同步已有项目。
func (m *ProjectManager) SetMaxHosts(n int) {
	if m == nil {
		return
	}
	for _, rs := range m.snapshot() {
		rs.SetMaxHosts(n)
	}
	m.mu.Lock()
	m.maxHosts = n
	m.mu.Unlock()
}

// SetMergeConfig 设置默认合并配置，并同步已有项目。
func (m *ProjectManager) SetMergeConfig(config MergeConfig) {
	if m == nil {
		return
	}
	for _, rs := range m.snapshot() {
		rs.SetMergeConfig(config)
	}
	m.mu.Lock()
	m.config = config
	m.mu.Unlock()
}

// SetResourceLimits 设置默认资源上限，并同步已有项目。
func (m *ProjectManager) SetResourceLimits(limits ResourceLimits) {
	if m == nil {
		return
	}
	for _, rs := range m.snapshot() {
		rs.SetResourceLimits(limits)
	}
	m.mu.Lock()
	m.limits = limits
	m.mu.Unlock()
}

// SetRedactConfig 设置默认脱敏名单，并同步已有项目。
func (m *ProjectManager) SetRedactConfig(cfg RedactConfig) {
	if m == nil {
		return
	}
	for _, rs := range m.snapshot() {
		rs.SetRedactConfig(cfg)
	}
	m.mu.Lock()
	m.redact = cfg
	m.mu.Unlock()
}

// SetInferenceRule 设置默认类型推断规则，并同步已有项目。
func (m *ProjectManager) SetInferenceRule(rule inference.TypeInferenceRule) {
	if m == nil || rule == nil {
		return
	}
	for _, rs := range m.snapshot() {
		rs.SetInferenceRule(rule)
	}
	m.mu.Lock()
	m.rule = rule
	m.mu.Unlock()
}

// SetMergeRule 设置自定义合并规则，并同步已有项目。
func (m *ProjectManager) SetMergeRule(rule MergeRule) {
	if m == nil {
		return
	}
	for _, rs := range m.snapshot() {
		rs.SetMergeRule(rule)
	}
	m.mu.Lock()
	m.mergeRule = rule
	m.mu.Unlock()
}

// SetLogger 设置日志器，并同步已有项目；nil 会关闭日志。
func (m *ProjectManager) SetLogger(logger *RouterLogger) {
	if m == nil {
		return
	}
	for _, rs := range m.snapshot() {
		rs.SetLogger(logger)
	}
	m.mu.Lock()
	m.logger = logger
	m.logged = true
	m.mu.Unlock()
}

// SetLogLevel 设置日志级别，并同步已有项目。
func (m *ProjectManager) SetLogLevel(level LogLevel) {
	if m == nil {
		return
	}
	for _, rs := range m.snapshot() {
		rs.SetLogLevel(level)
	}
	m.mu.Lock()
	pl := level
	m.logLevel = &pl
	m.mu.Unlock()
}

// snapshot 返回当前项目 RouterSet 快照，避免配置传播时持有管理锁。
func (m *ProjectManager) snapshot() []*RouterSet {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*RouterSet, 0, len(m.projects))
	for _, rs := range m.projects {
		result = append(result, rs)
	}
	return result
}

// ReverseHttpRequest 将请求喂入指定项目的路由树（项目不存在时自动创建）。
func (m *ProjectManager) ReverseHttpRequest(projectID string, req *request.HttpRequest) error {
	if req == nil {
		return fmt.Errorf("请求不能为nil")
	}
	if m == nil {
		return fmt.Errorf("ProjectManager不能为nil")
	}
	rs, err := m.Project(projectID)
	if err != nil {
		return err
	}
	return rs.ReverseHttpRequest(req)
}

// ReverseCurls 解析并分发一批 curl 请求到指定项目。
func (m *ProjectManager) ReverseCurls(projectID string, curls []string) BatchResult {
	result := BatchResult{}
	rs, err := m.Project(projectID)
	if err != nil {
		result.Failed = len(curls)
		appendBatchError(&result, 0, "", err)
		return result
	}
	for i, raw := range curls {
		req, err := request.ParseCurl(raw)
		if err != nil {
			result.Failed++
			appendBatchError(&result, i, raw, err)
			continue
		}
		if err = rs.ReverseHttpRequest(req); err != nil {
			result.Failed++
			appendBatchError(&result, i, raw, err)
			continue
		}
		result.Processed++
	}
	return result
}

// NormalizeURL 在指定项目内归一化请求。
// 项目不存在返回 false 且不建项目（读操作无副作用）。
// 需要失败原因时用 NormalizeURLDetailed。
func (m *ProjectManager) NormalizeURL(projectID string, req *request.HttpRequest) (NormalizedRoute, bool) {
	route, reason := m.NormalizeURLDetailed(projectID, req)
	return route, reason == NormalizeOK
}

// ProjectStats 返回指定项目的各 host 统计快照；项目不存在返回空 map。
func (m *ProjectManager) ProjectStats(projectID string) map[string]StatsSnapshot {
	if m == nil {
		return map[string]StatsSnapshot{}
	}
	m.mu.RLock()
	rs := m.projects[normalizeProjectID(projectID)]
	m.mu.RUnlock()
	if rs == nil {
		return map[string]StatsSnapshot{}
	}
	return rs.Stats()
}

// Stats 返回全项目的统计：项目 ID → host → 快照。
func (m *ProjectManager) Stats() map[string]map[string]StatsSnapshot {
	result := make(map[string]map[string]StatsSnapshot)
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
		result[id] = rs.Stats()
	}
	return result
}

// Health 返回全项目的健康报告：项目 ID → host → 健康报告。
func (m *ProjectManager) Health() map[string]map[string]HealthReport {
	result := make(map[string]map[string]HealthReport)
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
		result[id] = rs.Health()
	}
	return result
}
