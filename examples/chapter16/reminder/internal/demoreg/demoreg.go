// Package demoreg 是報名服務那一側的示範 schema：活動、報名，以及提醒服務讀取的唯讀檢視。
//
// 示範用，正式環境由報名服務維護：資料表與檢視由報名服務的 migration 建立，
// 提醒服務不建立、也不修改它們，只讀檢視 reminder_candidates。
// 這個套件讓書附的測試、冒煙腳本與壓測工具有資料可讀。
//
// 檢視有 email 欄位，範例不使用：範例只實作推播管道；
// email 走同一套寄送紀錄與重試，差別在供應商的限制是每日額度。
package demoreg

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"time"
)

//go:embed schema.sql
var schema string

// Apply 建立示範用的資料表與檢視。可以重複呼叫。
func Apply(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("apply demo registration schema: %w", err)
	}
	return nil
}

// AddEvent 新增一場活動，startsAt 是活動開始時間。
func AddEvent(ctx context.Context, db *sql.DB, eventID int64, startsAt time.Time) error {
	_, err := db.ExecContext(ctx, `INSERT INTO events (id, starts_at) VALUES (?, ?)`, eventID, startsAt.Unix())
	if err != nil {
		return fmt.Errorf("add event %d: %w", eventID, err)
	}
	return nil
}

// Seed 在一個交易裡新增一場活動與 registrants 位報名者（編號 1 到 registrants，全部是報名成功），
// 給冒煙腳本與壓測工具用：一筆一筆呼叫 AddRegistration，每筆各自提交，灌兩萬人要慢很多。
// 裝置 token 與 email 是示範用的假值。活動已存在時回傳錯誤，不寫入任何資料。
func Seed(ctx context.Context, db *sql.DB, eventID int64, startsAt time.Time, registrants int) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("seed event %d: %w", eventID, err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO events (id, starts_at) VALUES (?, ?)`, eventID, startsAt.Unix()); err != nil {
		return fmt.Errorf("seed event %d: %w", eventID, err)
	}
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO registrations (event_id, registrant_id, device_token, email, status) VALUES (?, ?, ?, ?, 'confirmed')`)
	if err != nil {
		return fmt.Errorf("seed event %d: %w", eventID, err)
	}
	defer stmt.Close()
	for id := 1; id <= registrants; id++ {
		if _, err := stmt.ExecContext(ctx, eventID, id, fmt.Sprintf("token-%d-%d", eventID, id), fmt.Sprintf("registrant%d@example.com", id)); err != nil {
			return fmt.Errorf("seed registrant %d of event %d: %w", id, eventID, err)
		}
	}
	return tx.Commit()
}

// Registration 是一筆報名。
type Registration struct {
	EventID      int64  // 活動編號
	RegistrantID int64  // 報名者編號
	DeviceToken  string // 推播用的裝置 token
	Email        string // 範例不使用，見套件說明
	Waitlisted   bool   // 候補者不在檢視裡，也不會收到提醒
}

// AddRegistration 新增一筆報名。同一場活動同一位報名者只能有一筆。
func AddRegistration(ctx context.Context, db *sql.DB, r Registration) error {
	status := "confirmed"
	if r.Waitlisted {
		status = "waitlisted"
	}
	_, err := db.ExecContext(ctx,
		`INSERT INTO registrations (event_id, registrant_id, device_token, email, status) VALUES (?, ?, ?, ?, ?)`,
		r.EventID, r.RegistrantID, r.DeviceToken, r.Email, status)
	if err != nil {
		return fmt.Errorf("add registration %d/%d: %w", r.EventID, r.RegistrantID, err)
	}
	return nil
}
