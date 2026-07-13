package generator

import (
	"math/rand"
	"strings"
	"testing"
)

// TestGenOneValue_AllPatterns 确定性覆盖 genOneValue 的全部 pattern 分支。
// 此前 genPhone/genPlate 0%、genOneValue 60%（部分分支靠 e2e 随机命中，不稳定）。
// 各 pattern 校验输出格式特征，避免断言固定值（随机 seed 但格式可测）。
func TestGenOneValue_AllPatterns(t *testing.T) {
	rnd := rand.New(rand.NewSource(42))

	cases := []struct {
		name    string
		pattern string
		check   func(t *testing.T, got string)
	}{
		{"integer", patternInteger, func(t *testing.T, got string) {
			if got == "" {
				t.Error("integer 不应为空")
			}
			for _, r := range got {
				if r < '0' || r > '9' {
					t.Errorf("integer 应全数字，得 %s", got)
					break
				}
			}
		}},
		{"uuid", patternUUID, func(t *testing.T, got string) {
			// 形如 8-4-4-4-12
			parts := strings.Split(got, "-")
			if len(parts) != 5 {
				t.Errorf("uuid 应有 5 段，得 %d 段: %s", len(parts), got)
			}
			expectedLens := []int{8, 4, 4, 4, 12}
			for i, p := range parts {
				if len(p) != expectedLens[i] {
					t.Errorf("uuid 第 %d 段长度应为 %d，得 %d (%s)", i, expectedLens[i], len(p), got)
				}
			}
		}},
		{"phone", patternPhone, func(t *testing.T, got string) {
			if len(got) != 11 {
				t.Errorf("phone 应 11 位，得 %d: %s", len(got), got)
			}
			if got[0] != '1' {
				t.Errorf("phone 应以 1 开头，得 %s", got)
			}
			second := got[1]
			if second < '3' || second > '9' {
				t.Errorf("phone 第二位应在 3-9，得 %c", second)
			}
		}},
		{"idcard", patternIDCard, func(t *testing.T, got string) {
			if len(got) != 18 {
				t.Errorf("idcard 应 18 位，得 %d: %s", len(got), got)
			}
			if got[0] == '0' {
				t.Errorf("idcard 首位不应为 0，得 %s", got)
			}
		}},
		{"bankcard", patternBankCard, func(t *testing.T, got string) {
			if got[0] != '6' {
				t.Errorf("bankcard 应以 6 开头，得 %s", got)
			}
			if len(got) < 16 || len(got) > 19 {
				t.Errorf("bankcard 应 16-19 位，得 %d: %s", len(got), got)
			}
		}},
		{"plate", patternPlate, func(t *testing.T, got string) {
			// 省+字母+5-6 位
			// 首字符应是汉字（省），第二位应是字母
			if len(got) < 7 {
				t.Errorf("plate 长度异常: %s", got)
			}
			// 验证首字符是已知省份
			provinces := "京沪粤川苏浙"
			if !strings.ContainsRune(provinces, []rune(got)[0]) {
				t.Errorf("plate 首字符应是省份汉字，得 %s", got)
			}
		}},
		{"prefix", patternPrefix, func(t *testing.T, got string) {
			if !strings.HasPrefix(got, "user_") {
				t.Errorf("prefix 应以 user_ 开头，得 %s", got)
			}
		}},
		{"suffix", patternSuffix, func(t *testing.T, got string) {
			if !strings.HasSuffix(got, "_user") {
				t.Errorf("suffix 应以 _user 结尾，得 %s", got)
			}
		}},
		{"default", "unknown_pattern", func(t *testing.T, got string) {
			if got == "" {
				t.Error("default 分支应返回数字串")
			}
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := genOneValue(rnd, c.pattern, 0)
			c.check(t, got)
		})
	}
}

// TestGenSimilarLengthValues 确定性覆盖 genSimilarLengthValues（突破/不突破由
// 调用方传入 n 决定，非随机）。记忆中"合并场景不稳定点"正是此函数，
// 直接测避免 e2e 概率性命中。
func TestGenSimilarLengthValues(t *testing.T) {
	rnd := rand.New(rand.NewSource(7))

	// 不突破：n=3-5（<6）
	for _, n := range []int{3, 4, 5} {
		vals := genSimilarLengthValues(rnd, n)
		if len(vals) != n {
			t.Errorf("n=%d 应返回 %d 个值，得 %d", n, n, len(vals))
		}
		// 每个值应 5-6 字符
		for _, v := range vals {
			if len(v) < 5 || len(v) > 6 {
				t.Errorf("相似长度值应 5-6 字符，得 %d: %s", len(v), v)
			}
		}
	}

	// 突破：n>=6（>=6 触发分档突破，由 deriveExpectations 判定）
	for _, n := range []int{6, 7, 8} {
		vals := genSimilarLengthValues(rnd, n)
		if len(vals) != n {
			t.Errorf("n=%d 应返回 %d 个值，得 %d", n, n, len(vals))
		}
	}
}
