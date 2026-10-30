package loans_test

import (
	"context"
	"database/sql"
	"fmt"

	loans "github.com/holmeslin/learning-go-to-senior/examples/chapter11/02_sql_injection/after"
)

// ANCHOR: recorder

// recorder 包住 [loans.Querier]，記下每次送進資料庫的 SQL 與參數。
type recorder struct {
	loans.Querier
	queries []string
	args    [][]any
}

func (r *recorder) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	r.queries = append(r.queries, query)
	r.args = append(r.args, args)
	return r.Querier.QueryContext(ctx, query, args...)
}

// ANCHOR_END: recorder

// ANCHOR: example

func ExampleStore_ByMember() {
	db, err := openDB()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	rec := &recorder{Querier: db}
	store := loans.NewStore(rec)
	ctx := context.Background()

	got, err := store.ByMember(ctx, "x' OR '1'='1", "id")
	fmt.Println(rec.queries[0])
	fmt.Println(rec.args[0])
	fmt.Println(len(got), err)

	got, err = store.ByTitles(ctx, "ann", []string{"go concurrency", "sqlite internals"})
	fmt.Println(rec.queries[1])
	fmt.Println(rec.args[1])
	fmt.Println(len(got), err)
	// Output:
	// SELECT id, member, title FROM loans WHERE member = ? ORDER BY id
	// [x' OR '1'='1]
	// 0 <nil>
	// SELECT id, member, title FROM loans WHERE member = ? AND title IN (?, ?) ORDER BY id
	// [ann go concurrency sqlite internals]
	// 2 <nil>
}

// ANCHOR_END: example
