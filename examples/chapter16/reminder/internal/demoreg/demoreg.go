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
