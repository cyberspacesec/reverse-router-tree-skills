# 核心算法设计：ReverseHttpRequest

## 算法概述

`ReverseHttpRequest` 是整个项目的核心方法，它的作用是：**将一个HTTP请求逆向工程到路由树中**，即在路由树中找到或创建对应的节点，同时识别路径变量和参数模式。

## 输入/输出

```go
func (x *ReverseRouter) ReverseHttpRequest(request *request.HttpRequest) error
```

- **输入**：一个HTTP请求（URL、方法、Headers、Body）
- **输出**：无（直接修改路由树）
- **副作用**：
  - 在路由树中创建新节点
  - 合并相似路径为路径变量节点
  - 更新节点的请求计数
  - 触发类型推断
  - 处理Header和Cookie路由

## 算法步骤（9步）

### 第1步：解析URL

```go
paths, params, err := urlParser.Parse()
```

将URL解析为有序的路径段数组和参数列表。

处理：
- URL编码自动解码（`%E7%94%A8%E6%88%B7` → `用户`）
- 参数名统一小写（`Page` → `page`）
- 路径遍历过滤（`.` 和 `..` 段被忽略）
- 多值参数展开（`?tag=go&tag=web` → 2个HttpParam）

### 第2步：路径匹配/创建

对于路径段 `["api", "users", "123"]`：

```
root → api → users → 123
```

逐段在路由树中查找：
- **如果找到匹配的路径节点**：继续往下
- **如果没找到**：
  - 检查是否有路径变量节点可以匹配（使用 `IsMatch()` 严格匹配）
  - 如果都没有，创建新的路径节点
  - 检查是否需要合并兄弟节点为路径变量

### 第3步：路径变量识别（核心难点）

当一个父节点下有**多个同层兄弟节点**时，可能需要合并为路径变量：

**触发条件**：
- 同一父节点下，相同类型的子节点数量超过阈值（默认3个）
- 这些子节点的值具有相似的模式（如都是数字、都是UUID）

**选择性合并策略**：
1. 使用 `PatternDetector` 检测兄弟节点的模式
2. `similar_length_strings` 模式默认不合并（如 admin/manager/guest）；但当兄弟节点数 >= `SimilarLengthBreakThreshold`（默认6）时突破合并（城市名/人名等变量值集合）
3. 整体匹配率 >= 0.8：合并全部兄弟节点
4. 整体匹配率 0.4-0.8：只合并匹配模式的子集（保留固定路径）
5. 整体匹配率 < 0.4：不合并

**示例**：
```
合并前：                    合并后：
users                      users
 ├── list                   ├── list [保留]
 │    └── GET               ├── create [保留]
 ├── create                 └── {users_id} (type=integer)
 │    └── GET                    ├── GET
 ├── 101                        └── ...
 │    └── GET
 ├── 102
 │    └── GET
 └── 103
      └── GET
```

### 第4步：HTTP方法节点

在路径节点下创建方法节点：
```
users
 ├── GET
 └── POST
```

### 第5步：参数节点

在方法节点下创建参数节点，参数来源有三：查询参数、路径嵌入参数、请求体参数：
- **大小写不敏感**：`Page=1` 和 `page=2` 合并到同一节点
- **多值参数**：`?tag=go&tag=web` 记录多个值
- **类型推断**：观察值后自动推断物理类型和逻辑类型
- **请求体参数**：POST/PUT/PATCH 的 body 由 `BodyParser` 按 Content-Type 解析（表单/JSON/multipart），与查询参数合并处理

```
POST /api/users?page=1 (body: name=alice&age=30)

users
 └── POST
      ├── page [Param, type=integer]      ← 查询参数
      ├── name [Param]                    ← body 参数
      └── age [Param]                     ← body 参数
```

JSON body 的嵌套会被扁平化为点号连接的参数名：
```
body: {"user":{"name":"bob"},"tags":["vip"]}

POST
 ├── user.name [Param]
 └── tags.0 [Param]
```

### 第6步：Content-Type节点

仅对 POST/PUT/PATCH 方法创建 Content-Type 节点：
```
POST /api/users (Content-Type: application/json)

users
 └── POST
      └── application/json [ContentType]
```

### 第7步：Header路由节点

处理影响路由决策的Header，使用两层结构：

```
GET /api/data (Accept: application/json)
GET /api/data (Accept: text/html)

data
 └── GET
      └── Accept [Header]
           ├── application/json [HeaderValue]
           └── text/html [HeaderValue]
```

Header值规范化：
- Accept → 取第一个MIME类型，忽略质量因子
- Authorization → 只取认证方案（Bearer/Basic/Token）
- Accept-Language → 取第一个语言标签
- X-Api-Version / X-Requested-With → 原值不变

### 第8步：Cookie路由节点

解析Cookie header，每个Cookie作为路由维度：

