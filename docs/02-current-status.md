# 当前实现状态

> 最后更新：2026-07-21

## 编译状态

**✅ 项目编译通过，所有测试通过（含竞态检测）**

## 测试覆盖率

**全量口径（`-coverpkg=./...`）总覆盖率 92.0%**（较上轮 89.6% 提升 2.4pp），`go test -race ./...` 全绿，staticcheck 全仓库清零。

本轮（2026-07-21）补全所有非辅助器 0% 函数：HttpRequestParam 全 getter/setter、curlParseError.Error、ReleasePath、Cookie/Header 节点 String 与 GetValueMetric、quickstart main() smoke test、generator/inference 辅助函数边界。删除 Deprecated 死存根 `ReverseRouter.FindNode`（始终返回 nil,nil，无调用方）。剩余未覆盖为 `assertion.go` 的 `Check` 测试辅助器失败分支（靠 `t.Errorf` 触发但不 fatal，强行补会制造"故意失败"噪音，是"测试器测自身"反模式，不纳入 100% 目标）。

| 包 | 单包覆盖率 |
|------|--------|
| **pkg/exporter** | 97.2% |
| **pkg/tree** | 94.7% |
| **pkg/node** | 91.8% |
| **pkg/router** | 90.3% |
| **pkg/request** | 91.4% |
| **pkg/inference** | 89.4% |
| **pkg/generator** | 88.9% |
| **pkg/value** | 100.0% |

## 吞吐量基线（2026-07-18，Phase 2）

环境：AMD Ryzen 9 5950X 32 核，Go 1.23.2。

| 场景 | Phase 1 末 | Phase 2 末 | 单核吞吐 |
|------|-----------|-----------|----------|
| 纯路径命中（真实流量，ID 有重复） | 1.27μs/op, 14 allocs | **0.69μs/op, 6 allocs** | **~146万 URL/s** |
| Merge（10000 ID 合并） | 1.5μs/op, 15 allocs（1000 ID） | 1.65μs/op, 7 allocs（10000 ID） | ~60万 URL/s |
| POST + JSON body | 3.55μs/op, 39 allocs | 2.68μs/op, 31 allocs | ~37万 URL/s |
| curl 解析 | 1.92μs/op, 30 allocs | 1.11μs/op, 25 allocs | ~90万 curl/s |
| 32 核并发（命中已存在变量节点） | 受 mergeMu 限制 | 160μs/op, 8 allocs | 1→8 核 12x 加速，16/32 核稳定 |

**Phase 2 瓶颈根因**（pprof）：`net/url.Parse` 占 40.87% CPU（全功能解析开销大）、`NewHttpRequestPath` 每路径段一次堆分配（占 33% 内存）、`processRoutingHeaders` 5×O(n) Get、`paths` slice append 扩容。

**Phase 2 优化手段**：
1. **`fast_url_parser`** 轻量解析替代 `net/url.Parse`——仅提取 path+query 不构造 `*url.URL`，纯路径零分配快路径；行为对齐 `net/url`（非法 `%xx` 报 `errInvalidEscape`、空 scheme 报 `errMissingScheme`，双 oracle 测试校验）
2. **`HttpRequestPath` + `paths` slice 双 `sync.Pool`**——每段 `*HttpRequestPath` 与容器 slice 复用，消除每段堆分配与 append 扩容
3. **`processRoutingHeaders` 单次遍历**——`routingHeaderList` 有序切片 + `canonicalName`，消除 5×O(n) Get
4. **`allParams` slice 预分配**——`len(pathParams)+len(params)` 一次到位避免扩容
5. **`findOrCreatePathNode` double-check 修复**——既有并发竞态（多 goroutine 重复 AddChild 导致 children 重复 key + 合并误判），未命中创建分支用 `mergeMu` 串行化 + 锁内 double-check，命中已存在节点走无锁快路径不受影响

**产品目标达成**："每秒处理几十万条 URL/cURL" —— 单核纯路径 **~146万/s**，较 Phase 1（~78万/s）提升 87%，远超目标；真实流量 ID 有重复走此路径。8 核并发 12x 加速。

**已知约束**：并发合并受 `mergeMu` 串行临界区限制。合并低频（每 N 请求触发），router 级串行化吞吐代价可接受。命中已存在变量节点的纯路径请求不进合并临界区，可完全并行（PurePath 6 allocs 无 mergeMu 开销）。

