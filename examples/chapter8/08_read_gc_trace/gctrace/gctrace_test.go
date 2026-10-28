package gctrace_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter8/08_read_gc_trace/gctrace"
)

// ANCHOR: example

func ExampleParse() {
	// 取自 GODEBUG=gctrace=1 的某一次輸出（macOS，14 個 P）。
	line := "gc 12 @0.133s 9%: 0.10+2.5+0.016 ms clock, 1.4+10/8.8/0.020+0.22 ms cpu, " +
		"349->414->201 MB, 410 MB goal, 0 MB stacks, 0 MB globals, 14 P"

	c, err := gctrace.Parse(line)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("第 %d 次 GC，程式啟動後 %.3f 秒，GC 占 %d%%\n", c.N, c.At, c.CPUPercent)
	fmt.Printf("牆上時間 (ms)：%v + %v + %v\n", c.SweepClock, c.MarkClock, c.TermClock)
	fmt.Printf("CPU 時間 (ms)：%v + %v/%v/%v + %v\n", c.SweepCPU, c.AssistCPU, c.BackgroundCPU, c.IdleCPU, c.TermCPU)
	fmt.Printf("heap：開始 %d、結束 %d、存活 %d、目標 %d (MB)\n", c.StartMB, c.EndMB, c.LiveMB, c.GoalMB)
	fmt.Printf("stacks %d MB、globals %d MB、%d 個 P、forced=%t\n", c.StacksMB, c.GlobalsMB, c.P, c.Forced)
	// Output:
	// 第 12 次 GC，程式啟動後 0.133 秒，GC 占 9%
	// 牆上時間 (ms)：0.1 + 2.5 + 0.016
	// CPU 時間 (ms)：1.4 + 10/8.8/0.02 + 0.22
	// heap：開始 349、結束 414、存活 201、目標 410 (MB)
	// stacks 0 MB、globals 0 MB、14 個 P、forced=false
}

// ANCHOR_END: example

func TestParseForced(t *testing.T) {
	line := "gc 1 @0.001s 0%: 0.011+0.45+0.002 ms clock, 0.13+0/0.21/0.37+0.027 ms cpu, " +
		"4->4->0 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 12 P (forced)"
	c, err := gctrace.Parse(line)
	if err != nil {
		t.Fatal(err)
	}
	if !c.Forced || c.N != 1 || c.P != 12 {
		t.Errorf("Parse() = %+v, want forced cycle 1 with 12 P", c)
	}
}

func TestParseRejectsOtherLines(t *testing.T) {
	for _, line := range []string{
		"",
		"scvg: 4 MB released",
		"gc 12 @0.133s 9%: 0.10+2.5+0.016 ms clock",
	} {
		if _, err := gctrace.Parse(line); !errors.Is(err, gctrace.ErrFormat) {
			t.Errorf("Parse(%q) error = %v, want ErrFormat", line, err)
		}
	}
}
