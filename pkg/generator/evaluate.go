package generator

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/router"
)

// EvalReport 一次还原率评估的结果。
// 期望来自 spec 冻结的已知答案，实际来自把 Requests 喂进 ReverseRouter 后的 ListAssets。
type EvalReport struct {
	// Expected 期望出现的方法+路径模板（去重、排序）。
	Expected []string
	// Actual 实际还原出的方法+路径模板。
	Actual []string
	// Missing 期望有、实际没有（漏还原）。
	Missing []string
	// Extra 实际有、期望没有（误合并或多余资产）。
	Extra []string
	// Recall 召回率 = 命中期望数 / 期望总数。期望为空时为 1。
	Recall float64
	// Precision 精确率 = 命中期望数 / 实际总数。实际为空时：期望也为空则 1，否则 0。
	Precision float64
}

// Passed 召回率和精确率都为 1。
func (r EvalReport) Passed() bool {
	return len(r.Missing) == 0 && len(r.Extra) == 0
}

// String 一行摘要，便于日志。
func (r EvalReport) String() string {
	return fmt.Sprintf("recall=%.3f precision=%.3f missing=%v extra=%v", r.Recall, r.Precision, r.Missing, r.Extra)
}

// ExpectedTemplates 从 spec 的已知答案推导期望模板。
// 会合并的路径变量收成 {变量名}；不合并的每个值各成一条固定路径。
func (s *Spec) ExpectedTemplates() []string {
	if s == nil {
		return nil
	}
	seen := map[string]struct{}{}
	var out []string
	add := func(method, template string) {
		key := method + " " + template
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	for _, res := range s.Resources {
		if res == nil {
			continue
		}
		base := "/" + strings.Join(append(append([]string{}, res.Prefix...), res.Name), "/")
		for _, op := range res.Operations {
			if op == nil {
				continue
			}
			method := op.Method
			if method == "" {
				method = "GET"
			}
			if op.PathVar == nil || len(op.PathVar.Values) == 0 {
				add(method, base)
				continue
			}
			if op.PathVar.ExpectMerge {
				name := op.PathVar.ExpectVarName
				if name == "" {
					name = "var"
				}
				add(method, base+"/{"+name+"}")
				continue
			}
			for _, v := range op.PathVar.Values {
				add(method, base+"/"+v)
			}
		}
	}
	sort.Strings(out)
	return out
}

// Evaluate 把 spec 生成的请求喂进一台新路由器，对照期望模板计算召回率与精确率。
// rnd 为 nil 时用 spec.Seed。
func Evaluate(s *Spec, rnd *rand.Rand) (EvalReport, error) {
	report := EvalReport{Expected: s.ExpectedTemplates()}
	if s == nil {
		report.Recall, report.Precision = 1, 1
		return report, nil
	}
	if rnd == nil {
		rnd = rand.New(rand.NewSource(s.Seed))
	}
	r := router.NewReverseRouter()
	for _, req := range s.Requests(rnd) {
		if err := r.ReverseHttpRequest(req); err != nil {
			return report, err
		}
	}
	r.InferRequiredParams()
	for _, asset := range r.ListAssets() {
		report.Actual = append(report.Actual, asset.AssetKey())
	}
	sort.Strings(report.Actual)
	report.Missing, report.Extra = diffKeys(report.Expected, report.Actual)
	hit := len(report.Expected) - len(report.Missing)
	report.Recall = ratio(hit, len(report.Expected))
	report.Precision = ratio(hit, len(report.Actual))
	return report, nil
}

func diffKeys(expected, actual []string) (missing, extra []string) {
	exp := map[string]struct{}{}
	act := map[string]struct{}{}
	for _, k := range expected {
		exp[k] = struct{}{}
	}
	for _, k := range actual {
		act[k] = struct{}{}
	}
	for _, k := range expected {
		if _, ok := act[k]; !ok {
			missing = append(missing, k)
		}
	}
	for _, k := range actual {
		if _, ok := exp[k]; !ok {
			extra = append(extra, k)
		}
	}
	return missing, extra
}

func ratio(hit, total int) float64 {
	if total == 0 {
		if hit == 0 {
			return 1
		}
		return 0
	}
	return float64(hit) / float64(total)
}