## curl 解析器真实场景健壮化 + 批量喂入（2026-07-20）

测绘平台导出的 curl 命令常带超时/重试/连接/输出等参数，原解析器把"带值 flag 的值"误当 URL，是上层集成直接阻塞的致命 bug。

### 修复的真实 bug

| 场景 | 修复前 | 修复后 |
|------|--------|--------|
| `curl --max-time 30 'http://x'` | URL="30"（致命） | URL=http://x，值 30 被消费丢弃 |
| `curl --connect-timeout 5 'http://x'` | URL="5" | URL=http://x |
| `curl -G 'http://x/search' -d 'q=go'` | body=q=go POST（语义错） | URL=http://x/search?q=go，GET，无 body |
| `curl --url 'http://explicit' http://positional` | URL=http://positional | URL=http://explicit（--url 优先） |

### curl flag 三分类

- **valueFlags**（消费下一个 token）：`--max-time`/`-m`/`--connect-timeout`/`--retry`/`--retry-delay`/`--retry-max-time`/`--max-redirs`/`--rate`/`--limit-rate`/`--speed-limit`/`--speed-time`/`--expect100-timeout`/`--resolve`/`--url`/`-o`/`--output`/`-e`/`--referer`/`-A`/`--user-agent`/`-u`/`--user`/`--cookie-jar`/`--cert`/`--key`/`--cacert`/`--capath`/`--ciphers`/`-x`/`--proxy`/`-U`/`--proxy-user`/`-b`/`--cookie`/`--dns-servers`/`--interface`/`--noproxy`/`--form`/`-F`/`--write-out`/`-w`/`--config`/`-K`
- **harmlessFlags**（跳过）：`--compressed`/`-s`/`--silent`/`-k`/`--insecure`/`-L`/`--location`/`-i`/`--include`/`-S`/`--show-error`/`-f`/`--fail`/`--fail-with-body`/`-v`/`--verbose`/`-q`/`--http1.1`/`--http2`/`-0`/`--http1.0`/`-N`/`--no-buffer`/`--tcp-nodelay`/`--tcp-fastopen`
- **未知 flag**（`--xxx`/`-x`）：整体跳过

两个 map 提到包级 `curlHarmlessFlags`/`curlValueFlags`，避免每次 `parseCurlTokens` 重建字面量分配（CurlParse 31→25 allocs）。

### 批量 fail-soft API

`pkg/router/batch.go` 新增 `ReverseRequests([]*request.HttpRequest) BatchResult` 与 `ReverseCurls([]string) BatchResult`：

- 逐条喂入，单条解析/处理失败不中断整批
- `BatchResult{Processed, Failed int; Errors []BatchError}`，`BatchError{Index, Raw, Err}`
- 失败详情上限 `maxBatchErrors=100`（超限只计数不记详情），`Raw` 截断到 `maxBatchErrorRawLen=128` 防超长撑爆日志
- 批次结束自动 `InferRequiredParams`
- 适合上层一次导出上万条 curl 的场景：坏样本被跳过并记入 Errors，不影响其余还原

## 模块完成度总览

| 模块 | 状态 | 完成度 | 说明 |
|------|------|--------|------|
| **node（节点层）** | ✅ | 98% | 所有节点类型完善，Header/Cookie两层结构，参数大小写不敏感 |
| **request（请求层）** | ✅ | 98% | URL解析、请求结构、Cookie解析、参数规范化完善 |
| **tree（树层）** | ✅ | 95% | Header/Cookie节点可视化、JSON、统计完善 |
| **router（路由层）** | ✅ | 95% | Header/Cookie路由、参数类型推断、IsNeedRequest完善、结构化日志+可观测性统计 |
| **inference（推断层）** | ✅ | 95% | 物理类型+逻辑类型+链式推断规则 |
| **value（值层）** | ✅ | 100% | 类型体系完善，ValueMetric并发安全 |
| **exporter（导出层）** | ✅ | 97% | OpenAPI 3.0.3 规范导出，路径变量还原+参数+请求体 |

## 各模块详细状态

### 1. node - 节点层 ✅

#### 已完成

