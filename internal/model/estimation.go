package model

import (
	"time"
)

// EstimationID is a unique identifier for an estimation project
type EstimationID string

// Estimation represents a project estimation with multiple tasks
type Estimation struct {
	ID          EstimationID      `yaml:"id"`
	Label       string            `yaml:"label"`
	Description string            `yaml:"description"`
	CreatedAt   time.Time         `yaml:"createdAt"`
	UpdatedAt   time.Time         `yaml:"updatedAt"`
	Ordering    []TaskID          `yaml:"ordering"`
	Tasks       map[TaskID]*Task  `yaml:"tasks"`
	Params      *EstimationParams `yaml:"params,omitempty"`
}

// EstimationParams contains project-specific parameters that override global config
type EstimationParams struct {
	TaskCategories     map[string]TaskCategory `yaml:"taskCategories,omitempty"`
	TimeUnit           *TimeUnit               `yaml:"timeUnit,omitempty"`
	Currency           string                  `yaml:"currency,omitempty"`
	RoundUpEstimations *bool                   `yaml:"roundUpEstimations,omitempty"`
}

// NewEstimation creates a new estimation with the given label
func NewEstimation(label string) *Estimation {
	now := time.Now()
	return &Estimation{
		ID:          EstimationID(generateID()),
		Label:       label,
		Description: "",
		CreatedAt:   now,
		UpdatedAt:   now,
		Ordering:    []TaskID{},
		Tasks:       make(map[TaskID]*Task),
		Params:      nil,
	}
}

// AddTask adds a new task to the estimation
func (e *Estimation) AddTask(task *Task) {
	e.Tasks[task.ID] = task
	e.Ordering = append(e.Ordering, task.ID)
	e.UpdatedAt = time.Now()
}

// RemoveTask removes a task from the estimation
func (e *Estimation) RemoveTask(id TaskID) {
	delete(e.Tasks, id)

	// Remove from ordering
	for i, taskID := range e.Ordering {
		if taskID == id {
			e.Ordering = append(e.Ordering[:i], e.Ordering[i+1:]...)
			break
		}
	}
	e.UpdatedAt = time.Now()
}

// MoveTask moves a task in the ordering by the specified offset
func (e *Estimation) MoveTask(id TaskID, offset int) bool {
	currentIndex := -1
	for i, taskID := range e.Ordering {
		if taskID == id {
			currentIndex = i
			break
		}
	}

	if currentIndex == -1 {
		return false
	}

	newIndex := currentIndex + offset
	if newIndex < 0 || newIndex >= len(e.Ordering) {
		return false
	}

	// Remove from current position
	e.Ordering = append(e.Ordering[:currentIndex], e.Ordering[currentIndex+1:]...)
	// Insert at new position
	e.Ordering = append(e.Ordering[:newIndex], append([]TaskID{id}, e.Ordering[newIndex:]...)...)

	e.UpdatedAt = time.Now()
	return true
}

// GetOrderedTasks returns tasks in the specified order
func (e *Estimation) GetOrderedTasks() []*Task {
	tasks := make([]*Task, 0, len(e.Tasks))
	for _, taskID := range e.Ordering {
		if task, ok := e.Tasks[taskID]; ok {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

// UpdateTask updates an existing task
func (e *Estimation) UpdateTask(task *Task) {
	if _, ok := e.Tasks[task.ID]; ok {
		e.Tasks[task.ID] = task
		e.UpdatedAt = time.Now()
	}
}

// Validate validates the entire estimation
func (e *Estimation) Validate() []string {
	var errors []string

	for _, task := range e.Tasks {
		if taskErrors := task.Validate(); len(taskErrors) > 0 {
			for _, err := range taskErrors {
				errors = append(errors, "task "+string(task.ID)+": "+err)
			}
		}
	}

	return errors
}
