package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/bluefunda/cai-cli/internal/api"
	"github.com/bluefunda/cai-cli/internal/config"
)

// StartSimpleChat starts a simple interactive chat (no TUI)
func StartSimpleChat(cfg *config.Config, chatID, model string) error {
	client := api.NewClient(cfg)
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("CAI Chat - Session: %s\n", chatID[:8])
	fmt.Printf("Model: %s\n", model)
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println("Type your message and press Enter. Type 'exit' to quit.\n")

	for {
		fmt.Print("You: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		if strings.ToLower(input) == "exit" || strings.ToLower(input) == "quit" {
			fmt.Println("Goodbye!")
			return nil
		}

		req := &api.ChatRequest{
			Model: model,
			Messages: []api.Message{
				{Role: "user", Content: input},
			},
		}

		fmt.Print("\nAI: ")
		err = client.SendMessage(chatID, req, func(event *api.StreamEvent) {
			if event.Type == "content" {
				fmt.Print(event.Content)
			} else if event.Type == "error" {
				fmt.Printf("\n[Error: %s]", event.Error)
			}
		})
		fmt.Println("\n")

		if err != nil {
			fmt.Printf("Error: %v\n\n", err)
		}
	}
}
