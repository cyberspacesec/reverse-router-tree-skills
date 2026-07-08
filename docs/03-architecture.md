# 架构设计

## 整体架构

```
┌─────────────────────────────────────────────────────────┐
│                    ReverseRouter                         │
│  核心入口：ReverseHttpRequest() / IsNeedRequest()        │
│  将HTTP请求逆向工程为路由树                                │
│  9步处理：URL→路径→方法→参数→ContentType→Header→Cookie    │
└───────────────────────┬─────────────────────────────────┘
                        │
          ┌─────────────┼─────────────┐
          │             │             │
          ▼             ▼             ▼
┌─────────────┐ ┌─────────────┐ ┌─────────────┐
│RequestPath   │ │RequestParam │ │RequestConten│
│Router        │ │Router       │ │tTypeRouter  │
│按路径查找    │ │按参数查找    │ │按Content-Type│
└──────┬──────┘ └──────┬──────┘ └──────┬──────┘
       │               │               │
       └───────────────┼───────────────┘
                       │
                       ▼
              ┌─────────────────┐
              │     Tree        │
              │  路由树容器      │
              │  Root → Node    │
              └────────┬────────┘
                       │
                       ▼
              ┌─────────────────┐
              │     Node        │
              │  路由树的节点    │
              │  树形结构组织    │
              └────────┬────────┘
                       │
        ┌──────────────┼──────────────┐
        │              │              │
        ▼              ▼              ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│RequestPath   │ │RequestMethod │ │RequestParam  │
│Node          │ │Node          │ │Node          │
│/api/users    │ │GET/POST/...  │ │page=1&size=10│
└──────┬───────┘ └──────────────┘ └──────────────┘
       │
       ▼
┌──────────────┐ ┌──────────────────┐ ┌──────────────┐
│RequestPath   │ │RequestPath       │ │RequestContent│
│VariableNode  │ │VariableNode      │ │TypeNode      │
│{id}          │ │{name}            │ │application/  │
│类型推断       │ │类型推断           │ │json          │
└──────────────┘ └──────────────────┘ └──────────────┘

┌──────────────┐  ┌─────────────────┐
│RequestHeader │  │RequestCookie    │
│Node          │  │Node             │
│Accept        │  │lang             │  ← 分组节点（key=名称）
│  ├──json     │  │  ├──zh-CN       │
│  └──html     │  │  └──en-US       │  ← 值子节点
└──────────────┘  └─────────────────┘

┌─────────────────────────────────────────────────────────┐
│                    Exporter（导出层）                     │
│  OpenAPIExporter.Export(tree) → OpenAPI 3.0.3 JSON       │
│  路径变量还原 → {var}，参数分类（query/path/header/cookie）│
│  请求体 schema，类型映射，稳定排序                          │
└─────────────────────────────────────────────────────────┘
```

## 路由树结构示例

对于以下请求：
```
GET  /api/users
GET  /api/users/123
GET  /api/users/456
POST /api/users (Content-Type: application/json)
GET  /api/users?page=1&size=10
GET  /api/data (Accept: application/json)
GET  /api/data (Accept: text/html)
GET  /api/home (Cookie: lang=zh-CN)
GET  /api/home (Cookie: lang=en-US)
```

还原出的路由树：
```
root
 └── api
      ├── users                    [RequestPathNode]
      │    ├── GET                  [RequestMethodNode]
      │    │    ├── (leaf)          [无路径变量、无参数]
      │    │    ├── {id}            [RequestPathVariableNode, type=integer]
      │    │    │    └── (leaf)
      │    │    └── ?page&size      [RequestParamNode]
      │    │         └── (leaf)
      │    └── POST                 [RequestMethodNode]
      │         └── application/json [RequestContentTypeNode]
      │              └── (leaf)
      ├── data                     [RequestPathNode]
      │    └── GET                  [RequestMethodNode]
      │         └── Accept          [RequestHeaderNode]
      │              ├── application/json [RequestHeaderValueNode]
      │              └── text/html       [RequestHeaderValueNode]
      └── home                     [RequestPathNode]
           └── GET                  [RequestMethodNode]
                └── lang            [RequestCookieNode]
                     ├── zh-CN      [RequestCookieValueNode]
                     └── en-US      [RequestCookieValueNode]
```

## 数据流

