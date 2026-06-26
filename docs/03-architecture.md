# 架构设计

## 整体架构

```
┌─────────────────────────────────────────────────────────┐
│                    ReverseRouter                         │
│  核心入口：ReverseHttpRequest() / IsNeedRequest()        │
│  将HTTP请求逆向工程为路由树                                │
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
```

## 路由树结构示例

对于以下请求：
```
GET  /api/users
GET  /api/users/123
GET  /api/users/456
POST /api/users (Content-Type: application/json)
GET  /api/users?page=1&size=10
```

还原出的路由树：
```
root
 └── api
      └── users                    [RequestPathNode]
           ├── GET                  [RequestMethodNode]
           │    ├── (leaf)          [无路径变量、无参数]
           │    ├── {id}            [RequestPathVariableNode, type=integer]
           │    │    └── (leaf)
           │    └── ?page&size      [RequestParamNode]
           │         └── (leaf)
           └── POST                 [RequestMethodNode]
                └── application/json [RequestContentTypeNode]
                     └── (leaf)
```

## 数据流

```
HTTP请求 (原始URL)
    │
    ▼
UrlParser.Parse()
    │
    ├──→ []*HttpRequestPath  (路径段)
    └──→ []*HttpParam        (查询参数)
    │
    ▼
ReverseRouter.ReverseHttpRequest()
    │
    ├──→ 在路由树中查找/创建路径节点
    ├──→ 检测兄弟节点数量，识别路径变量
    ├──→ 合并相似路径为PathVariableNode
    ├──→ 创建方法节点
    ├──→ 创建参数节点
    └──→ 触发类型推断
    │
    ▼
路由树 (Tree)
```

## 模块依赖关系

```
tree → node → value
              ↗
router → node
      → request
      → inference → node → value
```

## 节点类型体系

所有节点类型都实现 `Node[NodeContext]` 接口，通过 `GetType()` 区分：

| 节点类型 | GetType() | Key含义 | Value含义 | IsDynamic |
|----------|-----------|---------|-----------|-----------|
| BaseNode | 自定义 | 自定义 | 自定义 | false |
| RequestPathNode | "request_path" | 路径段名 | 空 | false |
| RequestPathVariableNode | "request_path_variable" | 位置标识 | 空 | true |
| RequestMethodNode | "request_method" | HTTP方法 | 空 | false |
| RequestContentTypeNode | "request_content_type" | Content-Type | 空 | false |
| RequestParamNode | "request_param" | 参数名 | 默认值 | false |

## 类型推断体系

```
Value（值）
 ├── PhysicalType（物理类型）：string/integer/float/boolean/array/object/null
 └── LogicalType（逻辑类型）：date/time/datetime/email/url/uuid/.../enum/binary/reference

ValueMetric（值统计）
 └── valueMap: map[string]int  →  记录每个值出现的次数

TypeInferenceRule（类型推断规则）
 └── PhysicalTypeInferenceRule
      ├── 采样值 → 统计各类型匹配次数
      └── 选择匹配最多的类型作为推断结果
```

## 并发设计

- BaseNode：细粒度锁（propMu/parentMu/childMu/contextMu）
- BaseNodeContext：读写锁（mutex）
- ValueMetric：**当前无锁保护，需要修复**
- 请求计数：atomic操作
