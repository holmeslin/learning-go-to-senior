package loans_test

import (
	"context"
	"testing"

	loans "github.com/holmeslin/learning-go-to-senior/examples/chapter11/02_sql_injection/before"
)

// 書名也能跳出引號：條件變成 (member = 'ann' AND title IN ('x')) OR ('1'='1')，所有人的借閱都出來。
func TestByTitlesInjection(t *testing.T) {
	db, err := openDB()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	got, err := loans.NewStore(db).ByTitles(context.Background(), "ann", []string{"x') OR ('1'='1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Errorf("got %d loans, want all 5", len(got))
	}
}

// order 是一段會被執行的 SQL：這個運算式把書名以 go 開頭的排在前面。
func TestByMemberOrderIsExecuted(t *testing.T) {
	db, err := openDB()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	got, err := loans.NewStore(db).ByMember(context.Background(), "ann",
		"CASE WHEN title LIKE 'go%' THEN 0 ELSE 1 END")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Title != "go concurrency" {
		t.Errorf("got %+v, want go concurrency first", got)
	}
}

// 名字裡有單引號的會員（o'neil）連正常查詢都會壞掉。
func TestByMemberQuoteBreaksQuery(t *testing.T) {
	db, err := openDB()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := loans.NewStore(db).ByMember(context.Background(), "o'neil", "id"); err == nil {
		t.Error("want a syntax error for a name containing a quote")
	}
}