```
HTTP请求 (原始URL)
    │
    ▼
UrlParser.Parse()
    │
    ├──→ []*HttpRequestPath  (路径段，已URL解码，已过滤./..)
    └──→ []*HttpParam        (查询参数，参数名小写，多值展开)
    │
    ▼
BodyParser.Parse(contentType, body)            ← 请求体参数（按Content-Type分发）
    │
    └──→ []*HttpParam        (表单/JSON/multipart解析，参数名小写)
    │
    ▼
ReverseRouter.ReverseHttpRequest() — 9步处理
    │
    ├──→ 1. URL解析（UrlParser）
    ├──→ 2. 路径匹配/创建（尾部斜杠处理、URL解码）
    ├──→ 3. 路径参数识别（key=value 格式）
    ├──→ 4. 路径变量识别合并（选择性合并，固定路径保留）
    ├──→ 5. HTTP方法节点
    ├──→ 6. 查询参数+请求体参数节点（合并处理，大小写不敏感、多值、类型推断）
    ├──→ 7. Content-Type节点
    ├──→ 8. Header路由节点（两层结构：名称→值）
    └──→ 9. Cookie路由节点（两层结构：名称→值）
    │
    ▼
路由树 (Tree)
```

## Header路由设计

某些Web服务根据特定Header做路由决策，例如：
- Accept: `application/json` vs `text/html` → 返回不同格式
- Authorization: `Bearer` vs `Basic` → 不同认证方式
- X-Api-Version: `v1` vs `v2` → API版本路由
- Accept-Language: `zh-CN` vs `en-US` → 多语言路由

### Header值规范化

为避免相同语义的Header值创建不同节点，对值进行规范化：

| Header | 原始值 | 规范化值 | 规则 |
|--------|--------|----------|------|
| Accept | `application/json, text/html;q=0.9` | `application/json` | 取第一个MIME类型 |
| Authorization | `Bearer token123` | `Bearer` | 只取认证方案 |
| Accept-Language | `zh-CN,zh;q=0.9` | `zh-CN` | 取第一个语言标签 |
| X-Api-Version | `v2` | `v2` | 原值不变 |
| X-Requested-With | `XMLHttpRequest` | `XMLHttpRequest` | 原值不变 |

### 两层结构设计

```
方法节点 (GET)
 └── Accept [request_header]        ← 第一层：Header名称分组
      ├── application/json [request_header_value]  ← 第二层：Header值
      └── text/html [request_header_value]
```

这种设计的好处：
- 同一Header的不同值作为兄弟节点，便于后续变量合并
- 查找时先按名称找分组，再按值找具体节点
- 统计时可以区分Header分组数和值数

## Cookie路由设计

与Header路由类似，使用两层结构：

```
方法节点 (GET)
 └── lang [request_cookie]           ← 第一层：Cookie名称分组
      ├── zh-CN [request_cookie_value]  ← 第二层：Cookie值
      └── en-US [request_cookie_value]
```

## 参数识别设计

### 大小写不敏感

HTTP参数名不区分大小写是常见约定：
- `Page=1`, `page=2`, `PAGE=3` → 统一存储为 `page` 参数
- 在 `UrlParser.Parse()` 中将参数名转为小写
- 在 `RequestParamNode` 构造函数中也统一小写

### 多值参数

同一参数名出现多次（如 `?tag=go&tag=web`）：
- `UrlParser.Parse()` 展开为多个 `HttpParam{Name:"tag", Value:"go"}` 等
- `findOrCreateParamNode` 第一次创建节点，后续调用 `ObserveValue` 记录值
- `RequestParamNode.ExtractValue()` 使用 `extractParamValues()` 提取所有值
- 多值标记 `multiValue=true`，上下文中用逗号连接

### 参数值类型推断

在 `findOrCreateParamNode` 中：
- 创建新参数节点时，如果有值，立即推断类型
- 观察新值时，重新推断类型
- 使用 `ChainTypeInferenceRule.InferPhysicalAndLogical()` 获取物理和逻辑类型
- 每次参数出现累加 `presenceCount`（用于必需性推断）

### 必需参数自动推断

黑盒场景下通过参数出现频率推断必需性：
- `InferRequiredParams()` 遍历所有方法节点，对参数节点调用 `InferRequired(totalRequests, threshold)`
- 出现率 = presenceCount / 方法节点请求次数 >= 0.9（`RequiredParamThreshold`）→ 必需
- 样本不足（请求次数 <= 1）保持默认，避免误判
- 建议在导出/序列化前调用

### 请求体参数解析（BodyParser）

POST/PUT/PATCH 请求的参数常出现在请求体中而非 URL 查询串。`BodyParser` 按 Content-Type
将请求体解析为 `[]*HttpParam`，与查询参数同构，并入第6步统一处理：

| Content-Type | 解析方式 | 参数名规则 |
|--------------|----------|------------|
| application/x-www-form-urlencoded | `net/url.ParseQuery`，支持 URL 解码和多值 | 原参数名，小写化 |
| application/json | 递归扁平化，标量→name=value | 嵌套用点号连接（`address.city`），数组用索引（`tags.0`） |
| multipart/form-data | 按 boundary 分割 part，提取字段值 | name 属性值；文件字段以 filename 为值 |

