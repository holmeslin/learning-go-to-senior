// Package loans 查詢圖書館的借閱紀錄。
//
// 這個套件是刻意示範的 SQL injection 漏洞，不要照用：
// 它把呼叫端給的字串直接拼進 SQL。可以參考的寫法在 after。
package loans

import (
	"context"
	"database/sql"
	"fmt"
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
type Querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// Store 從資料庫讀取借閱紀錄。
type Store struct {
	q Querier
}

// NewStore 回傳用 q 查詢的 Store。
func NewStore(q Querier) *Store { return &Store{q: q} }

// ANCHOR: bymember

// ByMember 回傳 member 的借閱，依 order 排序。
// 刻意示範的漏洞：member 與 order 都直接拼進 SQL。
func (s *Store) ByMember(ctx context.Context, member, order string) ([]Loan, error) {
	return s.query(ctx, fmt.Sprintf(
		`SELECT id, member, title FROM loans WHERE member = '%s' ORDER BY %s`, member, order))
}

// ANCHOR_END: bymember

// ANCHOR: bytitles

// ByTitles 回傳 member 借過、書名在 titles 裡的紀錄。
// 刻意示範的漏洞：每個書名都拼進 IN (...)。
func (s *Store) ByTitles(ctx context.Context, member string, titles []string) ([]Loan, error) {
	if len(titles) == 0 {
		return nil, nil
	}
	quoted := make([]string, len(titles))
	for i, t := range titles {
		quoted[i] = "'" + t + "'"
	}
	return s.query(ctx, fmt.Sprintf(
		`SELECT id, member, title FROM loans WHERE member = '%s' AND title IN (%s) ORDER BY id`,
		member, strings.Join(quoted, ", ")))
}

// ANCHOR_END: bytitles

func (s *Store) query(ctx context.Context, query string) ([]Loan, error) {
	rows, err := s.q.QueryContext(ctx, query)
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
