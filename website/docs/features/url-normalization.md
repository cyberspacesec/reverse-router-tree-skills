# URL 归一化 API

> 归一化是本项目的核心输出：把同一接口的不同 URL 收成一条稳定的路由资产，
> 支撑测绘 URL 资产的去重、聚合与检索。

## 三个问题，三组 API

```
采集流量 ──ReverseHttpRequest──▶ 路由树 ──归一化API──▶ 资产清单
                                        │
         ┌──────────────────────────────┼──────────────────────────────┐
         │ "这条流量归到哪"              │ "归不上卡在哪"                │ "树里有哪些资产"          │
         │ NormalizeURL                 │ NormalizeURLDetailed         │ ListAssets                │
         │ NormalizeCurl                │ NormalizeReport              │  ProjectAssets            │
         │ NormalizeURLString           │  5 种机器可读原因             │                           │
         └──────────────────────────────┴──────────────────────────────┴───────────────────────────┘
```

## 单条归一化：失败原因可读

`NormalizeURL` 只返回 `bool`；要排查"为什么归不上"，用 `Detailed` 变体：

```go
route, reason := r.NormalizeURLDetailed(req)
if reason != router.NormalizeOK {
    // reason.String() 直接可打日志：
    // unknown_path（路径段未命中已知路由）
    log.Printf("归一化失败: %s url=%s", reason, req.Url)
}
```

| 原因 | 含义 | 通常的动作 |
|------|------|-----------|
| `ok` | 成功 | 取 `route.AssetKey()` 入库 |
| `invalid_request` | 请求为 nil 或 URL 解析失败 | 检查采集样本质量 |
| `unknown_host` | 该 host 尚无采集数据 | 该目标还没采够流量 |
| `unknown_project` | 项目不存在（仅 `ProjectManager`） | 检查项目 ID |
| `unknown_path` | 路径段未命中已知路由 | 新接口，继续采集 |
| `unknown_method` | 路径命中但该方法没采集过 | 同一路径的新方法，补采 |

`unknown_path` vs `unknown_method` 的区分是故意的：前者是"全新路径"，
后者是"已知路径开了新方法"，两者在测绘里的跟进动作不同。

## 批量归一化：明细报告替代静默丢弃

旧批量 API（`NormalizeURLs` / `NormalizeAssets`）失败样本直接跳过。
要排查采集缺口，用 `Detailed` 批量拿 `NormalizeReport`：

```go
report := set.NormalizeAssetsDetailed(reqs)
for key, urls := range report.Matched {
    db.UpsertAsset(key, urls)          // 命中：资产键 → 原始 URL
}
for _, miss := range report.Unmatched {
    log.Printf("idx=%d url=%s reason=%s", miss.Index, miss.URL, miss.Reason)
}
fmt.Println(report.MatchedCount(), report.UnmatchedCount())
```

- `RouterSet` 批量键为 `AssetKey()`；多目标场景用 `NormalizeAssetsDetailed`，
  键为 `HostAssetKey()`（`host + 方法 + 模板`）。
- 旧签名保持兼容：内部即调 `Detailed` 版本取 `Matched`，行为一致。

## 资产清单：不用逐条试探

`ListAssets` 遍历树枚举全部已知资产（变量段输出 `{变量名}`），按 `AssetKey` 稳定排序：

```go
for _, a := range r.ListAssets() {
    fmt.Println(a.AssetKey(), a.PathParams, a.QueryParams, a.RequiredParams)
}
// GET /api/users/{users_id} [users_id] [page] [page]
// GET /health [] [] []
```

`RouterSet.ListAssets()` 按 host 分组（`Host` 字段已回填），
`ProjectManager.ListAssets()` 按项目 → host 两级分组。
`RequiredParams` 依赖 `InferRequiredParams()` 是否已执行——批量喂入 API
（`ReverseRequests` / `ReverseCurls`）结束时会自动调用。

## 便捷入口：curl / 裸 URL 直达

手里只有 curl 或裸 URL 字符串时，不必手动构造 `HttpRequest`：

```go
route, ok := set.NormalizeCurl("curl 'http://target/api/users/42' -H 'Authorization: Bearer x'")
route, ok := set.NormalizeURLString("http://target/api/users/42", "GET") // 空方法视为 GET
```

## 只读保证：归一化不建桶

`RouterSet.NormalizeURL` 与 `ProjectManager.NormalizeURL` 是**只读操作**：

- 未知 host / 未知项目只返回失败原因，**不**懒建空桶、不建项目；
- `Hosts()` / `Stats()` / `Projects()` 不会被"查过但从未采集"的目标污染；
- `MaxHosts` / `MaxProjects` 配额不被读流量意外消耗。

源码：[`normalize.go`](https://github.com/cyberspacesec/reverse-router-tree-skills/blob/main/pkg/router/normalize.go)
