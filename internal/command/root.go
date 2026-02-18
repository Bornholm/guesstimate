package command

import (
	"fmt"
	"os"

	"github.com/bornholm/guesstimate/internal/store"
	"github.com/spf13/cobra"
)

var (
	configFile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "guesstimate",
	Short: "A CLI tool for 3-point estimation management",
	Long: `Guesstimate is a CLI tool for creating and managing 3-point estimations.

It allows you to:
- Create new estimation projects
- Add and manage tasks with optimistic, likely, and pessimistic estimates
- View estimation results with confidence intervals
- Generate markdown reports

Use "guesstimate [command] --help" for more information about a command.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", store.DefaultConfigFile, "configuration file path")
}

// getStore creates a new YAML store with the configured file
func getStore() *store.YAMLStore {
	return store.NewYAMLStore(configFile)
}
