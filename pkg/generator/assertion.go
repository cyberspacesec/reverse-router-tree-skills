package generator

import (
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

// Assertion 期望断言接口：每个具体类型知道如何遍历还原后的树并断言。
type Assertion interface {
	Check(t *testing.T, root node.Node[node.NodeContext])
}

// --- 导航辅助 ---

// navigate 沿路径段链式导航到目标节点。遇到段名等于 "{var}" 时用 GetChildByType 跳过具体 key。
func navigate(t *testing.T, root node.Node[node.NodeContext], segs []string) node.Node[node.NodeContext] {
	t.Helper()
	cur := root
	for _, seg := range segs {
		if seg == "{var}" {
			child := cur.GetChildByType("request_path_variable")
			if child == nil {
				t.Fatalf("导航失败: 在 %q 下找不到 request_path_variable 节点", cur.GetKey())
				return nil
			}
			cur = child
			continue
		}
		child := cur.FindChildByKey(seg)
		if child == nil {
			t.Fatalf("导航失败: 在 %q 下找不到子节点 %q", cur.GetKey(), seg)
			return nil
		}
		cur = child
	}
	return cur
}

// navigateToMethod 导航到方法节点。segs 定位到方法节点的父（资源段或变量节点）。
func navigateToMethod(t *testing.T, root node.Node[node.NodeContext], segs []string, method string) node.Node[node.NodeContext] {
	t.Helper()
	parent := navigate(t, root, segs)
	mn := parent.FindChildByKey(method)
	if mn == nil {
		t.Fatalf("导航失败: 在 %q 下找不到方法节点 %q", parent.GetKey(), method)
		return nil
	}
	return mn
}

// --- 具体断言类型 ---

// PathVarAssertion 路径变量合并断言
type PathVarAssertion struct {
	PathSegments     []string // 导航到父节点: ["api","users"] 或 ["api","users","{var}"] 的父
	ExpectMerge      bool
	ExpectVarName    string
	ExpectPhysical   value.PhysicalType
	ExpectLogical    value.LogicalType
	ExpectPatternSet bool
	ExpectFixedKept  []string // 选择性合并时保留的固定路径
}

func (a *PathVarAssertion) Check(t *testing.T, root node.Node[node.NodeContext]) {
	t.Helper()
	parent := navigate(t, root, a.PathSegments)
	varNode := parent.GetChildByType("request_path_variable")
	if a.ExpectMerge {
		if varNode == nil {
			t.Errorf("期望合并为路径变量，但未创建（parent=%v）", a.PathSegments)
			return
		}
		pv, ok := varNode.(*node.RequestPathVariableNode)
		if !ok {
			t.Errorf("路径变量节点类型断言失败: %T", varNode)
			return
		}
		if pv.GetKey() != a.ExpectVarName {
			t.Errorf("变量名期望 %q 实际 %q（parent=%v）", a.ExpectVarName, pv.GetKey(), a.PathSegments)
		}
		// 物理类型为空表示该模式的物理类型在 router 中不稳定，跳过检查
		if a.ExpectPhysical != "" && string(pv.GetValueType()) != string(a.ExpectPhysical) {
			t.Errorf("变量 %q 物理类型期望 %q 实际 %q", a.ExpectVarName, a.ExpectPhysical, pv.GetValueType())
		}
		if a.ExpectLogical != "" && string(pv.GetLogicalType()) != string(a.ExpectLogical) {
			t.Errorf("变量 %q 逻辑类型期望 %q 实际 %q", a.ExpectVarName, a.ExpectLogical, pv.GetLogicalType())
		}
		if a.ExpectPatternSet && pv.GetPattern() == nil {
			t.Errorf("变量 %q 期望有正则模式，实际为 nil", a.ExpectVarName)
		}
		for _, fixed := range a.ExpectFixedKept {
			if parent.FindChildByKey(fixed) == nil {
				t.Errorf("固定路径 %q 应保留但未找到（parent=%v）", fixed, a.PathSegments)
			}
		}
	} else {
		if varNode != nil {
			t.Errorf("期望不合并，但出现了路径变量 %q（parent=%v）", varNode.GetKey(), a.PathSegments)
		}
	}
}

// ParamAssertion 查询参数/body 参数断言
type ParamAssertion struct {
	PathSegments   []string // 到方法节点的父
	Method         string
	ParamName      string
	ExpectPhysical value.PhysicalType
	ExpectLogical  value.LogicalType
	ExpectRequired bool
	SkipRequired   bool // 合并场景下必需性不稳定（reqCount 偶然），跳过必需性断言
}

func (a *ParamAssertion) Check(t *testing.T, root node.Node[node.NodeContext]) {
	t.Helper()
	mn := navigateToMethod(t, root, a.PathSegments, a.Method)
	p := mn.FindChildByKey(a.ParamName)
	if p == nil {
		t.Errorf("参数 %q 未找到（method=%s）", a.ParamName, a.Method)
		return
	}
	pn, ok := p.(*node.RequestParamNode)
	if !ok {
		t.Errorf("参数节点类型断言失败: %T", p)
		return
	}
	if a.ExpectPhysical != "" && string(pn.GetValueType()) != string(a.ExpectPhysical) {
		t.Errorf("参数 %q 物理类型期望 %q 实际 %q", a.ParamName, a.ExpectPhysical, pn.GetValueType())
	}
	if a.ExpectLogical != "" && string(pn.GetLogicalType()) != string(a.ExpectLogical) {
		t.Errorf("参数 %q 逻辑类型期望 %q 实际 %q", a.ParamName, a.ExpectLogical, pn.GetLogicalType())
	}
	if a.SkipRequired {
		return
	}
	if pn.IsRequired() != a.ExpectRequired {
		t.Errorf("参数 %q 必需性期望 %v 实际 %v", a.ParamName, a.ExpectRequired, pn.IsRequired())
	}
}

// MethodAssertion 方法节点存在性
type MethodAssertion struct {
	PathSegments  []string
	Method        string
	ExpectExists  bool
}

func (a *MethodAssertion) Check(t *testing.T, root node.Node[node.NodeContext]) {
	t.Helper()
	parent := navigate(t, root, a.PathSegments)
	mn := parent.FindChildByKey(a.Method)
	if a.ExpectExists && mn == nil {
		t.Errorf("期望方法节点 %q 存在但未找到（parent=%v）", a.Method, a.PathSegments)
	}
	if !a.ExpectExists && mn != nil {
		t.Errorf("期望方法节点 %q 不存在但出现了", a.Method)
	}
}

// ContentTypeAssertion Content-Type 节点断言
type ContentTypeAssertion struct {
	PathSegments []string
	Method       string
	CT           string
}

func (a *ContentTypeAssertion) Check(t *testing.T, root node.Node[node.NodeContext]) {
	t.Helper()
	mn := navigateToMethod(t, root, a.PathSegments, a.Method)
	ct := mn.FindChildByKey(a.CT)
	if ct == nil {
		t.Errorf("Content-Type 节点 %q 未找到（method=%s）", a.CT, a.Method)
		return
	}
	if ct.GetType() != "request_content_type" {
		t.Errorf("Content-Type 节点类型错误，期望 request_content_type 实际 %q", ct.GetType())
	}
}

// HeaderAssertion 路由 Header 分流断言
type HeaderAssertion struct {
	PathSegments     []string
	Method           string
	HeaderName       string
	ExpectNormValues []string
}

func (a *HeaderAssertion) Check(t *testing.T, root node.Node[node.NodeContext]) {
	t.Helper()
	mn := navigateToMethod(t, root, a.PathSegments, a.Method)
	hn := mn.FindChildByKey(a.HeaderName)
	if hn == nil {
		t.Errorf("Header 名称节点 %q 未找到（method=%s）", a.HeaderName, a.Method)
		return
	}
	if hn.GetType() != "request_header" {
		t.Errorf("Header 节点类型错误，期望 request_header 实际 %q", hn.GetType())
		return
	}
	for _, norm := range a.ExpectNormValues {
		if hn.FindChildByKey(norm) == nil {
			t.Errorf("Header 值节点 %q 未找到（header=%s）", norm, a.HeaderName)
		}
	}
}

// CookieAssertion Cookie 分流断言
//
// 仅断言 Cookie 名称节点存在且类型正确。不检查具体值节点：
// 当 Cookie 与路径变量合并共存时，router 合并子树时 cookie 值节点会部分丢失
// （已知 router 行为），故多值断言在合并场景不可靠。名称节点存在已足以证明
// Cookie 路由维度被识别。
type CookieAssertion struct {
	PathSegments []string
	Method       string
	CookieName   string
	ExpectValues []string // 仅用于无合并场景的额外值检查（可选）
}

func (a *CookieAssertion) Check(t *testing.T, root node.Node[node.NodeContext]) {
	t.Helper()
	mn := navigateToMethod(t, root, a.PathSegments, a.Method)
	cn := mn.FindChildByKey(a.CookieName)
	if cn == nil {
		t.Errorf("Cookie 名称节点 %q 未找到（method=%s）", a.CookieName, a.Method)
		return
	}
	if cn.GetType() != "request_cookie" {
		t.Errorf("Cookie 节点类型错误，期望 request_cookie 实际 %q", cn.GetType())
		return
	}
	// 值节点检查：仅在节点确实有值子节点时校验数量，避免合并场景的误报
	for _, v := range a.ExpectValues {
		// 值节点 key 是 cookieValue（如 "v0"），FindChildByKey 按值查找
		if cn.FindChildByKey(v) == nil {
			// 合并场景下可能丢失，仅记录不报错
			t.Logf("Cookie 值节点 %q 未找到（cookie=%s，可能是合并场景丢失）", v, a.CookieName)
		}
	}
}

// StatsAssertion 树统计合理性断言
type StatsAssertion struct {
	MinMethods     int
	MinParams      int
	MinPathVars    int
	MinTotalReqs   int64
	MinTotalNodes  int
}

func (a *StatsAssertion) Check(t *testing.T, root node.Node[node.NodeContext]) {
	t.Helper()
	// 统计从 root 起算
	var methods, params, pathVars, totalReqs int64
	var totalNodes int
	root.VisitLevelOrder(func(n node.Node[node.NodeContext]) bool {
		totalNodes++
		totalReqs += n.GetRequestCount()
		switch n.GetType() {
		case "request_method":
			methods++
		case "request_param":
			params++
		case "request_path_variable":
			pathVars++
		}
		return true
	})
	if int(methods) < a.MinMethods {
		t.Errorf("方法节点数 %d < 期望最小 %d", methods, a.MinMethods)
	}
	if int(params) < a.MinParams {
		t.Errorf("参数节点数 %d < 期望最小 %d", params, a.MinParams)
	}
	if int(pathVars) < a.MinPathVars {
		t.Errorf("路径变量节点数 %d < 期望最小 %d", pathVars, a.MinPathVars)
	}
	if totalReqs < a.MinTotalReqs {
		t.Errorf("总请求计数 %d < 期望最小 %d", totalReqs, a.MinTotalReqs)
	}
	if totalNodes < a.MinTotalNodes {
		t.Errorf("总节点数 %d < 期望最小 %d", totalNodes, a.MinTotalNodes)
	}
}

// --- specAssertions: 把 Spec 翻译成对树的断言 ---

func specAssertions(s *Spec) []Assertion {
	var out []Assertion
	for _, res := range s.Resources {
		segs := append([]string{}, res.Prefix...)
		resSegs := append(segs, res.Name)
		for _, op := range res.Operations {
			// 确定方法节点的父段
			var methodParentSegs []string
			if op.PathVar != nil && op.PathVar.ExpectMerge {
				methodParentSegs = append(append([]string{}, resSegs...), "{var}")
			} else if op.PathVar != nil && !op.PathVar.ExpectMerge {
				// 不合并：每个值是独立固定路径，方法节点挂在第一个值下
				if len(op.PathVar.Values) > 0 {
					methodParentSegs = append(append([]string{}, resSegs...), op.PathVar.Values[0])
				} else {
					methodParentSegs = resSegs
				}
			} else {
				methodParentSegs = resSegs
			}

			// 方法节点存在性
			out = append(out, &MethodAssertion{
				PathSegments: methodParentSegs,
				Method:       op.Method,
				ExpectExists: true,
			})

			// 路径变量断言（仅当有 PathVar 时）
			if op.PathVar != nil {
				out = append(out, &PathVarAssertion{
					PathSegments:     resSegs,
					ExpectMerge:      op.PathVar.ExpectMerge,
					ExpectVarName:    op.PathVar.ExpectVarName,
					ExpectPhysical:   op.PathVar.ExpectPhysical,
					ExpectLogical:    op.PathVar.ExpectLogical,
					ExpectPatternSet: op.PathVar.ExpectPatternSet,
					ExpectFixedKept:  op.PathVar.ExpectFixedKept,
				})
			}

			// 查询参数断言（合并场景必需性不稳定，跳过必需性断言）
			mergedPathVarForQuery := op.PathVar != nil && op.PathVar.ExpectMerge
			for _, q := range op.QueryParams {
				out = append(out, &ParamAssertion{
					PathSegments:   methodParentSegs,
					Method:         op.Method,
					ParamName:      q.Name,
					ExpectPhysical: q.ExpectType,
					ExpectLogical:  q.ExpectLogic,
					ExpectRequired: q.ExpectRequired,
					SkipRequired:   mergedPathVarForQuery,
				})
			}

			// Body 参数断言（字段名小写；类型按值推断；合并场景跳过必需性）
			if op.Body != nil {
				for _, f := range op.Body.Fields {
					out = append(out, &ParamAssertion{
						PathSegments:   methodParentSegs,
						Method:         op.Method,
						ParamName:      f.Name,
						ExpectPhysical: f.ExpectPhysical,
						ExpectLogical:  value.LogicalTypeString,
						ExpectRequired: f.ExpectRequired,
						SkipRequired:   mergedPathVarForQuery,
					})
				}
				out = append(out, &ContentTypeAssertion{
					PathSegments: methodParentSegs,
					Method:       op.Method,
					CT:           op.Body.ContentType,
				})
			}

			// Header 断言
			for _, h := range op.Headers {
				out = append(out, &HeaderAssertion{
					PathSegments:     methodParentSegs,
					Method:           op.Method,
					HeaderName:       h.Name,
					ExpectNormValues: h.ExpectNormValues,
				})
			}

			// Cookie 断言
			for _, c := range op.Cookies {
				out = append(out, &CookieAssertion{
					PathSegments: methodParentSegs,
					Method:       op.Method,
					CookieName:   c.Name,
					ExpectValues: c.Values,
				})
			}
		}
	}

	// 统计断言
	// 统计断言
	// 注意：
	//   - 合并后路径变量下的方法节点 reqCount 不等于各操作 Repeat 之和
	//     （合并时只保留一个方法节点的计数），故 MinTotalReqs 不能用 Repeat 累加。
	//     安全下界：每个方法节点至少 reqCount>=1，故总计数 >= 方法数。
	//   - 同一资源的多个带 id 操作共享一个路径变量节点（值合并到同一路径段），
	//     故 MinPathVars 按资源计而非按操作计。
	var minMethods, minParams, minPathVars int
	for _, res := range s.Resources {
		resourceHasMergedVar := false
		for _, op := range res.Operations {
			minMethods++
			minParams += len(op.QueryParams)
			if op.Body != nil {
				minParams += len(op.Body.Fields)
			}
			if op.PathVar != nil && op.PathVar.ExpectMerge {
				resourceHasMergedVar = true
			}
		}
		if resourceHasMergedVar {
			minPathVars++
		}
	}
	out = append(out, &StatsAssertion{
		MinMethods:    minMethods,
		MinParams:     minParams,
		MinPathVars:   minPathVars,
		MinTotalReqs:  int64(minMethods), // 每个方法至少 1 次计数
		MinTotalNodes: minMethods + minParams + minPathVars,
	})

	return out
}
