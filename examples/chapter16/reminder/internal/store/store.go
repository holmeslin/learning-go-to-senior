// Package store 是提醒服務的資料層：寄送紀錄的 schema、migration 與狀態轉換。
//
// 寄送紀錄表由提醒服務擁有；報名資料只透過報名服務維護的唯讀檢視 reminder_candidates 讀取
// （欄位：活動、報名者、裝置 token、email、活動開始時間；範例不使用 email）。
// 所有查詢都用參數，時間以 unix 秒（UTC）存放。
// 方法都接收 now 而不是自己呼叫 time.Now，時間由呼叫端決定，測試不需要假時鐘。
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite" // 註冊 "sqlite" driver
)

const (
	// LeadTime 是預定時間比活動開始早多久：提醒在活動開始前 24 小時送出。
	LeadTime = 24 * time.Hour
	// SendWindow 是預定時間之後多久內要送出；超過就放棄，記為失敗。
	// 原則是「寧可不寄，也不重複」。
	SendWindow = time.Hour
)

// ANCHOR: states

// State 是寄送紀錄的狀態。
type State string

const (
	// StateSending 是寄送中：紀錄已寫、結果尚未確認。行程在這時中斷，紀錄就停在這裡，不再被挑出。
	StateSending State = "sending"
	// StateSent 是已寄出。
	StateSent State = "sent"
	// StateRetrying 是待重試：暫時失敗，等到 NextAttemptAt 之後的掃描再送。
	StateRetrying State = "retrying"
	// StateFailed 是失敗：永久失敗，或超過預定時間 SendWindow 仍沒有結果。終點狀態。
	StateFailed State = "failed"
)

// ANCHOR_END: states

var (
	// ErrNotFound 表示沒有這筆寄送紀錄。
	ErrNotFound = errors.New("store: delivery not found")
	// ErrStateChanged 表示紀錄已經不在預期的狀態，例如記結果時它已經被記為失敗。
	ErrStateChanged = errors.New("store: delivery is no longer in the expected state")
)

// Reminder 是一則要送出的提醒。
type Reminder struct {
	EventID      int64     // 活動編號
	RegistrantID int64     // 報名者編號
	DeviceToken  string    // 推播用的裝置 token，不可寫進日誌
	DueAt        time.Time // 預定時間
	Attempts     int       // 已經開始送出的次數；0 代表還沒有寄送紀錄
}

// Delivery 是一筆寄送紀錄。
type Delivery struct {
	EventID       int64     // 活動編號
	RegistrantID  int64     // 報名者編號
	State         State     // 目前的狀態
	DueAt         time.Time // 預定時間
	Attempts      int       // 已經開始送出的次數
	NextAttemptAt time.Time // 下次嘗試的時間；只有待重試時有值，其他是零值
	LastError     string    // 最近一次失敗的原因
	UpdatedAt     time.Time // 最後更新時間
}

// Store 讀寫寄送紀錄。可以在多個 goroutine 之間共用。
type Store struct {
	db *sql.DB
}

// New 回傳使用 db 的 Store；資料表須已用 Migrate 建好。db 由呼叫端開啟與關閉。
func New(db *sql.DB) *Store {
	return &Store{db: db}
}

