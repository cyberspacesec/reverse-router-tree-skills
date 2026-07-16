package router

import (
	"testing"

	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/node"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/request"
	"github.com/cyberspacesec/reverse-router-tree-skills/pkg/value"
)

// TestInferenceCache_PathVarIncremental 验证增量推断缓存语义：
// 喂重复值（uniqueCount 不变）不触发新推断，喂新值（uniqueCount 变化）才重算。
// 用 TypeInferences 统计指标间接验证调用次数。
func TestInferenceCache_PathVarIncremental(t *testing.T) {
	r := newSilentRouter()
	// 喂 3 个不同 ID 触发合并 + 推断
	for _, id := range []string{"101", "102", "103"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/users/"+id, nil, "GET", nil))
	}
	stats1 := r.GetStats()
	// 再喂已存在的值（重复 101），uniqueCount 不变，不应触发新推断
	r.ReverseHttpRequest(request.NewHttpRequest("/api/users/101", nil, "GET", nil))
	stats2 := r.GetStats()
	if stats2.TypeInferences != stats1.TypeInferences {
		t.Errorf("重复值不应触发新推断：before=%d after=%d", stats1.TypeInferences, stats2.TypeInferences)
	}
	// 喂新值 104，uniqueCount 变化，应触发推断
	r.ReverseHttpRequest(request.NewHttpRequest("/api/users/104", nil, "GET", nil))
	stats3 := r.GetStats()
	if stats3.TypeInferences <= stats2.TypeInferences {
		t.Errorf("新值应触发推断：before=%d after=%d", stats2.TypeInferences, stats3.TypeInferences)
	}
	// 最终类型仍正确（integer）——增量推断不改变最终类型结果
	usersNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("users")
	varNode := usersNode.GetChildByType("request_path_variable")
	if varNode == nil {
		t.Fatal("应存在路径变量节点")
	}
	pv := varNode.(*node.RequestPathVariableNode)
	if pv.GetValueType() != value.Type(value.PhysicalTypeInteger) {
		t.Errorf("增量推断后路径变量类型仍应为 integer，实际 %s", pv.GetValueType())
	}
}

// TestInferenceCache_ParamIncremental 验证参数节点的增量推断缓存。
// page=1 反复出现不应每次都重算，page=2 新值才触发。
func TestInferenceCache_ParamIncremental(t *testing.T) {
	r := newSilentRouter()
	// page=1 三次（同值），unique 不变
	for i := 0; i < 3; i++ {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/list?page=1", nil, "GET", nil))
	}
	stats1 := r.GetStats()
	// 再喂 page=1，不应触发新推断
	r.ReverseHttpRequest(request.NewHttpRequest("/api/list?page=1", nil, "GET", nil))
	stats2 := r.GetStats()
	if stats2.TypeInferences != stats1.TypeInferences {
		t.Errorf("参数重复值不应触发新推断：before=%d after=%d", stats1.TypeInferences, stats2.TypeInferences)
	}
	// 喂 page=2 新值，应触发
	r.ReverseHttpRequest(request.NewHttpRequest("/api/list?page=2", nil, "GET", nil))
	stats3 := r.GetStats()
	if stats3.TypeInferences <= stats2.TypeInferences {
		t.Errorf("参数新值应触发推断：before=%d after=%d", stats2.TypeInferences, stats3.TypeInferences)
	}
	// 参数类型仍正确（integer）
	listNode := r.Tree.Root.FindChildByKey("api").FindChildByKey("list")
	getNode := listNode.FindChildByKey("GET")
	if getNode == nil {
		t.Fatal("应存在 GET 方法节点")
	}
	pageNode := getNode.FindChildByKey("page")
	if pageNode == nil {
		t.Fatal("应存在 page 参数节点")
	}
	p := pageNode.(*node.RequestParamNode)
	if p.GetValueType() != value.Type(value.PhysicalTypeInteger) {
		t.Errorf("增量推断后 page 类型仍应为 integer，实际 %s", p.GetValueType())
	}
}

// TestInferenceCache_FirstValueTriggers 验证合并生成的新变量节点首次推断会被触发
// （mergeSiblings 推断后设 lastInferredUniqueCount=当前uniqueCount，使后续命中走增量）。
// 路径变量节点仅在合并（≥3 兄弟）后才存在，故用 3 个不同值触发合并验证首次推断非 0。
func TestInferenceCache_FirstValueTriggers(t *testing.T) {
	r := newSilentRouter()
	// 3 个不同 ID 触发合并，生成路径变量节点并首次推断
	for _, id := range []string{"1", "2", "3"} {
		r.ReverseHttpRequest(request.NewHttpRequest("/api/items/"+id, nil, "GET", nil))
	}
	stats := r.GetStats()
	if stats.TypeInferences == 0 {
		t.Error("合并后新变量节点应触发首次推断，TypeInferences 不应为 0")
	}
	// 合并后 cache 应已设置：再喂已合并范围内的重复值（如 1）不应触发新推断
	stats1 := r.GetStats()
	r.ReverseHttpRequest(request.NewHttpRequest("/api/items/1", nil, "GET", nil))
	stats2 := r.GetStats()
	if stats2.TypeInferences != stats1.TypeInferences {
		t.Errorf("合并后重复值不应触发新推断：before=%d after=%d", stats1.TypeInferences, stats2.TypeInferences)
	}
}
