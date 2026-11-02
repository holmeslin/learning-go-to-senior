// Command seed 替壓測與冒煙腳本灌資料：在資料庫檔裡建立報名服務那一側的示範 schema（活動、報名與唯讀檢視），
// 再新增一場活動與指定人數的報名者。活動的開始時間設成「現在加 -starts-in」，預設 24 小時：
// 提醒的預定時間是開始前 24 小時，所以服務一啟動，這些提醒就已經到期。
//
//	go run ./examples/chapter16/reminder/cmd/seed -db /tmp/reminder.db -event 1 -registrants 200
//
// 灌的是示範資料（裝置 token 與 email 都是假值），不是真的報名服務；旗標只在這個 main 處理，灌資料的邏輯在 demoreg.Seed。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/demoreg"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter16/reminder/internal/store"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "seed:", err)
		os.Exit(1)
	}
}

func run() error {
	dbPath := flag.String("db", "reminder.db", "資料庫檔的路徑（與服務的 REMINDER_DB_PATH 相同）")
	eventID := flag.Int64("event", 1, "活動編號")
	registrants := flag.Int("registrants", 200, "報名者人數")
	startsIn := flag.Duration("starts-in", 24*time.Hour, "活動在多久之後開始；24h 代表提醒現在就到期")
	flag.Parse()
	if *eventID <= 0 || *registrants < 1 {
		return errors.New("-event and -registrants must be positive")
	}

	ctx := context.Background()
	db, err := store.OpenDB(*dbPath, 1)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := demoreg.Apply(ctx, db); err != nil {
		return err
	}
	if err := demoreg.Seed(ctx, db, *eventID, time.Now().Add(*startsIn), *registrants); err != nil {
		return err
	}
	fmt.Printf("seeded event %d with %d registrants in %s\n", *eventID, *registrants, *dbPath)
	return nil
}
