package model

import (
	"fmt"
	"sort"
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
		Tasks:       make(map[TaskID]*Task),
		Params:      nil,
	}
}

// AddTask adds a new task to the estimation
func (e *Estimation) AddTask(task *Task) {
	// Set the order to the next available position
	if task.Order == 0 {
		task.Order = len(e.Tasks)
	}
	e.Tasks[task.ID] = task
	e.UpdatedAt = time.Now()
}

// RemoveTask removes a task from the estimation
func (e *Estimation) RemoveTask(id TaskID) {
	deletedOrder := -1
	if task, ok := e.Tasks[id]; ok {
		deletedOrder = task.Order
	}
	delete(e.Tasks, id)

	// Reindex orders for tasks that came after the deleted one
	if deletedOrder >= 0 {
		for _, task := range e.Tasks {
			if task.Order > deletedOrder {
				task.Order--
			}
		}
	}
	e.UpdatedAt = time.Now()
}

// MoveTask moves a task in the ordering by the specified offset
func (e *Estimation) MoveTask(id TaskID, offset int) bool {
	task, ok := e.Tasks[id]
	if !ok {
		return false
	}

	currentOrder := task.Order
	newOrder := currentOrder + offset

	if newOrder < 0 || newOrder >= len(e.Tasks) {
		return false
	}

	// Find the task at the new position and swap orders
	for _, otherTask := range e.Tasks {
		if otherTask.Order == newOrder {
			otherTask.Order = currentOrder
			break
		}
	}

	task.Order = newOrder
	e.UpdatedAt = time.Now()
	return true
}

// GetOrderedTasks returns tasks sorted by their Order field
func (e *Estimation) GetOrderedTasks() []*Task {
	tasks := make([]*Task, 0, len(e.Tasks))
	for _, task := range e.Tasks {
		tasks = append(tasks, task)
	}
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Order < tasks[j].Order
	})
	return tasks
}

// ReorderTasks reorders tasks according to the provided list of task IDs
// Tasks not in the list will be appended at the end in their current order
func (e *Estimation) ReorderTasks(taskIDs []TaskID) error {
	// Validate all task IDs exist
	for _, id := range taskIDs {
		if _, ok := e.Tasks[id]; !ok {
			return fmt.Errorf("task with ID '%s' not found", id)
		}
	}

	// Create a set of provided IDs for quick lookup
	providedIDs := make(map[TaskID]bool)
	for _, id := range taskIDs {
		providedIDs[id] = true
	}

	// Assign new orders
	order := 0
	for _, id := range taskIDs {
		if task, ok := e.Tasks[id]; ok {
			task.Order = order
			order++
		}
	}

	// Append remaining tasks not in the provided list
	remainingTasks := make([]*Task, 0)
	for _, task := range e.Tasks {
		if !providedIDs[task.ID] {
			remainingTasks = append(remainingTasks, task)
		}
	}
	// Sort remaining tasks by their current order
	sort.Slice(remainingTasks, func(i, j int) bool {
		return remainingTasks[i].Order < remainingTasks[j].Order
	})
	for _, task := range remainingTasks {
		task.Order = order
		order++
	}

	e.UpdatedAt = time.Now()
	return nil
}

// SortMode defines how tasks should be sorted
type SortMode string

const (
	SortByCategory SortMode = "category"
	SortByCost     SortMode = "cost"
	SortByLabel    SortMode = "label"
)

// SortTasks sorts tasks according to the specified mode using the provided config.
// If descending is true, the sort order is reversed.
func (e *Estimation) SortTasks(mode SortMode, config *Config, descending bool) {
	tasks := e.GetOrderedTasks()

	var less func(i, j int) bool
	switch mode {
	case SortByCategory:
		less = func(i, j int) bool {
			catI := config.GetTaskCategory(tasks[i].Category)
			catJ := config.GetTaskCategory(tasks[j].Category)
			if catI.Label == catJ.Label {
				return tasks[i].Order < tasks[j].Order
			}
			return catI.Label < catJ.Label
		}
	case SortByCost:
		less = func(i, j int) bool {
			catI := config.GetTaskCategory(tasks[i].Category)
			catJ := config.GetTaskCategory(tasks[j].Category)
			costI := tasks[i].WeightedMean() * catI.CostPerTimeUnit
			costJ := tasks[j].WeightedMean() * catJ.CostPerTimeUnit
			if costI == costJ {
				return tasks[i].Order < tasks[j].Order
			}
			return costI < costJ
		}
	case SortByLabel:
		less = func(i, j int) bool {
			if tasks[i].Label == tasks[j].Label {
				return tasks[i].Order < tasks[j].Order
			}
			return tasks[i].Label < tasks[j].Label
		}
	default:
		return
	}

	// If descending, reverse the comparison
	if descending {
		originalLess := less
		less = func(i, j int) bool {
			return !originalLess(i, j)
		}
	}

	sort.Slice(tasks, less)

	// Reassign orders
	for i, task := range tasks {
		task.Order = i
	}

	e.UpdatedAt = time.Now()
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