```
GET /api/home (Cookie: lang=zh-CN)
GET /api/home (Cookie: lang=en-US)

home
 └── GET
      └── lang [Cookie]
           ├── zh-CN [CookieValue]
           └── en-US [CookieValue]
```

### 第9步：增加请求计数

增加路径节点和方法节点的请求计数。

## IsNeedRequest 算法

```go
func (x *ReverseRouter) IsNeedRequest(request *request.HttpRequest) bool
```

判断某个URL是否还需要请求，检查维度：

1. **路径**：找不到对应节点 → 需要请求
2. **方法**：找不到方法节点 → 需要请求
3. **参数**：参数名不存在 → 需要请求
4. **Content-Type**：Content-Type不存在 → 需要请求
5. **Header路由**：Header名称或值不存在 → 需要请求
6. **Cookie路由**：Cookie名称或值不存在 → 需要请求
7. **请求计数**：已请求过 → 不需要请求

这个方法的核心价值是**减少重复请求**，让爬虫/扫描器只请求真正需要的URL。

## 必需参数推断算法

```go
func (x *ReverseRouter) InferRequiredParams() int
```

黑盒场景下无法直接知道参数是否必需，通过统计参数在同一路由请求中的**出现频率**来推断。

### 推断依据

```
出现率 = 参数出现次数 (presenceCount) / 路由总请求次数 (methodNode.GetRequestCount)

出现率 >= RequiredParamThreshold (默认0.9) → 必需
出现率 <  阈值                              → 可选
样本不足 (总请求次数 <= 1)                  → 保持默认（不轻易判定）
```

### 示例

```
GET /api/users?page=1                 (page 出现)
GET /api/users?page=2&size=10         (page, size 出现)
GET /api/users?page=3&size=20         (page, size 出现)
...（共10次请求，page必现，size出现6次，callback出现2次）

InferRequiredParams() 后：
  page     10/10=1.0  → 必需   显示为 page*
  size      6/10=0.6  → 可选
  callback  2/10=0.2  → 可选
```

### 使用时机

建议在路由树构建完成、导出/序列化之前调用，以获得最准确的推断结果（样本越多越准）。
该方法遍历所有方法节点统一推断，返回推断的必需参数数量。

## 路径边界条件处理

| 边界条件 | 示例 | 处理方式 | 处理位置 |
|----------|------|----------|----------|
| 尾部斜杠 | `/api/users/` | Trim(path, "/") | UrlParser.Parse() |
| 连续斜杠 | `//api///users` | 循环替换 // | UrlParser.Parse() |
| URL编码 | `%E7%94%A8%E6%88%B7` | url.PathUnescape() | normalizePathSegment() |
| 路径遍历 | `.` 和 `..` | 过滤忽略 | normalizePathSegment() |
| 文件扩展名 | `data.json` | 排除不作为变量 | hasFileExtension() |
| 路径参数 | `action=delete` | 识别为参数 | HttpRequestPath.detectPathParam() |

## 关键设计决策

### 1. 路径变量合并时机

- **即时合并**：每次添加新路径时检查是否需要合并（当前方案）
- 设置较高的阈值（默认3个兄弟节点）避免误合并

### 2. 选择性合并策略

- `similar_length_strings` 模式永不合并
- 整体匹配率高（>=0.8）全部合并
- 整体匹配率中等（0.4-0.8）只合并匹配模式的子集
- 保留固定路径节点（如 list、create）
- **前缀/后缀合并**：当结构化模式（integer/uuid 等）匹配率不足（<0.5）时，回退检测公共前缀+变量后缀（`user_001`/`user_002`）或不同前缀+公共后缀（`001_user`/`002_user`），匹配率 >=0.6 时合并

### 3. Header/Cookie两层结构

- 第一层：名称分组节点（key=名称，如 "Accept"）
- 第二层：值子节点（key=值，如 "application/json"）
- 好处：同一名称的不同值作为兄弟节点，便于后续变量合并

### 4. 参数名大小写不敏感

- HTTP参数名不区分大小写是常见约定
- 在 UrlParser.Parse() 和 RequestParamNode 构造函数中统一小写
- IsMatch() 使用 EqualFold 匹配

### 5. 文件扩展名排除

- 有文件扩展名的路径段（如 `data.json`、`style.css`）通常是固定资源
- 在 RequestPathVariableNode.IsMatch() 中排除
- 但有正则模式的变量节点不检查扩展名（模式优先）

## 中国特有格式识别

### 支持的格式

| 格式 | 模式名 | LogicalType | 正则特征 |
|------|--------|-------------|----------|
| 手机号 | phone | phone | `(?:\+?86|0086)?1[3-9]\d{9}` |
| 座机号 | phone | phone | `0\d{2,3}[1-9]\d{6,7}`（区号+号码，与手机号统一） |
| 身份证号 | idcard | idcard | 18位含日期结构，或15位旧版 |
| 银行卡号 | bankcard | bankcard | `[3-6]\d{15,18}` |
| 车牌号 | plate | plate | `[\p{Han}][A-Z][A-Z0-9]{5,6}` |

