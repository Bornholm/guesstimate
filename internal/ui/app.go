package ui

import (
	"fmt"
	"strings"

	"github.com/bornholm/guesstimate/internal/model"
	"github.com/bornholm/guesstimate/internal/stats"
	"github.com/bornholm/guesstimate/internal/store"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// App represents the main tview application
type App struct {
	app        *tview.Application
	store      store.Store
	config     *model.Config
	estimation *model.Estimation
	filePath   string

	// UI Components
	pages      *tview.Pages
	layout     *tview.Flex
	header     *tview.TextView
	taskTable  *TaskTable
	preview    *tview.TextView
	footer     *tview.TextView
	commandBar *tview.InputField

	// State
	hasUnsavedChanges bool
	commandMode       bool
	modalVisible      bool
}

// NewApp creates a new App instance
func NewApp(s store.Store, config *model.Config, estimation *model.Estimation, filePath string) *App {
	a := &App{
		app:        tview.NewApplication(),
		store:      s,
		config:     config,
		estimation: estimation,
		filePath:   filePath,
	}

	a.setupUI()

	return a
}

// setupUI creates and configures all UI components
func (a *App) setupUI() {
	// Header
	a.header = tview.NewTextView()
	a.header.SetDynamicColors(true)
	a.header.SetTextAlign(tview.AlignCenter)
	a.updateHeader()

	// Task table
	a.taskTable = NewTaskTable(a.estimation, a.config)
	a.taskTable.OnTaskChanged = a.onTaskChanged
	a.taskTable.OnTaskAdded = a.onTaskAdded
	a.taskTable.OnTaskRemoved = a.onTaskRemoved

	// Preview
	a.preview = tview.NewTextView()
	a.preview.SetDynamicColors(true)
	a.preview.SetBorder(true)
	a.preview.SetTitle(" Estimation Preview ")
	a.updatePreview()

	// Command bar (hidden by default)
	a.commandBar = tview.NewInputField()
	a.commandBar.SetLabel(":")
	a.commandBar.SetFieldWidth(40)
	a.commandBar.SetDoneFunc(a.handleCommand)

	// Footer
	a.footer = tview.NewTextView()
	a.footer.SetDynamicColors(true)
	a.updateFooter()

	// Main content (two columns)
	mainContent := tview.NewFlex().SetDirection(tview.FlexColumn)
	mainContent.AddItem(a.taskTable, 0, 3, true) // Left: tasks table (3/4 width)
	mainContent.AddItem(a.preview, 0, 1, false)  // Right: estimation preview (1/4 width)

	// Layout
	a.layout = tview.NewFlex().SetDirection(tview.FlexRow)
	a.layout.AddItem(a.header, 3, 0, false)
	a.layout.AddItem(mainContent, 0, 1, true)
	a.layout.AddItem(a.footer, 1, 0, false)

	// Pages for modal dialogs
	a.pages = tview.NewPages()
	a.pages.AddPage("main", a.layout, true, true)
}

// updateFooter updates the footer text
func (a *App) updateFooter() {
	a.footer.SetText("[yellow]:w[white] Save  [yellow]:q[white] Quit  [yellow]:q![white] Force Quit  [yellow]a[white] Add Task  [yellow]e[white] Edit  [yellow]d[white] Delete  [yellow]?[white] Help")
}

// Run starts the application
func (a *App) Run() error {
	// Set up input capture on the pages (not layout)
	a.pages.SetInputCapture(a.handleInput)

	// Prevent Ctrl+C from quitting the app
	a.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlC {
			// Ignore Ctrl+C, user must use :q or :q! to quit
			return nil
		}
		return event
	})

	a.app.SetRoot(a.pages, true)
	a.app.SetFocus(a.taskTable)
	return a.app.Run()
}

