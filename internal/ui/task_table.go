package ui

import (
	"fmt"

	"github.com/bornholm/guesstimate/internal/model"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TaskTable is a tview table component for displaying tasks
type TaskTable struct {
	*tview.Table

	estimation *model.Estimation
	config     *model.Config

	// Callbacks
	OnTaskChanged func(task *model.Task)
	OnTaskAdded   func(task *model.Task)
	OnTaskRemoved func(taskID model.TaskID)

	// State
	tasks []*model.Task
}

// NewTaskTable creates a new TaskTable
func NewTaskTable(estimation *model.Estimation, config *model.Config) *TaskTable {
	t := &TaskTable{
		Table:      tview.NewTable(),
		estimation: estimation,
		config:     config,
		tasks:      estimation.GetOrderedTasks(),
	}

	t.SetBorder(true)
	t.SetTitle(" Tasks ")
	t.SetSelectable(true, true)
	t.SetFixed(1, 0) // Fixed header row

	t.setupColumns()
	t.populate()
	t.setupKeyBindings()

	return t
}

// setupColumns sets up the table columns
func (t *TaskTable) setupColumns() {
	headers := []string{"Task", "Category", "Optimistic", "Likely", "Pessimistic", "Mean", "SD"}

	for i, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)

		if i >= 2 {
			cell = cell.SetAlign(tview.AlignRight)
		}

		t.SetCell(0, i, cell)
	}
}

// populate fills the table with tasks
func (t *TaskTable) populate() {
	// Clear existing rows (keep header)
	for i := t.GetRowCount() - 1; i > 0; i-- {
		t.RemoveRow(i)
	}

	// Refresh tasks from estimation
	t.tasks = t.estimation.GetOrderedTasks()

	// Add tasks
	for i, task := range t.tasks {
		t.addTaskRow(i+1, task)
	}
}

// addTaskRow adds a row for a task
func (t *TaskTable) addTaskRow(row int, task *model.Task) {
	cat := t.config.GetTaskCategory(task.Category)
	mean := task.WeightedMean()
	sd := task.StandardDeviation()

	// Task label (editable)
	t.SetCell(row, 0, tview.NewTableCell(task.Label).
		SetTextColor(tcell.ColorWhite).
		SetExpansion(2).
		SetReference(task.ID))

	// Category
	t.SetCell(row, 1, tview.NewTableCell(cat.Label).
		SetTextColor(tcell.ColorWhite).
		SetReference(task.ID))

	// Optimistic
	t.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("%.1f", task.Estimations.Optimistic)).
		SetTextColor(tcell.ColorWhite).
		SetAlign(tview.AlignRight).
		SetReference(task.ID))

	// Likely
	t.SetCell(row, 3, tview.NewTableCell(fmt.Sprintf("%.1f", task.Estimations.Likely)).
		SetTextColor(tcell.ColorWhite).
		SetAlign(tview.AlignRight).
		SetReference(task.ID))

	// Pessimistic
	t.SetCell(row, 4, tview.NewTableCell(fmt.Sprintf("%.1f", task.Estimations.Pessimistic)).
		SetTextColor(tcell.ColorWhite).
		SetAlign(tview.AlignRight).
		SetReference(task.ID))

	// Mean (calculated)
	t.SetCell(row, 5, tview.NewTableCell(fmt.Sprintf("%.2f", mean)).
		SetTextColor(tcell.ColorGreen).
		SetAlign(tview.AlignRight).
		SetSelectable(false).
		SetReference(task.ID))

	// SD (calculated)
	t.SetCell(row, 6, tview.NewTableCell(fmt.Sprintf("%.2f", sd)).
		SetTextColor(tcell.ColorGreen).
		SetAlign(tview.AlignRight).
		SetSelectable(false).
		SetReference(task.ID))
}

// setupKeyBindings sets up keyboard navigation
func (t *TaskTable) setupKeyBindings() {
	t.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyUp:
			row, col := t.GetSelection()
			if row > 1 {
				t.Select(row-1, col)
			}
			return nil
		case tcell.KeyDown:
			row, col := t.GetSelection()
			if row < t.GetRowCount()-1 {
				t.Select(row+1, col)
			}
			return nil
		case tcell.KeyLeft:
			row, col := t.GetSelection()
			if col > 0 {
				t.Select(row, col-1)
			}
			return nil
		case tcell.KeyRight:
			row, col := t.GetSelection()
			if col < 6 {
				t.Select(row, col+1)
			}
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'j':
				row, col := t.GetSelection()
				if row < t.GetRowCount()-1 {
					t.Select(row+1, col)
				}
				return nil
			case 'k':
				row, col := t.GetSelection()
				if row > 1 {
					t.Select(row-1, col)
				}
				return nil
			case 'h':
				row, col := t.GetSelection()
				if col > 0 {
					t.Select(row, col-1)
				}
				return nil
			case 'l':
				row, col := t.GetSelection()
				if col < 6 {
					t.Select(row, col+1)
				}
				return nil
			}
		}

		return event
	})
}

// deleteSelectedTask deletes the currently selected task
func (t *TaskTable) deleteSelectedTask() {
	row, _ := t.GetSelection()

	if row < 1 || row > len(t.tasks) {
		return
	}

	task := t.tasks[row-1]

	// Remove task
	t.estimation.RemoveTask(task.ID)

	// Notify listener
	if t.OnTaskRemoved != nil {
		t.OnTaskRemoved(task.ID)
	}

	// Refresh table
	t.populate()

	// Adjust selection
	if row >= t.GetRowCount() {
		t.Select(t.GetRowCount()-1, 0)
	} else {
		t.Select(row, 0)
	}
}

// moveTaskUp moves the selected task up in the ordering
func (t *TaskTable) moveTaskUp() {
	row, _ := t.GetSelection()

	if row < 2 || row > len(t.tasks) {
		return
	}

	task := t.tasks[row-1]
	t.estimation.MoveTask(task.ID, -1)

	// Refresh table
	t.populate()

	// Notify listener
	if t.OnTaskChanged != nil {
		t.OnTaskChanged(task)
	}

	// Restore selection
	t.Select(row-1, 0)
}

// moveTaskDown moves the selected task down in the ordering
func (t *TaskTable) moveTaskDown() {
	row, _ := t.GetSelection()

	if row < 1 || row >= len(t.tasks) {
		return
	}

	task := t.tasks[row-1]
	t.estimation.MoveTask(task.ID, 1)

	// Refresh table
	t.populate()

	// Notify listener
	if t.OnTaskChanged != nil {
		t.OnTaskChanged(task)
	}

	// Restore selection
	t.Select(row+1, 0)
}

// AddTask adds a new task to the table
func (t *TaskTable) AddTask(task *model.Task) {
	t.estimation.AddTask(task)
	t.populate()

	// Notify listener
	if t.OnTaskAdded != nil {
		t.OnTaskAdded(task)
	}

	// Select the new task
	t.Select(len(t.tasks), 0)
}

// GetSelectedTask returns the currently selected task
func (t *TaskTable) GetSelectedTask() *model.Task {
	row, _ := t.GetSelection()
	if row < 1 || row > len(t.tasks) {
		return nil
	}
	return t.tasks[row-1]
}

// GetTaskCount returns the number of tasks
func (t *TaskTable) GetTaskCount() int {
	return len(t.tasks)
}

// Refresh refreshes the table display
func (t *TaskTable) Refresh() {
	t.populate()
}