- **Node接口**：完整的节点接口定义
- **BaseNode实现**：约900行，Node接口的通用实现
- **BaseNodeContext实现**：节点上下文的键值存储
- **RequestPathNode**：请求路径节点
- **RequestMethodNode**：HTTP方法节点
- **RequestContentTypeNode**：Content-Type节点
- **RequestParamNode**：查询参数节点
  - 精确参数名匹配（避免 page 匹配 page_size）
  - 参数名大小写不敏感（统一小写存储）
  - 多值参数支持（如 `?tag=go&tag=web`）
  - 默认值自动观察
  - 物理类型和逻辑类型推断
  - `ExtractValue` 支持多值提取
- **RequestPathVariableNode**：路径变量节点
  - 正则模式匹配和类型推断
  - 文件扩展名排除（.json/.xml/.html等不作为变量）
  - 逻辑类型字段和 Get/SetLogicalType 方法
- **RequestHeaderNode**：Header路由分组节点（两层结构）
  - 第一层：Header名称节点（key=headerName，如 "Accept"）
  - 第二层：Header值节点（RequestHeaderValueNode）
  - `FindOrCreateValueNode()` 查找或创建值子节点
- **RequestHeaderValueNode**：Header值节点
  - 存储规范化后的Header值
  - 值观察和统计（ValueMetric）
  - `IsMatch()` 精确匹配
- **RequestCookieNode**：Cookie路由分组节点（两层结构）
  - 第一层：Cookie名称节点（key=cookieName，如 "lang"）
  - 第二层：Cookie值节点（RequestCookieValueNode）
  - `FindOrCreateValueNode()` 查找或创建值子节点
- **RequestCookieValueNode**：Cookie值节点
  - 存储Cookie值
  - 值观察和统计（ValueMetric）
  - `IsMatch()` 精确匹配

### 2. request - 请求层 ✅

- **UrlParser**：
  - URL路径解码（`%E7%94%A8%E6%88%B7` → `用户`）
  - 参数名统一小写（HTTP参数名不区分大小写）
  - 路径段规范化（`.` 和 `..` 段过滤）
  - 多值参数解析（`?tag=go&tag=web` → 3个HttpParam）
  - 尾部斜杠自动去除
  - 连续斜杠规范化
- **Headers**：HTTP头集合
  - 大小写不敏感的 Get/Set/Has
  - 便捷方法：GetAccept/GetAuthorization/GetAuthScheme/GetXRequestedWith/GetXForwardedFor/GetXApiVersion/GetAcceptLanguage/IsAjax
  - ToHttpHeader() 转换为标准 http.Header
- **CurlParser**（新增 2026-07-17）：`ParseCurl(curl string)` 将一条 curl 命令解析为 HttpRequest
  - 手写 shell-token 切分（零外部依赖），处理单/双引号、反斜杠转义、反斜杠续行
  - 支持 `-X`/`--request`、`-H`/`--header`（多个）、`-d`/`--data`/`--data-raw`/`--data-binary`/`--data-ascii`
  - 有 `-d` 无显式 `-X` 默认 POST，无 Content-Type 时默认 `application/x-www-form-urlencoded`
  - 无害 flag 跳过（`--compressed`/`-s`/`-k`/`-L`/`--insecure`/`--silent` 等）；`-XPOST` 紧凑形式
  - **flag 三分类**（2026-07-20）：valueFlags 消费下 token（`--max-time`/`--connect-timeout`/`-A`/`-o`/`--url` 等），harmlessFlags 跳过，未知 flag 整体跳过——消除"带值 flag 的值被误当 URL"致命 bug
  - **`-G`/`--get`**：`-d` 值作为 query 串附加到 URL（含 `?` 用 `&` 拼接）而非 body
  - **`--url`**：显式 URL 优先于位置 URL
  - 网络空间测绘场景核心输入格式（抓包常以 curl 形态留存）
  - **批量喂入**（见 router 层）：`ReverseCurls([]string) BatchResult` fail-soft 逐条解析，坏样本跳过不中断
- **Cookies**：Cookie集合
  - ParseCookies() 解析 Cookie header 字符串
  - Get/Has/GetAll/String 方法
- **HttpRequestPath**：
  - 路径参数识别（`key=value` 格式检测）
  - 合法参数名校验（字母/下划线开头）
