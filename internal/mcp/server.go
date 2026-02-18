package mcp

import (
	"context"
	"fmt"

	"github.com/bornholm/guesstimate/internal/model"
	"github.com/bornholm/guesstimate/internal/stats"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server represents the MCP server for guesstimate operations
type Server struct {
	server *mcp.Server
	store  *ChrootedStore
	config *model.Config
}

// ServerOptions contains options for the MCP server
type ServerOptions struct {
	RootDir string
	Config  *model.Config
}

// NewServer creates a new MCP server for guesstimate operations
func NewServer(opts *ServerOptions) (*Server, error) {
	rootDir := opts.RootDir
	if rootDir == "" {
		rootDir = "."
	}

	store, err := NewChrootedStore(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create chrooted store: %w", err)
	}

	// Use provided config or default
	config := opts.Config
	if config == nil {
		config = model.DefaultConfig()
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "guesstimate",
		Version: "1.0.0",
	}, nil)

	s := &Server{
		server: server,
		store:  store,
		config: config,
	}

	// Register tools
	s.registerTools()

	return s, nil
}

// Run starts the MCP server on stdio transport
func (s *Server) Run(ctx context.Context) error {
	return s.server.Run(ctx, &mcp.StdioTransport{})
}

// Close closes the server and releases resources
func (s *Server) Close() error {
	return s.store.Close()
}

func (s *Server) registerTools() {
	// Estimation tools
	s.registerListEstimationsTool()
	s.registerCreateEstimationTool()
	s.registerGetEstimationTool()
	s.registerDeleteEstimationTool()
	s.registerGetEstimationSummaryTool()

	// Task tools
	s.registerListTasksTool()
	s.registerAddTaskTool()
	s.registerUpdateTaskTool()
	s.registerRemoveTaskTool()

	// Config tools
	s.registerGetConfigTool()
}

// list_estimations tool
type listEstimationsArgs struct {
	Dir string `json:"dir,omitempty" jsonschema:"the directory to list estimations from, defaults to current directory"`
}

func (s *Server) registerListEstimationsTool() {
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "list_estimations",
		Description: "List all estimation files in a directory",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args listEstimationsArgs) (*mcp.CallToolResult, any, error) {
		dir := args.Dir
		if dir == "" {
			dir = "."
		}

		files, err := s.store.ListEstimations(dir)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to list estimations: %w", err)
		}

		if len(files) == 0 {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: "No estimation files found."},
				},
			}, nil, nil
		}

		result := "Estimation files:\n"
		for _, f := range files {
			result += fmt.Sprintf("- %s\n", f)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}, nil, nil
	})
}

// create_estimation tool
type createEstimationArgs struct {
	Path        string `json:"path" jsonschema:"required,the file path for the estimation"`
	Label       string `json:"label" jsonschema:"required,the label/name for the estimation"`
	Description string `json:"description,omitempty" jsonschema:"optional description for the estimation"`
}

func (s *Server) registerCreateEstimationTool() {
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "create_estimation",
		Description: "Create a new estimation file",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args createEstimationArgs) (*mcp.CallToolResult, any, error) {
		estimation := model.NewEstimation(args.Label)
		estimation.Description = args.Description

		if err := s.store.SaveEstimation(args.Path, estimation); err != nil {
			return nil, nil, fmt.Errorf("failed to create estimation: %w", err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Created estimation '%s' at %s with ID %s", args.Label, args.Path, estimation.ID)},
			},
		}, nil, nil
	})
}

// get_estimation tool
type getEstimationArgs struct {
	Path string `json:"path" jsonschema:"required,the file path to the estimation"`
}

func (s *Server) registerGetEstimationTool() {
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "get_estimation",
		Description: "Get details of an estimation file",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args getEstimationArgs) (*mcp.CallToolResult, any, error) {
		estimation, err := s.store.LoadEstimation(args.Path)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load estimation: %w", err)
		}

		result := fmt.Sprintf("Estimation: %s\n", estimation.Label)
		result += fmt.Sprintf("ID: %s\n", estimation.ID)
		if estimation.Description != "" {
			result += fmt.Sprintf("Description: %s\n", estimation.Description)
		}
		result += fmt.Sprintf("Tasks: %d\n", len(estimation.Tasks))
		result += fmt.Sprintf("Created: %s\n", estimation.CreatedAt.Format("2006-01-02 15:04:05"))
		result += fmt.Sprintf("Updated: %s\n", estimation.UpdatedAt.Format("2006-01-02 15:04:05"))

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}, nil, nil
	})
}

// delete_estimation tool
type deleteEstimationArgs struct {
	Path string `json:"path" jsonschema:"required,the file path to the estimation to delete"`
}

func (s *Server) registerDeleteEstimationTool() {
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "delete_estimation",
		Description: "Delete an estimation file",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args deleteEstimationArgs) (*mcp.CallToolResult, any, error) {
		if err := s.store.DeleteEstimation(args.Path); err != nil {
			return nil, nil, fmt.Errorf("failed to delete estimation: %w", err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Deleted estimation at %s", args.Path)},
			},
		}, nil, nil
	})
}

