package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	pb "github.com/bluefunda/cai-cli/api/proto/bff"
	caigrpc "github.com/bluefunda/cai-cli/internal/grpc"
	"github.com/bluefunda/cai-cli/internal/ui"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Chat operations",
}

// --- chat list ---

var chatListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all chat sessions",
	RunE:  runChatList,
}

func runChatList(cmd *cobra.Command, args []string) error {
	conn, cfg, err := bffConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx, cancel := caigrpc.ContextWithTimeout()
	defer cancel()

	resp, err := conn.Client.GetChatIds(ctx, &pb.GetChatIdsRequest{})
	if err != nil {
		return fmt.Errorf("get chats: %w", err)
	}

	p := printer(cfg)
	if p.Format == ui.FormatJSON {
		p.ProtoJSON(resp)
		return nil
	}

	headers := []string{"CHAT_ID", "TITLE", "MODEL", "CREATED"}
	rows := make([][]string, 0, len(resp.GetChats()))
	for _, c := range resp.GetChats() {
		rows = append(rows, []string{
			c.GetChatId(),
			truncate(c.GetChatTitle(), 40),
			c.GetModel(),
			c.GetCreatedAt(),
		})
	}
	p.Table(headers, rows)
	return nil
}

// --- chat start (interactive REPL) ---

var (
	chatModel     string
	chatNew       bool
	chatMCPServer string
)

var chatStartCmd = &cobra.Command{
	Use:   "start [chatId]",
	Short: "Start an interactive chat session with gRPC streaming",
	Long:  "Start an interactive chat REPL. Use --new to create a new chat, or pass an existing chat ID to continue.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runChatStart,
}

func init() {
	chatStartCmd.Flags().StringVar(&chatModel, "model", "", "LLM model to use")
	chatStartCmd.Flags().BoolVar(&chatNew, "new", false, "Force new chat (generate UUID)")
	chatStartCmd.Flags().StringVar(&chatMCPServer, "mcp-server", "", "MCP server name")

	chatCmd.AddCommand(chatListCmd, chatStartCmd, chatHistoryCmd, chatContextCmd, chatTitleCmd, chatStopCmd)
}

func runChatStart(cmd *cobra.Command, args []string) error {
	conn, cfg, err := bffConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	model := chatModel
	if model == "" {
		model = cfg.Defaults.Model
	}
	if model == "" {
		model = "openai"
	}

	var chatID string
	isNewChat := true

	if len(args) > 0 && !chatNew {
		chatID = args[0]
		isNewChat = false
	} else {
		chatID = uuid.New().String()
	}

	p := printer(cfg)
	p.Info(fmt.Sprintf("Chat %s (model: %s)", chatID, model))
	p.Info("Type /exit or Ctrl+D to quit. Use \\ at end of line for multiline input.")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for {
		fmt.Print("You> ")
		var lines []string
		gotInput := false

		for scanner.Scan() {
			gotInput = true
			line := scanner.Text()
			if strings.HasSuffix(line, "\\") {
				lines = append(lines, strings.TrimSuffix(line, "\\"))
				fmt.Print("...> ")
				continue
			}
			lines = append(lines, line)
			break
		}

		if err := scanner.Err(); err != nil {
			return fmt.Errorf("read input: %w", err)
		}
		if !gotInput {
			fmt.Println()
			return nil
		}

		input := strings.TrimSpace(strings.Join(lines, "\n"))
		if input == "" {
			continue
		}
		if input == "/exit" {
			return nil
		}

		req := &pb.ChatRequest{
			ChatId:        chatID,
			Prompt:        input,
			Model:         model,
			IsNewChat:     isNewChat,
			McpServerName: chatMCPServer,
		}

		ctx, cancel := context.WithCancel(context.Background())
		stream, err := conn.Client.Chat(ctx, req)
		if err != nil {
			cancel()
			p.Error("Request failed: " + err.Error())
			continue
		}

		if err := ui.RenderGRPCStream(stream, cancel); err != nil {
			p.Error("Stream error: " + err.Error())
		}

		isNewChat = false
		fmt.Println()
	}
}

