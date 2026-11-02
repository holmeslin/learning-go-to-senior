package signup

import "errors"

// ErrFull 表示活動名額與候補名單都已額滿。
var ErrFull = errors.New("signup: event and waitlist are full")

// ANCHOR: waitlisted

// ErrWaitlisted 表示活動名額已滿，這次報名排進了候補名單。
//
// 它不是失敗：報名已經被記下。Register 把它當錯誤值回傳，是為了讓呼叫端非得決定怎麼處理，
// 分辨的方式是 errors.Is(err, ErrWaitlisted)。
var ErrWaitlisted = errors.New("signup: event is full, placed on the waitlist")

// ANCHOR_END: waitlisted

// ErrAlreadyRegistered 表示這個人已經報名過了，不論是在正式名單還是候補名單。
var ErrAlreadyRegistered = errors.New("signup: already registered")

// ErrNotRegistered 表示沒有這個人的報名紀錄。
var ErrNotRegistered = errors.New("signup: not registered")

// ErrEmptyName 表示報名者的名稱是空的。
var ErrEmptyName = errors.New("signup: empty name")

// ErrEventExists 表示同一個代號的活動已經存在。
var ErrEventExists = errors.New("signup: event already exists")