// get_estimation_summary tool
type getEstimationSummaryArgs struct {
	Path string `json:"path" jsonschema:"required,the file path to the estimation"`
}

func (s *Server) registerGetEstimationSummaryTool() {
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "get_estimation_summary",
		Description: "Get a summary of the estimation with confidence intervals and cost estimates",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args getEstimationSummaryArgs) (*mcp.CallToolResult, any, error) {
		estimation, err := s.store.LoadEstimation(args.Path)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load estimation: %w", err)
		}

		projectEst := stats.CalculateProjectEstimation(estimation)
		costs := stats.CalculateMinMaxCosts(estimation, s.config, stats.Confidence997)
		distribution := stats.CalculateCategoryDistribution(estimation, s.config)

		result := fmt.Sprintf("Project: %s\n", estimation.Label)
		result += fmt.Sprintf("Tasks: %d\n\n", len(estimation.Tasks))

		result += "Time Estimation:\n"
		result += fmt.Sprintf("  99.7%% confidence: %.2f ± %.2f %s\n", projectEst.WeightedMean, projectEst.StandardDeviation*3, s.config.TimeUnit.Acronym)
		result += fmt.Sprintf("  90%% confidence:   %.2f ± %.2f %s\n", projectEst.WeightedMean, projectEst.StandardDeviation*1.645, s.config.TimeUnit.Acronym)
		result += fmt.Sprintf("  68%% confidence:   %.2f ± %.2f %s\n\n", projectEst.WeightedMean, projectEst.StandardDeviation, s.config.TimeUnit.Acronym)

		if len(distribution) > 0 {
			result += "Category Repartition:\n"
			for _, dist := range distribution {
				if dist.Percentage > 0 {
					result += fmt.Sprintf("  %s: %.1f%% (%.2f %s)\n", dist.CategoryLabel, dist.Percentage, dist.Time, s.config.TimeUnit.Acronym)
				}
			}
			result += "\n"
		}

		result += "Cost Estimation (99.7% confidence):\n"
		result += fmt.Sprintf("  Maximum: %.2f %s (%.2f %s)\n", costs.Max.TotalCost, s.config.Currency, costs.Max.TotalTime, s.config.TimeUnit.Acronym)
		result += fmt.Sprintf("  Minimum: %.2f %s (%.2f %s)\n", costs.Min.TotalCost, s.config.Currency, costs.Min.TotalTime, s.config.TimeUnit.Acronym)

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}, nil, nil
	})
}

// list_tasks tool
type listTasksArgs struct {
	Path string `json:"path" jsonschema:"required,the file path to the estimation"`
}

func (s *Server) registerListTasksTool() {
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "list_tasks",
		Description: "List all tasks in an estimation",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args listTasksArgs) (*mcp.CallToolResult, any, error) {
		estimation, err := s.store.LoadEstimation(args.Path)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load estimation: %w", err)
		}

		if len(estimation.Tasks) == 0 {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: "No tasks found in this estimation."},
				},
			}, nil, nil
		}

		result := "Tasks:\n"
		for _, task := range estimation.GetOrderedTasks() {
			cat := s.config.GetTaskCategory(task.Category)
			mean := task.WeightedMean()
			sd := task.StandardDeviation()
			result += fmt.Sprintf("  [%s] %s (%s)\n", task.ID, task.Label, cat.Label)
			result += fmt.Sprintf("      O: %.2f, L: %.2f, P: %.2f => Mean: %.2f, SD: %.2f\n",
				task.Estimations.Optimistic, task.Estimations.Likely, task.Estimations.Pessimistic,
				mean, sd)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}, nil, nil
	})
}

// add_task tool
type addTaskArgs struct {
	Path        string  `json:"path" jsonschema:"required,the file path to the estimation"`
	Label       string  `json:"label" jsonschema:"required,the task label"`
	Category    string  `json:"category,omitempty" jsonschema:"optional task category, defaults to first category in config"`
	Optimistic  float64 `json:"optimistic,omitempty" jsonschema:"optional optimistic estimate, defaults to 0"`
	Likely      float64 `json:"likely,omitempty" jsonschema:"optional likely estimate, defaults to 0"`
	Pessimistic float64 `json:"pessimistic,omitempty" jsonschema:"optional pessimistic estimate, defaults to 0"`
}

func (s *Server) registerAddTaskTool() {
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "add_task",
		Description: "Add a new task to an estimation. If only some estimation values are provided, the missing ones will be auto-calculated using the configured multiplier (default 33%).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args addTaskArgs) (*mcp.CallToolResult, any, error) {
		estimation, _, err := s.store.LoadOrCreateEstimation(args.Path, args.Path)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load estimation: %w", err)
		}

		category := args.Category
		if category == "" {
			category = s.config.GetFirstCategoryID()
		}

		task := model.NewTask(args.Label, category)
		task.SetEstimations(args.Optimistic, args.Likely, args.Pessimistic, s.config.GetAutoEstimationMultiplier())

		estimation.AddTask(task)

		if err := s.store.SaveEstimation(args.Path, estimation); err != nil {
			return nil, nil, fmt.Errorf("failed to save estimation: %w", err)
		}

		result := fmt.Sprintf("Task '%s' added with ID %s\n", args.Label, task.ID)
		result += fmt.Sprintf("Estimations: O=%.2f, L=%.2f, P=%.2f",
			task.Estimations.Optimistic, task.Estimations.Likely, task.Estimations.Pessimistic)

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}, nil, nil
	})
}

