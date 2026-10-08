package router

import (
	"sync"
	"testing"
)

func TestGapProject_DoubleCheckUnderLock(t *testing.T) {
	m := NewProjectManager()
	const n = 64
	var wg sync.WaitGroup
	got := make([]*RouterSet, n)
	errs := make([]error, n)
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			got[i], errs[i] = m.Project("same")
		}(i)
	}
	close(start)
	wg.Wait()
	for i, err := range errs {
		if err != nil || got[i] == nil {
			t.Fatalf("并发建项目失败 i=%d err=%v", i, err)
		}
		if got[i] != got[0] {
			t.Fatal("同一项目应返回同一个 RouterSet")
		}
	}
}