- **BodyParser**：请求体参数解析
  - application/x-www-form-urlencoded：表单编码，复用 net/url.ParseQuery，支持 URL 解码和多值
  - application/json：扁平化为 name=value，嵌套对象用点号连接（address.city），数组用索引（tags.0）
  - multipart/form-data：提取表单字段值，文件字段以文件名为值（不读取文件内容）
  - 参数名统一小写（与查询参数一致）
  - Content-Type 带 charset 时正确识别主类型
  - MaxParams 上限防止参数爆炸
  - 不支持的类型（text/plain 等）不解析

### 3. tree - 树层 ✅

- `NewTree()` ✅
- `AddNode()` ✅：根据路径添加节点，自动创建中间路径节点
- `FindNodeByPath()` ✅：根据路径查找节点
- `String()` ✅：树形文本输出（含Header/Cookie节点显示）
- `Print()` ✅：打印到标准输出
- `ToJSON()` ✅：JSON导出（含路径变量类型信息、Header/Cookie节点）
- `FromJSON()` ✅：JSON导入（含Header/Cookie节点反序列化）
- `Stats()` ✅：路由树统计信息（含HeaderNodes/HeaderValueNodes/CookieNodes/CookieValueNodes）

### 4. router - 路由层 ✅

- **ReverseRouter**：
  - `ReverseHttpRequest()` ✅：9步核心方法
    1. URL解析
    2. 路径匹配/创建（含尾部斜杠处理、URL解码）
    3. 路径参数识别（key=value 格式）
    4. 路径变量识别合并（选择性合并，固定路径保留）
    5. HTTP方法节点
    6. 查询参数节点（大小写不敏感、多值参数、类型推断）
    7. Content-Type节点
    8. Header路由节点（两层结构：名称分组→值子节点）
    9. Cookie路由节点（两层结构：名称分组→值子节点）
  - `IsNeedRequest()` ✅：检查路径、方法、参数、Content-Type、Header路由、Cookie路由
  - **`ReverseRequests`/`ReverseCurls`** ✅（2026-07-20）：批量 fail-soft 喂入，单条失败不中断整批，`BatchResult` 聚合 Processed/Failed/Errors（详情上限 100、Raw 截断 128），批次结束自动 `InferRequiredParams`
  - **Header路由** ✅：
    - 支持的路由Header：Accept、Authorization、X-Api-Version、Accept-Language、X-Requested-With
    - Header值规范化：
      - Accept → 取第一个MIME类型，忽略质量因子
      - Authorization → 只取认证方案（Bearer/Basic/Token）
      - Accept-Language → 取第一个语言标签
      - X-Api-Version / X-Requested-With → 原值不变
  - **Cookie路由** ✅：解析Cookie header，每个Cookie作为路由维度
  - **参数类型推断** ✅：在参数节点创建时和观察值时自动推断物理类型和逻辑类型
  - **MergeConfig** ✅：可配置的合并阈值和模式相似度阈值
  - **PatternDetector** ✅：模式检测器
    - 支持 integer/uuid/float/date/email/ip/version/alphanumeric/similar_length_strings 模式
    - 前缀/后缀模式：结构化模式匹配不足（<0.5）时，检测公共前缀+变量后缀 / 变量前缀+公共后缀
  - **RouterLogger** ✅：结构化日志（封装 log/slog）
    - 五级日志：Debug（决策细节）/Info（关键事件）/Warn（异常兼容）/Error（错误）/Off
    - 默认 Warn 级别保持低噪音，调试时用 SetLogLevel(LogLevelDebug)
    - 可配置输出流，nil 安全
    - 关键埋点：请求处理、路径变量识别、模式检测、参数创建、类型推断、必需性推断、合并决策、异常 body
  - **RouterStats** ✅：可观测性统计指标（atomic 线程安全）
    - 11 项计数：请求数、路径变量数、模式检测数、参数数、类型推断数、body参数数、必需参数数、合并尝试/跳过数、警告/错误数
    - GetStats() 返回快照，支持 JSON 序列化，Reset() 清零

### 5. inference - 推断层 ✅

- **PhysicalTypeInferenceRule** ✅：推断算法完整实现
- **LogicalTypeInferenceRule** ✅：
  - 支持 UUID、IP地址、邮箱、日期、时间、日期时间、URL、JSON、XML 模式
  - 支持百分比、货币、精确小数等数值扩展类型
  - 枚举值类型检测