// handleInput handles global key input
func (a *App) handleInput(event *tcell.EventKey) *tcell.EventKey {
	// If modal is visible, pass all keys to modal
	if a.modalVisible {
		return event
	}

	// If in command mode, pass all keys to command bar
	if a.commandMode {
		return event
	}

	switch event.Key() {
	case tcell.KeyRune:
		switch event.Rune() {
		case ':':
			// Start command mode
			a.startCommandMode()
			return nil
		case '?':
			a.showHelp()
			return nil
		case 'a':
			a.addNewTask()
			return nil
		case 'e', 'i':
			a.editSelectedTask()
			return nil
		case 'd':
			a.deleteSelectedTask()
			return nil
		case 'J':
			a.moveTaskDown()
			return nil
		case 'K':
			a.moveTaskUp()
			return nil
		}
	}

	// Pass through to task table for navigation
	return event
}

// startCommandMode enters command mode
func (a *App) startCommandMode() {
	a.commandMode = true
	a.commandBar.SetText("")

	// Replace footer with command bar
	a.layout.RemoveItem(a.footer)
	a.layout.AddItem(a.commandBar, 1, 0, true)
	a.app.SetFocus(a.commandBar)
}

// exitCommandMode exits command mode
func (a *App) exitCommandMode() {
	a.commandMode = false
	a.commandBar.SetText("")

	// Restore footer
	a.layout.RemoveItem(a.commandBar)
	a.layout.AddItem(a.footer, 1, 0, false)
	a.app.SetFocus(a.taskTable)
}

// handleCommand processes the command entered in command mode
func (a *App) handleCommand(key tcell.Key) {
	if key != tcell.KeyEnter {
		a.exitCommandMode()
		return
	}

	command := strings.TrimSpace(a.commandBar.GetText())

	switch command {
	case "w":
		a.save()
		a.exitCommandMode()
	case "q":
		if a.hasUnsavedChanges {
			// Show error in command bar, don't exit
			a.commandBar.SetText("[red]Error: Unsaved changes. Use :q! to force quit.[white]")
			a.commandBar.SetLabel(":")
		} else {
			a.app.Stop()
		}
	case "q!":
		a.app.Stop()
	case "wq", "x":
		if err := a.store.SaveEstimation(a.filePath, a.estimation); err == nil {
			a.app.Stop()
		} else {
			a.commandBar.SetText(fmt.Sprintf("[red]Error: Failed to save: %v[white]", err))
			a.commandBar.SetLabel(":")
		}
	default:
		a.exitCommandMode()
	}
}

// deleteSelectedTask deletes the currently selected task
func (a *App) deleteSelectedTask() {
	row, _ := a.taskTable.GetSelection()
	if row < 1 || row > a.taskTable.GetTaskCount() {
		return
	}

	task := a.taskTable.GetSelectedTask()
	if task == nil {
		return
	}

	// Delete directly without confirmation
	a.estimation.RemoveTask(task.ID)
	a.taskTable.Refresh()
	a.hasUnsavedChanges = true
	a.updateHeader()
	a.updatePreview()
}

// moveTaskUp moves the selected task up
func (a *App) moveTaskUp() {
	row, _ := a.taskTable.GetSelection()
	if row < 2 {
		return
	}

	task := a.taskTable.GetSelectedTask()
	if task == nil {
		return
	}

	a.estimation.MoveTask(task.ID, -1)
	a.taskTable.Refresh()
	a.hasUnsavedChanges = true
	a.updateHeader()
	a.updatePreview()
	a.taskTable.Select(row-1, 0)
}

// moveTaskDown moves the selected task down
func (a *App) moveTaskDown() {
	row, _ := a.taskTable.GetSelection()
	if row >= a.taskTable.GetTaskCount() {
		return
	}

	task := a.taskTable.GetSelectedTask()
	if task == nil {
		return
	}

	a.estimation.MoveTask(task.ID, 1)
	a.taskTable.Refresh()
	a.hasUnsavedChanges = true
	a.updateHeader()
	a.updatePreview()
	a.taskTable.Select(row+1, 0)
}

// updateHeader updates the header text
func (a *App) updateHeader() {
	title := a.estimation.Label
	if title == "" {
		title = "Untitled Project"
	}

	saved := ""
	if a.hasUnsavedChanges {
		saved = " [red](unsaved changes)[white]"
	}

	a.header.SetTitle(fmt.Sprintf(" Guesstimate - %s%s ", title, saved))
	a.header.SetBorder(true)
}

