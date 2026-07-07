package generator

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"strings"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

// 合并触发阈值（与 router.DefaultMergeConfig 对齐）
const (
	siblingMergeThreshold       = 3 // 同层兄弟 >=3 才尝试合并
	similarLengthBreakThreshold = 6 // similar_length_strings >=6 才突破合并
	requiredParamThreshold      = 0.9
)

// patternXXX 是 PathVarSpec.Pattern 的取值常量
const (
	patternInteger          = "integer"
	patternUUID             = "uuid"
	patternPhone            = "phone"
	patternIDCard           = "idcard"
	patternBankCard         = "bankcard"
	patternPlate            = "plate"
	patternPrefix           = "prefix"
	patternSuffix           = "suffix"
	patternSimilarLength    = "similar_length"
	patternFixedWords       = "fixed_words"
	patternMixedIntFixed    = "mixed_int_fixed"
)

// DerivePathVarExpectations 是 deriveExpectations 的导出包装，供外部测试包
// 构造定向 spec 时触发期望推导（手写 spec 未经过 Generate，需手动冻结期望）。
func DerivePathVarExpectations(pv *PathVarSpec, parentKey string, siblingCount int) {
	deriveExpectations(pv, parentKey, siblingCount)
}

// DeriveBodyFieldExpectations 推导单个 body 字段的期望物理类型与必需性。
// 供外部测试包构造定向 spec 时调用。
func DeriveBodyFieldExpectations(f *BodyFieldSpec, repeat int) {
	deriveBodyFieldExpectations(f, repeat)
}

// deriveBodyFieldExpectations 镜像 router 对 body 参数的处理：
//   - 物理类型按首个值推断：纯数字且 len<16 → integer，否则 string
//   - 必需性：buildRequest 每次请求都带所有字段，故 presence=repeat/repeat=1.0 → 必需
//     （仅当 repeat>=2 时才可靠推断，对齐 requestCount<=1 保持默认）
func deriveBodyFieldExpectations(f *BodyFieldSpec, repeat int) {
	f.ExpectPhysical = inferPhysicalFromValue(firstNonEmpty(f.Values))
	f.ExpectRequired = repeat >= 2
}