- **ChainTypeInferenceRule** ✅：
  - 链式组合多个推断规则
  - `InferPhysicalAndLogical()` 方法分别获取物理和逻辑类型

### 6. value - 值层 ✅

- **ValueMetric** ✅：并发安全（sync.RWMutex）
- **LogicalType 类型体系** ✅：25种逻辑类型定义

### 7. exporter - 导出层 ✅

- **OpenAPIExporter** ✅：将路由树导出为 OpenAPI 3.0.3 规范
  - 路径变量还原为 `{var}` 形式拼进 path
  - 四类参数：query（查询参数）、path（路径变量，required=true）、header（Header 路由，同名去重）、cookie（Cookie 路由，同名去重）
  - POST/PUT/PATCH 请求体生成 object schema（含 properties 和 required）
  - Content-Type 去 charset 规范化（application/json; charset=utf-8 → application/json）
  - 物理/逻辑类型映射为 OpenAPI schema type+format（date→date-time、email→email、uuid→uuid、url→uri 等）
  - operationId 自动生成（`{method}_{sanitized_path}`）
  - 路径按字母序稳定输出，多次导出结果一致
  - 可配置：标题、版本、描述、ServerURL、是否包含可选参数

## 测试覆盖

| 测试文件 | 状态 | 覆盖内容 |
|----------|------|----------|
| `base_node_test.go` | ✅ 通过 | 基本属性、父子关系、查找、遍历、克隆、合并、并发 |
| `base_node_context_test.go` | ✅ 通过 | 键值操作、并发、边界条件 |
| `request_nodes_test.go` | ✅ 通过 | 各请求节点类型、Header/Cookie两层结构、参数大小写不敏感、多值参数、文件扩展名排除 |
| `url_parser_test.go` | ✅ 通过 | URL解析、路径参数检测、URL解码、参数大小写规范化、多值参数、路径遍历 |
| `request_test.go` | ✅ 通过 | Headers/Cookies所有方法、HttpRequest/HttpParam/HttpRequestPath |
| `body_parser_test.go` | ✅ 通过 | 表单/JSON/multipart解析、嵌套扁平化、多值、charset、空body、非法JSON、参数名小写、MaxParams上限 |
| `reverse_router_test.go` | ✅ 通过 | 基本路径、方法、参数、Content-Type、路径变量合并、Header路由、Cookie路由、参数边界条件、路径边界条件、IsNeedRequest(含Header/Cookie)、前缀/后缀合并、Body参数解析(表单/JSON/multipart) |
| `logger_test.go` | ✅ 通过 | 统计指标（请求/路径变量/合并/参数/body/类型推断/必需性/错误/警告/Reset/JSON）、日志级别、Off静默、路径变量日志、nil安全、综合统计 |
| `tree_test.go` | ✅ 通过 | AddNode、FindNodeByPath、可视化、JSON导出/导入、统计、Header/Cookie节点可视化/JSON/统计 |
| `value_test.go` | ✅ 通过 | ValueMetric操作、并发安全 |
| `physical_type_inference_rule_test.go` | ✅ 通过 | 整数/浮点/布尔/字符串/null推断、混合类型 |
| `logical_type_inference_rule_test.go` | ✅ 通过 | UUID/日期/时间/日期时间/邮箱/IP/URL/百分比/货币/JSON/XML/枚举/精确小数推断 |
| `chain_type_inference_rule_test.go` | ✅ 通过 | 链式推断、自定义规则链、物理+逻辑分别推断 |
| `openapi_test.go` | ✅ 通过 | OpenAPI基础结构、ServerURL、路径变量还原、固定路径保留、query/header/cookie参数、必需参数、请求体、charset规范化、多方法、schema类型映射、operationId、稳定排序、空树/nil树 |

## 已修复的问题

