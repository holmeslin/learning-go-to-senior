// Package roster 是會員名冊服務：名冊長駐在記憶體，每個請求帶一批會員 ID，回傳報表。
package roster

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
)

type member struct {
	name string
	tier string
}

// Roster 是服務啟動時載入、之後一直留在記憶體裡的會員名冊。
type Roster struct {
	byID map[int]*member
}

// Load 建立有 n 位會員（ID 為 0 到 n-1）的名冊。真實服務會從資料庫載入，這裡用固定的合成資料代替。
func Load(n int) *Roster {
	tiers := [...]string{"free", "pro", "team"}
	byID := make(map[int]*member, n)
	for id := range n {
		byID[id] = &member{name: "member-" + strconv.Itoa(id), tier: tiers[id%len(tiers)]}
	}
	return &Roster{byID: byID}
}

// ANCHOR: handle

// Handle 處理一批會員 ID，回傳每位會員一行的報表，不在名冊裡的 ID 略過。
// 怕處理完的批次堆在記憶體裡，所以每批結束就手動做一次 GC。
func (r *Roster) Handle(ids []int) []byte {
	lines := make([]string, 0, len(ids))
	for _, id := range ids {
		if m, ok := r.byID[id]; ok {
			lines = append(lines, fmt.Sprintf("%d,%s,%s", id, m.name, m.tier))
		}
	}
	out := []byte(strings.Join(lines, "\n"))
	runtime.GC()
	return out
}

// ANCHOR_END: handle
