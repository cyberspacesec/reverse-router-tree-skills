package inference

import (
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

// 测试UUID逻辑类型推断
func TestLogicalType_UUID(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	pathVarNode := node.NewRequestPathVariableNode("id", "[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}")
	uuids := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		"6ba7b811-9dad-11d1-80b4-00c04fd430c8",
	}
	for _, uuid := range uuids {
		pathVarNode.ObserveValue(uuid)
	}

	inferred, err := rule.Infer(pathVarNode)
	if err != nil {
		t.Fatalf("推断失败: %v", err)
	}

	if inferred != value.Type(value.LogicalTypeUUID) {
		t.Errorf("UUID值应该推断为 'uuid'，实际: '%s'", inferred)
	}
}

// 测试日期逻辑类型推断
func TestLogicalType_Date(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	pathVarNode := node.NewRequestPathVariableNode("date", "")
	dates := []string{"2024-01-15", "2024-02-20", "2024-03-25"}
	for _, d := range dates {
		pathVarNode.ObserveValue(d)
	}

	inferred, err := rule.Infer(pathVarNode)
	if err != nil {
		t.Fatalf("推断失败: %v", err)
	}

	if inferred != value.Type(value.LogicalTypeDate) {
		t.Errorf("日期值应该推断为 'date'，实际: '%s'", inferred)
	}
}

// 测试日期时间逻辑类型推断
func TestLogicalType_DateTime(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	pathVarNode := node.NewRequestPathVariableNode("ts", "")
	dateTimes := []string{
		"2024-01-15T10:30:00Z",
		"2024-02-20T14:45:30Z",
		"2024-03-25T08:15:00Z",
	}
	for _, dt := range dateTimes {
		pathVarNode.ObserveValue(dt)
	}

	inferred, err := rule.Infer(pathVarNode)
	if err != nil {
		t.Fatalf("推断失败: %v", err)
	}

	if inferred != value.Type(value.LogicalTypeDateTime) {
		t.Errorf("日期时间值应该推断为 'datetime'，实际: '%s'", inferred)
	}
}

// 测试邮箱逻辑类型推断
func TestLogicalType_Email(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	pathVarNode := node.NewRequestPathVariableNode("email", "")
	emails := []string{
		"user@example.com",
		"admin@test.org",
		"dev@company.co",
	}
	for _, e := range emails {
		pathVarNode.ObserveValue(e)
	}

	inferred, err := rule.Infer(pathVarNode)
	if err != nil {
		t.Fatalf("推断失败: %v", err)
	}

	if inferred != value.Type(value.LogicalTypeEmail) {
		t.Errorf("邮箱值应该推断为 'email'，实际: '%s'", inferred)
	}
}

// 测试IP地址逻辑类型推断
func TestLogicalType_IPAddress(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	pathVarNode := node.NewRequestPathVariableNode("ip", "")
	ips := []string{"192.168.1.1", "10.0.0.1", "172.16.0.1"}
	for _, ip := range ips {
		pathVarNode.ObserveValue(ip)
	}

	inferred, err := rule.Infer(pathVarNode)
	if err != nil {
		t.Fatalf("推断失败: %v", err)
	}

	if inferred != value.Type(value.LogicalTypeIPAddress) {
		t.Errorf("IP地址值应该推断为 'ipaddress'，实际: '%s'", inferred)
	}
}

// 测试URL逻辑类型推断
func TestLogicalType_URL(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	pathVarNode := node.NewRequestPathVariableNode("url", "")
	urls := []string{
		"https://example.com/api",
		"http://test.org/resource",
		"https://api.service.io/v1",
	}
	for _, u := range urls {
		pathVarNode.ObserveValue(u)
	}

	inferred, err := rule.Infer(pathVarNode)
	if err != nil {
		t.Fatalf("推断失败: %v", err)
	}

	if inferred != value.Type(value.LogicalTypeURL) {
		t.Errorf("URL值应该推断为 'url'，实际: '%s'", inferred)
	}
}

// 测试百分比逻辑类型推断
func TestLogicalType_Percentage(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	pathVarNode := node.NewRequestPathVariableNode("rate", "")
	percentages := []string{"50%", "75%", "100%"}
	for _, p := range percentages {
		pathVarNode.ObserveValue(p)
	}

	inferred, err := rule.Infer(pathVarNode)
	if err != nil {
		t.Fatalf("推断失败: %v", err)
	}

	if inferred != value.Type(value.LogicalTypePercentage) {
		t.Errorf("百分比值应该推断为 'percentage'，实际: '%s'", inferred)
	}
}

