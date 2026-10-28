package visits_test

import (
	"fmt"
	"sync"
	"time"

	visits "github.com/holmeslin/learning-go-to-senior/examples/chapter3/06_embedding_or_delegation/before"
)

// ANCHOR: counter

func ExampleCounter() {
	var c visits.Counter
	_, isLocker := any(&c).(sync.Locker)
	fmt.Println("is a sync.Locker:", isLocker)

	c.Lock() // 外面的人也能鎖住它，Counter 擋不住
	done := make(chan struct{})
	go func() {
		c.Incr("/home")
		close(done)
	}()
	select {
	case <-done:
		fmt.Println("Incr finished")
	case <-time.After(50 * time.Millisecond):
		fmt.Println("Incr is blocked until the caller unlocks")
	}
	c.Unlock()
	<-done
	// Output:
	// is a sync.Locker: true
	// Incr is blocked until the caller unlocks
}

// ANCHOR_END: counter

// ANCHOR: audited

func ExampleAudited() {
	var c visits.Counter
	log := func(msg string) { fmt.Println("audit:", msg) }

	a := &visits.Audited{Store: &c, Log: log}
	a.Incr("/home")  // 有覆寫：有稽核
	a.Reset("/home") // 沒覆寫：直接穿過去，沒有稽核
	fmt.Println("count after reset:", a.Count("/home"))

	forgot := &visits.Audited{Log: log} // 忘了設 Store，編譯器不會提醒
	defer func() { fmt.Println("panic:", recover()) }()
	forgot.Count("/home")
	// Output:
	// audit: incr /home
	// count after reset: 0
	// panic: runtime error: invalid memory address or nil pointer dereference
}

// ANCHOR_END: audited
