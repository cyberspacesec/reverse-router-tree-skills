package router

import (
	"encoding/json"
	"fmt"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/tree"
)

// snapshotVersion 整库快照格式版本。只增不改语义；读到更高版本直接报错。
const snapshotVersion = 1

// routerSetSnapshot RouterSet 的可续喂快照：每个 host 一棵带版本信封的路由树。
// 合并阈值、脱敏名单等运行配置不进快照，由导入方当前配置接管。
type routerSetSnapshot struct {
	Version int                               `json:"version"`
	Hosts   map[string]*tree.TreeJSONEnvelope `json:"hosts"`
}

// projectSnapshot ProjectManager 的可续喂快照：项目 → host → 路由树。
type projectSnapshot struct {
	Version  int                                          `json:"version"`
	Projects map[string]map[string]*tree.TreeJSONEnvelope `json:"projects"`
}

// ExportJSON 导出全部 host 路由树。空集合导出空 hosts，而不是 nil。
func (s *RouterSet) ExportJSON() ([]byte, error) {
	hosts := map[string]*tree.TreeJSONEnvelope{}
	if s != nil {
		hosts = s.exportHosts()
	}
	snap := routerSetSnapshot{Version: snapshotVersion, Hosts: hosts}
	return json.MarshalIndent(snap, "", "  ")
}

// exportHosts 取出每个有树的 host。无锁快照后拷贝，避免序列化时持有集合锁。
func (s *RouterSet) exportHosts() map[string]*tree.TreeJSONEnvelope {
	out := map[string]*tree.TreeJSONEnvelope{}
	s.mu.RLock()
	hosts := make([]string, 0, len(s.routers))
	routers := make(map[string]*ReverseRouter, len(s.routers))
	for h, r := range s.routers {
		hosts = append(hosts, h)
		routers[h] = r
	}
	s.mu.RUnlock()
	for _, h := range hosts {
		r := routers[h]
		if r == nil || r.Tree == nil {
			continue
		}
		out[h] = &tree.TreeJSONEnvelope{Version: tree.TreeJSONVersion, Tree: r.Tree.ExportRoot()}
	}
	return out
}

// ImportJSON 用快照替换当前全部 host 桶。
// 新建的路由器继承集合当前的合并/护栏/脱敏/日志配置。
// 未知更高版本报错且不改动现有数据。
func (s *RouterSet) ImportJSON(data []byte) error {
	if s == nil {
		return fmt.Errorf("RouterSet不能为nil")
	}
	var snap routerSetSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return fmt.Errorf("快照解析失败: %w", err)
	}
	if snap.Version > snapshotVersion {
		return fmt.Errorf("不支持的快照版本 %d（当前最高 v%d）", snap.Version, snapshotVersion)
	}
	routers, err := s.routersFromHosts(snap.Hosts)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.routers = routers
	s.mu.Unlock()
	return nil
}

// routersFromHosts 按当前配置把 host→树信封建成路由器。版本过高返回错误，不改动现有数据。
func (s *RouterSet) routersFromHosts(hosts map[string]*tree.TreeJSONEnvelope) (map[string]*ReverseRouter, error) {
	routers := make(map[string]*ReverseRouter, len(hosts))
	for host, env := range hosts {
		if env == nil || env.Tree == nil {
			continue
		}
		if env.Version > tree.TreeJSONVersion {
			return nil, fmt.Errorf("host %s 的路由树版本 %d 不受支持", host, env.Version)
		}
		t := tree.NewTree()
		t.ImportRoot(env.Tree)
		r := s.newConfiguredRouter()
		r.Tree = t
		routers[host] = r
	}
	return routers, nil
}

// newConfiguredRouter 按集合当前配置新建一台路由器（不登记进表）。
func (s *RouterSet) newConfiguredRouter() *ReverseRouter {
	r := NewReverseRouter()
	s.mu.RLock()
	cfg, limits, redact := s.config, s.limits, s.redact
	rule, mergeRule := s.rule, s.mergeRule
	logger, logged, level := s.logger, s.loggerConfigured, s.logLevel
	s.mu.RUnlock()
	r.SetMergeConfig(cfg)
	r.SetResourceLimits(limits)
	r.SetRedactConfig(redact)
	if rule != nil {
		r.SetInferenceRule(rule)
	}
	if mergeRule != nil {
		r.SetMergeRule(mergeRule)
	}
	if logged {
		r.SetLogger(logger)
	}
	if level != nil {
		r.SetLogLevel(*level)
	}
	return r
}

// ExportJSON 导出全部项目的路由树。
func (m *ProjectManager) ExportJSON() ([]byte, error) {
	snap := projectSnapshot{Version: snapshotVersion, Projects: map[string]map[string]*tree.TreeJSONEnvelope{}}
	if m == nil {
		return json.MarshalIndent(snap, "", "  ")
	}
	m.mu.RLock()
	projects := make(map[string]*RouterSet, len(m.projects))
	for id, rs := range m.projects {
		projects[id] = rs
	}
	m.mu.RUnlock()
	for id, rs := range projects {
		snap.Projects[id] = rs.exportHosts()
	}
	return json.MarshalIndent(snap, "", "  ")
}

// ImportJSON 用快照替换当前全部项目。
// 每个项目继承管理器当前配置。未知更高版本报错且不改动现有数据。
func (m *ProjectManager) ImportJSON(data []byte) error {
	if m == nil {
		return fmt.Errorf("ProjectManager不能为nil")
	}
	var snap projectSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return fmt.Errorf("快照解析失败: %w", err)
	}
	if snap.Version > snapshotVersion {
		return fmt.Errorf("不支持的快照版本 %d（当前最高 v%d）", snap.Version, snapshotVersion)
	}
	projects := make(map[string]*RouterSet, len(snap.Projects))
	for id, hosts := range snap.Projects {
		rs := m.newConfiguredSet()
		routers, err := rs.routersFromHosts(hosts)
		if err != nil {
			return fmt.Errorf("导入项目 %s 失败: %w", id, err)
		}
		rs.mu.Lock()
		rs.routers = routers
		rs.mu.Unlock()
		projects[id] = rs
	}
	m.mu.Lock()
	m.projects = projects
	m.mu.Unlock()
	return nil
}

// newConfiguredSet 按管理器当前配置新建一个 RouterSet（不登记进表）。
func (m *ProjectManager) newConfiguredSet() *RouterSet {
	m.mu.RLock()
	defer m.mu.RUnlock()
	rs := NewRouterSet()
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
	return rs
}
