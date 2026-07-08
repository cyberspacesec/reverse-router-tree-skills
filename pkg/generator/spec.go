// Package generator 提供"随机数据生成 + 端到端能力验证"的数据模型。
//
// 核心设计：Spec 是一个"虚拟 API 契约"，它同时承载两个用途：
//   - Requests(): 按 spec 派生 HTTP 请求序列，喂给 ReverseRouter 还原成路由树；
//   - Assertions(): 按 spec 派生对还原树的期望断言（"已知答案"）。
//
// 期望在 spec 生成时就由 deriveExpectations 按 router 的合并/推断规则确定性计算并冻结，
// 而非事后观察。固定 seed 保证可复现。
package generator

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

// Spec 一组虚拟 API 的完整描述。生成后即冻结为"已知答案"。
type Spec struct {
	Seed      int64       // 生成本 spec 用的 seed，失败时可复现
	Resources []*Resource // 一到多个资源
}

// Resource 一个 RESTful 资源（如 users/orders/products）
type Resource struct {
	Name       string       // 资源名，如 "users"。用于路径前缀和变量名前缀
	Prefix     []string     // 路径前缀段，如 ["api"] 或 ["api","v1"]
	Operations []*Operation // CRUD 操作集合
}

// OpKind CRUD 操作类型
type OpKind uint8

const (
	OpList   OpKind = iota // GET /resource            列表
	OpGetOne               // GET /resource/{id}       详情
	OpCreate               // POST /resource           创建
	OpUpdate               // PUT /resource/{id}       更新
	OpDelete               // DELETE /resource/{id}    删除
)

// String 返回 OpKind 的人类可读名称
func (k OpKind) String() string {
	switch k {
	case OpList:
		return "List"
	case OpGetOne:
		return "GetOne"
	case OpCreate:
		return "Create"
	case OpUpdate:
		return "Update"
	case OpDelete:
		return "Delete"
	default:
		return "Unknown"
	}
}

// Operation 一个 CRUD 操作
type Operation struct {
	Method      string          // GET/POST/PUT/PATCH/DELETE
	Kind        OpKind          // 决定路径形态与是否带 body
	PathVar     *PathVarSpec    // 仅 GetOne/Update/Delete 有；nil 表示无路径变量（如 List）
	QueryParams []*QueryParamSpec
	Body        *BodySpec       // 仅 Create/Update 有
	Headers     []*HeaderSpec   // 路由 Header 维度
	Cookies     []*CookieSpec   // Cookie 维度
	Repeat      int             // 该方法被请求的次数（必需参数推断需样本量 >= 2）
}

// PathVarSpec 路径变量维度——同时支撑"造值"和"断言合并结果"
type PathVarSpec struct {
	Pattern string   // integer/uuid/phone/idcard/bankcard/plate/prefix/suffix/similar_length/fixed_words/mixed_int_fixed
	Values  []string // 用于派生请求的具体值；数量决定能否触发合并（>=3 才合并, similar_length 需 >=6）

	// Expect 字段：spec 生成时由 deriveExpectations 按 router 规则确定性填充的"已知答案"
	ExpectMerge      bool               // 期望是否合并为 request_path_variable
	ExpectVarName    string             // 期望变量名（如 "users_id"）
	ExpectPhysical   value.PhysicalType // 期望物理类型
	ExpectLogical    value.LogicalType  // 期望逻辑类型
	ExpectPatternSet bool               // 期望 GetPattern() != nil
	ExpectFixedKept  []string           // 选择性合并时保留的固定路径（如 list/create）
}

// QueryParamSpec 查询参数维度
type QueryParamSpec struct {
	Name           string  // 参数名
	Values         []string // 派生请求时轮换使用的值
	Presence       float64 // 0.0-1.0，派生请求时按概率出现
	ExpectType     value.PhysicalType // 期望物理类型（基于 Values 推断）
	ExpectLogic    value.LogicalType  // 期望逻辑类型
	ExpectRequired bool               // Presence>=0.9 && Repeat>=2 → true
}

// BodySpec 请求体维度
type BodySpec struct {
	ContentType string           // application/json 或 application/x-www-form-urlencoded
	Fields      []*BodyFieldSpec // 扁平字段
}

// BodyFieldSpec 请求体字段
type BodyFieldSpec struct {
	Name   string   // 字段名
	Values []string // 字段值轮换

	// Expect 字段：spec 生成时按值推断的"已知答案"
	ExpectPhysical value.PhysicalType // 按值推断：纯数字→integer，否则 string
	ExpectRequired bool               // Repeat 内每次都带该字段→true
}

// HeaderSpec 路由 Header 维度（Name 必须在 router 路由 Header 白名单内）
type HeaderSpec struct {
	Name             string   // Accept/Authorization/X-Api-Version/Accept-Language/X-Requested-With
	Values           []string // 原始值
	ExpectNormValues []string // 规范化后期望的值节点 key（由 deriveHeaderNorm 推导）
}

// CookieSpec Cookie 维度
type CookieSpec struct {
	Name   string   // Cookie 名称
	Values []string // 每个值对应一个 request_cookie_value 子节点
}

// String 返回人类可读的 spec dump，失败时调试用。
func (s *Spec) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Spec seed=%d resources=%d\n", s.Seed, len(s.Resources))
	for i, res := range s.Resources {
		fmt.Fprintf(&b, "  [Resource %d] %s prefix=%v ops=%d\n", i, res.Name, res.Prefix, len(res.Operations))
		for j, op := range res.Operations {
			fmt.Fprintf(&b, "    [Op %d] %s(%s) repeat=%d", j, op.Method, op.Kind, op.Repeat)
			if op.PathVar != nil {
				fmt.Fprintf(&b, " pathvar{pattern=%s n=%d merge=%v name=%s phys=%s logic=%s}",
					op.PathVar.Pattern, len(op.PathVar.Values), op.PathVar.ExpectMerge,
					op.PathVar.ExpectVarName, op.PathVar.ExpectPhysical, op.PathVar.ExpectLogical)
			}
			fmt.Fprintf(&b, " query=%d body=%v headers=%d cookies=%d\n",
				len(op.QueryParams), op.Body != nil, len(op.Headers), len(op.Cookies))
			for _, q := range op.QueryParams {
				fmt.Fprintf(&b, "      query %s presence=%.2f required=%v phys=%s logic=%s\n",
					q.Name, q.Presence, q.ExpectRequired, q.ExpectType, q.ExpectLogic)
			}
			if op.Body != nil {
				fmt.Fprintf(&b, "      body ct=%s fields=%d\n", op.Body.ContentType, len(op.Body.Fields))
			}
			for _, h := range op.Headers {
				fmt.Fprintf(&b, "      header %s norms=%v\n", h.Name, h.ExpectNormValues)
			}
			for _, c := range op.Cookies {
				fmt.Fprintf(&b, "      cookie %s vals=%v\n", c.Name, c.Values)
			}
		}
	}
	return b.String()
}

// Requests 按 spec 派生 HTTP 请求序列（实现见 derive.go）。
func (s *Spec) Requests(rnd *rand.Rand) []*request.HttpRequest {
	return specRequests(s, rnd)
}

// Assertions 按 spec 派生对还原树的期望断言（实现见 assertion.go）。
func (s *Spec) Assertions() []Assertion {
	return specAssertions(s)
}
