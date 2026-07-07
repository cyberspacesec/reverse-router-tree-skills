package generator

import (
	"encoding/json"
	"math/rand"
	"net/url"
	"strings"
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

// TestDeriveExpectations_Integer 纯整数 >=3 合并
func TestDeriveExpectations_Integer(t *testing.T) {
	pv := &PathVarSpec{Pattern: patternInteger, Values: []string{"123", "456", "789"}}
	deriveExpectations(pv, "users", 3)
	if !pv.ExpectMerge {
		t.Error("integer >=3 应合并")
	}
	if pv.ExpectVarName != "users_id" {
		t.Errorf("变量名期望 users_id 实际 %s", pv.ExpectVarName)
	}
	if pv.ExpectPhysical != value.PhysicalTypeInteger {
		t.Errorf("物理类型期望 integer 实际 %s", pv.ExpectPhysical)
	}
	if !pv.ExpectPatternSet {
		t.Error("integer 应有正则模式")
	}
}

// TestDeriveExpectations_Integer_BelowThreshold 纯整数 <3 不合并
func TestDeriveExpectations_Integer_BelowThreshold(t *testing.T) {
	pv := &PathVarSpec{Pattern: patternInteger, Values: []string{"123", "456"}}
	deriveExpectations(pv, "users", 2)
	if pv.ExpectMerge {
		t.Error("integer <3 不应合并")
	}
}

// TestDeriveExpectations_UUID UUID 合并：物理 string，逻辑 uuid
func TestDeriveExpectations_UUID(t *testing.T) {
	pv := &PathVarSpec{Pattern: patternUUID, Values: []string{genUUID(rand.New(rand.NewSource(1))), "a", "b"}}
	// 用合法 UUID
	pv.Values = []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"550e8400-e29b-41d4-a716-446655440001",
		"550e8400-e29b-41d4-a716-446655440002",
	}
	deriveExpectations(pv, "res", 3)
	if !pv.ExpectMerge || pv.ExpectVarName != "res_uuid" {
		t.Errorf("UUID 合并错误: merge=%v name=%s", pv.ExpectMerge, pv.ExpectVarName)
	}
	if pv.ExpectPhysical != value.PhysicalTypeString {
		t.Errorf("UUID 物理类型期望 string 实际 %s", pv.ExpectPhysical)
	}
	if pv.ExpectLogical != value.LogicalTypeUUID {
		t.Errorf("UUID 逻辑类型期望 uuid 实际 %s", pv.ExpectLogical)
	}
}

// TestDeriveExpectations_Phone 手机号合并：物理 integer(11<16位阈值)，逻辑 phone
func TestDeriveExpectations_Phone(t *testing.T) {
	pv := &PathVarSpec{Pattern: patternPhone, Values: []string{"13812345678", "13912345678", "15012345678"}}
	deriveExpectations(pv, "users", 3)
	if !pv.ExpectMerge || pv.ExpectVarName != "users_phone" {
		t.Errorf("phone 合并错误: merge=%v name=%s", pv.ExpectMerge, pv.ExpectVarName)
	}
	if pv.ExpectPhysical != value.PhysicalTypeInteger {
		t.Errorf("phone 物理类型期望 integer(11位<16位阈值) 实际 %s", pv.ExpectPhysical)
	}
	if pv.ExpectLogical != value.LogicalTypePhoneNumber {
		t.Errorf("phone 逻辑类型期望 phone 实际 %s", pv.ExpectLogical)
	}
}

// TestDeriveExpectations_IDCard 身份证合并：物理 string（18位>=16位阈值降级），逻辑 idcard
func TestDeriveExpectations_IDCard(t *testing.T) {
	pv := &PathVarSpec{Pattern: patternIDCard, Values: []string{"110101199001011234", "310101198501012345", "44010119920303123X"}}
	deriveExpectations(pv, "users", 3)
	if !pv.ExpectMerge || pv.ExpectVarName != "users_idcard" {
		t.Errorf("idcard 合并错误: merge=%v name=%s", pv.ExpectMerge, pv.ExpectVarName)
	}
	if pv.ExpectPhysical != value.PhysicalTypeString {
		t.Errorf("idcard 物理类型期望 string(18位降级) 实际 %s", pv.ExpectPhysical)
	}
	if pv.ExpectLogical != value.LogicalTypeIDCard {
		t.Errorf("idcard 逻辑类型期望 idcard 实际 %s", pv.ExpectLogical)
	}
}

