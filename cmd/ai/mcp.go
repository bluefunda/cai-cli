package ai

import (
	"fmt"
	"strings"

	"github.com/bluefunda/cai-cli/internal/api"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Manage MCP servers",
	Long:  `List and manage Model Context Protocol (MCP) servers.`,
}

var mcpListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available MCP servers",
	Long:  `Display all available MCP servers.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := RequireAuth(); err != nil {
			return err
		}

		client := api.NewClient(cfg)
		mcp, err := client.GetMCPServers()
		if err != nil {
			return err
		}

		if len(mcp.Servers) == 0 {
			fmt.Println("No MCP servers available.")
			return nil
		}

		fmt.Println("Available MCP Servers:")
		fmt.Println(strings.Repeat("-", 60))
		for _, server := range mcp.Servers {
			fmt.Printf("  %s\n", server.Name)
			if server.Description != "" {
				fmt.Printf("    %s\n", server.Description)
			}
			fmt.Printf("    ID: %s\n\n", server.ID)
		}

		return nil
	},
}

var mcpUserCmd = &cobra.Command{
	Use:   "user",
	Short: "List your MCP subscriptions",
	Long:  `Display MCP servers you are subscribed to.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := RequireAuth(); err != nil {
			return err
		}

		client := api.NewClient(cfg)
		mcp, err := client.GetUserMCPSubscriptions()
		if err != nil {
			return err
		}

		if len(mcp.Servers) == 0 {
			fmt.Println("You are not subscribed to any MCP servers.")
			fmt.Println("Use 'ai mcp select <server-id>' to subscribe.")
			return nil
		}

		fmt.Println("Your MCP Subscriptions:")
		fmt.Println(strings.Repeat("-", 60))
		for _, server := range mcp.Servers {
			fmt.Printf("  %s (%s)\n", server.Name, server.ID)
		}

		return nil
	},
}

var mcpSelectCmd = &cobra.Command{
	Use:   "select [server-id]",
	Short: "Select an MCP server",
	Long:  `Subscribe to an MCP server to use its tools in chat.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := RequireAuth(); err != nil {
			return err
		}

		serverID := args[0]
		client := api.NewClient(cfg)
		if err := client.SelectMCPServer(serverID); err != nil {
			return err
		}

		fmt.Printf("Successfully selected MCP server: %s\n", serverID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
	mcpCmd.AddCommand(mcpListCmd)
	mcpCmd.AddCommand(mcpUserCmd)
	mcpCmd.AddCommand(mcpSelectCmd)
}