// --- chat history ---

var chatHistoryCmd = &cobra.Command{
	Use:   "history <chatId>",
	Short: "Get message history for a chat",
	Args:  cobra.ExactArgs(1),
	RunE:  runChatHistory,
}

func runChatHistory(cmd *cobra.Command, args []string) error {
	conn, cfg, err := bffConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx, cancel := caigrpc.ContextWithTimeout()
	defer cancel()

	resp, err := conn.Client.GetChatHistory(ctx, &pb.GetChatHistoryRequest{ChatId: args[0]})
	if err != nil {
		return fmt.Errorf("get history: %w", err)
	}

	p := printer(cfg)
	if p.Format == ui.FormatJSON {
		p.ProtoJSON(resp)
		return nil
	}

	headers := []string{"ROLE", "CONTENT", "CREATED"}
	rows := make([][]string, 0, len(resp.GetMessages()))
	for _, m := range resp.GetMessages() {
		rows = append(rows, []string{
			m.GetRole(),
			truncate(m.GetContent(), 80),
			m.GetCreatedAt(),
		})
	}
	p.Table(headers, rows)
	return nil
}

// --- chat context ---

var chatContextCmd = &cobra.Command{
	Use:   "context <chatId>",
	Short: "Get context for a chat",
	Args:  cobra.ExactArgs(1),
	RunE:  runChatContext,
}

func runChatContext(cmd *cobra.Command, args []string) error {
	conn, cfg, err := bffConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx, cancel := caigrpc.ContextWithTimeout()
	defer cancel()

	resp, err := conn.Client.GetChatContext(ctx, &pb.GetChatContextRequest{ChatId: args[0]})
	if err != nil {
		return fmt.Errorf("get context: %w", err)
	}

	printer(cfg).ProtoJSON(resp)
	return nil
}

// --- chat title ---

var chatTitlePrompt string

var chatTitleCmd = &cobra.Command{
	Use:   "title <chatId>",
	Short: "Generate a title for a chat",
	Args:  cobra.ExactArgs(1),
	RunE:  runChatTitle,
}

func init() {
	chatTitleCmd.Flags().StringVar(&chatTitlePrompt, "prompt", "", "Prompt hint for title generation")
}

func runChatTitle(cmd *cobra.Command, args []string) error {
	conn, cfg, err := bffConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx, cancel := caigrpc.ContextWithTimeout()
	defer cancel()

	resp, err := conn.Client.GenerateTitle(ctx, &pb.GenerateTitleRequest{
		ChatId: args[0],
		Prompt: chatTitlePrompt,
	})
	if err != nil {
		return fmt.Errorf("generate title: %w", err)
	}

	if resp.GetError() != "" {
		return fmt.Errorf("title generation: %s", resp.GetError())
	}

	p := printer(cfg)
	if p.Format == ui.FormatJSON {
		p.ProtoJSON(resp)
	} else {
		p.Success(resp.GetGeneratedTitle())
	}
	return nil
}

// --- chat stop ---

var chatStopCmd = &cobra.Command{
	Use:   "stop <chatId>",
	Short: "Stop a streaming chat",
	Args:  cobra.ExactArgs(1),
	RunE:  runChatStop,
}

func runChatStop(cmd *cobra.Command, args []string) error {
	conn, cfg, err := bffConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx, cancel := caigrpc.ContextWithTimeout()
	defer cancel()

	resp, err := conn.Client.StopChat(ctx, &pb.StopChatRequest{ChatId: args[0]})
	if err != nil {
		return fmt.Errorf("stop chat: %w", err)
	}

	p := printer(cfg)
	if resp.GetSuccess() {
		p.Success("Chat stopped")
	} else {
		p.Error("Failed to stop chat")
	}
	return nil
}

// --- helpers ---

func truncate(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
