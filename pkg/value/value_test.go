package value

import (
	"sync"
	"testing"
)

func TestNewValueMetric(t *testing.T) {
	vm := NewValueMetric()
	if vm == nil {
		t.Fatal("NewValueMetric 不应返回 nil")
	}
	if !vm.IsEmpty() {
		t.Error("新创建的 ValueMetric 应该是空的")
	}
}

func TestValueMetric_AddValue(t *testing.T) {
	vm := NewValueMetric()

	vm.AddValue("hello")
	if vm.IsEmpty() {
		t.Error("添加值后不应该为空")
	}
	if vm.GetValueCount("hello") != 1 {
		t.Errorf("hello 的计数应该是1，实际: %d", vm.GetValueCount("hello"))
	}

	vm.AddValue("hello")
	if vm.GetValueCount("hello") != 2 {
		t.Errorf("重复添加后 hello 的计数应该是2，实际: %d", vm.GetValueCount("hello"))
	}

	vm.AddValue("world")
	if vm.GetUniqueValueCount() != 2 {
		t.Errorf("不同值的数量应该是2，实际: %d", vm.GetUniqueValueCount())
	}
}

func TestValueMetric_GetAllValues(t *testing.T) {
	vm := NewValueMetric()
	vm.AddValue("a")
	vm.AddValue("b")
	vm.AddValue("a")

	all := vm.GetAllValues()
	if len(all) != 2 {
		t.Errorf("应该有2个不同的值，实际: %d", len(all))
	}
	if all["a"] != 2 {
		t.Errorf("a 的计数应该是2，实际: %d", all["a"])
	}
	if all["b"] != 1 {
		t.Errorf("b 的计数应该是1，实际: %d", all["b"])
	}
}

func TestValueMetric_GetTotalCount(t *testing.T) {
	vm := NewValueMetric()
	vm.AddValue("a")
	vm.AddValue("b")
	vm.AddValue("a")
	vm.AddValue("c")

	if vm.GetTotalCount() != 4 {
		t.Errorf("总计数应该是4，实际: %d", vm.GetTotalCount())
	}
}

func TestValueMetric_ConcurrentAccess(t *testing.T) {
	vm := NewValueMetric()
	var wg sync.WaitGroup

	// 并发添加值
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			vm.AddValue(string(rune('a' + i%5)))
		}(i)
	}

	wg.Wait()

	if vm.GetTotalCount() != 100 {
		t.Errorf("并发添加后总计数应该是100，实际: %d", vm.GetTotalCount())
	}
}

func TestValueMetric_GetValueCountNonExistent(t *testing.T) {
	vm := NewValueMetric()
	if vm.GetValueCount("nonexistent") != 0 {
		t.Error("不存在的值的计数应该是0")
	}
}

// 测试中国特有逻辑类型常量
func TestLogicalType_ChineseFormats(t *testing.T) {
	tests := []struct {
		name    string
		ltype   LogicalType
		wantStr string
	}{
		{"手机号", LogicalTypePhoneNumber, "phone"},
		{"身份证号", LogicalTypeIDCard, "idcard"},
		{"银行卡号", LogicalTypeBankCard, "bankcard"},
		{"车牌号", LogicalTypePlateNumber, "plate"},
		{"邮政编码", LogicalTypePostalCode, "postalcode"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.ltype) != tt.wantStr {
				t.Errorf("%s 类型常量值错误，期望 '%s'，实际 '%s'", tt.name, tt.wantStr, string(tt.ltype))
			}
			// 确保类型不为空
			if tt.ltype == "" {
				t.Errorf("%s 类型常量不应为空", tt.name)
			}
			// 确保与其他类型不同
			if tt.ltype == LogicalTypeString {
				t.Errorf("%s 不应等于 string", tt.name)
			}
		})
	}
}