// TestDeriveExpectations_BankCard 银行卡合并：物理 string（16-19位>=16位阈值降级），逻辑 bankcard
func TestDeriveExpectations_BankCard(t *testing.T) {
	pv := &PathVarSpec{Pattern: patternBankCard, Values: []string{"6222021234567890123", "6225887654321098765", "6217001234567890123"}}
	deriveExpectations(pv, "cards", 3)
	if !pv.ExpectMerge || pv.ExpectVarName != "cards_bankcard" {
		t.Errorf("bankcard 合并错误: merge=%v name=%s", pv.ExpectMerge, pv.ExpectVarName)
	}
	if pv.ExpectPhysical != value.PhysicalTypeString {
		t.Errorf("bankcard 物理类型期望 string(16-19位降级) 实际 %s", pv.ExpectPhysical)
	}
	if pv.ExpectLogical != value.LogicalTypeBankCard {
		t.Errorf("bankcard 逻辑类型期望 bankcard 实际 %s", pv.ExpectLogical)
	}
}

// TestDeriveExpectations_Plate 车牌合并：物理 string（含汉字），逻辑 plate
func TestDeriveExpectations_Plate(t *testing.T) {
	pv := &PathVarSpec{Pattern: patternPlate, Values: []string{"京A12345", "沪B12345D", "粤B12345"}}
	deriveExpectations(pv, "vehicles", 3)
	if !pv.ExpectMerge || pv.ExpectVarName != "vehicles_plate" {
		t.Errorf("plate 合并错误: merge=%v name=%s", pv.ExpectMerge, pv.ExpectVarName)
	}
	if pv.ExpectPhysical != value.PhysicalTypeString {
		t.Errorf("plate 物理类型期望 string 实际 %s", pv.ExpectPhysical)
	}
	if pv.ExpectLogical != value.LogicalTypePlateNumber {
		t.Errorf("plate 逻辑类型期望 plate 实际 %s", pv.ExpectLogical)
	}
}

// TestDeriveExpectations_Prefix 前缀模式：user_001 → user_id
func TestDeriveExpectations_Prefix(t *testing.T) {
	pv := &PathVarSpec{Pattern: patternPrefix, Values: []string{"user_001", "user_002", "user_003"}}
	deriveExpectations(pv, "items", 3)
	if !pv.ExpectMerge {
		t.Error("prefix >=3 应合并")
	}
	if pv.ExpectVarName != "user_id" {
		t.Errorf("prefix 变量名期望 user_id 实际 %s", pv.ExpectVarName)
	}
}

// TestDeriveExpectations_Suffix 后缀模式：001_user → user_id
func TestDeriveExpectations_Suffix(t *testing.T) {
	pv := &PathVarSpec{Pattern: patternSuffix, Values: []string{"001_user", "002_user", "003_user"}}
	deriveExpectations(pv, "items", 3)
	if !pv.ExpectMerge {
		t.Error("suffix >=3 应合并")
	}
	// suffix 模式 router 从公共后缀推导，公共后缀是 "_user"，去开头数字和分隔符 → "user"
	if pv.ExpectVarName != "user_id" {
		t.Errorf("suffix 变量名期望 user_id 实际 %s", pv.ExpectVarName)
	}
}

// TestDeriveExpectations_SimilarLength_Break >=6 突破合并
func TestDeriveExpectations_SimilarLength_Break(t *testing.T) {
	pv := &PathVarSpec{Pattern: patternSimilarLength, Values: []string{"abcde", "fghij", "klmno", "pqrst", "uvwxy", "zabcd"}}
	deriveExpectations(pv, "city", 6)
	if !pv.ExpectMerge {
		t.Error("similar_length >=6 应突破合并")
	}
	if pv.ExpectVarName != "var_city" {
		t.Errorf("similar_length 变量名期望 var_city 实际 %s", pv.ExpectVarName)
	}
	if pv.ExpectPatternSet {
		t.Error("similar_length 不应有正则模式")
	}
}

// TestDeriveExpectations_SimilarLength_NoBreak 3-5 不合并
func TestDeriveExpectations_SimilarLength_NoBreak(t *testing.T) {
	pv := &PathVarSpec{Pattern: patternSimilarLength, Values: []string{"admin", "manager", "guest"}}
	deriveExpectations(pv, "roles", 3)
	if pv.ExpectMerge {
		t.Error("similar_length <6 不应合并")
	}
}

// TestDeriveExpectations_FixedWords 固定单词不合并
func TestDeriveExpectations_FixedWords(t *testing.T) {
	pv := &PathVarSpec{Pattern: patternFixedWords, Values: []string{"admin", "manager", "guest"}}
	deriveExpectations(pv, "roles", 3)
	if pv.ExpectMerge {
		t.Error("fixed_words 永不合并")
	}
}

