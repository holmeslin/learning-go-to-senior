package tally_test

import (
	"testing"

	tally "github.com/holmeslin/learning-go-to-senior/examples/chapter1/06_zero_value_or_constructor/after"
)

// ANCHOR: test

func TestCounterZeroValueInStruct(t *testing.T) {
	type server struct {
		hits tally.Counter // 沒有初始化，就像嵌在別的 struct 裡
	}
	var s server
	s.hits.Add("/home")
	if got := s.hits.Count("/home"); got != 1 {
		t.Fatalf("Count = %d, want 1", got)
	}
}

// ANCHOR_END: test

// ANCHOR: zeroreporter

func TestReporterZeroValueReturnsError(t *testing.T) {
	var c tally.Counter
	c.Add("/home")

	var r tally.Reporter // 沒走 NewReporter：沒有寫入目標
	if err := r.Report(&c); err == nil {
		t.Fatal("零值 Reporter 應該回傳錯誤，而不是 panic 或默默成功")
	}
	if err := tally.NewReporter(nil).Report(&c); err == nil {
		t.Fatal("NewReporter(nil) 之後 Report 應該回傳錯誤")
	}
}

// ANCHOR_END: zeroreporter

func TestCounterZeroValueReads(t *testing.T) {
	var c tally.Counter
	if got := c.Count("/home"); got != 0 {
		t.Fatalf("Count = %d, want 0", got)
	}
	if got := c.Paths(); len(got) != 0 {
		t.Fatalf("Paths = %v, want empty", got)
	}
}
