# reverse-router-tree-skills

> 从黑盒抓包流量还原 Web 应用的真实路由树，把散落 URL 归一化成稳定的路由资产，并可导出为 OpenAPI 3.0.3 规范。

[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8)](https://go.dev)
[![Go Test](https://github.com/cyberspacesec/reverse-router-tree-skills/actions/workflows/go-test.yml/badge.svg)](https://github.com/cyberspacesec/reverse-router-tree-skills/actions/workflows/go-test.yml)
[![License](https://img.shields.io/badge/license-MIT-blue)](./LICENSE)

> 🌐 **在线官网**：<https://cyberspacesec.github.io/reverse-router-tree-skills/> ｜ 📚 **教学文档**：<https://cyberspacesec.github.io/reverse-router-tree-skills/docs/>

给一组抓到的 HTTP 请求，还你一棵还原好的路由树——识别路径变量、查询参数、Content-Type/Header/Cookie 路由维度，推断参数的物理与逻辑类型；路由树进一步收敛为「方法 + 路径模板」的归一化资产，并导出成"黑盒版 Swagger"。

## 核心目标：网络空间测绘 URL 资产归一化

本项目唯一的核心目标，是服务网络安全空间测绘：从抓包流量中尽可能还原每个目标应用的路由树，将同一接口的不同 URL 归一化为稳定的路由模板，支撑 URL 资产去重、聚合、检索和后续查询参数规范化。OpenAPI 导出、类型推断等能力都服务于这个目标，而不是独立的终点。


爬虫把 `/api/users/123` 和 `/api/users/456` 当成两个 URL 请求两遍；安全扫描器对同一接口重复测试。本项目把这些散落的 URL **还原成目标服务器真实的路由结构**：

```
/api/users/123      ──┐
/api/users/456        ├──▶  /api/users/{users_id}   （变量，integer）
/api/users/789      ──┘
/api/users?page=1&size=20  ──▶  ?page(必需) & size(必需)
POST /api/users (json)     ──▶  requestBody: name, age
```

## 能力

- **路径变量识别**：纯数字/UUID/手机号/身份证号/银行卡号/车牌号/前缀后缀模式自动合并为 `{var}`；`v1`/`v2` 版本段默认保持固定路径，`MergeVersionSegments` 打开后才合并为 `{parent_version}`
- **选择性合并**：只合并匹配模式的兄弟节点，固定路径（`list`/`create`）不误合并
- **查询参数 + 请求体**：JSON/表单/multipart 解析，JSON 嵌套点号扁平化；参数名小写，`ids[]`/`filter[status]` 收成稳定键，`utm_*` 等追踪参数默认丢弃；`SignatureKey` 给出方法+模板+参数名签名
- **多维度路由**：Content-Type / Header（Accept 等）/ Cookie 作为子路由维度
- **两层类型推断**：物理类型（integer/string/...）+ 逻辑类型（uuid/phone/idcard/...）
- **必需参数推断**：基于出现频率，阈值可配
- **路由查询与资产归一化**：`IsNeedRequest` 判断是否需采集，`FindRouteNode` 查询命中节点；归一化提供 `Detailed` 变体返回机器可读失败原因（unknown_path/method/host/project/invalid_request）、`NormalizeReport` 批量明细替代静默丢弃、`ListAssets` 枚举全量资产、`NormalizeCurl`/`NormalizeURLString` 直达入口，读操作只读不建桶；`NormalizeURL`/`NormalizeURLs` 输出兼容的方法+路径模板资产，`NormalizeAssets` 输出包含 Host 的多目标资产键
- **多目标 Host 隔离**：`RouterSet` 按规范化 host 分桶（小写、去 userinfo、去 http:80/https:443），同一目标不因写法不同拆成多个桶；`HostAssetKey` 用于跨目标资产清单
- **项目级隔离**：`ProjectManager` 按安全测试项目管理多个 `RouterSet`，同 host 在不同项目间互不污染；`SetMaxProjects` 防项目爆炸，`Project/Delete/Projects` 治理生命周期，合并/上限/脱敏/host上限/日志配置向已有与新建项目传播，`Stats/Health` 按项目聚合
- **可续喂路由树**：单树 `ToJSON`/`FromJSON` 保留 ValueMetric 与请求计数；`RouterSet.ExportJSON`/`ImportJSON`、`ProjectManager.ExportJSON`/`ImportJSON` 导出整库快照（host→树），未知更高版本报错且不改动已有数据；运行配置不进快照，导入后由接收方当前配置接管
- **生产护栏（默认开启）**：`SetResourceLimits` 限单父节点子节点数 / 单 ValueMetric 不同值数 / 单路径段长度，超限 fail-soft（拒绝新建、占位计数、截断），`SetRedactConfig` 对敏感参数/cookie 只记结构不存原值（默认覆盖 password/passwd/pwd、sessionid），`RouterSet` 支持 `SetMaxHosts`/`Delete` 容量治理，配置向已有/新建 host 桶传播
- **OpenAPI 3.0.3 导出**：路径/参数/请求体/安全方案（从 Authorization 推断 Bearer/Basic/Digest）
- **并发安全**：`-race` 全量测试通过，多 goroutine 并发喂数据安全
- **可观测性**：结构化日志（slog）+ 16 项 atomic 统计指标（含累计/最大处理耗时、护栏拒绝/截断/脱敏细分），`Health()` 一次返回性能+规模+护栏健康报告（`RouterSet`/`ProjectManager` 按 host/项目聚合），`String()` 输出人类可读摘要
- **自定义合并规则**：`SetMergeRule` 注入业务专属的"什么算变量"判定
- **零外部依赖**：纯 Go 标准库

## 快速上手

```go
package main

import (
	"fmt"
	"log"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/exporter"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/router"
)

func main() {
	r := router.NewReverseRouter()

	// 喂入抓包流量（数字 ID 会被自动合并为 {users_id}）
	samples := []struct{ url, method, body string }{
		{"/api/users/123", "GET", ""},
		{"/api/users/456", "GET", ""},
		{"/api/users/789", "GET", ""},
		{"/api/users", "POST", `{"name":"alice","age":30}`},
	}
	for _, s := range samples {
		h := request.Headers{}
		h.Set("Authorization", "Bearer eyJhbGc...")
		if s.body != "" {
			h.Set("Content-Type", "application/json")
		}
		body := []byte(s.body)
		if s.body == "" {
			body = nil
		}
		req := request.NewHttpRequest(s.url, h, s.method, body)
		if err := r.ReverseHttpRequest(req); err != nil {
			log.Fatal(err)
		}
	}

	r.InferRequiredParams()
	fmt.Println(r.Tree.String())

	// 导出 OpenAPI 3.0.3
	out, _ := exporter.NewOpenAPIExporter().Export(r.Tree)
	fmt.Println(string(out))
}
```

完整可运行示例见 [`examples/quickstart`](./examples/quickstart)（`go run ./examples/quickstart`）。

## 核心包

| 包 | 职责 |
|---|---|
| `pkg/router` | `ReverseRouter` 主入口、`RouterSet` 多目标 Host 分桶、`ProjectManager` 项目级隔离，9 步逆向流程，归一化 API，合并策略，自定义合并规则，生产护栏，Health 健康报告 |
| `pkg/request` | `HttpRequest` / `Headers` / `UrlParser` / `BodyParser` |
| `pkg/node` | 路径/参数/变量/方法/Content-Type/Header/Cookie 节点，`BaseNode` 通用树 |
| `pkg/tree` | `Tree` 容器，JSON 序列化/反序列化（类型信息往返一致） |
| `pkg/inference` | 物理类型 + 逻辑类型推断规则，`ChainTypeInferenceRule` |
| `pkg/value` | `ValueMetric` 值统计，类型常量 |
| `pkg/exporter` | `OpenAPIExporter` OpenAPI 3.0.3 导出 |
| `pkg/generator` | 随机数据生成器（端到端测试用） |

## 交互式可视化演示

一个 `go run` 起全栈演示：网页里粘贴一批抓包流量（CURL / URL），后端现场归一化，D3 树形图实时画出还原出的路由树，附资产清单与统计。

```bash
go run ./demo          # 打开 http://localhost:47177/（默认监听所有网卡，局域网机器可访问 http://<本机IP>:47177/）
```

- **两个页面**：首页（`index.html`）负责解释「这是什么 / 怎么用 / 为什么信得住」；**全屏还原工作台**（`workbench.html`）负责干活——左输入右展示、铺满视口、节点可点击折叠/展开子树
- 支持单行 CURL、`METHOD URL`、裸 URL 混合输入，自动归类解析
- 树节点按类型染色：变量节点（`{var}` + 推断类型）、参数节点、请求体字段；悬停节点看类型与样本值
- 子路径部署：`go run ./demo -prefix /reverse-router-tree-skills/demo`（前端按页面路径自动推断 API 前缀，静态资源全部走相对链接）
- 前端为纯静态页（D3 v7 已本地化为 `demo/web/d3.v7.min.js`，CDN 仅作离线兜底），后端仅 Go 标准库，零额外依赖

## 文档

完整教学站（含 Mermaid 图解算法全流程）：见 [`website/`](./website) 目录，本地预览：

```bash
cd website && npm install && npm run dev
```

关键文档：
- [9 步逆向流程](./website/docs/features/reverse-flow.md) — 从请求到路由树的全链路
- [路径变量识别](./website/docs/features/path-variable.md) — 凭什么 `/api/users/123` 是变量
- [选择性合并](./website/docs/features/selective-merge.md) — 为什么 `list`/`create` 不被误合并
- [并发设计](./website/docs/architecture/concurrency.md) — 锁与无锁策略
- [自定义合并规则](./website/docs/features/custom-merge-rule.md) — 业务专属判定
- [OpenAPI 导出](./website/docs/features/openapi-export.md)

## 状态

生产就绪。核心功能、并发安全、性能、序列化、OpenAPI 导出、可观测性、端到端示例均完成。`go test -race ./...` 全绿。

## 许可证

MIT，见 [LICENSE](./LICENSE)。
