package signup

import "errors"

// ErrFull 表示活動名額已滿。
var ErrFull = errors.New("signup: event is full")

// ErrAlreadyRegistered 表示這個人已經報名過了。
var ErrAlreadyRegistered = errors.New("signup: already registered")

// ErrNotRegistered 表示沒有這個人的報名紀錄。
var ErrNotRegistered = errors.New("signup: not registered")

// ErrEmptyName 表示報名者的名稱是空的。
var ErrEmptyName = errors.New("signup: empty name")

// ErrEventExists 表示同一個代號的活動已經存在。
var ErrEventExists = errors.New("signup: event already exists")
