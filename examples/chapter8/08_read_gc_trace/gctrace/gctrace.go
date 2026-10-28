// Package gctrace 把 GODEBUG=gctrace=1 印出的一行 GC trace 拆成欄位，方便逐欄對照 runtime 套件文件。
//
// trace 的格式可能隨 Go 版本改變，這個套件只對應 Go 1.27 的格式；
// 長期觀察 GC 請讀 runtime/metrics，不要靠解析 trace。
package gctrace

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

// ErrFormat 表示這一行不是（這個版本的）gctrace 格式。
var ErrFormat = errors.New("gctrace: unrecognized line format")

// ANCHOR: cycle

// Cycle 是一次 GC 的 trace，欄位順序與 runtime 套件文件的說明一致。
type Cycle struct {
	N          int     // gc #：第幾次 GC
	At         float64 // @#s：程式啟動後經過的秒數
	CPUPercent int     // #%：程式啟動以來，花在 GC 的時間比例

	// 三個階段的牆上時間（毫秒）：stop-the-world 的 sweep termination、並行的 mark and scan、
	// stop-the-world 的 mark termination。
	SweepClock, MarkClock, TermClock float64

	// 各階段的 CPU 時間（毫秒）。mark and scan 再分成 assist（GC 工作插在配置裡，由呼叫端的
	// goroutine 負擔）、background（GC 專用的 worker）與 idle（借用閒置的 P）。
	SweepCPU, AssistCPU, BackgroundCPU, IdleCPU, TermCPU float64

	// heap 在 GC 開始、結束時的大小，標記後存活的量，與 GC 的目標（MB）。
	StartMB, EndMB, LiveMB, GoalMB int

	StacksMB, GlobalsMB int  // 估計要掃描的 stacks 與全域變數大小（MB）
	P                   int  // 使用的處理器（P）數量
	Forced              bool // 是否由 runtime.GC() 強制觸發
}

// ANCHOR_END: cycle

var line = regexp.MustCompile(`^gc (\d+) @([\d.]+)s (\d+)%: ([\d.]+)\+([\d.]+)\+([\d.]+) ms clock, ` +
	`([\d.]+)\+([\d.]+)/([\d.]+)/([\d.]+)\+([\d.]+) ms cpu, ` +
	`(\d+)->(\d+)->(\d+) MB, (\d+) MB goal, (\d+) MB stacks, (\d+) MB globals, (\d+) P( \(forced\))?$`)

// Parse 解析一行 gctrace；不是這個格式的行回傳 [ErrFormat]。
func Parse(s string) (Cycle, error) {
	m := line.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return Cycle{}, ErrFormat
	}
	// 正規表示式只放行數字，所以轉換不會失敗。
	f := make([]float64, 18)
	for i := range f {
		f[i], _ = strconv.ParseFloat(m[i+1], 64)
	}
	return Cycle{
		N: int(f[0]), At: f[1], CPUPercent: int(f[2]),
		SweepClock: f[3], MarkClock: f[4], TermClock: f[5],
		SweepCPU: f[6], AssistCPU: f[7], BackgroundCPU: f[8], IdleCPU: f[9], TermCPU: f[10],
		StartMB: int(f[11]), EndMB: int(f[12]), LiveMB: int(f[13]), GoalMB: int(f[14]),
		StacksMB: int(f[15]), GlobalsMB: int(f[16]), P: int(f[17]),
		Forced: m[19] != "",
	}, nil
}
