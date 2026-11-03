-- 示範用，正式環境由報名服務維護。
-- 提醒服務只讀 reminder_candidates 這個檢視；它是兩個服務之間的契約，欄位與意義的改動由報名服務的 migration 負責。
-- 時間一律是 unix 秒（UTC）。

CREATE TABLE IF NOT EXISTS events (
    id        INTEGER PRIMARY KEY,
    starts_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS registrations (
    event_id      INTEGER NOT NULL REFERENCES events (id),
    registrant_id INTEGER NOT NULL,
    device_token  TEXT    NOT NULL,
    email         TEXT    NOT NULL,
    status        TEXT    NOT NULL CHECK (status IN ('confirmed', 'waitlisted')),
    PRIMARY KEY (event_id, registrant_id)
);

-- ANCHOR: view

-- 唯讀檢視：報名成功（不含候補）的人，加上活動開始時間。
-- email 欄位範例不使用：範例只實作推播管道；email 走同一套寄送紀錄與重試，差別在供應商的限制是每日額度。
CREATE VIEW IF NOT EXISTS reminder_candidates AS
SELECT r.event_id, r.registrant_id, r.device_token, r.email, e.starts_at
FROM registrations AS r
JOIN events AS e ON e.id = r.event_id
WHERE r.status = 'confirmed';

-- ANCHOR_END: view
