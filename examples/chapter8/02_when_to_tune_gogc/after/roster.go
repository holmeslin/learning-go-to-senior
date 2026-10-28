// Package roster 是會員名冊服務：名冊長駐在記憶體，每個請求帶一批會員 ID，回傳報表。
package roster

import "strconv"

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

// avgLineLen 是一行報表的估計長度，用來一次預留緩衝區；估少了 append 會自己擴張，結果不變。
const avgLineLen = 32

// Handle 處理一批會員 ID，回傳每位會員一行的報表，不在名冊裡的 ID 略過。
// 緩衝區一次配好、直接寫入，不另外做 GC：回收交給 runtime。
func (r *Roster) Handle(ids []int) []byte {
	out := make([]byte, 0, len(ids)*avgLineLen)
	for _, id := range ids {
		m, ok := r.byID[id]
		if !ok {
			continue
		}
		if len(out) > 0 {
			out = append(out, '\n')
		}
		out = strconv.AppendInt(out, int64(id), 10)
		out = append(out, ',')
		out = append(out, m.name...)
		out = append(out, ',')
		out = append(out, m.tier...)
	}
	return out
}

// ANCHOR_END: handle
