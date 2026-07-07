# 待修复问题清单

> 优先级从高到低排列

## ~~P0 - 编译错误（已全部修复）~~

所有 P0 编译错误已修复。

## ~~P1 - 功能Bug（已全部修复）~~

所有 P1 功能 bug 已修复。

## ~~P2 - 核心功能未实现（已全部实现）~~

所有 P2 核心功能已实现。

## ~~P3 - 改进项（已全部完成）~~

### 1. ✅ RequestParamNode.IsMatch() 对非必需参数过于宽松

**文件**：`pkg/node/request_param_node.go`

**修复**：
- 可选参数只在参数名实际出现时才匹配
- 精确参数名匹配（避免 page 匹配 page_size）
- 添加 ValueMetric 用于值观察和类型推断

### 2. 🔶 路由树的序列化/反序列化（部分完成）

**文件**：`pkg/tree/tree.go`

**已完成**：
- ✅ JSON 导出/导入基本实现
- ✅ 路径变量类型信息导出
- ✅ 参数 required 信息导出

**待完善**：
- [ ] 完整的逻辑类型信息导出
- [x] OpenAPI/Swagger 格式导出 — pkg/exporter.OpenAPIExporter 输出 OpenAPI 3.0.3 规范

### 3. ✅ 更丰富的逻辑类型推断

**文件**：`pkg/inference/logical_type_inference_rule.go`

**已完成**：
- ✅ LogicalTypeInferenceRule 完整实现
- ✅ UUID、IP地址、邮箱、日期、时间、日期时间、URL、JSON、XML 模式检测
- ✅ 百分比、货币、精确小数等数值扩展类型
- ✅ 枚举值检测
- ✅ ChainTypeInferenceRule 链式组合规则

### 4. ✅ 路径参数识别（key=value格式）

**文件**：`pkg/request/http_request_path.go`

**已完成**：
- ✅ `IsPathParam()` 检测路径段中的 key=value 格式
- ✅ 合法参数名校验
- ✅ ReverseHttpRequest 集成路径参数处理

### 5. ✅ 合并阈值可配置化

**文件**：`pkg/router/reverse_router.go`

**已完成**：
- ✅ MergeConfig 结构体（SiblingMergeThreshold + PatternSimilarityThreshold）
- ✅ SetMergeConfig() / GetMergeConfig() 方法
- ✅ PatternDetector 模式检测器

### 6. ✅ 路径变量合并的回退机制（已满足：共存式软回退）

**原问题**：合并后的路径变量节点不能"分裂"回多个固定节点

**现状分析**：当前实现采用"共存式软回退"，已实质满足需求：
- 变量节点建立后，不符合模式的新值（如 `special`）作为固定路径与之共存，不破坏已有合并
- 符合模式的新值（如 `205`）仍正确归入变量节点
- 固定路径与变量节点可同层共存（如 `{users_id}` + `list` + `create`）
- 选择性合并在固定路径已存在时仍能合并匹配的子集

**不实现硬回退的理由**：拆分已合并的变量节点需要保留全部原始值历史、重做模式检测，
风险高且触发场景罕见（需要"先大量数字合并、后证明全部是固定路径"的极端顺序）。
当前软回退已覆盖现实业务中的绝大多数情况，硬回退属过度工程。

如未来出现明确的硬回退需求，可考虑：保留原始值统计 + 当固定路径占比超过阈值时拆分变量节点。

## P4 - 未来增强

### 1. ~~合并回退机制~~（已满足：共存式软回退 ✅）

~~当新观察到的值不符合变量模式时，支持合并回退。~~
已采用共存式软回退：不符合模式的新值作为固定路径与变量节点共存，不破坏已有合并。
不实现硬回退（拆分已合并节点），理由是风险高、场景罕见，属过度工程。详见上文 P3-6。

### 2. ~~OpenAPI/Swagger 导出~~（已完成 ✅）

~~将路由树导出为 OpenAPI 规范格式。~~
已实现 `pkg/exporter.OpenAPIExporter`，输出 OpenAPI 3.0.3 规范：
- 路径变量还原为 `{var}`，固定路径保留
- query/path/header/cookie 四类参数，路径变量 required=true
- POST 请求体生成 object schema（含 properties/required）
- Content-Type 去 charset 规范化
- 物理/逻辑类型映射为 schema type+format
- operationId 自动生成，路径按字母序稳定输出
- 可配置标题/版本/ServerURL/是否包含可选参数

### 3. ~~更多合并策略~~（部分完成）

- [x] 基于前缀/后缀的合并 — detectPrefixPattern/detectSuffixPattern 已实现，结构化模式匹配不足时回退检测公共前缀/后缀
- [ ] 基于正则模式的合并
- [ ] 自定义合并规则

### 4. ~~RequestParamNode 必需参数自动推断~~（已完成 ✅）

~~根据参数出现频率自动推断是否为必需参数。~~
已实现 InferRequiredParams，基于 presenceCount/总请求次数 >= 0.9 判定必需，样本不足保持默认。

### 5. ~~日志和可观测性~~（已完成 ✅）

~~添加结构化日志，支持调试和监控。~~
已实现两层可观测性：
- **RouterLogger**：封装 log/slog 的结构化日志，五级（Debug/Info/Warn/Error/Off），
  默认 Warn 级别保持低噪音。关键决策埋点：请求处理、路径变量识别、模式检测、
  参数创建、类型推断、必需性推断、合并决策（尝试/跳过）、异常 body 兼容、错误。
  可配置输出流，nil 安全。
- **RouterStats**：11 项 atomic 线程安全计数指标（请求数、路径变量数、模式检测数、
  参数数、类型推断数、body 参数数、必需参数数、合并尝试/跳过数、警告/错误数）。
  `GetStats()` 返回快照，支持 JSON 序列化，`Reset()` 清零。
- 通过 `SetLogger`/`SetLogLevel`/`GetStats`/`ResetStats` 配置。

### 6. ~~OpenAPI 安全方案（securitySchemes）~~（已完成 ✅）

~~从抓包流量推断鉴权方案，导出 OpenAPI securitySchemes。~~
已实现：router 规范化 Authorization 头（提取 Bearer/Basic/Digest 方案名），exporter
据此在 operation 上挂 `security` 声明，并在 `components.securitySchemes` 注册
`http` 方案定义。仅识别 OpenAPI 3.0.3 标准 http 方案；未识别方案（如自定义 Token）
回退为普通 header 参数。保持 exporter 无状态可重入。
