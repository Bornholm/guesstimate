package command

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/bornholm/guesstimate/internal/model"
	"github.com/bornholm/guesstimate/internal/store"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management commands",
	Long:  `Manage the guesstimate configuration file.`,
}

// configInitCmd represents the config init command
var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration file",
	Long:  `Create a default configuration file in the config directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := getStore()

		// Check if config already exists
		configPath := configFile
		if configPath == "" {
			configPath = store.DefaultConfigFile
		}
		if _, err := os.Stat(configPath); err == nil {
			force, _ := cmd.Flags().GetBool("force")
			if !force {
				return fmt.Errorf("configuration file already exists at %s, use --force to overwrite", configPath)
			}
		}

		// Create default config
		config := model.DefaultConfig()

		if err := s.SaveConfig(config); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}

		fmt.Printf("Configuration file created at %s\n", configPath)
		return nil
	},
}

// configViewCmd represents the config view command
var configViewCmd = &cobra.Command{
	Use:   "view",
	Short: "View current configuration",
	Long:  `Display the current configuration settings.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := getStore()

		config, err := s.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		format, _ := cmd.Flags().GetString("format")

		switch format {
		case "json":
			data, err := json.MarshalIndent(config, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal config to JSON: %w", err)
			}
			fmt.Println(string(data))
		case "yaml":
			data, err := yaml.Marshal(config)
			if err != nil {
				return fmt.Errorf("failed to marshal config to YAML: %w", err)
			}
			fmt.Print(string(data))
		default:
			fmt.Println("Task Categories:")
			for id, cat := range config.TaskCategories {
				fmt.Printf("  %s: %s (%.2f per time unit)\n", id, cat.Label, cat.CostPerTimeUnit)
			}
			fmt.Printf("\nTime Unit: %s (%s)\n", config.TimeUnit.Label, config.TimeUnit.Acronym)
			fmt.Printf("Currency: %s\n", config.Currency)
			fmt.Printf("Round Up Estimations: %v\n", config.RoundUpEstimations)
		}

		return nil
	},
}

// configCategoryCmd represents the config category command
var configCategoryCmd = &cobra.Command{
	Use:   "category",
	Short: "Category management commands",
	Long:  `Manage task categories in the configuration.`,
}

// configCategoryAddCmd represents the config category add command
var configCategoryAddCmd = &cobra.Command{
	Use:   "add <id> <label>",
	Short: "Add a new task category",
	Long:  `Add a new task category to the configuration.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := getStore()

		config, err := s.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		id := args[0]
		label := args[1]
		cost, _ := cmd.Flags().GetFloat64("cost")

		if _, exists := config.TaskCategories[id]; exists {
			return fmt.Errorf("category with id '%s' already exists", id)
		}

		config.TaskCategories[id] = model.TaskCategory{
			ID:              id,
			Label:           label,
			CostPerTimeUnit: cost,
		}

		if err := s.SaveConfig(config); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}

		fmt.Printf("Category '%s' added successfully\n", id)
		return nil
	},
}

// configCategoryRemoveCmd represents the config category remove command
var configCategoryRemoveCmd = &cobra.Command{
	Use:   "remove <id>",
	Short: "Remove a task category",
	Long:  `Remove a task category from the configuration.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := getStore()

		config, err := s.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		id := args[0]

		if _, exists := config.TaskCategories[id]; !exists {
			return fmt.Errorf("category with id '%s' does not exist", id)
		}

		delete(config.TaskCategories, id)

		if err := s.SaveConfig(config); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}

		fmt.Printf("Category '%s' removed successfully\n", id)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configViewCmd)
	configCmd.AddCommand(configCategoryCmd)
	configCategoryCmd.AddCommand(configCategoryAddCmd)
	configCategoryCmd.AddCommand(configCategoryRemoveCmd)

	configInitCmd.Flags().BoolP("force", "f", false, "Force overwrite existing configuration")
	configViewCmd.Flags().StringP("format", "f", "yaml", "Output format (yaml, json)")
	configCategoryAddCmd.Flags().Float64P("cost", "c", 500, "Cost per time unit")
}
