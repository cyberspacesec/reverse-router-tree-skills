# reverse-router-tree-skills

> 从黑盒抓包流量还原 Web 应用的真实路由树，并导出为 OpenAPI 3.0.3 规范。

[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-blue)](./LICENSE)

给一组抓到的 HTTP 请求，还你一棵还原好的路由树——识别路径变量、查询参数、Content-Type/Header/Cookie 路由维度，推断参数的物理与逻辑类型，最终导出成"黑盒版 Swagger"。

## 为什么需要

爬虫把 `/api/users/123` 和 `/api/users/456` 当成两个 URL 请求两遍；安全扫描器对同一接口重复测试。本项目把这些散落的 URL **还原成目标服务器真实的路由结构**：

```
/api/users/123      ──┐
/api/users/456        ├──▶  /api/users/{users_id}   （变量，integer）
/api/users/789      ──┘
/api/users?page=1&size=20  ──▶  ?page(必需) & size(必需)
POST /api/users (json)     ──▶  requestBody: name, age
```

## 能力

- **路径变量识别**：纯数字/UUID/手机号/身份证号/银行卡号/车牌号/前缀后缀模式自动合并为 `{var}`
- **选择性合并**：只合并匹配模式的兄弟节点，固定路径（`list`/`create`）不误合并
- **查询参数 + 请求体**：JSON/表单/multipart 解析，JSON 嵌套点号扁平化，参数名大小写不敏感
- **多维度路由**：Content-Type / Header（Accept 等）/ Cookie 作为子路由维度
- **两层类型推断**：物理类型（integer/string/...）+ 逻辑类型（uuid/phone/idcard/...）
- **必需参数推断**：基于出现频率，阈值可配
- **OpenAPI 3.0.3 导出**：路径/参数/请求体/安全方案（从 Authorization 推断 Bearer/Basic/Digest）
- **并发安全**：`-race` 全量测试通过，多 goroutine 并发喂数据安全
- **可观测性**：结构化日志（slog）+ 11 项 atomic 统计指标
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
| `pkg/router` | `ReverseRouter` 主入口，9 步逆向流程，合并策略，自定义合并规则 |
| `pkg/request` | `HttpRequest` / `Headers` / `UrlParser` / `BodyParser` |
| `pkg/node` | 路径/参数/变量/方法/Content-Type/Header/Cookie 节点，`BaseNode` 通用树 |
| `pkg/tree` | `Tree` 容器，JSON 序列化/反序列化（类型信息往返一致） |
| `pkg/inference` | 物理类型 + 逻辑类型推断规则，`ChainTypeInferenceRule` |
| `pkg/value` | `ValueMetric` 值统计，类型常量 |
| `pkg/exporter` | `OpenAPIExporter` OpenAPI 3.0.3 导出 |
| `pkg/generator` | 随机数据生成器（端到端测试用） |

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