// updatePreview updates the estimation preview
func (a *App) updatePreview() {
	var sb strings.Builder

	projectEst := stats.CalculateProjectEstimation(a.estimation)
	roundUp := a.config.RoundUpEstimations

	sb.WriteString(fmt.Sprintf("[yellow]Tasks:[white] %d\n\n", len(a.estimation.Tasks)))

	sb.WriteString("[yellow]Time Estimation:[white]\n")
	sb.WriteString(fmt.Sprintf("  99.7%%: %s ± %s %s\n",
		formatFloat(projectEst.WeightedMean, roundUp),
		formatFloat(projectEst.StandardDeviation*3, roundUp),
		a.config.TimeUnit.Acronym))
	sb.WriteString(fmt.Sprintf("  90%%:   %s ± %s %s\n",
		formatFloat(projectEst.WeightedMean, roundUp),
		formatFloat(projectEst.StandardDeviation*1.645, roundUp),
		a.config.TimeUnit.Acronym))
	sb.WriteString(fmt.Sprintf("  68%%:   %s ± %s %s\n",
		formatFloat(projectEst.WeightedMean, roundUp),
		formatFloat(projectEst.StandardDeviation, roundUp),
		a.config.TimeUnit.Acronym))

	// Category distribution
	distribution := stats.CalculateCategoryDistribution(a.estimation, a.config)
	if len(distribution) > 0 {
		sb.WriteString("\n[yellow]Category Repartition:[white]\n")
		for _, dist := range distribution {
			if dist.Percentage > 0 {
				sb.WriteString(fmt.Sprintf("  %s: %.1f%% (%s %s)\n",
					dist.CategoryLabel,
					dist.Percentage,
					formatFloat(dist.Time, roundUp),
					a.config.TimeUnit.Acronym))
			}
		}
	}

	costs := stats.CalculateMinMaxCosts(a.estimation, a.config, stats.Confidence997)
	sb.WriteString(fmt.Sprintf("\n[yellow]Cost (99.7%%):[white]\n"))
	sb.WriteString(fmt.Sprintf("  Max: %s %s (%s %s)\n",
		formatFloat(costs.Max.TotalCost, false), a.config.Currency,
		formatFloat(costs.Max.TotalTime, roundUp), a.config.TimeUnit.Acronym))
	sb.WriteString(fmt.Sprintf("  Min: %s %s (%s %s)",
		formatFloat(costs.Min.TotalCost, false), a.config.Currency,
		formatFloat(costs.Min.TotalTime, roundUp), a.config.TimeUnit.Acronym))

	a.preview.SetText(sb.String())
}

// onTaskChanged is called when a task is modified
func (a *App) onTaskChanged(task *model.Task) {
	// Task is already modified in place (it's a pointer to the task in the estimation)
	a.hasUnsavedChanges = true
	a.updateHeader()
	a.updatePreview()
}

// onTaskAdded is called when a new task is added
func (a *App) onTaskAdded(task *model.Task) {
	// Task is already added by TaskTable.AddTask
	a.hasUnsavedChanges = true
	a.updateHeader()
	a.updatePreview()
}

// onTaskRemoved is called when a task is removed
func (a *App) onTaskRemoved(taskID model.TaskID) {
	// Task is already removed by TaskTable.deleteSelectedTask
	a.hasUnsavedChanges = true
	a.updateHeader()
	a.updatePreview()
}

// save saves the estimation to file
func (a *App) save() {
	if err := a.store.SaveEstimation(a.filePath, a.estimation); err != nil {
		// Show error in command bar
		a.commandBar.SetText(fmt.Sprintf("[red]Error: Failed to save: %v[white]", err))
		return
	}
	a.hasUnsavedChanges = false
	a.updateHeader()
}

// quit exits the application (now handled in handleCommand)
func (a *App) quit() {
	if a.hasUnsavedChanges {
		// This shouldn't be called anymore, but keep for safety
		return
	}
	a.app.Stop()
}