> **注意**：邮政编码(postalcode)已从自动识别中移除。6位纯数字无法与普通数字ID、
> 验证码、订单号等可靠区分，纯正则识别误判率太高（如 `123456`、`789012` 会被
> 100%误判为邮政编码）。`LogicalTypePostalCode` 常量保留以供未来基于参数名语义
> 的识别使用，但当前不参与自动模式匹配。

### 模式检测优先级

PatternDetector 的模式顺序至关重要——更具体的模式放在前面，通用模式（integer）放最后：

```
1. uuid          (最具体，有明确结构)
2. email         (@ 和域名)
3. ip            (点分十进制)
4. date          (连字符日期)
5. phone         (手机号11位特定前缀 / 座机号0+区号)
6. idcard        (18位含日期)
7. bankcard      (16-19位特定开头)
8. plate         (汉字+字母)
9. version       (v+数字)
10. alphanumeric (字母+数字)
11. float        (带小数点)
12. integer      (纯数字，最通用，最后)
```

当多个模式都匹配时（如身份证号同时匹配 integer 和 idcard），DetectPattern 选择匹配率最高的，匹配率相同时保留先出现的（更具体的）。

### 格式区分机制

**身份证号 vs 银行卡号**：
- 身份证号以 1 开头，且第7-14位是合法出生日期
- 银行卡号以 3-6 开头，无日期结构
- 二者正则互斥，不会误判

**手机号 vs 纯整数**：
- 手机号：11位，1开头，第二位3-9
- 纯整数：任意长度数字
- 11位手机号匹配 phone（在前），不回退到 integer

### 两层推断协同

物理类型推断和逻辑类型推断协同工作：

```
路径变量合并场景：
  /api/users/13812345678
  /api/users/15912345678
  /api/users/18612345678

物理层（PatternDetector）：
  → 检测到 phone 模式匹配率 100%
  → 创建变量节点，模式正则 = (?:\+?86)?1[3-9][0-9]{9}
  → 变量名 = users_phone

逻辑层（LogicalTypeInferenceRule）：
  → 3个值都匹配 phone 正则
  → 逻辑类型 = phone

参数值推断场景：
  /api/sms/send?phone=13812345678

参数节点创建时：
  → ObserveValue("13812345678")
  → InferPhysicalAndLogical()
  → 物理类型 = integer（纯数字）
  → 逻辑类型 = phone（匹配手机号正则）
```

### 异常数据兼容

当数据混合了合法与非法格式时：

```
/api/users/13812345678   (合法手机号)
/api/users/15912345678   (合法手机号)
/api/users/12345678901   (非法手机号，第二位是2)
/api/users/18612345678   (合法手机号)

物理层：
  phone 匹配率 3/4 = 0.75
  integer 匹配率 4/4 = 1.0
  → 选择 integer（更高匹配率），变量名 = users_id

逻辑层：
  phone 匹配率 3/4 = 0.75 >= 0.6 阈值
  → 逻辑类型 = phone

结果：{users_id} [Var, phone]
```

这种设计确保了：
- 即使有异常数据，仍能合并为变量（物理层宽容）
- 语义识别仍精确（逻辑层严格但有阈值容忍）

### 长数字串物理类型降级

16位及以上纯数字串（银行卡号16-19位、身份证号18位、超长业务ID>19位）在物理类型
推断中降级为 `string` 而非 `integer`：

```
身份证号 110101199003072314 (18位)
  → 物理类型 = string（标识符语义，非算术整数）
  → 逻辑类型 = idcard

银行卡号 6222021234567890123 (19位)
  → 物理类型 = string
  → 逻辑类型 = bankcard

超长ID 1234567890123456789012345 (25位)
  → 物理类型 = string
  → 逻辑类型 = string
```

理由：
- 这些长度的数字串本质是标识符，业务系统普遍以 string 存储
- int64 最大值 9223372036854775807（19位），16位以上存在溢出风险
- 16位是"算术整数"与"标识符数字串"的合理分界线
- 逻辑层仍识别 idcard/bankcard 等语义类型

短数字（≤15位，如手机号11位、普通ID）仍识别为 `integer`。

### 手机号格式归一化

现实中电话号码常以分隔符形式出现（用户输入、展示格式），逻辑推断在匹配前做归一化：

