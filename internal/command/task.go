package command

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/bornholm/guesstimate/internal/model"
	"github.com/spf13/cobra"
)

// taskCmd represents the task command
var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Task management commands",
	Long:  `Manage tasks within an estimation file.`,
}

// taskAddCmd represents the task add command
var taskAddCmd = &cobra.Command{
	Use:   "add <file> <label>",
	Short: "Add a new task",
	Long:  `Add a new task to an estimation file.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		file := args[0]
		label := args[1]

		s := getStore()

		// Load or create estimation
		estimation, created, err := s.LoadOrCreateEstimation(file, file)
		if err != nil {
			return fmt.Errorf("failed to load estimation: %w", err)
		}
		if created {
			fmt.Printf("Created new estimation file: %s\n", file)
		}

		// Load config to get default category
		config, err := s.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Get flags
		category, _ := cmd.Flags().GetString("category")
		optimistic, _ := cmd.Flags().GetFloat64("optimistic")
		likely, _ := cmd.Flags().GetFloat64("likely")
		pessimistic, _ := cmd.Flags().GetFloat64("pessimistic")

		// Use default category if not specified
		if category == "" {
			category = config.GetFirstCategoryID()
		}

		// Create task
		task := model.NewTask(label, category)
		task.SetEstimations(optimistic, likely, pessimistic, config.GetAutoEstimationMultiplier())

		// Add task to estimation
		estimation.AddTask(task)

		// Save estimation
		if err := s.SaveEstimation(file, estimation); err != nil {
			return fmt.Errorf("failed to save estimation: %w", err)
		}

		fmt.Printf("Task '%s' added with ID %s\n", label, task.ID)
		return nil
	},
}

// taskUpdateCmd represents the task update command
var taskUpdateCmd = &cobra.Command{
	Use:   "update <file> <task-id>",
	Short: "Update a task",
	Long:  `Update an existing task in an estimation file.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		file := args[0]
		taskID := model.TaskID(args[1])

		s := getStore()

		// Load estimation
		estimation, err := s.LoadEstimation(file)
		if err != nil {
			return fmt.Errorf("failed to load estimation: %w", err)
		}

		// Find task
		task, ok := estimation.Tasks[taskID]
		if !ok {
			return fmt.Errorf("task with ID '%s' not found", taskID)
		}

		// Get flags
		label, _ := cmd.Flags().GetString("label")
		category, _ := cmd.Flags().GetString("category")
		optimistic, _ := cmd.Flags().GetFloat64("optimistic")
		likely, _ := cmd.Flags().GetFloat64("likely")
		pessimistic, _ := cmd.Flags().GetFloat64("pessimistic")

		// Update fields if provided
		if label != "" {
			task.Label = label
		}
		if category != "" {
			task.Category = category
		}

		// Load config for multiplier
		config, err := s.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Check if any estimation flags were provided and update with constraints
		optimisticSet := cmd.Flags().Changed("optimistic")
		likelySet := cmd.Flags().Changed("likely")
		pessimisticSet := cmd.Flags().Changed("pessimistic")

		if optimisticSet || likelySet || pessimisticSet {
			// Get current values if not set
			o := task.Estimations.Optimistic
			l := task.Estimations.Likely
			p := task.Estimations.Pessimistic

			if optimisticSet {
				o = optimistic
			}
			if likelySet {
				l = likely
			}
			if pessimisticSet {
				p = pessimistic
			}

			task.SetEstimations(o, l, p, config.GetAutoEstimationMultiplier())
		}

		// Save estimation
		if err := s.SaveEstimation(file, estimation); err != nil {
			return fmt.Errorf("failed to save estimation: %w", err)
		}

		fmt.Printf("Task %s updated\n", taskID)
		return nil
	},
}

