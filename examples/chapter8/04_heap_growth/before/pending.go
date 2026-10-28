package ingest

// Delivery 是一筆等待送達的通知。
type Delivery struct {
	Endpoint string
	Attempts int
}

// ANCHOR: pending

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

// Done 移除已送達的通知。
func (p *Pending) Done(id uint64) {
	delete(p.m, id)
}

// ANCHOR_END: pending

// Len 回傳還在等待的通知數。
func (p *Pending) Len() int {
	return len(p.m)
}
