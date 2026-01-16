package ai

import (
	"fmt"
	"os"
	"strings"

	"github.com/bluefunda/cai-cli/internal/api"
	"github.com/bluefunda/cai-cli/internal/ui"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Chat with AI",
	Long:  `Start an interactive chat session or manage chat history.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := RequireAuth(); err != nil {
			return err
		}

		// Start interactive chat
		chatID, _ := cmd.Flags().GetString("id")
		if chatID == "" {
			chatID = uuid.New().String()
		}

		model, _ := cmd.Flags().GetString("model")
		if model == "" {
			model = "openai"
		}

		// Use simple chat by default, TUI with --tui flag
		tui, _ := cmd.Flags().GetBool("tui")
		if tui {
			return ui.StartInteractiveChat(cfg, chatID, model)
		}
		return ui.StartSimpleChat(cfg, chatID, model)
	},
}

var chatListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all chat sessions",
	Long:  `Display all your chat sessions.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := RequireAuth(); err != nil {
			return err
		}

		client := api.NewClient(cfg)
		chats, err := client.GetChats()
		if err != nil {
			return err
		}

		if len(chats.Chats) == 0 {
			fmt.Println("No chat sessions found.")
			return nil
		}

		fmt.Println("Chat Sessions:")
		fmt.Println(strings.Repeat("-", 60))
		for _, chat := range chats.Chats {
			title := chat.Title
			if title == "" {
				title = "(untitled)"
			}
			fmt.Printf("  %s  %s\n", chat.ID[:8], title)
		}

		return nil
	},
}

var chatHistoryCmd = &cobra.Command{
	Use:   "history [chat-id]",
	Short: "View chat history",
	Long:  `Display the message history for a specific chat session.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := RequireAuth(); err != nil {
			return err
		}

		chatID := args[0]
		client := api.NewClient(cfg)
		history, err := client.GetChatHistory(chatID)
		if err != nil {
			return err
		}

		if len(history.ChatHistory) == 0 {
			fmt.Println("No messages in this chat.")
			return nil
		}

		fmt.Printf("Chat History (%s):\n", chatID[:8])
		fmt.Println(strings.Repeat("-", 60))
		for _, msg := range history.ChatHistory {
			role := strings.ToUpper(msg.Role)
			fmt.Printf("[%s]\n%s\n\n", role, msg.Content)
		}

		return nil
	},
}

var chatNewCmd = &cobra.Command{
	Use:   "new",
	Short: "Start a new chat session",
	Long:  `Create a new chat session and optionally send an initial message.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := RequireAuth(); err != nil {
			return err
		}

		chatID := uuid.New().String()
		model, _ := cmd.Flags().GetString("model")
		if model == "" {
			model = "openai"
		}

		message, _ := cmd.Flags().GetString("message")
		if message != "" {
			// Send message directly
			client := api.NewClient(cfg)
			req := &api.ChatRequest{
				Model: model,
				Messages: []api.Message{
					{Role: "user", Content: message},
				},
			}

			fmt.Printf("Chat ID: %s\n", chatID)
			fmt.Println(strings.Repeat("-", 60))
			fmt.Printf("[USER]\n%s\n\n[ASSISTANT]\n", message)

			err := client.SendMessage(chatID, req, func(event *api.StreamEvent) {
				if event.Type == "content" {
					fmt.Print(event.Content)
				} else if event.Type == "error" {
					fmt.Fprintf(os.Stderr, "\nError: %s\n", event.Error)
				}
			})
			fmt.Println()
			return err
		}

		// Start interactive mode
		tui, _ := cmd.Flags().GetBool("tui")
		if tui {
			return ui.StartInteractiveChat(cfg, chatID, model)
		}
		return ui.StartSimpleChat(cfg, chatID, model)
	},
}

var chatSendCmd = &cobra.Command{
	Use:   "send [chat-id] [message]",
	Short: "Send a message to a chat",
	Long:  `Send a message to an existing chat session.`,
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := RequireAuth(); err != nil {
			return err
		}

		chatID := args[0]
		message := strings.Join(args[1:], " ")
		model, _ := cmd.Flags().GetString("model")
		if model == "" {
			model = "openai"
		}

		client := api.NewClient(cfg)
		req := &api.ChatRequest{
			Model: model,
			Messages: []api.Message{
				{Role: "user", Content: message},
			},
		}

		fmt.Printf("[ASSISTANT]\n")
		err := client.SendMessage(chatID, req, func(event *api.StreamEvent) {
			if event.Type == "content" {
				fmt.Print(event.Content)
			} else if event.Type == "error" {
				fmt.Fprintf(os.Stderr, "\nError: %s\n", event.Error)
			}
		})
		fmt.Println()
		return err
	},
}

var chatContextCmd = &cobra.Command{
	Use:   "context [chat-id]",
	Short: "View chat context",
	Long:  `Display the context for a specific chat session.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := RequireAuth(); err != nil {
			return err
		}

		chatID := args[0]
		client := api.NewClient(cfg)
		context, err := client.GetChatContext(chatID)
		if err != nil {
			return err
		}

		fmt.Printf("Chat Context (%s):\n", chatID[:8])
		fmt.Println(strings.Repeat("-", 60))
		for key, value := range context {
			fmt.Printf("  %s: %v\n", key, value)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(chatCmd)
	chatCmd.AddCommand(chatListCmd)
	chatCmd.AddCommand(chatHistoryCmd)
	chatCmd.AddCommand(chatNewCmd)
	chatCmd.AddCommand(chatSendCmd)
	chatCmd.AddCommand(chatContextCmd)

	chatCmd.Flags().StringP("id", "i", "", "Chat ID to resume")
	chatCmd.Flags().StringP("model", "m", "openai", "Model to use")
	chatCmd.Flags().Bool("tui", false, "Use TUI mode (experimental)")

	chatNewCmd.Flags().StringP("model", "m", "openai", "Model to use")
	chatNewCmd.Flags().Bool("tui", false, "Use TUI mode (experimental)")
	chatNewCmd.Flags().String("message", "", "Initial message to send")

	chatSendCmd.Flags().StringP("model", "m", "openai", "Model to use")
}
