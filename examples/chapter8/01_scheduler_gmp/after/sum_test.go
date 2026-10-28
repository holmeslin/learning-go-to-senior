package blobsum_test

import (
	"crypto/sha256"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"testing/synctest"

	blobsum "github.com/holmeslin/learning-go-to-senior/examples/chapter8/01_scheduler_gmp/after"
	before "github.com/holmeslin/learning-go-to-senior/examples/chapter8/01_scheduler_gmp/before"
)

// ANCHOR: example

func ExampleSum() {
	sums := blobsum.Sum([][]byte{[]byte("abc"), []byte("")})
	fmt.Printf("%x\n", sums[0][:4])
	fmt.Printf("%x\n", sums[1][:4])
	// Output:
	// ba7816bf
	// e3b0c442
}

// ANCHOR_END: example

// ANCHOR: follows

// 並行數跟著 GOMAXPROCS：設成 p，同時執行的 fn 最多就是 p。
// 測試改了全域設定，所以用 t.Cleanup 還原，也不呼叫 t.Parallel。
func TestForEachFollowsGOMAXPROCS(t *testing.T) {
	for _, procs := range []int{1, 2, 4} {
		t.Run(fmt.Sprintf("procs=%d", procs), func(t *testing.T) {
			prev := runtime.GOMAXPROCS(procs)
			t.Cleanup(func() { runtime.GOMAXPROCS(prev) })

			synctest.Test(t, func(t *testing.T) {
				var mu sync.Mutex
				inflight, peak := 0, 0
				release := make(chan struct{})
				done := make(chan struct{})
				go func() {
					defer close(done)
					blobsum.ForEach(procs*4, func(int) {
						mu.Lock()
						inflight++
						peak = max(peak, inflight)
						mu.Unlock()

						<-release // 真的阻塞，直到測試放行

						mu.Lock()
						inflight--
						mu.Unlock()
					})
				}()

				synctest.Wait() // 全部卡住了：每個 worker 各卡在一個 fn，其餘編號沒人領
				mu.Lock()
				if inflight != procs {
					t.Errorf("fn in flight while blocked = %d, want %d", inflight, procs)
				}
				mu.Unlock()

				close(release)
				<-done
				if peak != procs {
					t.Errorf("peak concurrency = %d, want %d", peak, procs)
				}
			})
		})
	}
}

// ANCHOR_END: follows

func TestForEachCallsEveryIndexOnce(t *testing.T) {
	const n = 1000
	var calls [n]int
	blobsum.ForEach(n, func(i int) { calls[i]++ })
	for i, c := range calls {
		if c != 1 {
			t.Fatalf("fn(%d) called %d times, want 1", i, c)
		}
	}
}

func TestForEachEmpty(t *testing.T) {
	blobsum.ForEach(0, func(int) { t.Error("fn must not be called when n is 0") })
}

// 結果必須和 before 一致：改並行數不能改變輸出。
func TestSumMatchesBefore(t *testing.T) {
	blobs := make([][]byte, 50)
	for i := range blobs {
		blobs[i] = fmt.Appendf(nil, "blob-%d", i)
	}
	got, want := blobsum.Sum(blobs), before.Sum(blobs)
	for i := range blobs {
		if got[i] != want[i] || got[i] != sha256.Sum256(blobs[i]) {
			t.Errorf("Sum()[%d] = %x, want %x", i, got[i], want[i])
		}
	}
}