// 测试货币逻辑类型推断
func TestLogicalType_Currency(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	pathVarNode := node.NewRequestPathVariableNode("price", "")
	currencies := []string{"$99.99", "€49.50", "¥199.00"}
	for _, c := range currencies {
		pathVarNode.ObserveValue(c)
	}

	inferred, err := rule.Infer(pathVarNode)
	if err != nil {
		t.Fatalf("推断失败: %v", err)
	}

	if inferred != value.Type(value.LogicalTypeCurrency) {
		t.Errorf("货币值应该推断为 'currency'，实际: '%s'", inferred)
	}
}

// 测试JSON逻辑类型推断
func TestLogicalType_JSON(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	pathVarNode := node.NewRequestPathVariableNode("data", "")
	jsons := []string{
		`{"name": "test"}`,
		`[1, 2, 3]`,
		`{"id": 1, "value": "abc"}`,
	}
	for _, j := range jsons {
		pathVarNode.ObserveValue(j)
	}

	inferred, err := rule.Infer(pathVarNode)
	if err != nil {
		t.Fatalf("推断失败: %v", err)
	}

	if inferred != value.Type(value.LogicalTypeJSON) {
		t.Errorf("JSON值应该推断为 'json'，实际: '%s'", inferred)
	}
}

// 测试枚举类型推断
func TestLogicalType_Enum(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	pathVarNode := node.NewRequestPathVariableNode("status", "")
	// 重复出现的几个有限值，很像枚举
	// 3个唯一值 / 12个总采样 = 0.25 < 0.3 阈值
	statuses := []string{
		"active", "inactive", "pending",
		"active", "inactive", "pending",
		"active", "inactive", "pending",
		"active", "active", "pending",
	}
	for _, s := range statuses {
		pathVarNode.ObserveValue(s)
	}

	inferred, err := rule.Infer(pathVarNode)
	if err != nil {
		t.Fatalf("推断失败: %v", err)
	}

	if inferred != value.Type(value.LogicalTypeEnum) {
		t.Errorf("重复有限值应该推断为 'enum'，实际: '%s'", inferred)
	}
}

// 测试普通字符串不被误判
func TestLogicalType_PlainString(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	pathVarNode := node.NewRequestPathVariableNode("name", "")
	names := []string{"Alice", "Bob", "Charlie", "David", "Eve"}
	for _, n := range names {
		pathVarNode.ObserveValue(n)
	}

	inferred, err := rule.Infer(pathVarNode)
	if err != nil {
		t.Fatalf("推断失败: %v", err)
	}

	// 普通字符串不应该被推断为枚举（唯一值太多）
	// 应该返回 string
	if inferred != value.Type(value.LogicalTypeString) {
		t.Errorf("普通字符串应该推断为 'string'，实际: '%s'", inferred)
	}
}

// 测试空节点
func TestLogicalType_EmptyNode(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	pathVarNode := node.NewRequestPathVariableNode("empty", "")

	inferred, err := rule.Infer(pathVarNode)
	if err != nil {
		t.Fatalf("推断失败: %v", err)
	}

	if inferred != value.Type(value.LogicalTypeString) {
		t.Errorf("空节点应该推断为 'string'，实际: '%s'", inferred)
	}
}

// 测试时间逻辑类型推断
func TestLogicalType_Time(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	pathVarNode := node.NewRequestPathVariableNode("time", "")
	times := []string{"10:30:00", "14:45:30", "08:15:00"}
	for _, tm := range times {
		pathVarNode.ObserveValue(tm)
	}

	inferred, err := rule.Infer(pathVarNode)
	if err != nil {
		t.Fatalf("推断失败: %v", err)
	}

	if inferred != value.Type(value.LogicalTypeTime) {
		t.Errorf("时间值应该推断为 'time'，实际: '%s'", inferred)
	}
}

// === 链式推断规则测试 ===

// 测试链式推断：先物理类型再逻辑类型
func TestChainInference_UUID(t *testing.T) {
	chain := NewChainTypeInferenceRule()

	pathVarNode := node.NewRequestPathVariableNode("id", "")
	uuids := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		"6ba7b811-9dad-11d1-80b4-00c04fd430c8",
	}
	for _, uuid := range uuids {
		pathVarNode.ObserveValue(uuid)
	}

	inferred, err := chain.Infer(pathVarNode)
	if err != nil {
		t.Fatalf("推断失败: %v", err)
	}

	// 链式推断应该返回更具体的逻辑类型 uuid
	if inferred != value.Type(value.LogicalTypeUUID) {
		t.Errorf("链式推断UUID应该返回 'uuid'，实际: '%s'", inferred)
	}
}