// OpenDB 開啟 sqlite 資料庫並設定連線池。path 是檔案路徑或 sqlite 的 file: URI，
// 例如記憶體資料庫的 "file:name?mode=memory&cache=shared"。maxConns 是連線池的上限。
//
// 每條連線都帶 busy_timeout 與 WAL：多條連線同時寫入時，後到的等待而不是立刻回 SQLITE_BUSY。
// sqlite 同一時間只有一個寫入者，連線再多也只是排隊；送出時的連線只在寫紀錄與記結果時各占一小段，
// 呼叫供應商的期間不占連線，所以連線池不必跟著送出的並行上限放大。
func OpenDB(path string, maxConns int) (*sql.DB, error) {
	dsn := path
	if !strings.HasPrefix(dsn, "file:") {
		dsn = "file:" + dsn
	}
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	db, err := sql.Open("sqlite", dsn+sep+"_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(maxConns)
	return db, nil
}

// CheckCandidates 確認報名服務的唯讀檢視存在、契約的欄位都在。啟動時呼叫：
// 檢視不見了或欄位被改掉，服務要在這時就拒絕啟動，而不是等到第一次掃描才出錯；
// 開關關閉的部署也一樣會檢查，上線的第一步就是確認「讀得到檢視」。
func (s *Store) CheckCandidates(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT event_id, registrant_id, device_token, email, starts_at FROM reminder_candidates LIMIT 0`)
	if err != nil {
		return fmt.Errorf("read view reminder_candidates: %w", err)
	}
	return rows.Close()
}

// ANCHOR: due

// 新的提醒：已到預定時間、還在 SendWindow 之內、還沒有寄送紀錄。
const dueNewSQL = `
SELECT c.event_id, c.registrant_id, c.device_token, c.starts_at - :lead AS due_at, 0 AS attempts
FROM reminder_candidates AS c
WHERE c.starts_at - :lead <= :now
  AND :now < c.starts_at - :lead + :window
  AND NOT EXISTS (
      SELECT 1 FROM reminder_deliveries AS d
      WHERE d.event_id = c.event_id AND d.registrant_id = c.registrant_id)
ORDER BY due_at, c.event_id, c.registrant_id`

// 該重試的提醒：待重試、下次嘗試的時間到了、還在 SendWindow 之內。
// 報名已被取消（不在檢視裡）的人不再重試，紀錄留著，到期後記為失敗。
const dueRetrySQL = `
SELECT d.event_id, d.registrant_id, c.device_token, d.due_at, d.attempts
FROM reminder_deliveries AS d
JOIN reminder_candidates AS c ON c.event_id = d.event_id AND c.registrant_id = d.registrant_id
WHERE d.state = 'retrying'
  AND d.next_attempt_at <= :now
  AND :now < d.due_at + :window
ORDER BY d.next_attempt_at, d.event_id, d.registrant_id`

// Due 回傳 now 這個時間點該送出的提醒：先是該重試的，再是新的。
// 它只讀不寫；要不要送由 Begin 的結果決定。
//
// shortcut: 一整批載入記憶體；一場活動約 2 萬筆、每筆約百位元組，是數 MB，可以接受。
// 活動大到放不下時，改成每次只讀一頁。
func (s *Store) Due(ctx context.Context, now time.Time) ([]Reminder, error) {
	args := []any{
		sql.Named("now", now.Unix()),
		sql.Named("lead", int64(LeadTime/time.Second)),
		sql.Named("window", int64(SendWindow/time.Second)),
	}
	var out []Reminder
	for _, query := range []string{dueRetrySQL, dueNewSQL} {
		rs, err := s.queryReminders(ctx, query, args)
		if err != nil {
			return nil, err
		}
		out = append(out, rs...)
	}
	return out, nil
}

// queryReminders 執行一個回傳提醒的查詢，並確實關閉 rows：沒關的 rows 會一直占著連線。
func (s *Store) queryReminders(ctx context.Context, query string, args []any) ([]Reminder, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query due reminders: %w", err)
	}
	defer rows.Close()
	var out []Reminder
	for rows.Next() {
		var r Reminder
		var dueAt int64
		if err := rows.Scan(&r.EventID, &r.RegistrantID, &r.DeviceToken, &dueAt, &r.Attempts); err != nil {
			return nil, fmt.Errorf("scan due reminder: %w", err)
		}
		r.DueAt = fromUnix(dueAt)
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read due reminders: %w", err)
	}
	return out, nil
}

// fromUnix 把資料庫裡的 unix 秒轉回 UTC 的時間。
func fromUnix(sec int64) time.Time { return time.Unix(sec, 0).UTC() }

// ANCHOR_END: due

// ANCHOR: begin

// Begin 在送出之前寫下紀錄，回傳 true 才可以送；回傳 false 代表有人處理過了，不要送。
//
// r.Attempts 是 0 時，這是第一次嘗試：INSERT 一筆寄送中的紀錄，主鍵的唯一限制擋住重複，
// 寫不進去（已經有紀錄）就回傳 false。r.Attempts 大於 0 時是重試：只有紀錄還是待重試才轉回寄送中。
// 一個陳述式就完成「檢查加寫入」，沒有先查再寫的空隙。
func (s *Store) Begin(ctx context.Context, r Reminder, now time.Time) (bool, error) {
	var res sql.Result
	var err error
	if r.Attempts == 0 {
		res, err = s.db.ExecContext(ctx, `
INSERT INTO reminder_deliveries (event_id, registrant_id, state, due_at, attempts, updated_at)
VALUES (?, ?, 'sending', ?, 1, ?)
ON CONFLICT (event_id, registrant_id) DO NOTHING`,
			r.EventID, r.RegistrantID, r.DueAt.Unix(), now.Unix())
	} else {
		res, err = s.db.ExecContext(ctx, `
UPDATE reminder_deliveries
SET state = 'sending', attempts = attempts + 1, next_attempt_at = NULL, updated_at = ?
WHERE event_id = ? AND registrant_id = ? AND state = 'retrying'`,
			now.Unix(), r.EventID, r.RegistrantID)
	}
	if err != nil {
		return false, fmt.Errorf("begin delivery %d/%d: %w", r.EventID, r.RegistrantID, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("begin delivery %d/%d: %w", r.EventID, r.RegistrantID, err)
	}
	return n == 1, nil
}

// ANCHOR_END: begin

// MarkSent 把寄送中的紀錄記為已寄出。紀錄不在寄送中時回傳 ErrStateChanged。
func (s *Store) MarkSent(ctx context.Context, r Reminder, now time.Time) error {
	return s.finish(ctx, r, StateSent, sql.NullInt64{}, "", now)
}

// MarkRetry 把寄送中的紀錄記為待重試，nextAttempt 之後的掃描才會再挑出它；cause 是這次失敗的原因。
// 紀錄不在寄送中時回傳 ErrStateChanged。
func (s *Store) MarkRetry(ctx context.Context, r Reminder, nextAttempt time.Time, cause string, now time.Time) error {
	return s.finish(ctx, r, StateRetrying, sql.NullInt64{Int64: nextAttempt.Unix(), Valid: true}, cause, now)
}

// MarkFailed 把寄送中的紀錄記為失敗（永久失敗，不再重試）；cause 是失敗的原因。
// 紀錄不在寄送中時回傳 ErrStateChanged。
func (s *Store) MarkFailed(ctx context.Context, r Reminder, cause string, now time.Time) error {
	return s.finish(ctx, r, StateFailed, sql.NullInt64{}, cause, now)
}

// finish 只轉換「寄送中」的紀錄：已經被記為失敗的不能被晚到的結果蓋掉。
func (s *Store) finish(ctx context.Context, r Reminder, state State, next sql.NullInt64, cause string, now time.Time) error {
	res, err := s.db.ExecContext(ctx, `
UPDATE reminder_deliveries
SET state = ?, next_attempt_at = ?, last_error = ?, updated_at = ?
WHERE event_id = ? AND registrant_id = ? AND state = 'sending'`,
		string(state), next, cause, now.Unix(), r.EventID, r.RegistrantID)
	if err != nil {
		return fmt.Errorf("mark delivery %d/%d %s: %w", r.EventID, r.RegistrantID, state, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark delivery %d/%d %s: %w", r.EventID, r.RegistrantID, state, err)
	}
	if n != 1 {
		return fmt.Errorf("mark delivery %d/%d %s: %w", r.EventID, r.RegistrantID, state, ErrStateChanged)
	}
	return nil
}

// ANCHOR: expire

// 寄送中與待重試的紀錄超過期限就記為失敗；原因已經有的（上一次失敗）保留。
const expireSQL = `
UPDATE reminder_deliveries
SET state = 'failed', next_attempt_at = NULL, updated_at = :now,
    last_error = CASE WHEN last_error = '' THEN 'no result before the send window closed' ELSE last_error END
WHERE state IN ('sending', 'retrying') AND due_at + :window <= :now`

// 從未嘗試、期限已過、活動還沒開始的提醒（例如服務停機超過 SendWindow）：直接寫一筆失敗的紀錄，不送出。
const missedSQL = `
INSERT INTO reminder_deliveries (event_id, registrant_id, state, due_at, attempts, last_error, updated_at)
SELECT c.event_id, c.registrant_id, 'failed', c.starts_at - :lead, 0, 'never attempted before the send window closed', :now
FROM reminder_candidates AS c
WHERE c.starts_at - :lead + :window <= :now AND :now < c.starts_at
ON CONFLICT (event_id, registrant_id) DO NOTHING`

// FailOverdue 把超過期限的提醒記為失敗，每次掃描在挑出到期的提醒之前先做。
// expired 是原本寄送中或待重試的筆數，missed 是從未嘗試過的筆數；兩者都計入失敗。
// 寄送中而逾期的，通常是行程中斷、停在寄送中的那一筆：它不補寄，期限到了就記為失敗。
func (s *Store) FailOverdue(ctx context.Context, now time.Time) (expired, missed int, err error) {
	args := []any{
		sql.Named("now", now.Unix()),
		sql.Named("lead", int64(LeadTime/time.Second)),
		sql.Named("window", int64(SendWindow/time.Second)),
	}
	res, err := s.db.ExecContext(ctx, expireSQL, args...)
	if err != nil {
		return 0, 0, fmt.Errorf("fail overdue deliveries: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, 0, fmt.Errorf("fail overdue deliveries: %w", err)
	}
	res, err = s.db.ExecContext(ctx, missedSQL, args...)
	if err != nil {
		return int(n), 0, fmt.Errorf("record missed reminders: %w", err)
	}
	m, err := res.RowsAffected()
	if err != nil {
		return int(n), 0, fmt.Errorf("record missed reminders: %w", err)
	}
	return int(n), int(m), nil
}

// ANCHOR_END: expire

// Get 回傳一筆寄送紀錄；沒有這筆時回傳 ErrNotFound。
func (s *Store) Get(ctx context.Context, eventID, registrantID int64) (Delivery, error) {
	d := Delivery{EventID: eventID, RegistrantID: registrantID}
	var state string
	var dueAt, updatedAt int64
	var next sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
SELECT state, due_at, attempts, next_attempt_at, last_error, updated_at
FROM reminder_deliveries WHERE event_id = ? AND registrant_id = ?`, eventID, registrantID).
		Scan(&state, &dueAt, &d.Attempts, &next, &d.LastError, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Delivery{}, fmt.Errorf("get delivery %d/%d: %w", eventID, registrantID, ErrNotFound)
	}
	if err != nil {
		return Delivery{}, fmt.Errorf("get delivery %d/%d: %w", eventID, registrantID, err)
	}
	d.State, d.DueAt, d.UpdatedAt = State(state), fromUnix(dueAt), fromUnix(updatedAt)
	if next.Valid {
		d.NextAttemptAt = fromUnix(next.Int64)
	}
	return d, nil
}

