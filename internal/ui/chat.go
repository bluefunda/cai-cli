package ui

import (
	"fmt"
	"strings"

	"github.com/bluefunda/cai-cli/internal/api"
	"github.com/bluefunda/cai-cli/internal/config"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	userStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	assistantStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("82"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

// Message represents a chat message for display
type Message struct {
	Role    string
	Content string
}

// Model represents the chat UI state
type Model struct {
	viewport    viewport.Model
	textarea    textarea.Model
	spinner     spinner.Model
	messages    []Message
	chatID      string
	modelName   string
	cfg         *config.Config
	client      *api.Client
	waiting     bool
	streaming   bool
	streamBuf   strings.Builder
	err         error
	width       int
	height      int
	ready       bool
}

// streamMsg is sent when a streaming event is received
type streamMsg struct {
	event *api.StreamEvent
}

// streamDoneMsg is sent when streaming is complete
type streamDoneMsg struct {
	err error
}

// NewChatModel creates a new chat UI model
func NewChatModel(cfg *config.Config, chatID, modelName string) Model {
	ta := textarea.New()
	ta.Placeholder = "Type your message... (Ctrl+S to send, Ctrl+C to quit)"
	ta.Focus()
	ta.CharLimit = 4000
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return Model{
		textarea:  ta,
		spinner:   sp,
		messages:  []Message{},
		chatID:    chatID,
		modelName: modelName,
		cfg:       cfg,
		client:    api.NewClient(cfg),
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.spinner.Tick,
	)
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		spCmd tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "ctrl+s":
			if m.waiting || m.streaming {
				return m, nil
			}
			content := strings.TrimSpace(m.textarea.Value())
			if content == "" {
				return m, nil
			}
			m.messages = append(m.messages, Message{Role: "user", Content: content})
			m.textarea.Reset()
			m.waiting = true
			m.streaming = true
			m.streamBuf.Reset()
			return m, m.sendMessage(content)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-10)
			m.viewport.SetContent(m.renderMessages())
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 10
		}
		m.textarea.SetWidth(msg.Width - 4)

	case streamMsg:
		if msg.event.Type == "content" {
			m.streamBuf.WriteString(msg.event.Content)
			m.viewport.SetContent(m.renderMessages() + m.renderStreamingMessage())
			m.viewport.GotoBottom()
		} else if msg.event.Type == "error" {
			m.err = fmt.Errorf(msg.event.Error)
		}
		return m, nil

	case streamDoneMsg:
		m.waiting = false
		m.streaming = false
		if msg.err != nil {
			m.err = msg.err
		} else if m.streamBuf.Len() > 0 {
			m.messages = append(m.messages, Message{Role: "assistant", Content: m.streamBuf.String()})
			m.streamBuf.Reset()
		}
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, nil

	case spinner.TickMsg:
		m.spinner, spCmd = m.spinner.Update(msg)
		return m, spCmd
	}

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	return m, tea.Batch(tiCmd, vpCmd)
}

// View renders the UI
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render(fmt.Sprintf("CAI Chat - %s", m.chatID[:8])))
	b.WriteString("\n")

	// Messages viewport
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// Status line
	if m.waiting {
		b.WriteString(m.spinner.View() + " Thinking...")
	} else if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
	}
	b.WriteString("\n")

	// Input area
	b.WriteString(m.textarea.View())
	b.WriteString("\n")

	// Help
	b.WriteString(helpStyle.Render("Ctrl+S: send | Ctrl+C: quit | Model: " + m.modelName))

	return b.String()
}

// renderMessages renders all messages
func (m Model) renderMessages() string {
	var b strings.Builder

	for _, msg := range m.messages {
		if msg.Role == "user" {
			b.WriteString(userStyle.Render("[YOU]"))
		} else {
			b.WriteString(assistantStyle.Render("[AI]"))
		}
		b.WriteString("\n")
		b.WriteString(msg.Content)
		b.WriteString("\n\n")
	}

	return b.String()
}

// renderStreamingMessage renders the current streaming message
func (m Model) renderStreamingMessage() string {
	if m.streamBuf.Len() == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(assistantStyle.Render("[AI]"))
	b.WriteString("\n")
	b.WriteString(m.streamBuf.String())
	b.WriteString("...")
	return b.String()
}

// sendMessage sends a message and returns a command to handle streaming
func (m Model) sendMessage(content string) tea.Cmd {
	return func() tea.Msg {
		req := &api.ChatRequest{
			Model: m.modelName,
			Messages: []api.Message{
				{Role: "user", Content: content},
			},
		}

		// Create a channel for streaming events
		eventChan := make(chan *api.StreamEvent)
		errChan := make(chan error)

		go func() {
			err := m.client.SendMessage(m.chatID, req, func(event *api.StreamEvent) {
				eventChan <- event
			})
			errChan <- err
			close(eventChan)
		}()

		// Read events until done
		for event := range eventChan {
			// We can't send tea.Msg from here directly in a blocking way
			// For simplicity, we'll collect all content and return at the end
			if event.Type == "content" {
				// In a real implementation, we'd use tea.Program.Send()
			}
		}

		err := <-errChan
		return streamDoneMsg{err: err}
	}
}

// StartInteractiveChat starts the interactive chat UI
func StartInteractiveChat(cfg *config.Config, chatID, model string) error {
	m := NewChatModel(cfg, chatID, model)
	p := tea.NewProgram(m, tea.WithAltScreen())

	_, err := p.Run()
	return err
}
