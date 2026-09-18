package router

import (
	"strings"
	"sync/atomic"
)

// ResourceLimits 生产护栏：限制单棵路由树的资源增长，防止不可信流量撑爆内存。
//
// 三项上限均为 0 表示不限制。默认值保守（DefaultResourceLimits），
// 既覆盖真实测绘流量，也远高于现有极端测试的用量（500 参数 / 500 层深 / 200KB 单段）。
type ResourceLimits struct {
	// MaxChildrenPerNode 单个父节点的最大子节点数，超限时拒绝新建子节点（fail-soft 返回错误）。
	MaxChildrenPerNode int
	// MaxValuesPerMetric 单个 ValueMetric 的最大不同值（unique）数，超限后新值仅计数不存原值。
	MaxValuesPerMetric int
	// MaxSegmentLen 单个路径段的最大字节数，超限截断后继续处理。
	MaxSegmentLen int
}

// DefaultResourceLimits 默认资源上限。
var DefaultResourceLimits = ResourceLimits{
	MaxChildrenPerNode: 10000,
	MaxValuesPerMetric: 10000,
	MaxSegmentLen:      262144, // 256KB：覆盖 G05 200KB 单段用例
}

// RedactConfig 敏感数据脱敏配置：命中名单的参数/cookie 只保留结构（节点名），不存原值。
type RedactConfig struct {
	// Params 参数名名单（大小写不敏感），如 password/token
	Params []string
	// Cookies cookie 名名单（大小写不敏感），如 sessionid
	Cookies []string
}

// DefaultRedactConfig 默认脱敏名单：只覆盖高确定性的密码凭据类字段，默认开启。
// 注意：session/token 等名太常见（现有 Cookie 路由测试依赖 session 值节点），
// 不进默认名单，由调用方按需追加。
func DefaultRedactConfig() RedactConfig {
	return RedactConfig{
		Params:  []string{"password", "passwd", "pwd"},
		Cookies: []string{"sessionid"},
	}
}

// ResourceLimitsHolder 用 atomic.Pointer 持有 ResourceLimits，供无锁快路径读取。
type ResourceLimitsHolder struct {
	p atomic.Pointer[ResourceLimits]
}

// StoreLimits 原子写入一份上限拷贝。
func (h *ResourceLimitsHolder) StoreLimits(l ResourceLimits) {
	cp := l
	h.p.Store(&cp)
}

// LoadLimits 原子读取当前上限；从未设置时返回 DefaultResourceLimits。
func (h *ResourceLimitsHolder) LoadLimits() ResourceLimits {
	if v := h.p.Load(); v != nil {
		return *v
	}
	return DefaultResourceLimits
}

// redactedSets 脱敏名单的不可变快照（小写集合）。
type redactedSets struct {
	params  map[string]bool
	cookies map[string]bool
}

// RedactHolder 用 atomic.Pointer 持有脱敏名单，供无锁快路径读取。
type RedactHolder struct {
	p atomic.Pointer[redactedSets]
}

// StoreRedact 原子写入名单快照。
func (h *RedactHolder) StoreRedact(cfg RedactConfig) {
	h.p.Store(&redactedSets{
		params:  buildRedactSet(cfg.Params),
		cookies: buildRedactSet(cfg.Cookies),
	})
}

// loadSets 读取名单快照；从未设置时返回空名单。
func (h *RedactHolder) loadSets() *redactedSets {
	if v := h.p.Load(); v != nil {
		return v
	}
	return &redactedSets{}
}

// IsParamRedacted 判断参数名是否命中脱敏名单（大小写不敏感，无锁）。
func (h *RedactHolder) IsParamRedacted(name string) bool {
	return h.loadSets().params[strings.ToLower(name)]
}

// IsCookieRedacted 判断 cookie 名是否命中脱敏名单（大小写不敏感，无锁）。
func (h *RedactHolder) IsCookieRedacted(name string) bool {
	return h.loadSets().cookies[strings.ToLower(name)]
}

// CappedMetricKey ValueMetric 超限后的占位键（存计数、不存原值）。
const CappedMetricKey = "[capped]"

// RedactedCookieValue 脱敏 cookie 在树中的占位值：保留结构，不存原值。
const RedactedCookieValue = "[redacted]"

// DefaultMaxHosts RouterSet 默认最大 host 桶数，0 表示不限制。
// 覆盖现有测试用量（≤4），远高于单次测绘常规目标数，防 host 爆炸 OOM。
const DefaultMaxHosts = 10000

// DefaultMaxProjects ProjectManager 默认最大项目数，0 表示不限制。
// 单个安全测试项目对应一个 RouterSet；多项目并发时防项目爆炸 OOM。
const DefaultMaxProjects = 1000

// buildRedactSet 名单转小写集合。
func buildRedactSet(names []string) map[string]bool {
	set := make(map[string]bool, len(names))
	for _, n := range names {
		if n = strings.ToLower(strings.TrimSpace(n)); n != "" {
			set[n] = true
		}
	}
	return set
}
