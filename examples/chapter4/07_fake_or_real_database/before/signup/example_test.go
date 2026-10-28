package signup_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "modernc.org/sqlite"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter4/07_fake_or_real_database/before/signup"
)

// ANCHOR: example

func Example_fakeAndSQLite() {
	ctx := context.Background()
	emails := []string{"ann@example.com", "bob@example.com", "ann@example.com"}

	added, skipped, err := signup.NewService(newFakeStore()).Invite(ctx, emails)
	fmt.Printf("fake   added=%d skipped=%v err=%v\n", added, skipped, err)

	db, err := openMemory(ctx, "before_fake_and_sqlite")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()
	added, skipped, err = signup.NewService(signup.NewSQLStore(db)).Invite(ctx, emails)
	fmt.Printf("sqlite added=%d skipped=%v err=%v\n", added, skipped, err)
	// Output:
	// fake   added=3 skipped=[] err=<nil>
	// sqlite added=2 skipped=[] err=insert ann@example.com: constraint failed: UNIQUE constraint failed: users.email (2067)
}

// ANCHOR_END: example

// openMemory 開一個共享快取、名稱獨一無二的記憶體資料庫，並建好資料表。
// 每個 Example 用自己的名稱，才不會互相干擾；一條連線就夠用。
func openMemory(ctx context.Context, name string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", "file:"+name+"?mode=memory&cache=shared")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err = db.ExecContext(ctx, signup.Schema); err != nil {
		return nil, errors.Join(fmt.Errorf("apply schema: %w", err), db.Close())
	}
	return db, nil
}
