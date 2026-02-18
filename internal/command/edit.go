package command

import (
	"fmt"

	"github.com/bornholm/guesstimate/internal/ui"
	"github.com/spf13/cobra"
)

// editCmd represents the edit command
var editCmd = &cobra.Command{
	Use:   "edit <file>",
	Short: "Edit an estimation interactively",
	Long:  `Open an interactive terminal UI to edit an estimation file.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		file := args[0]

		s := getStore()

		// Load or create estimation
		estimation, created, err := s.LoadOrCreateEstimation(file, file)
		if err != nil {
			return fmt.Errorf("failed to load estimation: %w", err)
		}
		if created {
			fmt.Printf("Created new estimation file: %s\n", file)
		}

		// Load config
		config, err := s.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Create and run UI
		app := ui.NewApp(s, config, estimation, file)
		if err := app.Run(); err != nil {
			return fmt.Errorf("failed to run UI: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(editCmd)
}
