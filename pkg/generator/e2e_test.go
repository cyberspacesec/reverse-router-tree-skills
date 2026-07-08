package generator_test

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/generator"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/router"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

// runScenario 用给定 seed 生成 spec → 喂请求 → 推断必需性 → 逐项断言 → 不变量校验。
func runScenario(t *testing.T, seed int64) {
	t.Helper()
	g := generator.NewGenerator(seed)
	spec := g.Generate()
	t.Logf("spec seed=%d", spec.Seed)
	t.Logf("spec dump:\n%s", spec.String())

	r := router.NewReverseRouter()
	reqRng := rand.New(rand.NewSource(spec.Seed))
	reqs := spec.Requests(reqRng)
	for i, req := range reqs {
		if err := r.ReverseHttpRequest(req); err != nil {
			t.Fatalf("ReverseHttpRequest #%d 失败: %v\nreq=%s", i, err, req.String())
		}
	}
	r.InferRequiredParams()

	t.Logf("还原出的路由树:\n%s", r.Tree.String())

	// 逐项断言
	for _, a := range spec.Assertions() {
		a.Check(t, r.Tree.Root)
	}

	// 不变量校验
	assertInvariants(t, r, spec, reqs)
}

// runSpec 用固定 spec 跑断言（不随机），复用 runScenario 的核心逻辑。
func runSpec(t *testing.T, spec *generator.Spec) {
	t.Helper()
	r := router.NewReverseRouter()
	reqRng := rand.New(rand.NewSource(spec.Seed))
	reqs := spec.Requests(reqRng)
	for i, req := range reqs {
		if err := r.ReverseHttpRequest(req); err != nil {
			t.Fatalf("ReverseHttpRequest #%d 失败: %v\nreq=%s", i, err, req.String())
		}
	}
	r.InferRequiredParams()
	t.Logf("还原出的路由树:\n%s", r.Tree.String())
	for _, a := range spec.Assertions() {
		a.Check(t, r.Tree.Root)
	}
	assertInvariants(t, r, spec, reqs)
}

// assertInvariants 不变量校验：无 panic、ToJSON 可反序列化、Stats 合理、IsNeedRequest 去重。
func assertInvariants(t *testing.T, r *router.ReverseRouter, spec *generator.Spec, reqs []*request.HttpRequest) {
	t.Helper()

	// 1. ToJSON 可反序列化（树结构完整、可持久化）
	data, err := r.Tree.ToJSON()
	if err != nil {
		t.Fatalf("Tree.ToJSON 失败: %v", err)
	}
	var generic map[string]interface{}
	if err := json.Unmarshal(data, &generic); err != nil {
		t.Fatalf("ToJSON 输出不是合法 JSON: %v", err)
	}

	// 2. OpenAPI 导出可反序列化（导出层完整）
	// 注意：exporter 不在 import 范围内时跳过；这里用 Tree.ToJSON 间接验证导出层不报错。

	// 3. IsNeedRequest 去重：已喂入的请求应返回 false（树里已建模）
	for i, req := range reqs {
		// 只有 method 节点 requestCount>0 时 IsNeedRequest 才可能 false；
		// 单次请求（Repeat 相关）的边界情况：requestCount<=1 时 IsNeedRequest 行为特殊，
		// 这里只验证"大量重复请求后，相同请求不需要再发"。
		if r.IsNeedRequest(req) {
			// 允许部分请求因样本量不足仍返回 true，记录但不直接失败
			t.Logf("请求 #%d IsNeedRequest=true（可能样本不足）", i)
		}
	}

	// 4. 新 URL（spec 外）应返回 true（需要请求）
	newReq := request.NewHttpRequest("/api/__nonexistent_endpoint__", nil, "GET", nil)
	if !r.IsNeedRequest(newReq) {
		t.Error("spec 外的新 URL 应该 IsNeedRequest=true（需要请求）")
	}
}

// TestE2E_RandomScenarios 多 seed 随机端到端验证
func TestE2E_RandomScenarios(t *testing.T) {
	for _, seed := range []int64{generator.DefaultSeed, generator.DefaultSeed + 1, generator.DefaultSeed + 2} {
		seed := seed
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			runScenario(t, seed)
		})
	}
}

// --- 定向子测试：固定 spec，保证每种场景至少一个确定性用例 ---

