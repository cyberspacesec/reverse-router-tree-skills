package generator

import (
	"fmt"
	"math/rand"
	"strconv"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

// DefaultSeed 默认随机种子，暴露为常量便于复现。
const DefaultSeed int64 = 20260704

// 资源名池
var resourceNames = []string{"users", "orders", "products", "articles", "cards", "vehicles", "resources", "items"}

// CRUD 方法模板
type crudTemplate struct {
	kind   OpKind
	method string
}

var allCRUD = []crudTemplate{
	{OpList, "GET"},
	{OpGetOne, "GET"},
	{OpCreate, "POST"},
	{OpUpdate, "PUT"},
	{OpDelete, "DELETE"},
}

// ID 模式池（用于随机选 PathVar.Pattern）
var idPatternPool = []string{
	patternInteger, patternUUID, patternPhone, patternIDCard, patternBankCard,
	patternPlate, patternPrefix, patternSuffix, patternSimilarLength,
	patternFixedWords, patternMixedIntFixed,
}

// Generator 随机 spec 生成器
type Generator struct {
	rng *rand.Rand
}

// NewGenerator 创建一个固定 seed 的生成器。
func NewGenerator(seed int64) *Generator {
	return &Generator{rng: rand.New(rand.NewSource(seed))}
}

// Generate 随机生成一个 spec（资源数、ID 模式、参数都随机，但生成后确定）。
func (g *Generator) Generate() *Spec {
	spec := &Spec{Seed: g.rng.Int63()}
	n := 1 + g.rng.Intn(3) // 1-3 个资源
	used := make(map[string]bool, n)
	for i := 0; i < n; i++ {
		spec.Resources = append(spec.Resources, g.genResource(used))
	}
	return spec
}

func (g *Generator) genResource(used map[string]bool) *Resource {
	// 资源名必须唯一，否则同路径段下不同资源的值会混合，破坏模式检测
	name := resourceNames[g.rng.Intn(len(resourceNames))]
	for used[name] {
		name = resourceNames[g.rng.Intn(len(resourceNames))]
	}
	used[name] = true

	// 前缀：["api"] 或 ["api","v1"]
	prefix := []string{"api"}
	if g.rng.Intn(2) == 0 {
		prefix = []string{"api", "v1"}
	}

	// CRUD 子集：至少 2 个方法，确保方法节点共存
	templates := pickAtLeast(g.rng, allCRUD, 2)

	// 该资源所有带 id 的操作(GetOne/Update/Delete)共享同一个 PathVar。
	// 否则不同操作用不同 pattern 的值会混在同一路径段，router 的模式检测
	// 看到的是混合值并集，行为不可预测，"已知答案"无法对齐。
	var sharedPathVar *PathVarSpec
	for _, t := range templates {
		if t.kind == OpGetOne || t.kind == OpUpdate || t.kind == OpDelete {
			if sharedPathVar == nil {
				sharedPathVar = g.genPathVar(name)
			}
		}
	}

	res := &Resource{Name: name, Prefix: prefix}
	for _, t := range templates {
		res.Operations = append(res.Operations, g.genOperation(name, t, sharedPathVar))
	}
	return res
}

func (g *Generator) genOperation(resName string, t crudTemplate, sharedPathVar *PathVarSpec) *Operation {
	op := &Operation{
		Method: t.method,
		Kind:   t.kind,
		Repeat: 2 + g.rng.Intn(3), // 2-4 次，保证必需参数推断有样本量
	}

	// 路径变量：仅 GetOne/Update/Delete 有，复用资源级共享 PathVar
	if t.kind == OpGetOne || t.kind == OpUpdate || t.kind == OpDelete {
		if sharedPathVar != nil {
			op.PathVar = sharedPathVar
			// 合并触发取决于实际喂入的不同值数（buildRequest 按 iter%len 轮换）。
			// 必须保证 repeat >= len(Values)，否则部分值不喂入，兄弟数不足阈值，
			// 期望（按 len(Values) 推导）与实际合并行为不符。
			if n := len(sharedPathVar.Values); op.Repeat < n {
				op.Repeat = n
			}
		}
	}

	// 查询参数：List/GetOne 常带
	if t.kind == OpList || t.kind == OpGetOne {
		n := g.rng.Intn(3) // 0-2 个
		for i := 0; i < n; i++ {
			op.QueryParams = append(op.QueryParams, g.genQueryParam(op.Repeat))
		}
	}

	// Body：Create/Update 带
	if t.kind == OpCreate || t.kind == OpUpdate {
		op.Body = g.genBody(op.Repeat)
	}

	// 路由 Header：随机 0-1 个
	if g.rng.Intn(2) == 0 {
		op.Headers = append(op.Headers, g.genHeader())
	}

	// Cookie：随机 0-1 个
	if g.rng.Intn(2) == 0 {
		op.Cookies = append(op.Cookies, g.genCookie())
	}

	return op
}

// genPathVar 随机选一个 ID 模式，生成 Values 并调用 deriveExpectations 冻结期望。
func (g *Generator) genPathVar(parent string) *PathVarSpec {
	pattern := idPatternPool[g.rng.Intn(len(idPatternPool))]
	pv := &PathVarSpec{Pattern: pattern}

	// 生成 Values：相似串场景按是否突破分档；其余 >=3
	switch pattern {
	case patternSimilarLength:
		// 50% 概率突破(>=6)，50% 不突破(3-5)
		if g.rng.Intn(2) == 0 {
			pv.Values = genSimilarLengthValues(g.rng, 6+g.rng.Intn(3)) // 6-8
		} else {
			pv.Values = genSimilarLengthValues(g.rng, 3+g.rng.Intn(3)) // 3-5
		}
	case patternFixedWords:
		pv.Values = []string{"admin", "manager", "guest"}
	case patternMixedIntFixed:
		// 整数子集 + 固定路径
		n := 3 + g.rng.Intn(2) // 3-4 个整数
		vals := make([]string, 0, n+2)
		for i := 0; i < n; i++ {
			vals = append(vals, strconv.Itoa(100+g.rng.Intn(900)))
		}
		vals = append(vals, "list", "create")
		pv.Values = vals
	default:
		n := 3 + g.rng.Intn(3) // 3-5 个
		pv.Values = genValuesForPattern(g.rng, pattern, n)
	}

	// 计算同层兄弟数（mixed 模式含固定路径；其余兄弟数 = Values 数）
	siblingCount := len(pv.Values)
	if pattern == patternMixedIntFixed {
		// 整数子集 + 2 固定路径，全算兄弟
		siblingCount = len(pv.Values)
	}
	deriveExpectations(pv, parent, siblingCount)
	return pv
}

func (g *Generator) genQueryParam(repeat int) *QueryParamSpec {
	name := pickQueryParamName(g.rng)
	// 随机场景的 query 参数一律必现（presence=1.0），保证参数节点确定存在、
	// 必需性可稳定断言。可选参数的概率行为由 TestE2E_Directive_RequiredParamInference
	// 定向覆盖（固定 spec，可控）。
	presence := 1.0
	n := 1 + g.rng.Intn(3)
	vals := make([]string, n)
	for i := 0; i < n; i++ {
		vals[i] = strconv.Itoa(g.rng.Intn(1000))
	}
	q := &QueryParamSpec{
		Name:     name,
		Values:   vals,
		Presence: presence,
	}
	// 期望物理类型：纯数字 → integer
	q.ExpectType = value.PhysicalTypeInteger
	q.ExpectLogic = value.LogicalTypeString
	// 必需性：Presence>=0.9 且 repeat>=2
	q.ExpectRequired = presence >= requiredParamThreshold && repeat >= 2
	return q
}

func (g *Generator) genBody(repeat int) *BodySpec {
	var ct string
	if g.rng.Intn(2) == 0 {
		ct = "application/json"
	} else {
		ct = "application/x-www-form-urlencoded"
	}
	n := 1 + g.rng.Intn(3) // 1-3 个字段
	// 字段名必须唯一：router 解析 body 时同名字段会被合并为单个参数节点，
	// 若 spec 声明 N 个字段但其中重名，router 实际只建 N-k 个参数节点，
	// 导致 StatsAssertion.MinParams（按 len(Fields) 计）与实际不符。
	// 故从池中无放回抽取，保证每个字段名唯一。
	names := pickUniqueBodyFieldNames(g.rng, n)
	b := &BodySpec{ContentType: ct}
	for i := 0; i < n; i++ {
		field := &BodyFieldSpec{
			Name:   names[i],
			Values: []string{fmt.Sprintf("val%d", g.rng.Intn(1000))},
		}
		deriveBodyFieldExpectations(field, repeat)
		b.Fields = append(b.Fields, field)
	}
	return b
}

func (g *Generator) genHeader() *HeaderSpec {
	// 白名单内的 Header 名
	names := []string{"Accept", "Authorization", "X-Api-Version", "Accept-Language", "X-Requested-With"}
	name := names[g.rng.Intn(len(names))]
	vals := genHeaderRawValues(g.rng, name)
	return &HeaderSpec{
		Name:             name,
		Values:           vals,
		ExpectNormValues: deriveHeaderNorm(name, vals),
	}
}

func (g *Generator) genCookie() *CookieSpec {
	names := []string{"lang", "theme", "sessid", "region"}
	name := names[g.rng.Intn(len(names))]
	n := 2 + g.rng.Intn(2) // 2-3 个值
	vals := make([]string, n)
	for i := 0; i < n; i++ {
		vals[i] = fmt.Sprintf("v%d", i)
	}
	return &CookieSpec{Name: name, Values: vals}
}

// pickAtLeast 从 src 随机挑至少 min 个元素（保留原顺序）。
func pickAtLeast(rnd *rand.Rand, src []crudTemplate, min int) []crudTemplate {
	idxs := rnd.Perm(len(src))
	k := min + rnd.Intn(len(src)-min+1) // min..len(src)
	if k > len(src) {
		k = len(src)
	}
	out := make([]crudTemplate, 0, k)
	// 按 idxs 顺序取，但保持模板在原列表中的相对顺序
	picked := make(map[int]bool)
	for i := 0; i < k; i++ {
		picked[idxs[i]] = true
	}
	for i, t := range src {
		if picked[i] {
			out = append(out, t)
		}
	}
	return out
}

var queryParamNames = []string{"page", "size", "limit", "offset", "sort", "q"}
var bodyFieldNames = []string{"name", "age", "title", "desc", "status", "tag"}

func pickQueryParamName(rnd *rand.Rand) string {
	return queryParamNames[rnd.Intn(len(queryParamNames))]
}

// pickUniqueBodyFieldNames 从 bodyFieldNames 池中无放回抽取 n 个互不重复的字段名。
// 用 Fisher-Yates 部分洗牌：打乱池副本后取前 n 个。n 上限为池长度，超出则取全部。
func pickUniqueBodyFieldNames(rnd *rand.Rand, n int) []string {
	pool := make([]string, len(bodyFieldNames))
	copy(pool, bodyFieldNames)
	rnd.Shuffle(len(pool), func(i, j int) {
		pool[i], pool[j] = pool[j], pool[i]
	})
	if n > len(pool) {
		n = len(pool)
	}
	return pool[:n]
}

// genValuesForPattern 按模式生成 n 个值。
func genValuesForPattern(rnd *rand.Rand, pattern string, n int) []string {
	vals := make([]string, n)
	for i := 0; i < n; i++ {
		vals[i] = genOneValue(rnd, pattern, i)
	}
	return vals
}

func genOneValue(rnd *rand.Rand, pattern string, idx int) string {
	switch pattern {
	case patternInteger:
		return strconv.Itoa(rnd.Intn(1_000_000))
	case patternUUID:
		return genUUID(rnd)
	case patternPhone:
		return genPhone(rnd)
	case patternIDCard:
		return genIDCard(rnd)
	case patternBankCard:
		return genBankCard(rnd)
	case patternPlate:
		return genPlate(rnd)
	case patternPrefix:
		return fmt.Sprintf("user_%03d", idx+1)
	case patternSuffix:
		return fmt.Sprintf("%03d_user", idx+1)
	default:
		return strconv.Itoa(rnd.Intn(1_000_000))
	}
}

// genUUID 生成形如 xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx 的 UUID（小写十六进制）。
func genUUID(rnd *rand.Rand) string {
	b := make([]byte, 16)
	for i := range b {
		b[i] = byte(rnd.Intn(256))
	}
	const hex = "0123456789abcdef"
	out := make([]byte, 36)
	pos := 0
	for i, c := range b {
		out[pos] = hex[c>>4]
		out[pos+1] = hex[c&0x0f]
		pos += 2
		if i == 3 || i == 5 || i == 7 || i == 9 {
			out[pos] = '-'
			pos++
		}
	}
	return string(out[:36])
}

// genPhone 生成 1[3-9]xxxxxxxxxx 共 11 位手机号。
func genPhone(rnd *rand.Rand) string {
	second := 3 + rnd.Intn(7) // 3-9
	s := "1" + strconv.Itoa(second)
	for i := 0; i < 9; i++ {
		s += strconv.Itoa(rnd.Intn(10))
	}
	return s
}

// genIDCard 生成符合 router idcard 正则的 18 位身份证号：
//   [1-9]\d{5} (19|20)\d{2} (0[1-9]|1[0-2]) (0[1-9]|[12]\d|3[01]) \d{3} [\dXx]
// 必须合规，否则 DetectPattern 不识别为 idcard，合并与类型推断都会偏离。
func genIDCard(rnd *rand.Rand) string {
	// 地区码 6 位，首位非 0
	area := strconv.Itoa(1 + rnd.Intn(9))
	for i := 0; i < 5; i++ {
		area += strconv.Itoa(rnd.Intn(10))
	}
	// 出生年：19xx 或 20xx
	year := "19"
	if rnd.Intn(2) == 0 {
		year = "20"
	}
	year += fmt.Sprintf("%02d", rnd.Intn(100))
	// 月：01-12
	month := fmt.Sprintf("%02d", 1+rnd.Intn(12))
	// 日：01-28（避免非法日期简化处理）
	day := fmt.Sprintf("%02d", 1+rnd.Intn(28))
	// 顺序码 3 位
	seq := fmt.Sprintf("%03d", rnd.Intn(1000))
	// 校验位：数字或 X
	last := "X"
	if rnd.Intn(2) == 0 {
		last = strconv.Itoa(rnd.Intn(10))
	}
	return area + year + month + day + seq + last
}

// genBankCard 生成 6 开头的 16-19 位银行卡号。
func genBankCard(rnd *rand.Rand) string {
	length := 16 + rnd.Intn(4) // 16-19
	s := "6"
	for i := 1; i < length; i++ {
		s += strconv.Itoa(rnd.Intn(10))
	}
	return s
}

// genPlate 生成车牌号：汉字 + 字母 + 5-6 位字母数字。
func genPlate(rnd *rand.Rand) string {
	provinces := []string{"京", "沪", "粤", "川", "苏", "浙"}
	prov := provinces[rnd.Intn(len(provinces))]
	letters := "ABCDEFGHJKLMNPQRSTUVWXYZ"
	second := string(letters[rnd.Intn(len(letters))])
	rest := ""
	length := 5 + rnd.Intn(2) // 5-6
	alnum := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for i := 0; i < length; i++ {
		rest += string(alnum[rnd.Intn(len(alnum))])
	}
	return prov + second + rest
}

// genSimilarLengthValues 生成 n 个长度相似的随机字母串（5-6 字符）。
func genSimilarLengthValues(rnd *rand.Rand, n int) []string {
	letters := "abcdefghijklmnopqrstuvwxyz"
	length := 5 + rnd.Intn(2) // 5-6
	vals := make([]string, n)
	for i := 0; i < n; i++ {
		s := ""
		for j := 0; j < length; j++ {
			s += string(letters[rnd.Intn(len(letters))])
		}
		vals[i] = s
	}
	return vals
}

// genHeaderRawValues 生成某 Header 的原始值（含规范化前缀，便于检验规范化）。
func genHeaderRawValues(rnd *rand.Rand, name string) []string {
	switch name {
	case "Accept":
		opts := []string{"application/json", "text/html", "application/xml"}
		v := opts[rnd.Intn(len(opts))]
		return []string{v + ", text/plain;q=0.8"} // 故意带 q 因子
	case "Authorization":
		schemes := []string{"Bearer", "Basic", "Token"}
		scheme := schemes[rnd.Intn(len(schemes))]
		return []string{scheme + " token123abc"}
	case "Accept-Language":
		opts := []string{"zh-CN", "en-US", "ja-JP"}
		v := opts[rnd.Intn(len(opts))]
		return []string{v + ",zh;q=0.9"}
	case "X-Api-Version":
		return []string{"v1", "v2"}
	case "X-Requested-With":
		return []string{"XMLHttpRequest"}
	default:
		return []string{"x"}
	}
}