1. ✅ RequestPathVariableNode 编译错误
2. ✅ 测试文件调用签名不匹配
3. ✅ RequestPathRouter/RequestParamRouter 类型断言bug
4. ✅ RequestPathRouter 返回值类型错误
5. ✅ RequestContentTypeRouter 复制了错误代码
6. ✅ Tree.AddNode() 未实现
7. ✅ ReverseRouter.ReverseHttpRequest() 未实现
8. ✅ ReverseRouter.IsNeedRequest() 未实现
9. ✅ ValueMetric 缺并发安全保护
10. ✅ PhysicalTypeInferenceRule.Infer() 未连接节点上下文
11. ✅ VisitChildren 并发数据竞态
12. ✅ 路径合并判定中 similar_length_strings 被短路（原 shouldMergeAsVariable，现 findMergeableSiblings）
13. ✅ RequestParamNode.IsMatch() 对非必需参数过于宽松
14. ✅ 路径参数识别（key=value 格式路径段）
15. ✅ ReverseRouter.urlParser 并发竞态 bug
16. ✅ 混合路径错误合并（选择性合并策略）
17. ✅ Header/Cookie节点结构设计问题（改为两层结构）

## 新增功能（本次迭代）

1. ✅ **Header路由支持**：Accept/Authorization/X-Api-Version/Accept-Language/X-Requested-With
2. ✅ **Header值规范化**：Accept取首MIME、Authorization取方案、Accept-Language取首标签
3. ✅ **Cookie路由支持**：自动解析Cookie header作为路由维度
4. ✅ **两层Header/Cookie树结构**：名称分组→值子节点，便于变量合并
5. ✅ **参数名大小写不敏感**：Page/page/PAGE 统一为 page
6. ✅ **多值参数支持**：`?tag=go&tag=web` 合并到同一参数节点
7. ✅ **URL编码自动解码**：路径段和参数值自动URL解码
8. ✅ **参数值类型推断**：参数节点创建时和观察值时自动推断物理/逻辑类型
9. ✅ **路径遍历安全处理**：`.` 和 `..` 路径段自动过滤
10. ✅ **文件扩展名排除**：有扩展名的路径不作为变量合并
11. ✅ **IsNeedRequest完善**：检查Header路由和Cookie路由
12. ✅ **RequestParamNode默认值自动观察**：创建时自动观察默认值
13. ✅ **中国特有格式识别**：手机号/身份证号/银行卡号/车牌号（邮政编码已移除自动识别，避免6位数字误判）
14. ✅ **参数节点类型推断修复**：PhysicalTypeInferenceRule和LogicalTypeInferenceRule支持RequestParamNode
15. ✅ **模式检测优先级优化**：具体模式（phone/idcard等）优先于通用模式（integer）
16. ✅ **异常数据兼容**：混合合法/非法格式仍能识别（部分匹配阈值）
17. ✅ **邮政编码误判修复**：移除 postalcode 纯正则自动识别，6位数字回退为 integer
18. ✅ **长数字串物理类型降级**：16位及以上纯数字（银行卡16-19位、身份证18位、超长ID>19位）降级为 string，避免 int64 溢出，符合标识符语义
19. ✅ **手机号格式归一化**：识别带空格/横线/括号分隔的手机号（138-1234-5678、138 1234 5678），归一化后匹配
20. ✅ **中文/相似串合并突破规则**：6个及以上长度相似的字符串兄弟节点（城市名、人名等）合并为变量，少量（<6）固定路径名仍保护
21. ✅ **科学计数法识别**：1e5、1.5E-3 等科学计数法识别为 float 物理类型
22. ✅ **座机号识别**：区号+号码格式（010-12345678、(0755)12345678）识别为 phone，与手机号统一
23. ✅ **十六进制数值识别**：0x1A、0xDEADBEEF 等十六进制识别为 integer 物理类型
24. ✅ **必需参数自动推断**：InferRequiredParams 基于参数出现频率推断必需性（出现率>=0.9判定必需），样本不足保持默认
25. ✅ **路由树完整序列化/反序列化**：JSON 含物理/逻辑类型、必需性、出现计数、默认值、多值标记，往返一致
26. ✅ **参数节点可视化增强**：路由树显示标注必需参数（page*）和逻辑类型
27. ✅ **前缀/后缀合并策略**：相同前缀+变量后缀（user_001/user_002/user_003）或不同前缀+相同后缀（001_user/002_user）合并为变量，变量名基于公共前缀/后缀生成（user_id），生成精确正则（user_[0-9]+），无关固定路径保留不误合并
28. ✅ **请求体参数解析**：BodyParser 支持 application/x-www-form-urlencoded / application/json / multipart/form-data 三种格式，按 Content-Type 分发，参数名小写化，JSON 嵌套用点号连接（address.city）数组用索引（tags.0），multipart 文件字段以文件名为值，不支持的类型（text/plain 等）不解析，MaxParams 防参数爆炸
29. ✅ **OpenAPI 3.0.3 导出**：pkg/exporter 包将路由树导出为标准 OpenAPI 规范，路径变量还原为 {var}，query/path/header/cookie 四类参数，POST body 生成 schema，Content-Type 去 charset 规范化，operationId 自动生成，路径按字母序稳定输出，可被 Swagger UI/Redoc 直接渲染
30. ✅ **结构化日志（RouterLogger）**：封装 log/slog，五级日志（Debug/Info/Warn/Error/Off），默认 Warn 低噪音，关键决策埋点（请求处理、变量识别、模式检测、类型推断、必需性推断、合并决策、异常 body），可配置输出流，nil 安全
31. ✅ **可观测性统计（RouterStats）**：11 项 atomic 计数指标（请求数/路径变量数/模式检测数/参数数/类型推断数/body参数数/必需参数数/合并尝试跳过数/警告错误数），GetStats 快照支持 JSON 序列化，Reset 清零，线程安全