// update_task tool
type updateTaskArgs struct {
	Path        string   `json:"path" jsonschema:"required,the file path to the estimation"`
	TaskID      string   `json:"taskId" jsonschema:"required,the task ID to update"`
	Label       string   `json:"label,omitempty" jsonschema:"optional new task label"`
	Category    string   `json:"category,omitempty" jsonschema:"optional new task category"`
	Optimistic  *float64 `json:"optimistic,omitempty" jsonschema:"optional new optimistic estimate"`
	Likely      *float64 `json:"likely,omitempty" jsonschema:"optional new likely estimate"`
	Pessimistic *float64 `json:"pessimistic,omitempty" jsonschema:"optional new pessimistic estimate"`
}

func (s *Server) registerUpdateTaskTool() {
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "update_task",
		Description: "Update an existing task in an estimation. If estimation values are updated, missing/invalid ones will be auto-calculated using the configured multiplier (default 33%).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args updateTaskArgs) (*mcp.CallToolResult, any, error) {
		estimation, err := s.store.LoadEstimation(args.Path)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load estimation: %w", err)
		}

		taskID := model.TaskID(args.TaskID)
		task, ok := estimation.Tasks[taskID]
		if !ok {
			return nil, nil, fmt.Errorf("task with ID '%s' not found", args.TaskID)
		}

		if args.Label != "" {
			task.Label = args.Label
		}
		if args.Category != "" {
			task.Category = args.Category
		}

		// Check if any estimation values were provided
		if args.Optimistic != nil || args.Likely != nil || args.Pessimistic != nil {
			o := task.Estimations.Optimistic
			l := task.Estimations.Likely
			p := task.Estimations.Pessimistic

			if args.Optimistic != nil {
				o = *args.Optimistic
			}
			if args.Likely != nil {
				l = *args.Likely
			}
			if args.Pessimistic != nil {
				p = *args.Pessimistic
			}

			task.SetEstimations(o, l, p, s.config.GetAutoEstimationMultiplier())
		}

		estimation.UpdateTask(task)

		if err := s.store.SaveEstimation(args.Path, estimation); err != nil {
			return nil, nil, fmt.Errorf("failed to save estimation: %w", err)
		}

		result := fmt.Sprintf("Task %s updated\n", args.TaskID)
		result += fmt.Sprintf("Estimations: O=%.2f, L=%.2f, P=%.2f",
			task.Estimations.Optimistic, task.Estimations.Likely, task.Estimations.Pessimistic)

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}, nil, nil
	})
}

// remove_task tool
type removeTaskArgs struct {
	Path   string `json:"path" jsonschema:"required,the file path to the estimation"`
	TaskID string `json:"taskId" jsonschema:"required,the task ID to remove"`
}

func (s *Server) registerRemoveTaskTool() {
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "remove_task",
		Description: "Remove a task from an estimation",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args removeTaskArgs) (*mcp.CallToolResult, any, error) {
		estimation, err := s.store.LoadEstimation(args.Path)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load estimation: %w", err)
		}

		taskID := model.TaskID(args.TaskID)
		if _, ok := estimation.Tasks[taskID]; !ok {
			return nil, nil, fmt.Errorf("task with ID '%s' not found", args.TaskID)
		}

		estimation.RemoveTask(taskID)

		if err := s.store.SaveEstimation(args.Path, estimation); err != nil {
			return nil, nil, fmt.Errorf("failed to save estimation: %w", err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Task %s removed", args.TaskID)},
			},
		}, nil, nil
	})
}

// get_config tool
type getConfigArgs struct{}

func (s *Server) registerGetConfigTool() {
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "get_config",
		Description: "Get the current guesstimate configuration",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args getConfigArgs) (*mcp.CallToolResult, any, error) {
		result := "Configuration:\n"
		result += fmt.Sprintf("  Time Unit: %s (%s)\n", s.config.TimeUnit.Label, s.config.TimeUnit.Acronym)
		result += fmt.Sprintf("  Currency: %s\n", s.config.Currency)
		result += fmt.Sprintf("  Round Up Estimations: %v\n", s.config.RoundUpEstimations)
		result += fmt.Sprintf("  Auto Estimation Multiplier: %.0f%%\n\n", s.config.GetAutoEstimationMultiplier()*100)

		result += "Task Categories:\n"
		for id, cat := range s.config.TaskCategories {
			result += fmt.Sprintf("  %s: %s (%.2f per %s)\n", id, cat.Label, cat.CostPerTimeUnit, s.config.TimeUnit.Acronym)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: result},
			},
		}, nil, nil
	})
}
