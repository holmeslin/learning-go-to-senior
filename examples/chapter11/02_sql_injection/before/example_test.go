package loans_test

import (
	"context"
	"database/sql"
	"fmt"

	loans "github.com/holmeslin/learning-go-to-senior/examples/chapter11/02_sql_injection/before"
)

// ANCHOR: recorder

// recorder 包住 [loans.Querier]，記下每次送進資料庫的 SQL。
type recorder struct {
	loans.Querier
	queries []string
}

func (r *recorder) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	r.queries = append(r.queries, query)
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

	for i, member := range []string{"ann", "x' OR '1'='1"} {
		got, err := store.ByMember(context.Background(), member, "id")
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(rec.queries[i])
		for _, l := range got {
			fmt.Printf("  %s: %s\n", l.Member, l.Title)
		}
	}
	// Output:
	// SELECT id, member, title FROM loans WHERE member = 'ann' ORDER BY id
	//   ann: sqlite internals
	//   ann: go concurrency
	// SELECT id, member, title FROM loans WHERE member = 'x' OR '1'='1' ORDER BY id
	//   ann: sqlite internals
	//   ann: go concurrency
	//   bob: distributed systems
	//   bob: go concurrency
	//   o'neil: compilers
}

// ANCHOR_END: example
