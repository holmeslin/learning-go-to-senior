// Package loans 查詢圖書館的借閱紀錄。
// 這是 SQL injection 的防禦寫法：值一律走佔位符，不能參數化的部分走白名單。
package loans

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Loan 是一筆借閱紀錄。
type Loan struct {
	// ID 是借閱紀錄的編號。
	ID int
	// Member 是借書的會員。
	Member string
	// Title 是書名。
	Title string
}

// Querier 是 [Store] 需要的查詢能力，*sql.DB 與 *sql.Tx 都符合。
// 它也讓測試能包住它，看見實際送進資料庫的 SQL 與參數。
type Querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// ErrInvalidOrder 表示排序名稱不在白名單裡。
var ErrInvalidOrder = errors.New("invalid order")

// Store 從資料庫讀取借閱紀錄。
type Store struct {
	q Querier
}

// NewStore 回傳用 q 查詢的 Store。
func NewStore(q Querier) *Store { return &Store{q: q} }

// ANCHOR: bymember

// orderClauses 是排序名稱到 ORDER BY 片段的對照，只有這幾種排序能用。
var orderClauses = map[string]string{
	"id":     "id",
	"title":  "title, id",
	"newest": "id DESC",
}

// ByMember 回傳 member 的借閱，依 order 排序（"id"、"title" 或 "newest"）。
// order 不在白名單裡時回傳 [ErrInvalidOrder]，不送出查詢。
func (s *Store) ByMember(ctx context.Context, member, order string) ([]Loan, error) {
	clause, ok := orderClauses[order]
	if !ok {
		return nil, fmt.Errorf("by member: %w: %q", ErrInvalidOrder, order)
	}
	// member 走參數；clause 取自白名單，不是呼叫端給的字串。
	return s.query(ctx, `SELECT id, member, title FROM loans WHERE member = ? ORDER BY `+clause, member)
}

// ANCHOR_END: bymember

// ANCHOR: bytitles

// ByTitles 回傳 member 借過、書名在 titles 裡的紀錄。titles 是空的就不查詢。
func (s *Store) ByTitles(ctx context.Context, member string, titles []string) ([]Loan, error) {
	if len(titles) == 0 {
		return nil, nil // 空的 IN () 不是標準 SQL，沒有書名也不必查
	}
	// 每個書名一個 ?，書名本身放進 args；SQL 字串裡只有 ?，沒有任何資料。
	marks := strings.Join(slices.Repeat([]string{"?"}, len(titles)), ", ")
	args := []any{member}
	for _, t := range titles {
		args = append(args, t)
	}
	return s.query(ctx, `SELECT id, member, title FROM loans WHERE member = ? AND title IN (`+marks+`) ORDER BY id`, args...)
}

// ANCHOR_END: bytitles

func (s *Store) query(ctx context.Context, query string, args ...any) ([]Loan, error) {
	rows, err := s.q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query loans: %w", err)
	}
	defer rows.Close()

	var out []Loan
	for rows.Next() {
		var l Loan
		if err := rows.Scan(&l.ID, &l.Member, &l.Title); err != nil {
			return nil, fmt.Errorf("scan loan: %w", err)
		}
		out = append(out, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("query loans: %w", err)
	}
	return out, nil
}