// 测试链式推断：整数返回物理类型
func TestChainInference_Integer(t *testing.T) {
	chain := NewChainTypeInferenceRule()

	pathVarNode := node.NewRequestPathVariableNode("id", "[0-9]+")
	for _, id := range []string{"123", "456", "789"} {
		pathVarNode.ObserveValue(id)
	}

	inferred, err := chain.Infer(pathVarNode)
	if err != nil {
		t.Fatalf("推断失败: %v", err)
	}

	// 整数没有更具体的逻辑类型，应该返回 integer
	if inferred != value.Type(value.PhysicalTypeInteger) {
		t.Errorf("整数推断应该返回 'integer'，实际: '%s'", inferred)
	}
}

// 测试 InferPhysicalAndLogical 分别获取物理和逻辑类型
func TestChainInference_PhysicalAndLogical(t *testing.T) {
	chain := NewChainTypeInferenceRule()

	pathVarNode := node.NewRequestPathVariableNode("id", "")
	for _, uuid := range []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	} {
		pathVarNode.ObserveValue(uuid)
	}

	physicalType, logicalType, err := chain.InferPhysicalAndLogical(pathVarNode)
	if err != nil {
		t.Fatalf("推断失败: %v", err)
	}

	if physicalType != value.PhysicalTypeString {
		t.Errorf("UUID的物理类型应该是 'string'，实际: '%s'", physicalType)
	}

	if logicalType != value.LogicalTypeUUID {
		t.Errorf("UUID的逻辑类型应该是 'uuid'，实际: '%s'", logicalType)
	}
}

// 测试 InferPhysicalAndLogical 对整数
func TestChainInference_PhysicalAndLogical_Integer(t *testing.T) {
	chain := NewChainTypeInferenceRule()

	pathVarNode := node.NewRequestPathVariableNode("id", "[0-9]+")
	for _, id := range []string{"123", "456", "789"} {
		pathVarNode.ObserveValue(id)
	}

	physicalType, logicalType, err := chain.InferPhysicalAndLogical(pathVarNode)
	if err != nil {
		t.Fatalf("推断失败: %v", err)
	}

	if physicalType != value.PhysicalTypeInteger {
		t.Errorf("整数的物理类型应该是 'integer'，实际: '%s'", physicalType)
	}

	// 整数没有更具体的逻辑类型，应该返回 string（表示无更具体语义）
	if logicalType != value.LogicalTypeString {
		t.Errorf("整数的逻辑类型应该是 'string'（无更具体语义），实际: '%s'", logicalType)
	}
}

// 测试自定义规则链
func TestChainInference_CustomRules(t *testing.T) {
	// 只使用逻辑类型推断规则
	chain := NewChainTypeInferenceRuleWithRules(NewLogicalTypeInferenceRule())

	pathVarNode := node.NewRequestPathVariableNode("id", "")
	for _, ip := range []string{"192.168.1.1", "10.0.0.1", "172.16.0.1"} {
		pathVarNode.ObserveValue(ip)
	}

	inferred, err := chain.Infer(pathVarNode)
	if err != nil {
		t.Fatalf("推断失败: %v", err)
	}

	if inferred != value.Type(value.LogicalTypeIPAddress) {
		t.Errorf("自定义规则链应该推断为 'ipaddress'，实际: '%s'", inferred)
	}
}

// 测试精确小数逻辑类型推断
func TestLogicalType_Decimal(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	pathVarNode := node.NewRequestPathVariableNode("amount", "")
	decimals := []string{"99.99", "49.50", "199.00"}
	for _, d := range decimals {
		pathVarNode.ObserveValue(d)
	}

	inferred, err := rule.Infer(pathVarNode)
	if err != nil {
		t.Fatalf("推断失败: %v", err)
	}

	if inferred != value.Type(value.LogicalTypeDecimal) {
		t.Errorf("精确小数值应该推断为 'decimal'，实际: '%s'", inferred)
	}
}

