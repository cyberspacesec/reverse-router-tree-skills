# 待修复问题清单

> 优先级从高到低排列

## P0 - 编译错误（必须先修复）

### 1. RequestPathVariableNode 结构体字段不匹配

**文件**：`pkg/node/request_path_variable_node.go`

**问题**：
- 结构体声明了 `value value.Value` 字段
- 但构造函数 `NewRequestPathVariableNode` 中初始化了 `valueMetric`、`valueType`、`inferFunc` 三个不存在的字段
- 结构体缺少这三个字段的声明

**修复方案**：
```go
type RequestPathVariableNode struct {
    *BaseNode[NodeContext]

    // 值统计，用于记录观察到的路径变量值
    valueMetric *value.ValueMetric
    // 推断出的值类型
    valueType value.Type
    // 类型推断函数
    inferFunc func(node Node[NodeContext]) (value.Type, error)
}
```

同时需要调整构造函数的签名和实现，使其与测试文件兼容。

### 2. 测试文件调用签名不匹配

**文件**：`pkg/node/request_nodes_test.go`

**问题**：
- 测试中 `NewRequestPathVariableNode("id", "[0-9]+")` 传入正则字符串
- 实现中 `NewRequestPathVariableNode(position, inferFunc)` 接受位置标识和推断函数

**修复方案**：重新设计 RequestPathVariableNode 的构造函数，可能需要：
- 支持正则模式匹配
- 或者调整测试用例使用推断函数

## P1 - 功能Bug

### 3. RequestPathRouter 类型断言bug

**文件**：`pkg/router/request_path_router.go`

**问题**：
```go
requestPathNode, ok := node.(node.RequestPathNode)  // 值类型断言
```
应该是：
```go
requestPathNode, ok := node.(*node.RequestPathNode)  // 指针类型断言
```

### 4. RequestPathRouter 返回值类型错误

**文件**：`pkg/router/request_path_router.go`

**问题**：空路径时返回 `requestPathNode.GetKey() == ""` (bool)，但函数签名要求返回 `Node`。

### 5. RequestParamRouter 类型断言bug

**文件**：`pkg/router/request_param_router.go`

同问题3，值类型断言应改为指针类型断言。

### 6. RequestContentTypeRouter 复制了错误的代码

**文件**：`pkg/router/request_content_type_router.go`

**问题**：完全复制了RequestPathRouter的代码，没有实现Content-Type路由逻辑。

## P2 - 核心功能未实现

### 7. ReverseRouter.ReverseHttpRequest() 未实现

**文件**：`pkg/router/reverse_router.go`

这是项目的核心方法，目前只有注释描述算法思路。需要实现：
- 将HTTP请求解析为路径段和参数
- 在路由树中查找或创建对应节点
- 通过兄弟节点数量检测识别路径变量
- 合并相似路径为路径变量节点
- 推断路径变量的类型

### 8. ReverseRouter.IsNeedRequest() 未实现

**文件**：`pkg/router/reverse_router.go`

判断某个URL是否还需要请求。逻辑：
- 如果能在路由树上找到对应节点，且该节点已被请求过足够多次，则不再需要请求

### 9. Tree.AddNode() 未实现

**文件**：`pkg/tree/tree.go`

目前只返回nil。

### 10. PhysicalTypeInferenceRule.Infer() 未连接节点上下文

**文件**：`pkg/inference/physical_type_inference_rule.go`

Infer方法中从节点上下文获取值采样是TODO，目前使用空的ValueMetric。

## P3 - 改进项

### 11. ValueMetric 缺少并发安全保护

**文件**：`pkg/value/value.go`

valueMap 没有锁保护，并发写入可能出问题。

### 12. RequestPathVariableNode 需要更丰富的类型推断

当前只有简单的字符串类型推断，需要支持：
- 数字ID识别（纯数字 → integer）
- UUID识别（匹配UUID格式）
- 日期识别（匹配日期格式）
- 正则模式匹配

### 13. RequestParamNode.IsMatch() 对非必需参数过于宽松

当前实现：如果参数不是必需的，任何查询字符串都返回true。这可能导致误匹配。

### 14. 缺少路由树的序列化/反序列化

路由树需要能够持久化存储，当前只有BaseNode的基本序列化。
