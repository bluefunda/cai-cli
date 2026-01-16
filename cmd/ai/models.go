package ai

import (
	"fmt"
	"strings"

	"github.com/bluefunda/cai-cli/internal/api"
	"github.com/spf13/cobra"
)

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Manage AI models",
	Long:  `List and manage available AI models.`,
}

var modelsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available models",
	Long:  `Display all available AI models.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := RequireAuth(); err != nil {
			return err
		}

		client := api.NewClient(cfg)
		models, err := client.GetModels()
		if err != nil {
			return err
		}

		if len(models.LLMInfo) == 0 {
			fmt.Println("No models available.")
			return nil
		}

		fmt.Println("Available Models:")
		fmt.Println(strings.Repeat("-", 50))
		fmt.Printf("%-15s %-20s %s\n", "NAME", "OWNED BY", "ID")
		fmt.Println(strings.Repeat("-", 50))
		for _, model := range models.LLMInfo {
			fmt.Printf("%-15s %-20s %d\n", model.Name, model.OwnedBy, model.ModelID)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(modelsCmd)
	modelsCmd.AddCommand(modelsListCmd)
}
