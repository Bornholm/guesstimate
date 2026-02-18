package command

import (
	"fmt"
	"os"
	"strings"

	"github.com/bornholm/guesstimate/internal/format"
	"github.com/bornholm/guesstimate/internal/model"
	"github.com/bornholm/guesstimate/internal/stats"
	"github.com/spf13/cobra"
)

// newCmd represents the new command
var newCmd = &cobra.Command{
	Use:   "new <name>",
	Short: "Create a new estimation",
	Long:  `Create a new estimation file with the given name.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		output, _ := cmd.Flags().GetString("output")
		description, _ := cmd.Flags().GetString("description")

		// Generate output filename if not provided
		if output == "" {
			// Sanitize name for filename
			safeName := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
			output = safeName + ".estimation.yml"
		}

		s := getStore()

		// Check if file already exists
		if _, err := os.Stat(output); err == nil {
			force, _ := cmd.Flags().GetBool("force")
			if !force {
				return fmt.Errorf("file '%s' already exists, use --force to overwrite", output)
			}
		}

		// Create estimation
		estimation := model.NewEstimation(name)
		estimation.Description = description

		if err := s.SaveEstimation(output, estimation); err != nil {
			return fmt.Errorf("failed to create estimation: %w", err)
		}

		fmt.Printf("Created estimation '%s' at %s\n", name, output)
		return nil
	},
}

// viewCmd represents the view command
var viewCmd = &cobra.Command{
	Use:   "view <file>",
	Short: "View an estimation",
	Long:  `View an estimation in various formats (markdown, json, yaml).`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		file := args[0]
		formatType, _ := cmd.Flags().GetString("format")
		output, _ := cmd.Flags().GetString("output")

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

		var result string

		switch formatType {
		case "markdown", "md":
			formatter := format.NewMarkdownFormatter(config)
			result = formatter.Format(estimation)
		case "json":
			formatter := format.NewJSONFormatter(config)
			var err error
			result, err = formatter.Format(estimation)
			if err != nil {
				return fmt.Errorf("failed to format estimation as JSON: %w", err)
			}
		case "yaml", "yml":
			formatter := format.NewYAMLFormatter(config)
			var err error
			result, err = formatter.Format(estimation)
			if err != nil {
				return fmt.Errorf("failed to format estimation as YAML: %w", err)
			}
		default:
			formatter := format.NewMarkdownFormatter(config)
			result = formatter.Format(estimation)
		}

		// Output result
		if output != "" {
			if err := os.WriteFile(output, []byte(result), 0644); err != nil {
				return fmt.Errorf("failed to write output: %w", err)
			}
			fmt.Printf("Output written to %s\n", output)
		} else {
			fmt.Print(result)
		}

		return nil
	},
}

// summaryCmd represents the summary command
var summaryCmd = &cobra.Command{
	Use:   "summary <file>",
	Short: "Show estimation summary",
	Long:  `Show a quick summary of the estimation with confidence intervals.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		file := args[0]

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

		// Calculate estimation
		projectEst := stats.CalculateProjectEstimation(estimation)
		costs := stats.CalculateMinMaxCosts(estimation, config, stats.Confidence997)
		distribution := stats.CalculateCategoryDistribution(estimation, config)

		// Print summary
		fmt.Printf("Project: %s\n", estimation.Label)
		fmt.Printf("Tasks: %d\n", len(estimation.Tasks))
		fmt.Println()
		fmt.Println("Time Estimation:")
		fmt.Printf("  99.7%% confidence: %.2f ± %.2f %s\n", projectEst.WeightedMean, projectEst.StandardDeviation*3, config.TimeUnit.Acronym)
		fmt.Printf("  90%% confidence:   %.2f ± %.2f %s\n", projectEst.WeightedMean, projectEst.StandardDeviation*1.645, config.TimeUnit.Acronym)
		fmt.Printf("  68%% confidence:   %.2f ± %.2f %s\n", projectEst.WeightedMean, projectEst.StandardDeviation, config.TimeUnit.Acronym)
		fmt.Println()

		// Category distribution
		if len(distribution) > 0 {
			fmt.Println("Category Repartition:")
			for _, dist := range distribution {
				if dist.Percentage > 0 {
					fmt.Printf("  %s: %.1f%% (%.2f %s)\n", dist.CategoryLabel, dist.Percentage, dist.Time, config.TimeUnit.Acronym)
				}
			}
			fmt.Println()
		}

		fmt.Println("Cost Estimation (99.7% confidence):")
		fmt.Printf("  Maximum: %.2f %s (%.2f %s)\n", costs.Max.TotalCost, config.Currency, costs.Max.TotalTime, config.TimeUnit.Acronym)
		fmt.Printf("  Minimum: %.2f %s (%.2f %s)\n", costs.Min.TotalCost, config.Currency, costs.Min.TotalTime, config.TimeUnit.Acronym)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(viewCmd)
	rootCmd.AddCommand(summaryCmd)

	// new command flags
	newCmd.Flags().StringP("output", "o", "", "Output file path (default: <name>.estimation.yml)")
	newCmd.Flags().StringP("description", "d", "", "Project description")
	newCmd.Flags().BoolP("force", "f", false, "Force overwrite existing file")

	// view command flags
	viewCmd.Flags().StringP("format", "f", "markdown", "Output format (markdown, json, yaml)")
	viewCmd.Flags().StringP("output", "o", "", "Output file path (default: stdout)")
}
