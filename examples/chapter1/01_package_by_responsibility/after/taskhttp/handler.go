// Package taskhttp 把 task.Store 透過 HTTP 提供出去。
package taskhttp

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter1/01_package_by_responsibility/after/task"
)

// ANCHOR: handler
// NewHandler 回傳處理 /tasks/{id}/complete 的 http.Handler。
func NewHandler(store *task.Store) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /tasks/{id}/complete", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		err = store.Complete(id)
		if err != nil {
			if errors.Is(err, task.ErrNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	return mux
}

// ANCHOR_END: handler