// editSelectedTask opens a modal to edit the selected task
func (a *App) editSelectedTask() {
	task := a.taskTable.GetSelectedTask()
	if task == nil {
		return
	}

	// Store current selection
	row, col := a.taskTable.GetSelection()

	// Create form
	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle(fmt.Sprintf(" Edit Task: %s ", task.Label))
	form.SetTitleAlign(tview.AlignCenter)

	label := task.Label
	description := task.Description
	optimisticVal := task.Estimations.Optimistic
	likelyVal := task.Estimations.Likely
	pessimisticVal := task.Estimations.Pessimistic

	// Get category options
	var categoryOptions []string
	var categoryIDs []string
	var selectedCategoryIndex int
	for id, cat := range a.config.TaskCategories {
		categoryOptions = append(categoryOptions, cat.Label)
		categoryIDs = append(categoryIDs, id)
		if id == task.Category {
			selectedCategoryIndex = len(categoryOptions) - 1
		}
	}
	category := task.Category

	form.AddInputField("Label:", label, 40, nil, func(text string) {
		label = text
	})

	// Description as a text area (using InputField with larger width)
	form.AddTextArea("Description:", description, 60, 3, 0, func(text string) {
		description = text
	})

	form.AddDropDown("Category:", categoryOptions, selectedCategoryIndex, func(option string, index int) {
		category = categoryIDs[index]
	})

	// Create estimation input fields
	optimisticField := tview.NewInputField().
		SetLabel("Optimistic:").
		SetText(fmt.Sprintf("%.1f", optimisticVal)).
		SetFieldWidth(10)
	likelyField := tview.NewInputField().
		SetLabel("Likely:").
		SetText(fmt.Sprintf("%.1f", likelyVal)).
		SetFieldWidth(10)
	pessimisticField := tview.NewInputField().
		SetLabel("Pessimistic:").
		SetText(fmt.Sprintf("%.1f", pessimisticVal)).
		SetFieldWidth(10)

	// Add the input fields to the form
	form.AddFormItem(optimisticField)
	form.AddFormItem(likelyField)
	form.AddFormItem(pessimisticField)

	// Helper function to close modal
	closeModal := func() {
		a.modalVisible = false
		a.pages.RemovePage("modal")
		a.app.SetFocus(a.taskTable)
		a.taskTable.Select(row, col)
	}

	// Helper function to save and close
	saveAndClose := func() {
		task.Label = label
		task.Description = description
		task.Category = category
		// Get values from fields (they may have been updated)
		optimisticVal = parseFloat(optimisticField.GetText())
		likelyVal = parseFloat(likelyField.GetText())
		pessimisticVal = parseFloat(pessimisticField.GetText())
		task.SetEstimations(optimisticVal, likelyVal, pessimisticVal, a.config.GetAutoEstimationMultiplier())

		a.taskTable.Refresh()
		a.hasUnsavedChanges = true
		a.updateHeader()
		a.updatePreview()
		closeModal()
	}

	// Add vim-style command handling for the form
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Handle Escape to cancel
		if event.Key() == tcell.KeyEscape {
			closeModal()
			return nil
		}
		return event
	})

	form.AddButton("Save (Enter)", saveAndClose)
	form.AddButton("Cancel (Esc)", closeModal)

	form.SetCancelFunc(closeModal)

	// Center the form using a flex container
	flex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(form, 22, 1, true).
			AddItem(nil, 0, 1, false), 80, 1, true).
		AddItem(nil, 0, 1, false)

	a.modalVisible = true
	a.pages.AddPage("modal", flex, true, true)
	a.app.SetFocus(form)
}

