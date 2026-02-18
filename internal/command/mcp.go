package command

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bornholm/guesstimate/internal/mcp"
	"github.com/spf13/cobra"
)

var (
	mcpRootDir string
)

// mcpCmd represents the mcp command
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "MCP server management commands",
	Long:  `Manage MCP server for LLM integration.`,
}

func init() {
	rootCmd.AddCommand(mcpCmd)
	mcpCmd.AddCommand(mcpServerCmd)
	mcpServerCmd.Flags().StringVar(&mcpRootDir, "root", "", "Root directory for the MCP server (default: current working directory)")
}

// mcpServerCmd represents the mcp server command
var mcpServerCmd = &cobra.Command{
	Use:   "server",
	Short: "Run the MCP server",
	Long:  `Run the MCP server with specified configuration. The server uses stdio transport for communication.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		rootDir := mcpRootDir
		if rootDir == "" {
			var err error
			rootDir, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current working directory: %w", err)
			}
		}

		// Load configuration from the global config file (outside chroot)
		store := getStore()
		config, err := store.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Create the MCP server with the loaded config
		server, err := mcp.NewServer(&mcp.ServerOptions{
			RootDir: rootDir,
			Config:  config,
		})
		if err != nil {
			return fmt.Errorf("failed to create MCP server: %w", err)
		}
		defer server.Close()

		// Set up context with cancellation for graceful shutdown
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Handle signals for graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-sigChan
			cancel()
		}()

		// Run the server
		if err := server.Run(ctx); err != nil {
			return fmt.Errorf("MCP server error: %w", err)
		}

		return nil
	},
}