- 参数名统一小写，与查询参数一致
- Content-Type 带 `; charset=utf-8` 时，先 `normalizeContentType` 取主类型再分发
- multipart 的 boundary 从**原始** Content-Type 用 `extractBoundary` 提取
- 不支持的类型（`text/plain`、`application/octet-stream` 等）返回空列表不报错
- `MaxParams`（默认1000）上限防止恶意 body 导致参数爆炸

解析出的 body 参数与 URL 查询参数、路径嵌入参数合并为 `allParams`，统一进入 `processParams`，
享受类型推断、多值处理、必需性推断等全部能力。

### 路由树序列化（含类型信息）

`ToJSON()` / `FromJSON()` 完整保留节点类型信息：
- 路径变量：`inferred_type`（物理类型）、`logical_type`
- 参数节点：`required`、`physical_type`、`logical_type`、`presence_count`、`default_value`、`multi_value`
- 往返一致，可用于持久化和恢复路由树状态

### OpenAPI 3.0.3 导出

`pkg/exporter.OpenAPIExporter` 将路由树导出为标准 OpenAPI 规范，可被 Swagger UI / Redoc 直接渲染：

```
路由树                         OpenAPI 3.0.3
─────────                     ─────────────
request_path + path_variable  →  paths: { "/api/users/{users_id}": {...} }
request_method                →  pathItem.get/post/put/...
request_param (无body)        →  parameters[].in = "query"
request_path_variable         →  parameters[].in = "path" (required=true)
request_header                →  parameters[].in = "header" (同名去重)
request_cookie                →  parameters[].in = "cookie" (同名去重)
request_content_type + body   →  requestBody.content[ct].schema (object)
```

关键设计：
- **端点收集**：DFS 遍历树，遇 `request_method` 节点回溯路径段栈构造完整 path；路径变量段记录为 `{key}` 并保留类型信息
- **参数分类**：有 Content-Type 时 param 归入 requestBody，否则归入 query；Header/Cookie 从两层结构的子节点收集
- **类型映射**：逻辑类型优先决定 schema format（date→date-time、email→email、uuid→uuid、url→uri、ipaddress→ipv4）；回退到物理类型（integer/number/boolean/string）
- **稳定输出**：路径按字母序、参数按 `in`+`name` 排序，operationId = `{method}_{sanitized_path}`
- **可配置**：标题、版本、描述、ServerURL、是否包含可选参数

## 路径边界条件处理

| 边界条件 | 处理方式 |
|----------|----------|
| 尾部斜杠 `/api/users/` | 在 `UrlParser.Parse()` 中 `Trim(path, "/")` |
| 连续斜杠 `//api///users` | 循环替换 `//` 为 `/` |
| URL编码 `%E7%94%A8%E6%88%B7` | `url.PathUnescape()` 自动解码 |
| 路径遍历 `.` 和 `..` | `normalizePathSegment()` 过滤 |
| 文件扩展名 `.json/.xml/.html` | `hasFileExtension()` 排除，不作为变量 |
| 路径变量模式匹配 | `IsMatch()` 有模式时严格匹配，无模式时启发式 |

## 节点类型体系

所有节点类型都实现 `Node[NodeContext]` 接口，通过 `GetType()` 区分：

| 节点类型 | GetType() | Key含义 | Value含义 | IsDynamic |
|----------|-----------|---------|-----------|-----------|
| BaseNode | 自定义 | 自定义 | 自定义 | false |
| RequestPathNode | "request_path" | 路径段名 | 空 | false |
| RequestPathVariableNode | "request_path_variable" | 位置标识 | 空 | true |
| RequestMethodNode | "request_method" | HTTP方法 | 空 | false |
| RequestContentTypeNode | "request_content_type" | Content-Type | 空 | false |
| RequestParamNode | "request_param" | 参数名(小写) | 默认值 | false |
| RequestHeaderNode | "request_header" | Header名称 | Header名称 | false |
| RequestHeaderValueNode | "request_header_value" | 规范化值 | Header名称 | false |
| RequestCookieNode | "request_cookie" | Cookie名称 | Cookie名称 | false |
| RequestCookieValueNode | "request_cookie_value" | Cookie值 | Cookie名称 | false |

## 类型推断体系