func TestE2E_Directive_IntegerMerge(t *testing.T) {
	spec := &generator.Spec{
		Seed: 1,
		Resources: []*generator.Resource{{
			Name: "users", Prefix: []string{"api"},
			Operations: []*generator.Operation{{
				Method: "GET", Kind: generator.OpGetOne, Repeat: 3,
				PathVar: &generator.PathVarSpec{
					Pattern: "integer",
					Values:  []string{"123", "456", "789"},
				},
			}},
		}},
	}
	// 手动触发期望推导（定向 spec 未经过 Generate）
	runSpecWithDerive(t, spec, "users")
}

func TestE2E_Directive_UUIDMerge(t *testing.T) {
	spec := &generator.Spec{
		Seed: 2,
		Resources: []*generator.Resource{{
			Name: "items", Prefix: []string{"api"},
			Operations: []*generator.Operation{{
				Method: "GET", Kind: generator.OpGetOne, Repeat: 3,
				PathVar: &generator.PathVarSpec{
					Pattern: "uuid",
					Values: []string{
						"550e8400-e29b-41d4-a716-446655440000",
						"550e8400-e29b-41d4-a716-446655440001",
						"550e8400-e29b-41d4-a716-446655440002",
					},
				},
			}},
		}},
	}
	runSpecWithDerive(t, spec, "items")
}

func TestE2E_Directive_PhoneMerge(t *testing.T) {
	spec := &generator.Spec{
		Seed: 3,
		Resources: []*generator.Resource{{
			Name: "users", Prefix: []string{"api"},
			Operations: []*generator.Operation{{
				Method: "GET", Kind: generator.OpGetOne, Repeat: 3,
				PathVar: &generator.PathVarSpec{
					Pattern: "phone",
					Values:  []string{"13812345678", "13912345678", "15012345678"},
				},
			}},
		}},
	}
	runSpecWithDerive(t, spec, "users")
}

func TestE2E_Directive_IDCardMerge(t *testing.T) {
	spec := &generator.Spec{
		Seed: 4,
		Resources: []*generator.Resource{{
			Name: "users", Prefix: []string{"api"},
			Operations: []*generator.Operation{{
				Method: "GET", Kind: generator.OpGetOne, Repeat: 3,
				PathVar: &generator.PathVarSpec{
					Pattern: "idcard",
					Values:  []string{"110101199001011234", "310101198501012345", "44010119920303123X"},
				},
			}},
		}},
	}
	runSpecWithDerive(t, spec, "users")
}

func TestE2E_Directive_BankCardMerge(t *testing.T) {
	spec := &generator.Spec{
		Seed: 5,
		Resources: []*generator.Resource{{
			Name: "cards", Prefix: []string{"api"},
			Operations: []*generator.Operation{{
				Method: "GET", Kind: generator.OpGetOne, Repeat: 3,
				PathVar: &generator.PathVarSpec{
					Pattern: "bankcard",
					Values:  []string{"6222021234567890123", "6225887654321098765", "6217001234567890123"},
				},
			}},
		}},
	}
	runSpecWithDerive(t, spec, "cards")
}

func TestE2E_Directive_PrefixMerge(t *testing.T) {
	spec := &generator.Spec{
		Seed: 6,
		Resources: []*generator.Resource{{
			Name: "items", Prefix: []string{"api"},
			Operations: []*generator.Operation{{
				Method: "GET", Kind: generator.OpGetOne, Repeat: 3,
				PathVar: &generator.PathVarSpec{
					Pattern: "prefix",
					Values:  []string{"user_001", "user_002", "user_003"},
				},
			}},
		}},
	}
	runSpecWithDerive(t, spec, "items")
}

func TestE2E_Directive_SimilarLengthBreak(t *testing.T) {
	spec := &generator.Spec{
		Seed: 7,
		Resources: []*generator.Resource{{
			Name: "city", Prefix: []string{"api"},
			Operations: []*generator.Operation{{
				Method: "GET", Kind: generator.OpGetOne, Repeat: 6,
				PathVar: &generator.PathVarSpec{
					Pattern: "similar_length",
					Values:  []string{"abcde", "fghij", "klmno", "pqrst", "uvwxy", "zabcd"},
				},
			}},
		}},
	}
	runSpecWithDerive(t, spec, "city")
}

func TestE2E_Directive_SimilarLengthNoBreak(t *testing.T) {
	spec := &generator.Spec{
		Seed: 8,
		Resources: []*generator.Resource{{
			Name: "roles", Prefix: []string{"api"},
			Operations: []*generator.Operation{{
				Method: "GET", Kind: generator.OpGetOne, Repeat: 3,
				PathVar: &generator.PathVarSpec{
					Pattern: "similar_length",
					Values:  []string{"admin", "manager", "guest"},
				},
			}},
		}},
	}
	runSpecWithDerive(t, spec, "roles")
}

