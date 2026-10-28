package visits_test

import (
	"slices"
	"sync"
	"testing"

	visits "github.com/holmeslin/learning-go-to-senior/examples/chapter3/06_embedding_or_delegation/after"
)

// 改成具名欄位之後，Mutex 的方法不再是 Counter 的 API。
func TestCounterHidesMutex(t *testing.T) {
	var c any = &visits.Counter{}
	if _, ok := c.(sync.Locker); ok {
		t.Error("Counter still satisfies sync.Locker")
	}
	if _, ok := c.(interface{ TryLock() bool }); ok {
		t.Error("Counter still exposes TryLock")
	}
}

// 鎖是 Counter 自己的事：並行呼叫 Incr 不會漏算（搭配 -race）。
func TestCounterConcurrentIncr(t *testing.T) {
	var c visits.Counter
	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() { c.Incr("/home") })
	}
	wg.Wait()
	if got := c.Count("/home"); got != 50 {
		t.Fatalf("Count = %d, want 50", got)
	}
}

// 每一個會寫入的方法都有稽核，並且真的轉給底下的 Store。
func TestAuditedLogsEveryWrite(t *testing.T) {
	var c visits.Counter
	var log []string
	a := visits.NewAudited(&c, func(msg string) { log = append(log, msg) })

	a.Incr("/home")
	if got := c.Count("/home"); got != 1 {
		t.Fatalf("Count after Incr = %d, want 1", got)
	}
	a.Reset("/home")
	if got := a.Count("/home"); got != 0 {
		t.Fatalf("Count after Reset = %d, want 0", got)
	}
	if want := []string{"incr /home", "reset /home"}; !slices.Equal(log, want) {
		t.Fatalf("log = %q, want %q", log, want)
	}
}