// taskRemoveCmd represents the task remove command
var taskRemoveCmd = &cobra.Command{
	Use:   "remove <file> <task-id>",
	Short: "Remove a task",
	Long:  `Remove a task from an estimation file.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		file := args[0]
		taskID := model.TaskID(args[1])

		s := getStore()

		// Load estimation
		estimation, err := s.LoadEstimation(file)
		if err != nil {
			return fmt.Errorf("failed to load estimation: %w", err)
		}

		// Check if task exists
		if _, ok := estimation.Tasks[taskID]; !ok {
			return fmt.Errorf("task with ID '%s' not found", taskID)
		}

		// Remove task
		estimation.RemoveTask(taskID)

		// Save estimation
		if err := s.SaveEstimation(file, estimation); err != nil {
			return fmt.Errorf("failed to save estimation: %w", err)
		}

		fmt.Printf("Task %s removed\n", taskID)
		return nil
	},
}

// taskListCmd represents the task list command
var taskListCmd = &cobra.Command{
	Use:   "list <file>",
	Short: "List tasks",
	Long:  `List all tasks in an estimation file.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		file := args[0]
		format, _ := cmd.Flags().GetString("format")

		s := getStore()

		// Load estimation
		estimation, err := s.LoadEstimation(file)
		if err != nil {
			return fmt.Errorf("failed to load estimation: %w", err)
		}

		// Load config
		config, err := s.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		if len(estimation.Tasks) == 0 {
			fmt.Println("No tasks found.")
			return nil
		}

		switch format {
		case "json":
			tasks := estimation.GetOrderedTasks()
			data, err := json.MarshalIndent(tasks, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal tasks to JSON: %w", err)
			}
			fmt.Println(string(data))
		default:
			fmt.Println("Tasks:")
			for _, task := range estimation.GetOrderedTasks() {
				cat := config.GetTaskCategory(task.Category)
				mean := task.WeightedMean()
				sd := task.StandardDeviation()
				fmt.Printf("  [%s] %s (%s)\n", task.ID, task.Label, cat.Label)
				fmt.Printf("      O: %.2f, L: %.2f, P: %.2f => Mean: %.2f, SD: %.2f\n",
					task.Estimations.Optimistic, task.Estimations.Likely, task.Estimations.Pessimistic,
					mean, sd)
			}
		}

		return nil
	},
}

// taskMoveCmd represents the task move command
var taskMoveCmd = &cobra.Command{
	Use:   "move <file> <task-id> <offset>",
	Short: "Move a task",
	Long:  `Move a task up or down in the ordering. Use negative offset to move up, positive to move down.`,
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		file := args[0]
		taskID := model.TaskID(args[1])
		offset, err := strconv.Atoi(args[2])
		if err != nil {
			return fmt.Errorf("invalid offset: %w", err)
		}

		s := getStore()

		// Load estimation
		estimation, err := s.LoadEstimation(file)
		if err != nil {
			return fmt.Errorf("failed to load estimation: %w", err)
		}

		// Move task
		if !estimation.MoveTask(taskID, offset) {
			return fmt.Errorf("failed to move task %s by %d positions", taskID, offset)
		}

		// Save estimation
		if err := s.SaveEstimation(file, estimation); err != nil {
			return fmt.Errorf("failed to save estimation: %w", err)
		}

		fmt.Printf("Task %s moved by %d positions\n", taskID, offset)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(taskCmd)
	taskCmd.AddCommand(taskAddCmd)
	taskCmd.AddCommand(taskUpdateCmd)
	taskCmd.AddCommand(taskRemoveCmd)
	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskMoveCmd)

	// task add flags
	taskAddCmd.Flags().String("category", "", "Task category (default: first category in config)")
	taskAddCmd.Flags().Float64P("optimistic", "o", 0, "Optimistic estimate")
	taskAddCmd.Flags().Float64P("likely", "l", 0, "Likely estimate")
	taskAddCmd.Flags().Float64P("pessimistic", "p", 0, "Pessimistic estimate")

	// task update flags
	taskUpdateCmd.Flags().StringP("label", "l", "", "New task label")
	taskUpdateCmd.Flags().String("category", "", "New task category")
	taskUpdateCmd.Flags().Float64P("optimistic", "o", 0, "New optimistic estimate")
	taskUpdateCmd.Flags().Float64("likely", 0, "New likely estimate")
	taskUpdateCmd.Flags().Float64P("pessimistic", "p", 0, "New pessimistic estimate")

	// task list flags
	taskListCmd.Flags().StringP("format", "f", "table", "Output format (table, json)")
}