// TestDeriveExpectations_Mixed 选择性合并：保留 list/create
func TestDeriveExpectations_Mixed(t *testing.T) {
	pv := &PathVarSpec{Pattern: patternMixedIntFixed, Values: []string{"101", "102", "103", "list", "create"}}
	deriveExpectations(pv, "users", 5)
	if !pv.ExpectMerge {
		t.Error("mixed 应选择性合并")
	}
	if pv.ExpectVarName != "users_id" {
		t.Errorf("mixed 变量名期望 users_id 实际 %s", pv.ExpectVarName)
	}
	if len(pv.ExpectFixedKept) != 2 || pv.ExpectFixedKept[0] != "list" || pv.ExpectFixedKept[1] != "create" {
		t.Errorf("mixed 应保留 list/create，实际 %v", pv.ExpectFixedKept)
	}
}

// TestMarshalBody_JSON JSON body 必须合法可反序列化
func TestMarshalBody_JSON(t *testing.T) {
	b := &BodySpec{
		ContentType: "application/json",
		Fields: []*BodyFieldSpec{
			{Name: "name", Values: []string{"alice"}},
			{Name: "age", Values: []string{"30"}},
		},
	}
	data, err := marshalBody(b)
	if err != nil {
		t.Fatalf("marshalBody JSON 失败: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("生成的 JSON 不合法: %v\ndata=%s", err, string(data))
	}
	if m["name"] != "alice" {
		t.Errorf("JSON name 字段期望 alice 实际 %v", m["name"])
	}
}

// TestMarshalBody_Form 表单 body 必须合法可解析
func TestMarshalBody_Form(t *testing.T) {
	b := &BodySpec{
		ContentType: "application/x-www-form-urlencoded",
		Fields: []*BodyFieldSpec{
			{Name: "name", Values: []string{"alice"}},
			{Name: "age", Values: []string{"30"}},
		},
	}
	data, err := marshalBody(b)
	if err != nil {
		t.Fatalf("marshalBody form 失败: %v", err)
	}
	vals, err := url.ParseQuery(string(data))
	if err != nil {
		t.Fatalf("生成的 form 不合法: %v\ndata=%s", err, string(data))
	}
	if vals.Get("name") != "alice" {
		t.Errorf("form name 期望 alice 实际 %s", vals.Get("name"))
	}
}

// TestDeriveHeaderNorm Header 规范化
func TestDeriveHeaderNorm(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"Accept", "application/json, text/html;q=0.9", "application/json"},
		{"Authorization", "Bearer token123", "Bearer"},
		{"Authorization", "Basic dXNlcjpwYXNz", "Basic"},
		{"Accept-Language", "zh-CN,zh;q=0.9", "zh-CN"},
		{"X-Api-Version", "v2", "v2"},
		{"X-Requested-With", "XMLHttpRequest", "XMLHttpRequest"},
	}
	for _, c := range cases {
		got := deriveHeaderNorm(c.name, []string{c.raw})
		if len(got) != 1 || got[0] != c.want {
			t.Errorf("header %s 规范化期望 %q 实际 %v", c.name, c.want, got)
		}
	}
}

// TestGenerator_Generate_ProducesValidSpec 生成器生成的 spec 必须可派生请求且断言非空
func TestGenerator_Generate_ProducesValidSpec(t *testing.T) {
	g := NewGenerator(DefaultSeed)
	spec := g.Generate()
	if spec == nil || len(spec.Resources) == 0 {
		t.Fatal("生成的 spec 为空")
	}
	rnd := rand.New(rand.NewSource(spec.Seed))
	reqs := spec.Requests(rnd)
	if len(reqs) == 0 {
		t.Fatal("spec 派生的请求为空")
	}
	asserts := spec.Assertions()
	if len(asserts) == 0 {
		t.Fatal("spec 派生的断言为空")
	}
	// String() 应可打印且非空
	if strings.TrimSpace(spec.String()) == "" {
		t.Error("spec.String() 不应为空")
	}
}

// TestGenerator_Repeatable 同 seed 生成等价 spec
func TestGenerator_Repeatable(t *testing.T) {
	s1 := NewGenerator(42).Generate()
	s2 := NewGenerator(42).Generate()
	if s1.Seed != s2.Seed {
		t.Errorf("同 seed 应生成相同 Seed: %d vs %d", s1.Seed, s2.Seed)
	}
	if len(s1.Resources) != len(s2.Resources) {
		t.Errorf("同 seed 资源数应相同: %d vs %d", len(s1.Resources), len(s2.Resources))
	}
}