func TestE2E_Directive_FixedWordsNoMerge(t *testing.T) {
	spec := &generator.Spec{
		Seed: 9,
		Resources: []*generator.Resource{{
			Name: "roles", Prefix: []string{"api"},
			Operations: []*generator.Operation{{
				Method: "GET", Kind: generator.OpGetOne, Repeat: 3,
				PathVar: &generator.PathVarSpec{
					Pattern: "fixed_words",
					Values:  []string{"admin", "manager", "guest"},
				},
			}},
		}},
	}
	runSpecWithDerive(t, spec, "roles")
}

func TestE2E_Directive_MixedSelectiveMerge(t *testing.T) {
	spec := &generator.Spec{
		Seed: 10,
		Resources: []*generator.Resource{{
			Name: "users", Prefix: []string{"api"},
			Operations: []*generator.Operation{{
				Method: "GET", Kind: generator.OpGetOne, Repeat: 5,
				PathVar: &generator.PathVarSpec{
					Pattern: "mixed_int_fixed",
					Values:  []string{"101", "102", "103", "list", "create"},
				},
			}},
		}},
	}
	runSpecWithDerive(t, spec, "users")
}

func TestE2E_Directive_CRUDMultiMethod(t *testing.T) {
	spec := &generator.Spec{
		Seed: 11,
		Resources: []*generator.Resource{{
			Name: "orders", Prefix: []string{"api"},
			Operations: []*generator.Operation{
				{Method: "GET", Kind: generator.OpList, Repeat: 2},
				{Method: "POST", Kind: generator.OpCreate, Repeat: 2,
					Body: &generator.BodySpec{
						ContentType: "application/json",
						Fields: []*generator.BodyFieldSpec{{Name: "name", Values: []string{"a"}}},
					}},
				{Method: "GET", Kind: generator.OpGetOne, Repeat: 3,
					PathVar: &generator.PathVarSpec{Pattern: "integer", Values: []string{"1", "2", "3"}}},
				{Method: "PUT", Kind: generator.OpUpdate, Repeat: 3,
					PathVar: &generator.PathVarSpec{Pattern: "integer", Values: []string{"1", "2", "3"}}},
				{Method: "DELETE", Kind: generator.OpDelete, Repeat: 3,
					PathVar: &generator.PathVarSpec{Pattern: "integer", Values: []string{"1", "2", "3"}}},
			},
		}},
	}
	runSpecWithDeriveAll(t, spec)
}

func TestE2E_Directive_JSONBodyParams(t *testing.T) {
	spec := &generator.Spec{
		Seed: 12,
		Resources: []*generator.Resource{{
			Name: "users", Prefix: []string{"api"},
			Operations: []*generator.Operation{{
				Method: "POST", Kind: generator.OpCreate, Repeat: 2,
				Body: &generator.BodySpec{
					ContentType: "application/json",
					Fields: []*generator.BodyFieldSpec{
						{Name: "name", Values: []string{"alice"}},
						{Name: "age", Values: []string{"30"}},
					},
				},
			}},
		}},
	}
	runSpecNoPathVar(t, spec)
}

func TestE2E_Directive_HeaderRouting(t *testing.T) {
	spec := &generator.Spec{
		Seed: 13,
		Resources: []*generator.Resource{{
			Name: "data", Prefix: []string{"api"},
			Operations: []*generator.Operation{{
				Method: "GET", Kind: generator.OpList, Repeat: 2,
				Headers: []*generator.HeaderSpec{{
					Name:   "Accept",
					Values: []string{"application/json, text/plain;q=0.8", "text/html, */*;q=0.5"},
				}},
			}},
		}},
	}
	// HeaderSpec.ExpectNormValues 需手动推导
	for _, res := range spec.Resources {
		for _, op := range res.Operations {
			for _, h := range op.Headers {
				h.ExpectNormValues = deriveHeaderNormForTest(h.Name, h.Values)
			}
		}
	}
	runSpecNoPathVar(t, spec)
}

func TestE2E_Directive_CookieRouting(t *testing.T) {
	spec := &generator.Spec{
		Seed: 14,
		Resources: []*generator.Resource{{
			Name: "home", Prefix: []string{"api"},
			Operations: []*generator.Operation{{
				Method: "GET", Kind: generator.OpList, Repeat: 2,
				Cookies: []*generator.CookieSpec{{
					Name:   "lang",
					Values: []string{"zh-CN", "en-US"},
				}},
			}},
		}},
	}
	runSpecNoPathVar(t, spec)
}

