# 增量迭代与结论更正

> 路由树不是一次成型的产物，而是**随流量持续演化的推测**。每来一条新请求，
> 树都有机会修正之前的结论——但也存在不会自动回退的边界。本页用可复现的实验说明哪些结论会被更正、哪些不会。

## 一句话结论

| 结论类型 | 会被新请求自动更正吗 | 机制 |
|---------|-------------------|------|
| 树结构（合并出的变量） | ✅ 增量吸收，❌ 不自动拆分 | 每次建新节点后重跑合并检查 |
| 变量的物理/逻辑类型 | ✅ 会 | 新值命中 → 样本计数变化 → 重推断 |
| 参数必需性 | ✅ 会（重算时） | 按当前全部样本比例重新评估 |
| 合并决策本身 | ❌ 单向不可逆 | 无拆分/回退机制 |

## 类型结论被新证据推翻：实测

物理类型按观察值集合的**多数派**判定。同一个变量节点，先来的样本得出一个结论，
后来的样本可以把它推翻——树结构不变，类型字段原地翻转：

```
喂 object/deadbeef, object/cafebabe, object/abcdef01   （字母 hex ×3）

root
└── object [Path]
    └── {object_id} [Var, string]      ← 推断为 string
        └── GET [Method]

再喂 object/12345678, 87654321, 11112222, 33334444    （数字 hex ×4 反超）

root
└── object [Path]
    └── {object_id} [Var, integer]     ← 同一节点，结论翻转为 integer
        └── GET [Method]
```

触发条件是**样本计数变化**：新值命中已有变量节点（`IsMatch` 通过其正则）时
`ObserveValue` 累积观察值，unique 值数变化即触发链式推断重算
（`ChainTypeInferenceRule.InferPhysicalAndLogical`，见
`findOrCreatePathNode` 的增量推断分支）。unique 数不变的重复命中会跳过重算——
这是热路径优化，避免 O(N²) 重复推断。

## 必需参数结论：随样本量重算

必需性 = 参数出现次数 / 该方法节点总请求数 ≥ 阈值（默认 0.9）。
`InferRequiredParams` 每次**全量重算**，结论只反映"当前全部样本"：

```
第一轮：page 出现 2/3，size 出现 3/3
→ GET /list params=[page size] required=[size]

再喂 8 条后：page 9/11 ≈ 0.82，size 7/11 ≈ 0.64（双双跌破 0.9）
→ GET /list params=[page size] required=[]     ← size 的"必需"结论被撤销
```

注意时序：**谁排在最后喂入，对结论影响越大**。批量 API（`ReverseRequests` /
`ReverseCurls`）结束时会自动调用一次重算；单条 `ReverseHttpRequest` 不自动触发，
需要你在合适的时机（如每批喂完）手动调用。

## 树结构：增量吸收，不自动拆分

**会自动做的：**

- **增量合并**：每次创建新路径节点后都重跑兄弟合并检查，新证据可能触发新合并
  （1/2/3 合并成 `{users_id}` 后，再来 4/5 直接走进已有变量，不产生新节点）。
- **级联合并**：合并产生的子树归位会向下传播检查（`cascadeMergeLocked`），
  `u/1/posts, u/2/posts, u/3/posts` 合并后 `posts` 自动挂到 `{u_id}` 之下。
- **选择性合并**：只合并匹配主导模式的兄弟节点，`list`/`create` 等固定路径不误伤。

**不会自动做的：**

- **合并不可逆**：一旦合并成 `{users_id}`（模式 `[0-9]+`），不存在"发现合错了
  再拆回去"的机制。后续证据只能**新增**结构，不能撤销已有合并。
- **变量模式守门**：不匹配变量正则的新值不会被硬塞进变量——
  `/api/users/list` 会作为固定节点与 `{users_id}` **共存**，
  归一化/查询时各自独立命中：

```
GET /api/users/{users_id} [users_id]
GET /api/users/list
```

这个设计是有意取舍：拆分机制需要保留每个原始值到节点的完整映射，内存与
实现复杂度大幅上升；而误合并的实际风险已被"严格模式匹配 + 选择性合并"
压到很低。若业务上有已知固定路径，优先用 `SetMergeRule` 注入自定义规则
**阻止误合并发生**，而不是事后修正。

## 分批导入：结论跨批次持续演化

`ToVersionedJSON` 序列化时保留每个 `ValueMetric` 的样本计数，
`FromJSON` 加载后继续喂入新流量，类型/必需性推断会**基于累计样本**继续迭代——
适合"按天分批采集、周期性重建结论"的测绘采集管道。

## 实践建议

1. **喂入顺序影响结论**：小批量试喂 → 检查 `ListAssets` → 全量喂入，
   比一次性倒入更可控。
2. **结论要定期重算**：`InferRequiredParams` 是快照式重算，采集管道里
   建议每批结束后调用一次。
3. **防误合并优于事后修正**：对已知的固定路径段（`list`/`health`/`status`…），
   用 `SetMergeRule` 在合并决策时拦截。

源码：[`reverse_router.go`](https://github.com/cyberspacesec/reverse-router-tree-skills/blob/main/pkg/router/reverse_router.go)
（`findOrCreatePathNode` / `checkAndMergeSiblingsLocked` / `mergeSiblings`）、
[`InferRequiredParams`](https://github.com/cyberspacesec/reverse-router-tree-skills/blob/main/pkg/router/reverse_router.go)。
