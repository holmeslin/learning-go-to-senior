package ingest_test

import (
	"errors"
	"strconv"
	"testing"

	ingest "github.com/holmeslin/learning-go-to-senior/examples/chapter8/04_heap_growth/after"
	old "github.com/holmeslin/learning-go-to-senior/examples/chapter8/04_heap_growth/before"
)

// ANCHOR: cache

// 筆數永遠不超過上限；超過時丟掉最久沒被用到的。
func TestCacheEvictsLeastRecentlyUsed(t *testing.T) {
	c := ingest.NewCache(2)
	c.Put("a", "1")
	c.Put("b", "2")
	c.Get("a") // a 變成最近用過的
	c.Put("c", "3")

	if got, want := c.Len(), 2; got != want {
		t.Fatalf("Len() = %d, want %d", got, want)
	}
	if _, ok := c.Get("b"); ok {
		t.Error("b should be evicted")
	}
	for _, key := range []string{"a", "c"} {
		if _, ok := c.Get(key); !ok {
			t.Errorf("%s should stay", key)
		}
	}
}

// ANCHOR_END: cache

// 還沒到上限時，行為和 before 一致，已有的 key 只覆蓋、不增加筆數。
func TestCacheMatchesOldUnderLimit(t *testing.T) {
	c, o := ingest.NewCache(10), old.NewCache()
	for i := range 5 {
		key := strconv.Itoa(i % 3)
		c.Put(key, strconv.Itoa(i))
		o.Put(key, strconv.Itoa(i))
	}
	if c.Len() != o.Len() {
		t.Fatalf("Len() = %d, want %d", c.Len(), o.Len())
	}
	for _, key := range []string{"0", "1", "2", "3"} {
		got, gotOK := c.Get(key)
		want, wantOK := o.Get(key)
		if got != want || gotOK != wantOK {
			t.Errorf("Get(%q) = %q, %v; want %q, %v", key, got, gotOK, want, wantOK)
		}
	}
}

func TestCacheMinimumSize(t *testing.T) {
	c := ingest.NewCache(0)
	c.Put("a", "1")
	if got, want := c.Len(), 1; got != want {
		t.Errorf("Len() = %d, want %d", got, want)
	}
}

// ANCHOR: seen

// 記住的代號只占自己的 16 位元組，不再拉著整筆通知的陣列。
func TestSeenCopiesID(t *testing.T) {
	payload := make([]byte, 64<<10)
	copy(payload, "0123456789abcdef")

	var s ingest.Seen
	if err := s.Add(payload); err != nil {
		t.Fatal(err)
	}
	payload[0] = 'X' // 通知的緩衝區之後會被重複使用

	id := s.IDs()[0]
	if got, want := string(id), "0123456789abcdef"; got != want {
		t.Errorf("id = %q, want %q", got, want)
	}
	if got, want := cap(id), ingest.IDLen; got != want {
		t.Errorf("cap(id) = %d, want %d", got, want)
	}
}

// ANCHOR_END: seen

func TestSeenShortPayload(t *testing.T) {
	var s ingest.Seen
	err := s.Add(make([]byte, ingest.IDLen-1))
	if !errors.Is(err, ingest.ErrShortPayload) {
		t.Errorf("Add() error = %v, want %v", err, ingest.ErrShortPayload)
	}
	if got := len(s.IDs()); got != 0 {
		t.Errorf("len(IDs()) = %d, want 0", got)
	}
}

// 重建前後，還在等待的通知一筆不少，也沒有多出已送達的。
func TestPendingCompactKeepsEntries(t *testing.T) {
	p := ingest.NewPending()
	for id := range uint64(1000) {
		p.Start(id, ingest.Delivery{Endpoint: "https://example.com/hook", Attempts: int(id)})
	}
	for id := range uint64(990) {
		p.Done(id)
	}

	p.Compact()

	if got, want := p.Len(), 10; got != want {
		t.Fatalf("Len() = %d, want %d", got, want)
	}
	p.Done(995)
	if got, want := p.Len(), 9; got != want {
		t.Errorf("Len() after Done = %d, want %d", got, want)
	}
	p.Done(0) // 早就送達了，重建後不會回來
	if got, want := p.Len(), 9; got != want {
		t.Errorf("Len() after Done of a finished id = %d, want %d", got, want)
	}
}