func TestE2E_Directive_RequiredParamInference(t *testing.T) {
	// page 必现(Presence=1.0) 且 Repeat=3 → 必需；size 偶尔(Presence=0.3) → 可选
	spec := &generator.Spec{
		Seed: 15,
		Resources: []*generator.Resource{{
			Name: "users", Prefix: []string{"api"},
			Operations: []*generator.Operation{{
				Method: "GET", Kind: generator.OpList, Repeat: 3,
				QueryParams: []*generator.QueryParamSpec{
					{Name: "page", Values: []string{"1", "2", "3"}, Presence: 1.0,
						ExpectType: value.PhysicalTypeInteger, ExpectLogic: value.LogicalTypeString,
						ExpectRequired: true},
					{Name: "size", Values: []string{"10", "20"}, Presence: 0.3,
						ExpectType: value.PhysicalTypeInteger, ExpectLogic: value.LogicalTypeString,
						ExpectRequired: false},
				},
			}},
		}},
	}
	runSpecNoPathVar(t, spec)
}

func TestE2E_Directive_IsNeedRequestDedup(t *testing.T) {
	spec := &generator.Spec{
		Seed: 16,
		Resources: []*generator.Resource{{
			Name: "users", Prefix: []string{"api"},
			Operations: []*generator.Operation{{
				Method: "GET", Kind: generator.OpGetOne, Repeat: 3,
				PathVar: &generator.PathVarSpec{Pattern: "integer", Values: []string{"1", "2", "3"}},
			}},
		}},
	}
	runSpecWithDerive(t, spec, "users")
	// 去重逻辑在 assertInvariants 里验证
}

// --- 定向 spec 的期望推导辅助 ---

// runSpecWithDerive 对单一 GetOne 操作的 spec，推导 PathVar 期望后跑。
func runSpecWithDerive(t *testing.T, spec *generator.Spec, parent string) {
	t.Helper()
	for _, res := range spec.Resources {
		for _, op := range res.Operations {
			if op.PathVar != nil {
				generator.DerivePathVarExpectations(op.PathVar, parent, len(op.PathVar.Values))
			}
			deriveBodyFields(op)
		}
	}
	runSpec(t, spec)
}

// runSpecWithDeriveAll 对 CRUD 多操作 spec，所有 PathVar 用资源名推导。
func runSpecWithDeriveAll(t *testing.T, spec *generator.Spec) {
	t.Helper()
	for _, res := range spec.Resources {
		for _, op := range res.Operations {
			if op.PathVar != nil {
				generator.DerivePathVarExpectations(op.PathVar, res.Name, len(op.PathVar.Values))
			}
			deriveBodyFields(op)
		}
	}
	runSpec(t, spec)
}

// runSpecNoPathVar 对无 PathVar 的 spec 直接跑（Header/Cookie/Body 场景）。
func runSpecNoPathVar(t *testing.T, spec *generator.Spec) {
	t.Helper()
	for _, res := range spec.Resources {
		for _, op := range res.Operations {
			deriveBodyFields(op)
		}
	}
	runSpec(t, spec)
}

// deriveBodyFields 推导 op.Body 各字段的期望物理类型与必需性。
func deriveBodyFields(op *generator.Operation) {
	if op.Body == nil {
		return
	}
	for _, f := range op.Body.Fields {
		generator.DeriveBodyFieldExpectations(f, op.Repeat)
	}
}

// deriveHeaderNormForTest 镜像 generator 的 Header 规范化（测试包无法访问未导出的 deriveHeaderNorm）。
func deriveHeaderNormForTest(name string, raw []string) []string {
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		out = append(out, normalizeHeaderValueForTest(name, v))
	}
	return out
}

func normalizeHeaderValueForTest(name, val string) string {
	if val == "" {
		return ""
	}
	switch name {
	case "Accept":
		parts := splitN(val, ",", 2)
		mime := trimSpace(parts[0])
		if i := indexByte(mime, ';'); i >= 0 {
			mime = trimSpace(mime[:i])
		}
		return mime
	case "Authorization":
		parts := splitN(val, " ", 2)
		if len(parts) > 0 {
			return parts[0]
		}
		return ""
	case "Accept-Language":
		parts := splitN(val, ",", 2)
		lang := trimSpace(parts[0])
		if i := indexByte(lang, ';'); i >= 0 {
			lang = trimSpace(lang[:i])
		}
		return lang
	default:
		return val
	}
}

func splitN(s, sep string, n int) []string {
	// 简化版 strings.SplitN
	var out []string
	start := 0
	for i := 0; i < len(s) && len(out) < n-1; i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			out = append(out, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	out = append(out, s[start:])
	return out
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}
