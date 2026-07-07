package router

import (
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
)

// MergeDecision 自定义合并规则的决策结果。
//
// 当用户通过 SetMergeRule 注入了自定义规则时，findMergeableSiblings
// 会先调用该规则。规则通过返回的 Action 表达意图：
//
//   - MergeActionMerge：合并 Mergeable 中的节点（可为入参 children 的子集，
//     实现选择性合并；也可为全部，实现完全接管）。交还内置逻辑不再参与。
//   - MergeActionSkip：本次不合并（即便内置逻辑本会合并）。用于用户判定
//     "这些是固定路径，别动"。
//   - MergeActionDefault：交还内置逻辑判定（相当于未注入规则的默认行为）。
//     用于自定义规则只关心部分场景、其余放行的场景。
type MergeAction int

const (
	// MergeActionDefault 交还内置合并逻辑判定。
	MergeActionDefault MergeAction = iota
	// MergeActionMerge 合并 Mergeable 指定的节点子集。
	MergeActionMerge
	// MergeActionSkip 本次跳过合并，保留所有兄弟为固定路径节点。
	MergeActionSkip
)

// MergeContext 传给自定义合并规则的上下文。
//
// Pattern / Similarity 由内置 PatternDetector 预先计算好，避免用户重复
// 实现模式检测。用户可直接基于这些信息决策，也可完全忽略自行判定。
type MergeContext struct {
	// Parent 合并发生的父节点。变量名将基于 Parent.GetKey() 生成。
	Parent node.Node[node.NodeContext]
	// Siblings 同层全部 request_path 兄弟节点（未过滤）。
	Siblings []node.Node[node.NodeContext]
	// Values Siblings 各节点的路径段值，与 Siblings 一一对应。
	Values []string
	// Pattern 内置检测器对 Values 识别出的主导模式名（如 integer/uuid/
	// phone/prefix/suffix/similar_length_strings；无匹配时为空）。
	Pattern string
	// Similarity Values 中匹配 Pattern 的比例（0.0-1.0）。
	Similarity float64
	// Config 当前 MergeConfig，便于规则读取阈值等配置。
	Config MergeConfig
}

// MergeRule 自定义合并规则接口。
//
// 注入方式：r.SetMergeRule(myRule)。nil 表示使用内置逻辑。
//
// 设计原则：
//   - 规则只决定"哪些兄弟可合并"，不负责执行合并（执行仍由内置 mergeSiblings
//     完成，保证子树迁移、类型推断、统计等不变）。
//   - 规则应是无状态的：同一输入任何时候返回同一决策，否则在并发合并临界区
//     （mergeMu 保护）下行为不可预测。
//   - 返回 MergeActionDefault 可让规则仅对特定场景生效，其余放行。
type MergeRule interface {
	// Decide 根据上下文决定是否合并、合并哪些节点。
	//
	// 返回 (action, mergeable)：
	//   - MergeActionMerge：mergeable 必须非空且为 Siblings 的子集（顺序不限，
	//     内部会重新匹配）。若 mergeable 为空则等同于 MergeActionSkip。
	//   - MergeActionSkip / MergeActionDefault：mergeable 被忽略。
	Decide(ctx MergeContext) (action MergeAction, mergeable []node.Node[node.NodeContext])
}

// SetMergeRule 注入自定义合并规则。传 nil 清除，恢复内置逻辑。
//
// 规则在 checkAndMergeSiblings 临界区内（mergeMu 保护）被调用，
// 无需自行处理并发。规则决定合并子集后，执行仍走内置 mergeSiblings，
// 因此子树迁移、变量名推断、类型推断、统计计数等行为保持一致。
//
// 本方法持 mergeMu 与合并临界区互斥，确保规则不会在合并执行中途被换。
func (x *ReverseRouter) SetMergeRule(rule MergeRule) {
	x.mergeMu.Lock()
	defer x.mergeMu.Unlock()
	x.mergeRule = rule
}

// GetMergeRule 返回当前注入的自定义合并规则，未注入返回 nil。
//
// 不持 mergeMu 以避免与合并临界区死锁（findMergeableSiblings 在 mergeMu
// 内若调本方法会重入死锁）。接口指针读写为原子操作，最坏仅读到换规则前
// 的旧值，不影响正确性。
func (x *ReverseRouter) GetMergeRule() MergeRule {
	return x.mergeRule
}

// intersectNodes 返回 base 中同时也出现在 others 里的节点，保持 base 顺序。
// 用于过滤自定义规则返回的 mergeable，确保只含合法兄弟。
func intersectNodes(base, others []node.Node[node.NodeContext]) []node.Node[node.NodeContext] {
	if len(others) == 0 {
		return nil
	}
	seen := make(map[node.Node[node.NodeContext]]bool, len(others))
	for _, o := range others {
		seen[o] = true
	}
	out := make([]node.Node[node.NodeContext], 0, len(base))
	for _, b := range base {
		if seen[b] {
			out = append(out, b)
		}
	}
	return out
}
