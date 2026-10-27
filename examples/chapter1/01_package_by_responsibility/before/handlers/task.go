// Package handlers 放 HTTP 處理函式。
package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter1/01_package_by_responsibility/before/services"
)

// ANCHOR: handler
// NewHandler 回傳處理 /tasks/{id}/complete 的 http.Handler。
func NewHandler(svc *services.TaskService) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /tasks/{id}/complete", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		err = svc.Complete(id)
		if err != nil {
			if errors.Is(err, services.ErrNotFound) {
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
