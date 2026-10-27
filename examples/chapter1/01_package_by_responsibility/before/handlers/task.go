// Package handlers 放 HTTP 處理函式。
package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter1/01_package_by_responsibility/before/services"
)

type TaskHandler struct {
	svc *services.TaskService
}

func NewTaskHandler(svc *services.TaskService) *TaskHandler {
	return &TaskHandler{svc: svc}
}

// ANCHOR: complete
func (h *TaskHandler) Complete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	if err := h.svc.Complete(id); errors.Is(err, services.ErrNotFound) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ANCHOR_END: complete
