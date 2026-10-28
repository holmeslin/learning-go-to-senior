// Package ledger 加總訂單、退款與純數字：數字用泛型，訂單與退款各寫各的迴圈。
package ledger

// Order 是一筆訂單。
type Order struct {
	ID        string
	Cents     int
	Cancelled bool
}

// Refund 是一筆退款。
type Refund struct {
	ID       string
	Cents    int
	Approved bool
}

// ANCHOR: sum

// Number 是 Sum 能加總的數字型別。
type Number interface {
	~int | ~int64 | ~float64
}

// Sum 回傳 xs 所有元素的總和。
func Sum[N Number](xs []N) N {
	var total N
	for _, x := range xs {
		total += x
	}
	return total
}

// ANCHOR_END: sum

// ANCHOR: totals

// OrderTotal 加總沒有取消的訂單金額（分）。
func OrderTotal(orders []Order) int {
	total := 0
	for _, o := range orders {
		if !o.Cancelled {
			total += o.Cents
		}
	}
	return total
}

// RefundTotal 加總已核准的退款金額（分）。
func RefundTotal(refunds []Refund) int {
	total := 0
	for _, r := range refunds {
		if r.Approved {
			total += r.Cents
		}
	}
	return total
}

// ANCHOR_END: totals
