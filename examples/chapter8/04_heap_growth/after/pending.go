package ingest

// ANCHOR: pending

// Delivery 是一筆等待送達的通知。
type Delivery struct {
	Endpoint string
	Attempts int
}

// Pending 記住還沒送達的通知，送達後移除。
type Pending struct {
	m map[uint64]Delivery
}

// NewPending 回傳空的 Pending。
func NewPending() *Pending {
	return &Pending{m: make(map[uint64]Delivery)}
}

// Start 登記一筆等待送達的通知。
func (p *Pending) Start(id uint64, d Delivery) {
	p.m[id] = d
}

// Done 移除已送達的通知。map 刪除後不會縮小，尖峰過後由呼叫端用 Compact 重建。
func (p *Pending) Done(id uint64) {
	delete(p.m, id)
}

// Compact 把還在等待的通知搬進一張剛好夠大的新 map，舊的交給 GC。
func (p *Pending) Compact() {
	m := make(map[uint64]Delivery, len(p.m))
	for id, d := range p.m {
		m[id] = d
	}
	p.m = m
}

// ANCHOR_END: pending

// Len 回傳還在等待的通知數。
func (p *Pending) Len() int {
	return len(p.m)
}