// 测试XML逻辑类型推断
func TestLogicalType_XML(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	pathVarNode := node.NewRequestPathVariableNode("data", "")
	xmls := []string{
		"<root><item>test</item></root>",
		"<?xml version=\"1.0\"?><data/>",
		"<response><status>ok</status></response>",
	}
	for _, x := range xmls {
		pathVarNode.ObserveValue(x)
	}

	inferred, err := rule.Infer(pathVarNode)
	if err != nil {
		t.Fatalf("推断失败: %v", err)
	}

	if inferred != value.Type(value.LogicalTypeXML) {
		t.Errorf("XML值应该推断为 'xml'，实际: '%s'", inferred)
	}
}

// === 中国特有格式逻辑类型推断测试 ===

// 测试手机号逻辑类型推断
func TestLogicalType_PhoneNumber(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	pathVarNode := node.NewRequestPathVariableNode("phone", "")
	phones := []string{
		"13812345678",
		"15912345678",
		"18612345678",
		"17712345678",
	}
	for _, p := range phones {
		pathVarNode.ObserveValue(p)
	}

	inferred, err := rule.Infer(pathVarNode)
	if err != nil {
		t.Fatalf("推断失败: %v", err)
	}

	if inferred != value.Type(value.LogicalTypePhoneNumber) {
		t.Errorf("手机号应该推断为 'phone'，实际: '%s'", inferred)
	}
}

// 测试带国际前缀的手机号
func TestLogicalType_PhoneNumberInternational(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	pathVarNode := node.NewRequestPathVariableNode("phone", "")
	phones := []string{
		"+8613812345678",
		"+8615912345678",
		"+8618612345678",
	}
	for _, p := range phones {
		pathVarNode.ObserveValue(p)
	}

	inferred, err := rule.Infer(pathVarNode)
	if err != nil {
		t.Fatalf("推断失败: %v", err)
	}

	if inferred != value.Type(value.LogicalTypePhoneNumber) {
		t.Errorf("国际格式手机号应该推断为 'phone'，实际: '%s'", inferred)
	}
}

// 测试身份证号逻辑类型推断（18位）
func TestLogicalType_IDCard18(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	pathVarNode := node.NewRequestPathVariableNode("idcard", "")
	idcards := []string{
		"110101199001011234",
		"310101198501012345",
		"44010119920303123X",
		"510101198812120001",
	}
	for _, id := range idcards {
		pathVarNode.ObserveValue(id)
	}

	inferred, err := rule.Infer(pathVarNode)
	if err != nil {
		t.Fatalf("推断失败: %v", err)
	}

	if inferred != value.Type(value.LogicalTypeIDCard) {
		t.Errorf("身份证号应该推断为 'idcard'，实际: '%s'", inferred)
	}
}

// 测试身份证号（15位旧版）
func TestLogicalType_IDCard15(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	pathVarNode := node.NewRequestPathVariableNode("idcard", "")
	idcards := []string{
		"110101900101123",
		"310101850101234",
		"440101920303123",
	}
	for _, id := range idcards {
		pathVarNode.ObserveValue(id)
	}

	inferred, err := rule.Infer(pathVarNode)
	if err != nil {
		t.Fatalf("推断失败: %v", err)
	}

	if inferred != value.Type(value.LogicalTypeIDCard) {
		t.Errorf("15位身份证号应该推断为 'idcard'，实际: '%s'", inferred)
	}
}

// 测试银行卡号逻辑类型推断
func TestLogicalType_BankCard(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	pathVarNode := node.NewRequestPathVariableNode("bankcard", "")
	bankcards := []string{
		"6222021234567890123",
		"6225887654321098765",
		"6217001234567890123",
		"6228481234567890123",
	}
	for _, b := range bankcards {
		pathVarNode.ObserveValue(b)
	}

	inferred, err := rule.Infer(pathVarNode)
	if err != nil {
		t.Fatalf("推断失败: %v", err)
	}

	if inferred != value.Type(value.LogicalTypeBankCard) {
		t.Errorf("银行卡号应该推断为 'bankcard'，实际: '%s'", inferred)
	}
}

// 测试车牌号逻辑类型推断
func TestLogicalType_PlateNumber(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	pathVarNode := node.NewRequestPathVariableNode("plate", "")
	plates := []string{
		"京A12345",
		"沪B12345D",
		"粤B12345",
		"川A12345",
	}
	for _, p := range plates {
		pathVarNode.ObserveValue(p)
	}

	inferred, err := rule.Infer(pathVarNode)
	if err != nil {
		t.Fatalf("推断失败: %v", err)
	}

	if inferred != value.Type(value.LogicalTypePlateNumber) {
		t.Errorf("车牌号应该推断为 'plate'，实际: '%s'", inferred)
	}
}

