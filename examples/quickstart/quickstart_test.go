package main

import (
	"bufio"
	"bytes"
	"os"
	"strings"
	"testing"
)

// TestMain_Smoke 调用 main() 验证 quickstart 端到端可运行且无 panic。
// 捕获 stdout 避免污染测试输出。8 条 curl 全合法，main 不会触发 log.Fatalf。
func TestMain_Smoke(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	done := make(chan struct{})
	var panicked bool
	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		main()
	}()

	<-done
	w.Close()
	os.Stdout = old

	if panicked {
		t.Fatal("main() panic")
	}

	var buf bytes.Buffer
	scanner := bufio.NewScanner(r)
	found := false
	for scanner.Scan() {
		line := scanner.Text()
		buf.WriteString(line + "\n")
		// 批量喂入结果应显示成功 8 条
		if strings.Contains(line, "成功 8 条") {
			found = true
		}
	}
	if !found {
		t.Errorf("quickstart 输出未含'成功 8 条'，实际输出：\n%s", buf.String())
	}
}
