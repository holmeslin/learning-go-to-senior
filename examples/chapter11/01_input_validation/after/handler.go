package booking

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

// MaxBodyBytes 是請求本文的大小上限。匯出它，是因為呼叫端需要知道上限，
// 測試也要用它確認超大的本文沒有被整份讀完。
const MaxBodyBytes = 4 << 10

// ANCHOR: handler

// NewHandler 回傳處理預約的 handler。只有通過驗證的 [Booking] 會交給 save。
func NewHandler(save func(Booking)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, status, fe := decode(w, r)
		if fe != nil {
			writeErrors(w, status, *fe)
			return
		}
		b, err := req.Validate()
		if err != nil {
			writeErrors(w, http.StatusBadRequest, fieldErrors(err)...)
			return
		}
		save(b)
		w.WriteHeader(http.StatusCreated)
	})
}

// decode 讀出請求：限制大小、拒絕未知欄位、本文裡只能有一個 JSON 值。
func decode(w http.ResponseWriter, r *http.Request) (Request, int, *FieldError) {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxBodyBytes))
	dec.DisallowUnknownFields()
	var req Request
	err := dec.Decode(&req)
	if err == nil {
		if _, err = dec.Token(); err == nil { // 後面還有東西
			err = errors.New("trailing data")
		} else if err == io.EOF {
			return req, 0, nil
		}
	}
	status, fe := problem(err)
	return Request{}, status, &fe
}

// problem 把解碼錯誤翻成固定的訊息。`err.Error()` 不能直接交出去：
// 它會帶出 Go 的型別與欄位名稱。
func problem(err error) (int, FieldError) {
	if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
		return http.StatusRequestEntityTooLarge, FieldError{Message: "request body too large"}
	}
	if te, ok := errors.AsType[*json.UnmarshalTypeError](err); ok {
		return http.StatusBadRequest, FieldError{Field: te.Field, Message: "wrong type"}
	}
	if strings.HasPrefix(err.Error(), "json: unknown field") { // encoding/json 沒有為它定義型別
		return http.StatusBadRequest, FieldError{Message: "unknown field"}
	}
	return http.StatusBadRequest, FieldError{Message: "malformed request body"}
}

// fieldErrors 把 [Request.Validate] 合併的錯誤展開成清單。
func fieldErrors(err error) []FieldError {
	var out []FieldError
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		for _, e := range joined.Unwrap() {
			if fe, ok := errors.AsType[*FieldError](e); ok {
				out = append(out, *fe)
			}
		}
	}
	return out
}

func writeErrors(w http.ResponseWriter, status int, errs ...FieldError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(struct {
		Errors []FieldError `json:"errors"`
	}{errs})
}

// ANCHOR_END: handler