// 测试邮政编码不再被自动识别
// 6位纯数字无法与普通数字ID、验证码、订单号等可靠区分，
// 因此从自动模式识别中移除 postalcode。
// 这些6位数字应该回退到更通用的类型（string），而不是误判为邮政编码。
func TestLogicalType_PostalCode_NotAutoDetected(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	pathVarNode := node.NewRequestPathVariableNode("postalcode", "")
	// 这些6位数字可能是邮政编码，但也可能是验证码、短ID、订单号等
	postals := []string{
		"100000",
		"518000",
		"200000",
		"510000",
	}
	for _, p := range postals {
		pathVarNode.ObserveValue(p)
	}

	inferred, err := rule.Infer(pathVarNode)
	if err != nil {
		t.Fatalf("推断失败: %v", err)
	}

	// 不应该被误判为 postalcode
	if inferred == value.Type(value.LogicalTypePostalCode) {
		t.Errorf("6位数字不应被自动识别为 'postalcode'（无法与普通数字ID区分），实际: '%s'", inferred)
	}
	// 应该回退到 string（因为没有结构化模式匹配）
	if inferred != value.Type(value.LogicalTypeString) {
		t.Errorf("6位数字应回退为 'string'，实际: '%s'", inferred)
	}
}

// 测试6位数字ID不被误判为邮政编码
// 这是异常数据兼容性的关键测试：6位数字ID（如 123456、789012）
// 不应该被错误地分类为邮政编码。
func TestLogicalType_SixDigitID_NotPostalCode(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	pathVarNode := node.NewRequestPathVariableNode("id", "")
	ids := []string{
		"123456",
		"789012",
		"345678",
	}
	for _, id := range ids {
		pathVarNode.ObserveValue(id)
	}

	inferred, err := rule.Infer(pathVarNode)
	if err != nil {
		t.Fatalf("推断失败: %v", err)
	}

	// 不应该被误判为 postalcode
	if inferred == value.Type(value.LogicalTypePostalCode) {
		t.Errorf("6位数字ID不应被误判为 'postalcode'，实际: '%s'", inferred)
	}
}

// 测试身份证号与银行卡号的区分
// 身份证号以1开头，银行卡号以3-6开头
func TestLogicalType_IDCardVsBankCard(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	// 身份证号应该识别为 idcard 而非 bankcard
	idcardNode := node.NewRequestPathVariableNode("idcard", "")
	idcardNode.ObserveValue("110101199001011234")
	idcardNode.ObserveValue("310101198501012345")
	idcardNode.ObserveValue("44010119920303123X")

	inferred, _ := rule.Infer(idcardNode)
	if inferred != value.Type(value.LogicalTypeIDCard) {
		t.Errorf("身份证号应该推断为 'idcard'，实际: '%s'", inferred)
	}

	// 银行卡号应该识别为 bankcard 而非 idcard
	bankNode := node.NewRequestPathVariableNode("bankcard", "")
	bankNode.ObserveValue("6222021234567890123")
	bankNode.ObserveValue("6225887654321098765")
	bankNode.ObserveValue("6217001234567890123")

	inferred, _ = rule.Infer(bankNode)
	if inferred != value.Type(value.LogicalTypeBankCard) {
		t.Errorf("银行卡号应该推断为 'bankcard'，实际: '%s'", inferred)
	}
}

// 测试手机号与纯整数的区分
// 手机号是11位且1开头第二位3-9，纯整数是任意数字
func TestLogicalType_PhoneVsInteger(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	// 手机号应该识别为 phone 而非 integer
	phoneNode := node.NewRequestPathVariableNode("phone", "")
	phoneNode.ObserveValue("13812345678")
	phoneNode.ObserveValue("15912345678")
	phoneNode.ObserveValue("18612345678")

	inferred, _ := rule.Infer(phoneNode)
	if inferred != value.Type(value.LogicalTypePhoneNumber) {
		t.Errorf("手机号应该推断为 'phone'，实际: '%s'", inferred)
	}

	// 短数字应该识别为 integer
	intNode := node.NewRequestPathVariableNode("id", "")
	intNode.ObserveValue("123")
	intNode.ObserveValue("456")
	intNode.ObserveValue("789")

	inferred, _ = rule.Infer(intNode)
	// 短数字不匹配手机号模式，应该回退到 string 或其他类型
	t.Logf("短数字推断结果: %s", inferred)
}

