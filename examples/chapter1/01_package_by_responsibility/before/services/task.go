// Package services 放商業邏輯。
package services

import (
	"errors"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter1/01_package_by_responsibility/before/models"
)

var ErrNotFound = errors.New("task: not found")

type TaskService struct {
	nextID int
	tasks  map[int]*models.Task
}

func NewTaskService() *TaskService {
	return &TaskService{tasks: make(map[int]*models.Task)}
}

func (s *TaskService) Add(title string) models.Task {
	s.nextID++
	t := &models.Task{ID: s.nextID, Title: title}
	s.tasks[t.ID] = t
	return *t
}

// ANCHOR: complete

func (s *TaskService) Complete(id int) error {
	t, ok := s.tasks[id]
	if !ok {
		return ErrNotFound
	}
	t.Done = true
	return nil
}

// ANCHOR_END: complete