```
手机号：
138-1234-5678  → 去除横线 → 13812345678 → 匹配 phone
138 1234 5678  → 去除空格 → 13812345678 → 匹配 phone
(+86)138-1234 5678 → 去除分隔符 → +861381234 → 匹配 phone

座机号：
010-12345678   → 去除横线 → 01012345678 → 匹配 phone
(0755)12345678 → 去除括号 → 075512345678 → 匹配 phone
021 87654321   → 去除空格 → 02187654321 → 匹配 phone

归一化规则：保留数字和 + 号，去除空格、横线、括号、点等分隔符
```

手机号与座机号统一识别为 `phone` 逻辑类型，因为业务上二者同属电话号码，常混用。

混合格式（部分标准、部分带分隔符，或手机与座机混合）仍能识别，只要匹配率 >= 60%。

### 相似长度字符串合并突破规则

`similar_length_strings` 模式（长度相似但无结构化模式）默认不合并，避免固定路径名
（admin/manager/guest）被误合并。但当同层兄弟节点数量 >= `SimilarLengthBreakThreshold`
（默认6）时突破此规则：

```
/api/city/北京、上海、广州、深圳、杭州、成都  (6个，>=6)
  → 合并为 {var_city} [Var, string]

/api/roles/admin、manager、guest  (3个，<6)
  → 不合并，保留为固定路径
```

理由：大量长度相似的字符串兄弟节点强烈暗示是变量值集合（城市名、人名、商品名等），
固定路由名很少超过5个同层。阈值可通过 `MergeConfig.SimilarLengthBreakThreshold` 配置，
设为 0 禁用此突破规则。

### 前缀/后缀合并策略

现实业务中常见"固定前缀 + 变量后缀"或"变量前缀 + 固定后缀"的命名约定，这类值集合
没有结构化模式（不像纯数字/UUID），但具有明显的公共子串特征：

```
前缀模式：
  /api/user_001、/api/user_002、/api/user_003
  → 公共前缀 user_ + 变量后缀 001/002/003
  → 合并为 {user_id} [Var, string]，正则 user_[0-9]+

后缀模式：
  /api/001_user、/api/002_user、/api/003_user
  → 变量前缀 001/002/003 + 公共后缀 _user
  → 合并为 {user_id} [Var, string]，正则 [0-9]+_user
```

**触发条件**（在 `DetectPattern` 中）：
- 结构化模式（integer/uuid/phone 等）匹配率不足（`bestRatio < 0.5`）
- 公共前缀/后缀占值长度的比例 >= 0.6（`detectPrefixPattern`/`detectSuffixPattern`）
- 否则回退到 `similar_length_strings` 的默认不合并行为

**变量名生成**：
- 前缀模式：`trimTrailingDigits(公共前缀)` 去掉末尾数字，再 `trimTrailingSeparator` 去掉末尾分隔符，加 `_id`
  - `user_001` → 前缀 `user_` → trim 数字仍是 `user_` → trim 分隔符 → `user` → `user_id`
  - `ORD-2024-001` → 前缀 `ORD-2024-` → trim 数字 → `ORD-2024` → `ORD-2024_id`
- 后缀模式：对公共后缀做对称处理（去开头数字和分隔符）

**正则生成**：用 `regexp.QuoteMeta` 转义公共前缀/后缀，拼接变量部分 `[0-9]+`，使
`IsMatch` 能严格匹配同类新值。

**误合并防护**：前缀/后缀模式只合并真正匹配的兄弟节点，无关固定路径（如 `list`、
`create`）因不共享公共前缀/后缀而保留，不会误合并。

## 日志与可观测性

### 逆向过程日志

`ReverseHttpRequest` 在关键决策点输出结构化日志（封装 `log/slog`）：

| 阶段 | 级别 | 日志内容 |
|------|------|----------|
| 请求开始/完成 | Debug | url、method、参数数、body参数数 |
| 路径变量识别 | Info | parent、var_name、pattern、physical_type、logical_type、merged_count |
| 模式检测 | Debug | parent、pattern、similarity、values数 |
| 合并决策 | Debug | 尝试（parent、children数）/ 跳过（可合并不足） |
| 参数创建 | Debug | method、param、value、physical_type、logical_type |
| 必需性推断 | Info | required_count、threshold；Debug 级别记录每个标记为必需的参数 |
| 异常 body | Warn | url、content_type、error |
| 处理错误 | Error | url、error |

默认 Warn 级别（仅警告和错误），调试时 `SetLogLevel(LogLevelDebug)` 查看全链路决策。

### 逆向效果统计

`RouterStats` 用 atomic 计数器量化逆向效果，便于评估覆盖率与质量：

```
处理 1000 个请求 → 识别 12 个路径变量、创建 45 个参数、5 次合并跳过（固定路径）
→ 必需参数推断：8 个必需 → 警告 2（异常 body 已兼容）→ 错误 0
```

`merge_skipped` 高说明有大量固定路径未被误合并（健康）；
`warnings` 反映异常数据兼容次数；
`path_variables_identified / requests_processed` 反映变量识别密度。