## 中国特有格式识别能力

| 格式 | LogicalType | 正则特征 | 示例 |
|------|-------------|----------|------|
| 手机号 | phone | 11位，1开头第二位3-9，支持+86前缀 | 13812345678, +8613812345678 |
| 身份证号(18位) | idcard | 6地区码+8出生日期+3顺序码+1校验位 | 110101199001011234, 11010119900101123X |
| 身份证号(15位) | idcard | 6地区码+6出生日期+3顺序码（旧版） | 110101900101123 |
| 银行卡号 | bankcard | 16-19位纯数字，首位3-6 | 6222021234567890123 |
| 车牌号 | plate | 省份汉字+字母+5-6位字母数字 | 京A12345, 沪B12345D |

> **邮政编码(postalcode)已从自动识别中移除**：6位纯数字无法与普通数字ID、验证码、
> 订单号等可靠区分（`123456`、`789012` 会被100%误判为邮政编码）。当前6位数字回退
> 到 integer 模式，生成 `{xxx_id}` 变量名。`LogicalTypePostalCode` 常量保留待用。

### 格式区分机制

- **身份证号 vs 银行卡号**：身份证号以1开头且有日期结构，银行卡号以3-6开头
- **手机号 vs 纯整数**：手机号是11位且1[3-9]开头，纯整数任意长度
- **具体模式优先**：PatternDetector把具体格式放前面，通用integer放最后

### 异常数据兼容

- 混合合法/非法手机号（3/4合法）→ 物理类型integer（宽匹配），逻辑类型phone（60%阈值）
- 参数值单次观察也能推断（1/1=100%满足阈值）
- 两层推断协同：物理层用宽模式合并，逻辑层精确识别语义

## 仍需完善的功能

- [x] 路由树的完整序列化/反序列化（含类型信息）— ToJSON/FromJSON 已含物理/逻辑类型、必需性、出现计数、默认值、多值标记，往返一致
- [x] 路径变量合并回退机制（共存式软回退）— 不符合模式的新值作为固定路径与变量节点共存，不破坏已有合并；硬回退（拆分已合并节点）风险高场景罕见不实现
- [x] 更多的合并策略（如基于前缀/后缀的合并）— detectPrefixPattern/detectSuffixPattern 识别公共前缀+变量后缀 / 不同前缀+公共后缀，生成精确正则和基于前缀/后缀的变量名
- [x] RequestParamNode 必需参数自动推断 — InferRequiredParams 基于出现频率推断，阈值0.9
- [x] Body参数解析（POST表单数据、JSON body）— BodyParser 支持表单/JSON/multipart，按 Content-Type 分发，与查询参数合并处理，参数名小写化
- [x] OpenAPI/Swagger 格式导出 — pkg/exporter.OpenAPIExporter 输出 OpenAPI 3.0.3，路径变量还原+四类参数+请求体schema，可被 Swagger UI/Redoc 渲染
- [x] 更多日志和可观测性 — RouterLogger 结构化日志（log/slog，五级，默认 Warn）+ RouterStats 11项 atomic 统计指标（GetStats 快照/JSON/Reset），关键决策埋点
