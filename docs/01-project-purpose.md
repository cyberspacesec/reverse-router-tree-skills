# 项目初衷与核心问题

## 问题背景

在网络空间测绘（Cyberspace Mapping）领域，一个关键难题是：

**如何从抓包/流量中收集到的URL路径，还原出目标Web服务器的真实路由结构？**

## 具体场景

### 场景1：路径含变量

抓到的URL：
```
/api/users/123
/api/users/456
/api/users/789
```

实际对应同一个路由：
```
/api/users/{id}
```

### 场景2：查询参数变化

抓到的URL：
```
/list?page=1&size=10
/list?page=2&size=20
/list?page=3&size=30
```

实际对应同一个接口，参数 `page`（integer）和 `size`（integer）值不同。

### 场景3：Content-Type 变化

```
POST /api/users (Content-Type: application/json)
POST /api/users (Content-Type: application/xml)
```

同一个路径、同一个方法，但请求体的格式不同，可能对应不同的处理逻辑。

### 场景4：路径参数 vs 路径变量

```
/api/action=delete
/api/action=create
```

这里的 `action` 是路径中的参数（如Spring的 `@RequestParam`），需要识别出来。

## 核心目标

本项目的核心目标是：

- **输入**：已收集到的大量URL路径/HTTP请求
- **输出**：还原出的路由树结构（类似 Spring 的路由映射表）
- **方法**：通过树形结构 + 类型推断 + 模式识别，自动合并相似路径，识别路径变量，推断参数类型

## 为什么这个问题重要

在网络空间测绘中，能够还原出目标的路由结构意味着：

1. **更精准的接口测试** — 知道哪些是真正的接口，哪些只是参数变化，避免重复测试
2. **减少重复请求** — 同一接口不反复请求，节省资源
3. **更好的安全评估** — 完整还原攻击面，不遗漏接口
4. **智能爬虫** — 知道哪些URL还需要请求，哪些已经覆盖

## 与现有方案的区别

- **白盒方案**（如Swagger/OpenAPI）：需要目标主动暴露接口文档，大多数目标不会
- **传统黑盒爬虫**：只收集URL，不去重、不还原路由结构
- **本项目**：纯黑盒，从流量中自动推断路由结构

## 相关领域

- 网络空间测绘（Cyberspace Mapping）
- Web指纹识别
- API逆向工程
- 黑盒安全测试
- 智能爬虫
