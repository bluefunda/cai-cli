package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bluefunda/cai-cli/internal/config"
)

// Client handles API requests to the CAI gateway
type Client struct {
	cfg        *config.Config
	httpClient *http.Client
}

// NewClient creates a new API client
func NewClient(cfg *config.Config) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Model represents an LLM model
type Model struct {
	Name    string `json:"name"`
	ModelID int    `json:"modelId"`
	Object  string `json:"object"`
	OwnedBy string `json:"ownedBy"`
}

// ModelsResponse represents the response from /models
type ModelsResponse struct {
	LLMInfo []Model `json:"llmInfo"`
}

// Chat represents a chat session
type Chat struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// ChatsResponse represents the response from /chats
type ChatsResponse struct {
	Chats []Chat `json:"chats"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatHistoryResponse represents the response from /chats/{id}/messages
type ChatHistoryResponse struct {
	ChatHistory []Message `json:"chatHistory"`
}

// ChatRequest represents a request to send a chat message
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// MCPServer represents an MCP server
type MCPServer struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// MCPResponse represents the response from /mcp
type MCPResponse struct {
	Servers []MCPServer `json:"servers"`
}

// UserInfo represents user information
type UserInfo struct {
	Sub               string `json:"sub"`
	Name              string `json:"name"`
	Email             string `json:"email"`
	PreferredUsername string `json:"preferred_username"`
}

// StreamEvent represents a server-sent event
type StreamEvent struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	Error   string `json:"error,omitempty"`
}

// doRequest performs an authenticated HTTP request
func (c *Client) doRequest(method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	url := c.cfg.APIBaseURL + path
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.cfg.AccessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}

// Health checks the API health
func (c *Client) Health() error {
	resp, err := http.Get(c.cfg.APIBaseURL + "/health")
	if err != nil {
		return fmt.Errorf("failed to check health: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed: status %d", resp.StatusCode)
	}
	return nil
}

// GetUserInfo retrieves the current user's information
func (c *Client) GetUserInfo() (*UserInfo, error) {
	resp, err := c.doRequest("GET", "/userinfo", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user info: %s", string(body))
	}

	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &userInfo, nil
}

// GetModels retrieves available LLM models
func (c *Client) GetModels() (*ModelsResponse, error) {
	resp, err := c.doRequest("GET", "/models", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get models: %s", string(body))
	}

	var models ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &models, nil
}

// GetChats retrieves all chat sessions for the user
func (c *Client) GetChats() (*ChatsResponse, error) {
	resp, err := c.doRequest("GET", "/chats", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get chats: %s", string(body))
	}

	var chats ChatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&chats); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &chats, nil
}

// GetChatHistory retrieves messages for a specific chat
func (c *Client) GetChatHistory(chatID string) (*ChatHistoryResponse, error) {
	resp, err := c.doRequest("GET", "/chats/"+chatID+"/messages", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get chat history: %s", string(body))
	}

	var history ChatHistoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&history); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &history, nil
}

// GetChatContext retrieves context for a specific chat
func (c *Client) GetChatContext(chatID string) (map[string]interface{}, error) {
	resp, err := c.doRequest("GET", "/chats/"+chatID+"/context", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get chat context: %s", string(body))
	}

	var context map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&context); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return context, nil
}

// SendMessage sends a message and streams the response
func (c *Client) SendMessage(chatID string, req *ChatRequest, eventHandler func(*StreamEvent)) error {
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.cfg.APIBaseURL + "/chats/" + chatID
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.cfg.AccessToken)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	// Use a client without timeout for streaming
	streamClient := &http.Client{}
	resp, err := streamClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to send message: %s", string(body))
	}

	// Read SSE stream
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}

		// SSE format: "data: {...}"
		if len(line) > 6 && line[:6] == "data: " {
			var event StreamEvent
			if err := json.Unmarshal([]byte(line[6:]), &event); err != nil {
				continue
			}
			eventHandler(&event)

			if event.Type == "done" || event.Type == "error" {
				break
			}
		}
	}

	return scanner.Err()
}

// StopChat stops an ongoing chat stream
func (c *Client) StopChat(chatID string) error {
	resp, err := c.doRequest("POST", "/chats/"+chatID+"/stop", map[string]interface{}{})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to stop chat: %s", string(body))
	}

	return nil
}

// GetMCPServers retrieves available MCP servers
func (c *Client) GetMCPServers() (*MCPResponse, error) {
	resp, err := c.doRequest("GET", "/mcp", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get MCP servers: %s", string(body))
	}

	var mcp MCPResponse
	if err := json.NewDecoder(resp.Body).Decode(&mcp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &mcp, nil
}

// GetUserMCPSubscriptions retrieves user's MCP subscriptions
func (c *Client) GetUserMCPSubscriptions() (*MCPResponse, error) {
	resp, err := c.doRequest("GET", "/mcp/user", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user MCP subscriptions: %s", string(body))
	}

	var mcp MCPResponse
	if err := json.NewDecoder(resp.Body).Decode(&mcp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &mcp, nil
}

// SelectMCPServer selects an MCP server for the user
func (c *Client) SelectMCPServer(serverID string) error {
	resp, err := c.doRequest("POST", "/mcp/select", map[string]string{"server_id": serverID})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to select MCP server: %s", string(body))
	}

	return nil
}

// GetSettings retrieves user settings
func (c *Client) GetSettings() (map[string]interface{}, error) {
	resp, err := c.doRequest("GET", "/settings", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get settings: %s", string(body))
	}

	var settings map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return settings, nil
}

// GetRateLimit retrieves rate limit status
func (c *Client) GetRateLimit() (map[string]interface{}, error) {
	resp, err := c.doRequest("GET", "/rate-limit", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get rate limit: %s", string(body))
	}

	var rateLimit map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rateLimit); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return rateLimit, nil
}
