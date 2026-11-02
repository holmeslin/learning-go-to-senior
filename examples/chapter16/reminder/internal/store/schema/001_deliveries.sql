-- ANCHOR: table

-- 寄送紀錄：提醒服務自己擁有的表。主鍵是「活動 + 報名者」，唯一限制就是冪等鍵：
-- 寫得進去的才送，寫不進去代表已經有人處理過。時間一律是 unix 秒（UTC）。
CREATE TABLE reminder_deliveries (
    event_id        INTEGER NOT NULL,
    registrant_id   INTEGER NOT NULL,
    state           TEXT    NOT NULL CHECK (state IN ('sending', 'sent', 'retrying', 'failed')),
    due_at          INTEGER NOT NULL,           -- 預定時間：活動開始前 24 小時
    attempts        INTEGER NOT NULL DEFAULT 0, -- 已經開始送出的次數
    next_attempt_at INTEGER,                    -- 只有 retrying 才有值
    last_error      TEXT    NOT NULL DEFAULT '',
    updated_at      INTEGER NOT NULL,
    PRIMARY KEY (event_id, registrant_id)
) WITHOUT ROWID;

-- ANCHOR_END: table

-- 逾期與重試的查詢只看還沒結束的紀錄；已寄出與失敗的佔絕大多數，不放進索引。
CREATE INDEX reminder_deliveries_open ON reminder_deliveries (due_at) WHERE state IN ('sending', 'retrying');
