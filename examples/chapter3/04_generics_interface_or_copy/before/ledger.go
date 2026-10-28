// Package ledger 加總訂單、退款與純數字。這是「長得像的迴圈全部併進一個 Total」的版本。
package ledger

// ANCHOR: constraint

// Number 是 Total 能加總的數字型別。
type Number interface {
	~int | ~int64 | ~float64
}

// Entry 是 Total 能加總的紀錄。
type Entry[N Number] interface {
	Amount() N    // 為了訂單、退款與純數字：金額有 int 也有 float64，所以 Total 多了型別參數 N
	Counts() bool // 為了取消的訂單與未核准的退款：不是每一筆都要算
}

// Total 加總 entries 裡要算的紀錄。
func Total[N Number, E Entry[N]](entries []E) N {
	var sum N
	for _, e := range entries {
		if e.Counts() {
			sum += e.Amount()
		}
	}
	return sum
}

// ANCHOR_END: constraint

// ANCHOR: adapters

// Order 是一筆訂單。
type Order struct {
	ID        string
	Cents     int
	Cancelled bool
}

// Amount 回傳訂單金額（分）。
func (o Order) Amount() int { return o.Cents }

// Counts 回報這筆訂單要不要算：取消的不算。
func (o Order) Counts() bool { return !o.Cancelled }

// Refund 是一筆退款。
type Refund struct {
	ID       string
	Cents    int
	Approved bool
}

// Amount 回傳退款金額（分）。
func (r Refund) Amount() int { return r.Cents }

// Counts 回報這筆退款要不要算：只算已核准的。
func (r Refund) Counts() bool { return r.Approved }

// ANCHOR_END: adapters

// ANCHOR: wrappers

// Cents 只是為了讓純粹的 []int 也能傳進 Total，沒有任何規則。
type Cents int

// Amount 回傳自己的數值。
func (c Cents) Amount() int { return int(c) }

// Counts 永遠回傳 true。
func (c Cents) Counts() bool { return true }

// GB 只是為了讓純粹的 []float64 也能傳進 Total，沒有任何規則。
type GB float64

// Amount 回傳自己的數值。
func (g GB) Amount() float64 { return float64(g) }

// Counts 永遠回傳 true。
func (g GB) Counts() bool { return true }

// ANCHOR_END: wrappers
