package value

// 整个系统中的类型
type Type string

// 物理类型
type PhysicalType string

const (
	PhysicalTypeString  PhysicalType = "string"
	PhysicalTypeInteger PhysicalType = "integer"
	PhysicalTypeFloat   PhysicalType = "float"
	PhysicalTypeBoolean PhysicalType = "boolean"
	PhysicalTypeArray   PhysicalType = "array"
	PhysicalTypeObject  PhysicalType = "object"
	PhysicalTypeNull    PhysicalType = "null"
)

// LogicalType 表示值的逻辑类型，用于在物理类型之上提供更具体的语义
type LogicalType string

const (
	// 基本逻辑类型
	LogicalTypeString  LogicalType = "string"  // 普通字符串
	LogicalTypeInteger LogicalType = "integer" // 整数
	LogicalTypeFloat   LogicalType = "float"   // 浮点数
	LogicalTypeBoolean LogicalType = "boolean" // 布尔值
	LogicalTypeArray   LogicalType = "array"   // 数组
	LogicalTypeObject  LogicalType = "object"  // 对象
	LogicalTypeNull    LogicalType = "null"    // 空值

	// 扩展字符串逻辑类型
	LogicalTypeDate      LogicalType = "date"      // 日期，如 "2023-01-01"
	LogicalTypeTime      LogicalType = "time"      // 时间，如 "13:45:30"
	LogicalTypeDateTime  LogicalType = "datetime"  // 日期时间，如 "2023-01-01T13:45:30Z"
	LogicalTypeEmail     LogicalType = "email"     // 电子邮件地址
	LogicalTypeURL       LogicalType = "url"       // URL地址
	LogicalTypeUUID      LogicalType = "uuid"      // UUID字符串
	LogicalTypeRegex     LogicalType = "regex"     // 正则表达式
	LogicalTypeJSON      LogicalType = "json"      // JSON格式字符串
	LogicalTypeXML       LogicalType = "xml"       // XML格式字符串
	LogicalTypeIPAddress LogicalType = "ipaddress" // IP地址

	// 扩展数值逻辑类型
	LogicalTypeDecimal    LogicalType = "decimal"    // 精确小数
	LogicalTypeCurrency   LogicalType = "currency"   // 货币
	LogicalTypePercentage LogicalType = "percentage" // 百分比

	// 其他特殊逻辑类型
	LogicalTypeEnum      LogicalType = "enum"      // 枚举值
	LogicalTypeBinary    LogicalType = "binary"    // 二进制数据
	LogicalTypeReference LogicalType = "reference" // 引用类型

	// 中国特有逻辑类型
	LogicalTypePhoneNumber LogicalType = "phone"       // 手机号码，如 13812345678
	LogicalTypeIDCard      LogicalType = "idcard"      // 身份证号码，18位或15位
	LogicalTypeBankCard    LogicalType = "bankcard"    // 银行卡号，16-19位
	LogicalTypePlateNumber LogicalType = "plate"       // 车牌号，如 京A12345
	LogicalTypePostalCode  LogicalType = "postalcode"  // 邮政编码，6位数字
)
