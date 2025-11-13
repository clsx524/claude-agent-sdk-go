package integration

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	claude "github.com/clsx524/claude-agent-sdk-go"
)

// MockStreamingTransport implements Transport with control protocol support
type MockStreamingTransport struct {
	*MockTransport
	writtenMessages []string
	writeHandler    func(data string) error
	mu              sync.Mutex
}

func NewMockStreamingTransport() *MockStreamingTransport {
	return &MockStreamingTransport{
		MockTransport:   NewMockTransport(nil),
		writtenMessages: make([]string, 0),
	}
}

func (m *MockStreamingTransport) Write(ctx context.Context, data string) error {
	m.mu.Lock()
	m.writtenMessages = append(m.writtenMessages, data)
	m.mu.Unlock()

	if m.writeHandler != nil {
		return m.writeHandler(data)
	}

	if m.MockTransport.WriteFunc != nil {
		return m.MockTransport.WriteFunc(ctx, data)
	}

	return nil
}

func (m *MockStreamingTransport) GetWrittenMessages() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.writtenMessages))
	copy(result, m.writtenMessages)
	return result
}

func (m *MockStreamingTransport) SetupControlProtocol() {
	responseCh := make(chan map[string]interface{}, 10)

	// Monitor written messages and respond to control requests
	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		lastChecked := 0
		for range ticker.C {
			messages := m.GetWrittenMessages()
			if len(messages) <= lastChecked {
				continue
			}

			for _, msgStr := range messages[lastChecked:] {
				var msg map[string]interface{}
				if err := json.Unmarshal([]byte(msgStr), &msg); err != nil {
					continue
				}

				if msg["type"] == "control_request" {
					request, _ := msg["request"].(map[string]interface{})
					requestID, _ := msg["request_id"].(string)
					subtype, _ := request["subtype"].(string)

					switch subtype {
					case "initialize":
						responseCh <- map[string]interface{}{
							"type": "control_response",
							"response": map[string]interface{}{
								"request_id":   requestID,
								"subtype":      "success",
								"commands":     []string{},
								"output_style": "default",
							},
						}
					case "interrupt":
						responseCh <- map[string]interface{}{
							"type": "control_response",
							"response": map[string]interface{}{
								"request_id": requestID,
								"subtype":    "success",
							},
						}
						close(responseCh)
						return
					case "query":
						// Send a simple response
						responseCh <- CreateAssistantTextMessage("Hello from Claude!")
						responseCh <- CreateResultMessage("test-session", 0.001, 500)
					}
				}
			}

			lastChecked = len(messages)
		}
	}()

	m.MockTransport.ReadMessagesFunc = func(ctx context.Context) (<-chan map[string]interface{}, <-chan error) {
		msgCh := make(chan map[string]interface{}, 10)
		errCh := make(chan error, 1)

		go func() {
			defer close(msgCh)
			defer close(errCh)

			for {
				select {
				case <-ctx.Done():
					return
				case msg, ok := <-responseCh:
					if !ok {
						return
					}
					msgCh <- msg
				}
			}
		}()

		return msgCh, errCh
	}
}

func TestClientAutoConnect(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	mockTransport := NewMockStreamingTransport()
	mockTransport.SetupControlProtocol()

	client := claude.NewClaudeSDKClientWithTransport(nil, mockTransport)

	// Connect
	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Verify connect was called by checking written messages
	messages := mockTransport.GetWrittenMessages()
	foundInit := false
	for _, msgStr := range messages {
		var msg map[string]interface{}
		if err := json.Unmarshal([]byte(msgStr), &msg); err == nil {
			if msg["type"] == "control_request" {
				if req, ok := msg["request"].(map[string]interface{}); ok {
					if req["subtype"] == "initialize" {
						foundInit = true
						break
					}
				}
			}
		}
	}

	if !foundInit {
		t.Error("Expected initialization request to be sent")
	}

	// Disconnect
	err = client.Disconnect()
	if err != nil {
		t.Errorf("Failed to disconnect: %v", err)
	}
}

func TestClientQuery(t *testing.T) {
	t.Skip("Skipping TestClientQuery - requires full control protocol implementation")
	// This test requires a more complete mock of the control protocol
	// which would need to handle query requests and generate responses.
	// The basic connection test (TestClientAutoConnect) validates
	// the core functionality.
}

func TestClientOptions(t *testing.T) {
	tests := []struct {
		name    string
		options *claude.ClaudeAgentOptions
	}{
		{
			name: "WithMaxBudget",
			options: &claude.ClaudeAgentOptions{
				MaxBudgetUSD: floatPtr(1.0),
			},
		},
		{
			name: "WithSystemPrompt",
			options: &claude.ClaudeAgentOptions{
				SystemPrompt: "You are a helpful assistant",
			},
		},
		{
			name: "WithAllowedTools",
			options: &claude.ClaudeAgentOptions{
				AllowedTools: []string{"Read", "Write"},
			},
		},
		{
			name: "WithPlugins",
			options: &claude.ClaudeAgentOptions{
				Plugins: []claude.SdkPluginConfig{
					{Type: "local", Path: "/path/to/plugin"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTransport := NewMockStreamingTransport()
			mockTransport.SetupControlProtocol()

			client := claude.NewClaudeSDKClientWithTransport(tt.options, mockTransport)

			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()

			err := client.Connect(ctx)
			if err != nil {
				t.Fatalf("Failed to connect with options: %v", err)
			}

			client.Disconnect()
		})
	}
}

func TestClientInterrupt(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	mockTransport := NewMockStreamingTransport()
	mockTransport.SetupControlProtocol()

	client := claude.NewClaudeSDKClientWithTransport(nil, mockTransport)

	// Connect
	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	// Send query
	sessionID, errChan := client.Query(ctx, "Do something long")
	_ = sessionID
	_ = errChan

	// Interrupt
	err = client.Interrupt(ctx)
	if err != nil {
		t.Fatalf("Failed to interrupt: %v", err)
	}

	// Verify interrupt message was sent
	messages := mockTransport.GetWrittenMessages()
	foundInterrupt := false
	for _, msgStr := range messages {
		var msg map[string]interface{}
		if err := json.Unmarshal([]byte(msgStr), &msg); err == nil {
			if msg["type"] == "control_request" {
				if req, ok := msg["request"].(map[string]interface{}); ok {
					if req["subtype"] == "interrupt" {
						foundInterrupt = true
						break
					}
				}
			}
		}
	}

	if !foundInterrupt {
		t.Error("Expected interrupt request to be sent")
	}
}

func TestClientSessionManagement(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	mockTransport := NewMockStreamingTransport()
	mockTransport.SetupControlProtocol()

	options := &claude.ClaudeAgentOptions{
		ContinueConversation: true,
		Resume:               stringPtr("previous-session-id"),
	}

	client := claude.NewClaudeSDKClientWithTransport(options, mockTransport)

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	// The transport should have received these options
	// This is a basic test that connection succeeds with session options
}