// addNewTask opens a dialog to add a new task
func (a *App) addNewTask() {
	// Create form
	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle(" Add New Task ")
	form.SetTitleAlign(tview.AlignCenter)

	var label string
	var description string
	category := a.config.GetFirstCategoryID()

	// Get category options
	var categoryOptions []string
	var categoryIDs []string
	for id, cat := range a.config.TaskCategories {
		categoryOptions = append(categoryOptions, cat.Label)
		categoryIDs = append(categoryIDs, id)
	}

	form.AddInputField("Label:", "", 40, nil, func(text string) {
		label = text
	})

	// Description as a text area
	form.AddTextArea("Description:", "", 60, 3, 0, func(text string) {
		description = text
	})

	form.AddDropDown("Category:", categoryOptions, 0, func(option string, index int) {
		category = categoryIDs[index]
	})

	// Create estimation input fields
	optimisticField := tview.NewInputField().
		SetLabel("Optimistic:").
		SetText("0").
		SetFieldWidth(10)
	likelyField := tview.NewInputField().
		SetLabel("Likely:").
		SetText("0").
		SetFieldWidth(10)
	pessimisticField := tview.NewInputField().
		SetLabel("Pessimistic:").
		SetText("0").
		SetFieldWidth(10)

	// Add the input fields to the form
	form.AddFormItem(optimisticField)
	form.AddFormItem(likelyField)
	form.AddFormItem(pessimisticField)

	// Helper function to close modal
	closeModal := func() {
		a.modalVisible = false
		a.pages.RemovePage("modal")
		a.app.SetFocus(a.taskTable)
	}

	// Helper function to add task and close
	addAndClose := func() {
		task := model.NewTask(label, category)
		task.Description = description
		// Get values from fields
		optimisticVal := parseFloat(optimisticField.GetText())
		likelyVal := parseFloat(likelyField.GetText())
		pessimisticVal := parseFloat(pessimisticField.GetText())
		task.SetEstimations(optimisticVal, likelyVal, pessimisticVal, a.config.GetAutoEstimationMultiplier())

		a.taskTable.AddTask(task)
		a.hasUnsavedChanges = true
		a.updateHeader()
		a.updatePreview()
		closeModal()
	}

	// Add vim-style command handling for the form
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Handle Escape to cancel
		if event.Key() == tcell.KeyEscape {
			closeModal()
			return nil
		}
		return event
	})

	form.AddButton("Add (Enter)", addAndClose)
	form.AddButton("Cancel (Esc)", closeModal)

	form.SetCancelFunc(closeModal)

	// Center the form using a flex container
	flex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(form, 22, 1, true).
			AddItem(nil, 0, 1, false), 80, 1, true).
		AddItem(nil, 0, 1, false)

	a.modalVisible = true
	a.pages.AddPage("modal", flex, true, true)
	a.app.SetFocus(form)
}

// showHelp displays help information
func (a *App) showHelp() {
	// Use a TextView for better control over text alignment
	helpView := tview.NewTextView()
	helpView.SetDynamicColors(true)
	helpView.SetBorder(true)
	helpView.SetTitle(" Keyboard Shortcuts ")
	helpView.SetTitleAlign(tview.AlignCenter)
	helpView.SetTextAlign(tview.AlignLeft)

	// Build help text with consistent formatting
	helpText := `[yellow]Commands:[white]
  :w         Save estimation
  :q         Quit application
  :q!        Force quit (discard changes)
  :wq or :x  Save and quit

[yellow]Task Operations:[white]
  a          Add new task
  e or i     Edit selected task
  d          Delete selected task

[yellow]Navigation:[white]
  J          Move task down
  K          Move task up
  j/k/h/l    Navigate (vim-style)

[yellow]Other:[white]
  ?          Show this help

[gray]Press Escape or Enter to close[white]`

	helpView.SetText(helpText)

	// Handle key events to close
	helpView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || event.Key() == tcell.KeyEnter {
			a.modalVisible = false
			a.pages.RemovePage("modal")
			a.app.SetFocus(a.taskTable)
			return nil
		}
		return event
	})

	// Center the help view using a flex container
	flex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(helpView, 18, 1, true).
			AddItem(nil, 0, 1, false), 50, 1, true).
		AddItem(nil, 0, 1, false)

	a.modalVisible = true
	a.pages.AddPage("modal", flex, true, true)
	a.app.SetFocus(helpView)
}

func formatFloat(value float64, roundUp bool) string {
	if roundUp {
		return fmt.Sprintf("%.0f", value)
	}
	return fmt.Sprintf("%.2f", value)
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}
