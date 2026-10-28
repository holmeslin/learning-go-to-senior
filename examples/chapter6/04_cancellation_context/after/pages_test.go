package crawl_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"testing/synctest"

	crawl "github.com/holmeslin/learning-go-to-senior/examples/chapter6/04_cancellation_context/after"
)

func echo(_ context.Context, url string) string { return "page " + url }

func TestPagesDeliversAllInOrder(t *testing.T) {
	var got []string
	for page := range crawl.Pages(t.Context(), []string{"/a", "/b", "/c"}, echo) {
		got = append(got, page)
	}

	want := []string{"page /a", "page /b", "page /c"}
	if !slices.Equal(got, want) {
		t.Errorf("Pages() = %v, want %v", got, want)
	}
}

// ANCHOR: abandoned

// 呼叫端取消之後不再接收：送出那一行必須能被取消，goroutine 才會結束。
// 若不能，bubble 結束時還有 goroutine 卡在送出，synctest.Test 會 panic。
func TestPagesStopsWhenNobodyReceives(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		out := crawl.Pages(ctx, []string{"/a", "/b", "/c"}, echo)

		<-out // 收下第一筆就走
		cancel()
		synctest.Wait() // 等 bubble 裡其他 goroutine 都結束或卡住

		select {
		case _, ok := <-out:
			if ok {
				t.Error("received a page after cancel, want closed channel")
			}
		default:
			t.Error("channel still open after cancel, goroutine is stuck")
		}
	})
}

// ANCHOR_END: abandoned

// ANCHOR: cpu

// countingCtx 數 Err 被呼叫了幾次。
type countingCtx struct {
	context.Context
	checks int
}

func (c *countingCtx) Err() error {
	c.checks++
	return c.Context.Err()
}

func TestSumChecksOncePerChunk(t *testing.T) {
	values := make([]int, 10*crawl.CheckEvery)
	for i := range values {
		values[i] = 1
	}
	ctx := &countingCtx{Context: t.Context()}

	got, err := crawl.Sum(ctx, values)

	if err != nil || got != len(values) {
		t.Fatalf("Sum() = %d, %v, want %d, nil", got, err, len(values))
	}
	if want := 10; ctx.checks != want {
		t.Errorf("Sum() checked ctx %d times for %d values, want %d", ctx.checks, len(values), want)
	}
}

func TestSumStopsWhenCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := crawl.Sum(ctx, make([]int, 10*crawl.CheckEvery))

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Sum() error = %v, want context.Canceled", err)
	}
}

// ANCHOR_END: cpu
