package loans_test

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"
	"testing"

	loans "github.com/holmeslin/learning-go-to-senior/examples/chapter11/02_sql_injection/after"
)

func newDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := openDB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close db: %v", err)
		}
	})
	return db
}

func ids(ls []loans.Loan) []int {
	var out []int
	for _, l := range ls {
		out = append(out, l.ID)
	}
	return out
}

// ANCHOR: test

// 同樣的輸入在 before 讀出所有人的借閱；在 after 只是一個找不到的會員名字。
func TestByMemberTreatsInputAsData(t *testing.T) {
	store := loans.NewStore(newDB(t))
	tests := []struct {
		name   string
		member string
		want   []int
	}{
		{"plain", "ann", []int{1, 2}},
		{"or true", "x' OR '1'='1", nil},
		{"comment", "ann'--", nil},
		{"quote is data", "o'neil", []int{5}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.ByMember(context.Background(), tt.member, "id")
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(ids(got), tt.want) {
				t.Errorf("ids = %v, want %v", ids(got), tt.want)
			}
		})
	}
}

// ANCHOR_END: test

func TestByMemberOrders(t *testing.T) {
	store := loans.NewStore(newDB(t))
	tests := []struct {
		member, order string
		want          []int
	}{
		{"bob", "id", []int{3, 4}},
		{"bob", "newest", []int{4, 3}},
		{"ann", "title", []int{2, 1}}, // go concurrency 排在 sqlite internals 前面
	}
	for _, tt := range tests {
		got, err := store.ByMember(context.Background(), tt.member, tt.order)
		if err != nil {
			t.Fatalf("order %q: %v", tt.order, err)
		}
		if !slices.Equal(ids(got), tt.want) {
			t.Errorf("order %q: ids = %v, want %v", tt.order, ids(got), tt.want)
		}
	}
}

// ANCHOR: orders

// 不在白名單的排序一律拒絕，而且連查詢都不送出。
func TestByMemberRejectsUnknownOrder(t *testing.T) {
	rec := &recorder{Querier: newDB(t)}
	store := loans.NewStore(rec)
	for _, order := range []string{
		"", "nope", "ID", "member", "id DESC",
		"CASE WHEN title LIKE 'go%' THEN 0 ELSE 1 END",
		"id; DROP TABLE loans",
	} {
		_, err := store.ByMember(context.Background(), "ann", order)
		if !errors.Is(err, loans.ErrInvalidOrder) {
			t.Errorf("order %q: err = %v, want ErrInvalidOrder", order, err)
		}
	}
	if len(rec.queries) != 0 {
		t.Errorf("sent %d queries, want 0", len(rec.queries))
	}
}

// ANCHOR_END: orders

// 佔位符只能代表值：ORDER BY ? 收到 "title" 是一個常數字串，不是欄位，結果沒有排序。
func TestPlaceholderIsNotAColumn(t *testing.T) {
	db := newDB(t)
	rows, err := db.Query(`SELECT id FROM loans ORDER BY ?`, "title")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		got = append(got, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	// 依 title 排序會是 5、3、2、4、1。
	if want := []int{1, 2, 3, 4, 5}; !slices.Equal(got, want) {
		t.Errorf("ids = %v, want %v", got, want)
	}
}

// ANCHOR: in

// IN 的佔位符個數等於 titles 的個數，值只出現在參數裡。
func TestByTitlesPlaceholders(t *testing.T) {
	for n := range 4 {
		titles := slices.Repeat([]string{"go concurrency"}, n)
		rec := &recorder{Querier: newDB(t)}
		got, err := loans.NewStore(rec).ByTitles(context.Background(), "ann", titles)
		if err != nil {
			t.Fatal(err)
		}
		if n == 0 {
			if len(rec.queries) != 0 || len(got) != 0 {
				t.Errorf("no titles: %d queries, %d loans, want none", len(rec.queries), len(got))
			}
			continue
		}
		q := rec.queries[0]
		if c := strings.Count(q, "?"); c != n+1 || len(rec.args[0]) != n+1 {
			t.Errorf("%d titles: %d placeholders, %d args, want %d each", n, c, len(rec.args[0]), n+1)
		}
		if strings.Contains(q, "go concurrency") || strings.Contains(q, "ann") {
			t.Errorf("%d titles: query contains data: %s", n, q)
		}
	}
}

// ANCHOR_END: in

func TestByTitles(t *testing.T) {
	store := loans.NewStore(newDB(t))
	tests := []struct {
		name   string
		titles []string
		want   []int
	}{
		{"one", []string{"go concurrency"}, []int{2}}, // bob 也借了同一本，但不在 ann 的結果裡
		{"two", []string{"sqlite internals", "go concurrency"}, []int{1, 2}},
		{"unknown", []string{"nope"}, nil},
		{"escape attempt", []string{"x') OR ('1'='1"}, nil},
		{"quote is data", []string{"it's"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.ByTitles(context.Background(), "ann", tt.titles)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(ids(got), tt.want) {
				t.Errorf("ids = %v, want %v", ids(got), tt.want)
			}
		})
	}
}