```
Value（值）
 ├── PhysicalType（物理类型）：string/integer/float/boolean/array/object/null
 └── LogicalType（逻辑类型）：
      ├── 基本类型：date/time/datetime/email/url/uuid/json/xml/ipaddress
      ├── 数值扩展：decimal/currency/percentage
      ├── 特殊类型：enum/binary/reference
      └── 中国特有：phone/idcard/bankcard/plate

ValueMetric（值统计）
 └── valueMap: map[string]int  →  记录每个值出现的次数

TypeInferenceRule（类型推断规则）
 ├── PhysicalTypeInferenceRule：统计各类型匹配次数，选择最多的
 │    ├── 支持 RequestPathVariableNode 和 RequestParamNode
 │    ├── 长数字串降级：16位及以上纯数字 → string（标识符语义，避免int64溢出）
 │    ├── 十六进制：0x1A、0xDEADBEEF 等 → integer
 │    └── 科学计数法：1e5、1.5E-3 等 → float
 └── LogicalTypeInferenceRule：模式匹配 + 枚举检测
      ├── 支持 RequestPathVariableNode 和 RequestParamNode
      └── 电话号码归一化：手机号+座机号，匹配前去除空格/横线/括号分隔符
         （138-1234-5678 → 13812345678，010-12345678 → 01012345678）

ChainTypeInferenceRule（链式推断）
 └── 组合 PhysicalTypeInferenceRule + LogicalTypeInferenceRule
      └── InferPhysicalAndLogical() 分别返回物理和逻辑类型
```

## 中国特有格式识别

支持识别以下中国特有数据格式（在路径变量和参数值中）：

| 格式 | LogicalType | 示例 | 区分机制 |
|------|-------------|------|----------|
| 手机号 | phone | 13812345678, +8613812345678 | 11位1[3-9]开头，支持分隔符归一化 |
| 座机号 | phone | 010-12345678, (0755)12345678 | 0+区号+7-8位号码，与手机号统一为phone |
| 身份证号 | idcard | 110101199001011234, 11010119900101123X | 18位含日期结构 |
| 银行卡号 | bankcard | 6222021234567890123 | 16-19位3-6开头 |
| 车牌号 | plate | 京A12345, 沪B12345D | 汉字+字母+数字 |

> 邮政编码(postalcode)已从自动识别中移除：6位纯数字无法与普通数字ID、验证码等
> 可靠区分，误判率太高。`LogicalTypePostalCode` 常量保留，但当前不参与自动匹配。

**模式检测优先级**：具体格式（phone/idcard/bankcard等）优先于通用格式（integer），避免身份证号被误判为纯整数。

## 并发设计

- BaseNode：细粒度锁（propMu/parentMu/childMu/contextMu）
- BaseNodeContext：读写锁（mutex）
- ValueMetric：读写锁（sync.RWMutex）
- 请求计数：atomic操作
- UrlParser：改为局部变量，避免竞态
- RouterStats：所有计数器用 atomic 操作，线程安全

## 日志与可观测性

逆向工程过程是黑盒的，日志和统计让过程可追踪、效果可量化。

### 结构化日志（RouterLogger）

封装 Go 标准库 `log/slog`，输出带时间戳的结构化键值对日志：

```
time=2026-07-01T15:35:39 level=INFO msg=识别路径变量 parent=users var_name=users_id pattern=integer physical_type=integer logical_type=string merged_count=3
time=2026-07-01T15:35:39 level=WARN msg=解析请求体参数失败 url=/api/x content_type=application/json error="解析JSON失败: ..."
```

| 级别 | 用途 | 默认输出 |
|------|------|----------|
| Debug | 决策细节：每次合并、模式检测、类型推断、参数创建 | 否 |
| Info | 关键事件：路径变量识别、必需性推断完成 | 否 |
| Warn | 异常兼容：非法 body、模式匹配失败 | 是 |
| Error | 处理失败 | 是 |
| Off | 关闭 | - |

默认 **Warn** 级别保持低噪音（生产环境只看到真正需要关注的事件）；
调试逆向过程时用 `SetLogLevel(LogLevelDebug)` 查看全部决策。

埋点位置：请求处理（开始/完成）、路径变量识别、模式检测、参数创建、类型推断、
必需性推断、合并决策（尝试/跳过）、异常 body、错误。

### 可观测性统计（RouterStats）

11 项 atomic 计数指标，量化逆向效果：

| 指标 | 含义 |
|------|------|
| requests_processed | 已处理请求总数 |
| path_variables_identified | 识别的路径变量数（合并成功） |
| pattern_detections | 模式检测调用次数 |
| params_created | 创建的参数节点数 |
| type_inferences | 类型推断调用次数 |
| body_params_parsed | 从请求体解析的参数总数 |
| required_params_inferred | 推断为必需的参数数 |
| merge_attempts / merge_skipped | 合并尝试 / 跳过次数 |
| warnings / errors | 警告 / 错误事件数 |

```go
r.InferRequiredParams()
stats := r.GetStats()           // 返回快照
data, _ := json.Marshal(stats)  // 可序列化用于监控上报
fmt.Println(stats)              // requests=4, path_vars=1, params=4, ...
r.ResetStats()                  // 清零重新统计
```