// CountByState 回傳某場活動各狀態的寄送紀錄數量，內部 API 用它回答「這場活動的提醒送到哪了」。
// 四個狀態都會出現在結果裡，沒有紀錄的是 0。檢視裡沒有這場活動的報名者、也沒有任何紀錄時，
// 回傳 ErrNotFound：呼叫端才分得出「活動存在、還沒有紀錄」與「沒有這場活動」。
func (s *Store) CountByState(ctx context.Context, eventID int64) (map[State]int, error) {
	counts := map[State]int{StateSending: 0, StateSent: 0, StateRetrying: 0, StateFailed: 0}
	recorded, err := s.countRecorded(ctx, eventID, counts)
	if err != nil {
		return nil, err
	}
	if recorded > 0 {
		return counts, nil
	}
	var known bool
	err = s.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM reminder_candidates WHERE event_id = ?)`, eventID).Scan(&known)
	if err != nil {
		return nil, fmt.Errorf("check event %d: %w", eventID, err)
	}
	if !known {
		return nil, fmt.Errorf("count deliveries of event %d: %w", eventID, ErrNotFound)
	}
	return counts, nil
}

// countRecorded 把這場活動各狀態的筆數填進 counts，回傳總筆數；rows 在回傳前關閉，不占著連線。
func (s *Store) countRecorded(ctx context.Context, eventID int64, counts map[State]int) (int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT state, COUNT(*) FROM reminder_deliveries WHERE event_id = ? GROUP BY state`, eventID)
	if err != nil {
		return 0, fmt.Errorf("count deliveries of event %d: %w", eventID, err)
	}
	defer rows.Close()
	total := 0
	for rows.Next() {
		var state string
		var n int
		if err := rows.Scan(&state, &n); err != nil {
			return 0, fmt.Errorf("scan delivery count of event %d: %w", eventID, err)
		}
		counts[State(state)] = n
		total += n
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("count deliveries of event %d: %w", eventID, err)
	}
	return total, nil
}

// CountOverdue 回傳 now 這個時間點，已經過了預定時間、還沒寄出的提醒數量：
// 寄送中與待重試的紀錄，加上已到預定時間、還在 SendWindow 之內、卻還沒有紀錄的。
// 已寄出與已記為失敗的不算。取消報名的人若還留著未結束的紀錄，會算到期限到了記為失敗為止；數量很小，不另外排除。
func (s *Store) CountOverdue(ctx context.Context, now time.Time) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `
SELECT
  (SELECT COUNT(*) FROM reminder_deliveries WHERE state IN ('sending', 'retrying'))
+ (SELECT COUNT(*) FROM reminder_candidates AS c
   WHERE c.starts_at - :lead <= :now
     AND :now < c.starts_at - :lead + :window
     AND NOT EXISTS (
         SELECT 1 FROM reminder_deliveries AS d
         WHERE d.event_id = c.event_id AND d.registrant_id = c.registrant_id))`,
		sql.Named("now", now.Unix()),
		sql.Named("lead", int64(LeadTime/time.Second)),
		sql.Named("window", int64(SendWindow/time.Second)),
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count overdue reminders: %w", err)
	}
	return n, nil
}