func firstNonEmpty(vals []string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// inferPhysicalFromValue 镜像 PhysicalTypeInferenceRule.isInteger：
// 纯数字且长度 < 16 → integer，否则 string。
func inferPhysicalFromValue(val string) value.PhysicalType {
	if isIntegerLike(val) {
		return value.PhysicalTypeInteger
	}
	return value.PhysicalTypeString
}

// isIntegerLike 镜像 physical_type_inference_rule.go 的 isInteger：
// 长度 >= 16 降级为 string；首位可带符号；其余必须全为数字。
func isIntegerLike(val string) bool {
	if len(val) == 0 || len(val) >= 16 {
		return false
	}
	for i, c := range val {
		if i == 0 && (c == '+' || c == '-') {
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
// 把 PathVarSpec 的随机字段确定性推导成 Expect* 字段。这是整个"已知答案"的源头。
//
// 参数：
//   - pv: 待填充的 PathVarSpec（Pattern/Values 已设置）
//   - parentKey: 合并发生的父路径段 key（如 "users"），用于变量名前缀
//   - siblingCount: 同层 request_path 兄弟总数（含固定路径与变量值）
func deriveExpectations(pv *PathVarSpec, parentKey string, siblingCount int) {
	pv.ExpectMerge = false
	pv.ExpectVarName = ""
	pv.ExpectPhysical = ""
	pv.ExpectLogical = value.LogicalTypeString
	pv.ExpectPatternSet = false
	pv.ExpectFixedKept = nil

	switch pv.Pattern {
	case patternInteger:
		// 纯整数 >=3 合并，变量名 {parent}_id，物理 integer
		if siblingCount >= siblingMergeThreshold && len(pv.Values) >= siblingMergeThreshold {
			pv.ExpectMerge = true
			pv.ExpectVarName = parentKey + "_id"
			pv.ExpectPhysical = value.PhysicalTypeInteger
			pv.ExpectLogical = value.LogicalTypeString
			pv.ExpectPatternSet = true
		}
	case patternUUID:
		if siblingCount >= siblingMergeThreshold && len(pv.Values) >= siblingMergeThreshold {
			pv.ExpectMerge = true
			pv.ExpectVarName = parentKey + "_uuid"
			// UUID 物理类型为 string（底层是十六进制串）。
			// 历史上合并后被后续请求命中时物理会被逻辑 "uuid" 污染，已于 2026-07-08 修复
			// （ObserveValue 不再推断，findOrCreatePathNode 用 InferPhysicalAndLogical 回填），
			// 故恢复精确物理断言。
			pv.ExpectPhysical = value.PhysicalTypeString
			pv.ExpectLogical = value.LogicalTypeUUID
			pv.ExpectPatternSet = true
		}
	case patternPhone:
		// 11 位手机号：物理 integer（11 < 16 位阈值，isInteger 返回 true），
		// 逻辑 phone。变量名 {parent}_phone。
		if siblingCount >= siblingMergeThreshold && len(pv.Values) >= siblingMergeThreshold {
			pv.ExpectMerge = true
			pv.ExpectVarName = parentKey + "_phone"
			pv.ExpectPhysical = value.PhysicalTypeInteger
			pv.ExpectLogical = value.LogicalTypePhoneNumber
			pv.ExpectPatternSet = true
		}
	case patternIDCard:
		if siblingCount >= siblingMergeThreshold && len(pv.Values) >= siblingMergeThreshold {
			pv.ExpectMerge = true
			pv.ExpectVarName = parentKey + "_idcard"
			// 身份证 18 位 → 物理 string（18 >= 16 位阈值降级）。合并后物理稳定 string。
			pv.ExpectPhysical = value.PhysicalTypeString
			pv.ExpectLogical = value.LogicalTypeIDCard
			pv.ExpectPatternSet = true
		}
	case patternBankCard:
		if siblingCount >= siblingMergeThreshold && len(pv.Values) >= siblingMergeThreshold {
			pv.ExpectMerge = true
			pv.ExpectVarName = parentKey + "_bankcard"
			// 银行卡 16-19 位 → 物理 string（>= 16 位阈值降级）。合并后物理稳定 string。
			pv.ExpectPhysical = value.PhysicalTypeString
			pv.ExpectLogical = value.LogicalTypeBankCard
			pv.ExpectPatternSet = true
		}
	case patternPlate:
		if siblingCount >= siblingMergeThreshold && len(pv.Values) >= siblingMergeThreshold {
			pv.ExpectMerge = true
			pv.ExpectVarName = parentKey + "_plate"
			// 车牌含汉字 → 物理 string。合并后物理稳定 string。
			pv.ExpectPhysical = value.PhysicalTypeString
			pv.ExpectLogical = value.LogicalTypePlateNumber
			pv.ExpectPatternSet = true
		}
	case patternPrefix:
		// 前缀模式：>=3 且有公共前缀 → 合并，变量名从公共前缀推导为 {base}_id
		if siblingCount >= siblingMergeThreshold && len(pv.Values) >= siblingMergeThreshold {
			base := commonPrefixBase(pv.Values)
			if base == "" {
				base = parentKey
			}
			pv.ExpectMerge = true
			pv.ExpectVarName = base + "_id"
			pv.ExpectPhysical = value.PhysicalTypeString
			pv.ExpectLogical = value.LogicalTypeString
			pv.ExpectPatternSet = true
		}
	case patternSuffix:
		// 后缀模式：>=3 且有公共后缀 → 合并，变量名从公共后缀推导为 {base}_id
		if siblingCount >= siblingMergeThreshold && len(pv.Values) >= siblingMergeThreshold {
			base := commonSuffixBase(pv.Values)
			if base == "" {
				base = parentKey
			}
			pv.ExpectMerge = true
			pv.ExpectVarName = base + "_id"
			pv.ExpectPhysical = value.PhysicalTypeString
			pv.ExpectLogical = value.LogicalTypeString
			pv.ExpectPatternSet = true
		}
	case patternSimilarLength:
		// 长度相似串：默认不合并，>=6 才突破；突破后走 default → var_{parent}
		if len(pv.Values) >= similarLengthBreakThreshold {
			pv.ExpectMerge = true
			pv.ExpectVarName = "var_" + parentKey
			pv.ExpectPhysical = value.PhysicalTypeString
			pv.ExpectLogical = value.LogicalTypeString
			pv.ExpectPatternSet = false
		}
	case patternFixedWords:
		// 固定单词路径名（admin/manager/guest）：永不合并，保留为固定路径
		pv.ExpectMerge = false
	case patternMixedIntFixed:
		// 混合：整数子集 + 固定路径，选择性合并整数子集，保留固定路径
		if siblingCount >= siblingMergeThreshold && len(pv.Values) >= siblingMergeThreshold {
			pv.ExpectMerge = true
			pv.ExpectVarName = parentKey + "_id"
			pv.ExpectPhysical = value.PhysicalTypeInteger
			pv.ExpectLogical = value.LogicalTypeString
			pv.ExpectPatternSet = true
			pv.ExpectFixedKept = []string{"list", "create"}
		}
	}
}

// commonPrefixBase 镜像 router 的 inferVariableNameWithContext 对 prefix 模式的处理：
// 取最长公共前缀，去掉末尾数字，再去掉末尾分隔符。suffix 模式对称（去开头数字和分隔符）。
// 这里对 prefix/suffix 通用：都先求公共前缀再做 trim（生成器保证 suffix 场景的 Values 也能用前缀推导出 base）。
func commonPrefixBase(values []string) string {
	if len(values) == 0 {
		return ""
	}
	prefix := longestCommonPrefix(values)
	prefix = trimTrailingDigits(prefix)
	prefix = trimTrailingSeparator(prefix)
	return prefix
}

func longestCommonPrefix(values []string) string {
	if len(values) == 0 {
		return ""
	}
	min := values[0]
	for _, v := range values[1:] {
		if len(v) < len(min) {
			min = v
		}
	}
	for i := 0; i < len(min); i++ {
		c := min[i]
		for _, v := range values {
			if v[i] != c {
				return min[:i]
			}
		}
	}
	return min
}

func trimTrailingDigits(s string) string {
	i := len(s)
	for i > 0 && s[i-1] >= '0' && s[i-1] <= '9' {
		i--
	}
	return s[:i]
}

func trimTrailingSeparator(s string) string {
	i := len(s)
	for i > 0 {
		c := s[i-1]
		if c == '_' || c == '-' || c == '.' || c == '/' {
			i--
			continue
		}
		break
	}
	return s[:i]
}

// commonSuffixBase 镜像 router 的 inferVariableNameWithContext 对 suffix 模式的处理：
// 取最长公共后缀，去掉开头数字，再去掉开头分隔符。
func commonSuffixBase(values []string) string {
	if len(values) == 0 {
		return ""
	}
	suffix := longestCommonSuffix(values)
	suffix = trimLeadingDigits(suffix)
	suffix = trimLeadingSeparator(suffix)
	return suffix
}

func longestCommonSuffix(values []string) string {
	if len(values) == 0 {
		return ""
	}
	min := values[0]
	for _, v := range values[1:] {
		if len(v) < len(min) {
			min = v
		}
	}
	for i := len(min); i > 0; i-- {
		candidate := min[len(min)-i:]
		ok := true
		for _, v := range values {
			if len(v) < i || v[len(v)-i:] != candidate {
				ok = false
				break
			}
		}
		if ok {
			return candidate
		}
	}
	return ""
}

func trimLeadingDigits(s string) string {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	return s[i:]
}

func trimLeadingSeparator(s string) string {
	i := 0
	for i < len(s) {
		c := s[i]
		if c == '_' || c == '-' || c == '.' || c == '/' {
			i++
			continue
		}
		break
	}
	return s[i:]
}

// deriveHeaderNorm 镜像 router 的路由 Header 规范化逻辑，把原始值列表转成期望的值节点 key 列表。
func deriveHeaderNorm(name string, raw []string) []string {
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		out = append(out, normalizeHeaderValue(name, v))
	}
	return out
}

// normalizeHeaderValue 镜像 router.reverse_router.go 的 normalizeAccept/normalizeAuthorization/...
func normalizeHeaderValue(name, val string) string {
	if val == "" {
		return ""
	}
	switch name {
	case "Accept":
		// 取第一个 MIME，去 ;q= 因子
		parts := strings.SplitN(val, ",", 2)
		mime := strings.TrimSpace(parts[0])
		if i := strings.Index(mime, ";"); i >= 0 {
			mime = strings.TrimSpace(mime[:i])
		}
		return mime
	case "Authorization":
		// 只取认证方案（Bearer/Basic/Token）
		parts := strings.SplitN(val, " ", 2)
		if len(parts) > 0 {
			return parts[0]
		}
		return ""
	case "Accept-Language":
		// 取第一个语言标签，去 ;q= 因子
		parts := strings.SplitN(val, ",", 2)
		lang := strings.TrimSpace(parts[0])
		if i := strings.Index(lang, ";"); i >= 0 {
			lang = strings.TrimSpace(lang[:i])
		}
		return lang
	case "X-Api-Version", "X-Requested-With":
		// 原值不变
		return val
	default:
		return val
	}
}

// marshalBody 按 BodySpec 生成合法的请求体字节。
// JSON 用 json.Marshal(map) 保证合法；form 用 url.Values.Encode()。
func marshalBody(b *BodySpec) ([]byte, error) {
	if b == nil {
		return nil, nil
	}
	switch b.ContentType {
	case "application/json":
		m := make(map[string]interface{}, len(b.Fields))
		for _, f := range b.Fields {
			if len(f.Values) > 0 {
				m[f.Name] = f.Values[0]
			}
		}
		return json.Marshal(m)
	case "application/x-www-form-urlencoded":
		form := url.Values{}
		for _, f := range b.Fields {
			if len(f.Values) > 0 {
				form.Set(f.Name, f.Values[0])
			}
		}
		return []byte(form.Encode()), nil
	default:
		return nil, fmt.Errorf("不支持的 Content-Type: %s", b.ContentType)
	}
}

// specRequests 按 spec 派生 HTTP 请求序列。
//
// 派生逻辑：
//   - 每个操作按 Repeat 重复生成请求；
//   - 路径变量值按 Values 全部轮换喂入（不随机丢弃，保证合并触发）；
//   - 查询参数按 rnd.Float64() < Presence 决定该次是否带上；
//   - POST/PUT/PATCH 带 Content-Type + body；
//   - Header 只用白名单名；Cookie 拼 "name=value; ..."。
func specRequests(s *Spec, rnd *rand.Rand) []*request.HttpRequest {
	var reqs []*request.HttpRequest
	for _, res := range s.Resources {
		for _, op := range res.Operations {
			n := op.Repeat
			if n < 1 {
				n = 1
			}
			for i := 0; i < n; i++ {
				reqs = append(reqs, buildRequest(res, op, i, rnd))
			}
		}
	}
	return reqs
}

// buildRequest 为一次操作构造单个 HttpRequest。
func buildRequest(res *Resource, op *Operation, iter int, rnd *rand.Rand) *request.HttpRequest {
	// 构造路径：prefix + resourceName [+ pathVarValue]
	var segs []string
	segs = append(segs, res.Prefix...)
	segs = append(segs, res.Name)

	var pathVarValue string
	if op.PathVar != nil && len(op.PathVar.Values) > 0 {
		// 全部轮换喂入，保证合并触发
		pathVarValue = op.PathVar.Values[iter%len(op.PathVar.Values)]
		segs = append(segs, pathVarValue)
	}
	path := "/" + strings.Join(segs, "/")

	// 查询参数：按 Presence 概率带上
	var queryParts []string
	for _, q := range op.QueryParams {
		if len(q.Values) > 0 && rnd.Float64() < q.Presence {
			val := q.Values[iter%len(q.Values)]
			queryParts = append(queryParts, url.QueryEscape(q.Name)+"="+url.QueryEscape(val))
		}
	}
	if len(queryParts) > 0 {
		path += "?" + strings.Join(queryParts, "&")
	}

	// Headers
	headers := request.Headers{}
	bodyBytes := []byte(nil)
	if op.Body != nil {
		headers.Set("Content-Type", op.Body.ContentType)
		b, err := marshalBody(op.Body)
		if err == nil {
			bodyBytes = b
		}
	}
	for _, h := range op.Headers {
		if len(h.Values) > 0 {
			headers.Set(h.Name, h.Values[iter%len(h.Values)])
		}
	}
	for _, c := range op.Cookies {
		if len(c.Values) > 0 {
			cookieVal := c.Name + "=" + c.Values[iter%len(c.Values)]
			if existing := headers.Get("Cookie"); existing != "" {
				headers.Set("Cookie", existing+"; "+cookieVal)
			} else {
				headers.Set("Cookie", cookieVal)
			}
		}
	}

	method := op.Method
	if method == "" {
		method = "GET"
	}
	return request.NewHttpRequest(path, headers, method, bodyBytes)
}
