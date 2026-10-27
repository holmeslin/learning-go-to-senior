package taskhttp_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter1/01_package_by_responsibility/after/task"
	"github.com/holmeslin/learning-go-to-senior/examples/chapter1/01_package_by_responsibility/after/taskhttp"
)

func TestCompleteRoute(t *testing.T) {
	store := task.NewStore()
	added := store.Add("寫第 1 章")
	h := taskhttp.NewHandler(store)

	tests := []struct {
		name string
		path string
		want int
	}{
		{"存在的任務", fmt.Sprintf("/tasks/%d/complete", added.ID), http.StatusNoContent},
		{"不存在的任務", "/tasks/99/complete", http.StatusNotFound},
		{"id 不是數字", "/tasks/abc/complete", http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, tt.path, nil))
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}

	if got, _ := store.Get(added.ID); !got.Done {
		t.Fatal("任務應該已經標記完成")
	}
}