// 测试手机号格式归一化识别
// 现实中手机号常以分隔符形式出现（用户输入、展示格式），
// 应归一化后统一识别为 phone，提升异常/不规范数据兼容性。
func TestLogicalType_PhoneNumberNormalization(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	cases := []struct {
		name   string
		values []string
	}{
		{"带空格分隔", []string{"138 1234 5678", "159 1234 5678", "186 1234 5678"}},
		{"带横线分隔", []string{"138-1234-5678", "159-1234-5678", "186-1234-5678"}},
		{"带括号和空格", []string{"(+86)138 1234 5678", "(+86)159 1234 5678", "(+86)186 1234 5678"}},
		{"混合分隔符", []string{"138-1234 5678", "159-1234 5678", "186-1234 5678"}},
		{"带+86和横线", []string{"+86-138-1234-5678", "+86-159-1234-5678", "+86-186-1234-5678"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			phoneNode := node.NewRequestPathVariableNode("phone", "")
			for _, v := range c.values {
				phoneNode.ObserveValue(v)
			}

			inferred, err := rule.Infer(phoneNode)
			if err != nil {
				t.Fatalf("推断失败: %v", err)
			}

			if inferred != value.Type(value.LogicalTypePhoneNumber) {
				t.Errorf("%s 应归一化后识别为 'phone'，实际: '%s'", c.name, inferred)
			}
		})
	}
}

// 测试混合格式的手机号（部分带分隔符，部分标准）仍能识别
func TestLogicalType_MixedFormatPhone(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	phoneNode := node.NewRequestPathVariableNode("phone", "")
	phoneNode.ObserveValue("13812345678")      // 标准
	phoneNode.ObserveValue("159-1234-5678")    // 带横线
	phoneNode.ObserveValue("186 1234 5678")    // 带空格
	phoneNode.ObserveValue("abc12345")         // 噪声数据

	inferred, err := rule.Infer(phoneNode)
	if err != nil {
		t.Fatalf("推断失败: %v", err)
	}

	// 3/4 = 75% >= 60% 阈值，应识别为 phone
	if inferred != value.Type(value.LogicalTypePhoneNumber) {
		t.Errorf("混合格式手机号（3/4合法）应识别为 'phone'，实际: '%s'", inferred)
	}
}

// 测试座机号识别为 phone
// 座机号（区号+号码）是中国常见电话格式，与手机号同属电话号码语义。
// 支持多种分隔格式：横线、括号、纯数字。
func TestLogicalType_LandlinePhone(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	cases := []struct {
		name   string
		values []string
	}{
		{"3位区号-横线", []string{"010-12345678", "021-87654321", "022-12345678"}},
		{"4位区号-横线", []string{"0755-12345678", "0991-1234567", "0898-12345678"}},
		{"纯数字座机", []string{"01012345678", "02187654321", "075512345678"}},
		{"带括号区号", []string{"(010)12345678", "(021)87654321", "(0755)12345678"}},
		{"带空格", []string{"010 12345678", "021 87654321", "0755 12345678"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			phoneNode := node.NewRequestPathVariableNode("tel", "")
			for _, v := range c.values {
				phoneNode.ObserveValue(v)
			}

			inferred, err := rule.Infer(phoneNode)
			if err != nil {
				t.Fatalf("推断失败: %v", err)
			}

			if inferred != value.Type(value.LogicalTypePhoneNumber) {
				t.Errorf("%s 座机号应识别为 'phone'，实际: '%s'", c.name, inferred)
			}
		})
	}
}

// 测试手机号与座机号混合仍识别为 phone
func TestLogicalType_MobileAndLandlineMix(t *testing.T) {
	rule := NewLogicalTypeInferenceRule()

	phoneNode := node.NewRequestPathVariableNode("tel", "")
	phoneNode.ObserveValue("13812345678")      // 手机号
	phoneNode.ObserveValue("010-87654321")     // 座机号
	phoneNode.ObserveValue("15912345678")      // 手机号
	phoneNode.ObserveValue("021-12345678")     // 座机号

	inferred, err := rule.Infer(phoneNode)
	if err != nil {
		t.Fatalf("推断失败: %v", err)
	}

	// 4/4 都是电话号码，应识别为 phone
	if inferred != value.Type(value.LogicalTypePhoneNumber) {
		t.Errorf("手机号与座机号混合应识别为 'phone'，实际: '%s'", inferred)
	}
}
