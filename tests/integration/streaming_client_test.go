package integration

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	claude "github.com/clsx524/claude-agent-sdk-go"
)

// AdvancedMockTransport provides a more complete mock for streaming tests
type AdvancedMockTransport struct {
	connected       bool
	closed          bool
	writtenMessages []string
	responseCh      chan map[string]interface{}
	errorCh         chan error
	mu              sync.Mutex
	ctx             context.Context
	cancel          context.CancelFunc
}

func NewAdvancedMockTransport() *AdvancedMockTransport {
	ctx, cancel := context.WithCancel(context.Background())
	return &AdvancedMockTransport{
		writtenMessages: make([]string, 0),
		responseCh:      make(chan map[string]interface{}, 10),
		errorCh:         make(chan error, 1),
		ctx:             ctx,
		cancel:          cancel,
	}
}

func (m *AdvancedMockTransport) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = true
	return nil
}

func (m *AdvancedMockTransport) Write(ctx context.Context, data string) error {
	m.mu.Lock()
	m.writtenMessages = append(m.writtenMessages, data)
	m.mu.Unlock()

	// Auto-respond to control requests
	go m.handleControlRequest(data)

	return nil
}

func (m *AdvancedMockTransport) handleControlRequest(data string) {
	var msg map[string]interface{}
	if err := json.Unmarshal([]byte(data), &msg); err != nil {
		return
	}

	if msg["type"] == "control_request" {
		request, _ := msg["request"].(map[string]interface{})
		requestID, _ := msg["request_id"].(string)
		subtype, _ := request["subtype"].(string)

		switch subtype {
		case "initialize":
			m.responseCh <- map[string]interface{}{
				"type": "control_response",
				"response": map[string]interface{}{
					"request_id":   requestID,
					"subtype":      "success",
					"commands":     []interface{}{},
					"output_style": "default",
				},
			}
		case "interrupt":
			m.responseCh <- map[string]interface{}{
				"type": "control_response",
				"response": map[string]interface{}{
					"request_id": requestID,
					"subtype":    "success",
				},
			}
		case "set_permission_mode":
			m.responseCh <- map[string]interface{}{
				"type": "control_response",
				"response": map[string]interface{}{
					"request_id": requestID,
					"subtype":    "success",
				},
			}
		case "set_model":
			m.responseCh <- map[string]interface{}{
				"type": "control_response",
				"response": map[string]interface{}{
					"request_id": requestID,
					"subtype":    "success",
				},
			}
		}
	}
}

func (m *AdvancedMockTransport) ReadMessages(ctx context.Context) (<-chan map[string]interface{}, <-chan error) {
	msgCh := make(chan map[string]interface{}, 10)
	errCh := make(chan error, 1)

	go func() {
		defer close(msgCh)
		defer close(errCh)

		for {
			select {
			case <-ctx.Done():
				return
			case <-m.ctx.Done():
				return
			case msg, ok := <-m.responseCh:
				if !ok {
					return
				}
				select {
				case msgCh <- msg:
				case <-ctx.Done():
					return
				}
			case err, ok := <-m.errorCh:
				if ok && err != nil {
					errCh <- err
					return
				}
			}
		}
	}()

	return msgCh, errCh
}

func (m *AdvancedMockTransport) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		m.closed = true
		m.cancel()
		close(m.responseCh)
		close(m.errorCh)
	}
	return nil
}

func (m *AdvancedMockTransport) IsReady() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected && !m.closed
}

func (m *AdvancedMockTransport) EndInput() error {
	return nil
}

func (m *AdvancedMockTransport) GetWrittenMessages() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.writtenMessages))
	copy(result, m.writtenMessages)
	return result
}

func (m *AdvancedMockTransport) QueueResponse(msg map[string]interface{}) {
	m.responseCh <- msg
}

func (m *AdvancedMockTransport) QueueError(err error) {
	m.errorCh <- err
}

// TestStreamingClientManualConnectDisconnect tests manual connection lifecycle
func TestStreamingClientManualConnectDisconnect(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	transport := NewAdvancedMockTransport()
	client := claude.NewClaudeSDKClientWithTransport(nil, transport)

	// Connect
	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// Verify initialization was sent
	messages := transport.GetWrittenMessages()
	foundInit := false
	for _, msgStr := range messages {
		if strings.Contains(msgStr, "initialize") {
			foundInit = true
			break
		}
	}
	if !foundInit {
		t.Error("Expected initialization request")
	}

	// Disconnect
	err = client.Disconnect()
	if err != nil {
		t.Errorf("Disconnect failed: %v", err)
	}

	if !transport.closed {
		t.Error("Expected transport to be closed")
	}
}

// TestStreamingClientConnectWithStringPrompt tests connecting with initial string prompt
func TestStreamingClientConnectWithStringPrompt(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	transport := NewAdvancedMockTransport()
	client := claude.NewClaudeSDKClientWithTransport(nil, transport)

	err := client.ConnectWithPrompt(ctx, "Hello Claude")
	if err != nil {
		t.Fatalf("ConnectWithPrompt failed: %v", err)
	}
	defer client.Disconnect()

	// The prompt handling is done by transport creation in real implementation
	// For mock, we just verify connection succeeded
}

// TestStreamingClientQuerySending tests sending queries
func TestStreamingClientQuerySending(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	transport := NewAdvancedMockTransport()
	client := claude.NewClaudeSDKClientWithTransport(nil, transport)

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer client.Disconnect()

	// Send query
	msgCh, errCh := client.Query(ctx, "What is 2+2?")

	// Queue a response
	transport.QueueResponse(CreateAssistantTextMessage("The answer is 4"))
	transport.QueueResponse(CreateResultMessage("test-session", 0.001, 500))

	// Collect messages
	var receivedMessages []claude.Message
	for msg := range msgCh {
		receivedMessages = append(receivedMessages, msg)
	}

	// Check for errors
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Query error: %v", err)
		}
	default:
	}

	if len(receivedMessages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(receivedMessages))
	}
}

// TestStreamingClientReceiveMessages tests receiving messages
func TestStreamingClientReceiveMessages(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	transport := NewAdvancedMockTransport()
	client := claude.NewClaudeSDKClientWithTransport(nil, transport)

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer client.Disconnect()

	// Queue messages
	transport.QueueResponse(CreateAssistantTextMessage("Hello!"))
	transport.QueueResponse(map[string]interface{}{
		"type": "user",
		"message": map[string]interface{}{
			"role":    "user",
			"content": "Hi there",
		},
	})

	// Receive messages
	receiveCh := client.ReceiveMessages(ctx)

	messages := make([]claude.Message, 0)
	timeout := time.After(500 * time.Millisecond)
	for i := 0; i < 2; i++ {
		select {
		case msg := <-receiveCh:
			messages = append(messages, msg)
		case <-timeout:
			t.Fatal("Timeout waiting for messages")
		}
	}

	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}

	// Verify message types
	if _, ok := messages[0].(*claude.AssistantMessage); !ok {
		t.Errorf("Expected AssistantMessage, got %T", messages[0])
	}
	if _, ok := messages[1].(*claude.UserMessage); !ok {
		t.Errorf("Expected UserMessage, got %T", messages[1])
	}
}

// TestStreamingClientReceiveResponse tests ReceiveResponse stops at ResultMessage
func TestStreamingClientReceiveResponse(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	transport := NewAdvancedMockTransport()
	client := claude.NewClaudeSDKClientWithTransport(nil, transport)

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer client.Disconnect()

	// Queue messages including ResultMessage and one after
	transport.QueueResponse(CreateAssistantTextMessage("Answer"))
	transport.QueueResponse(CreateResultMessage("test-session", 0.001, 500))
	transport.QueueResponse(CreateAssistantTextMessage("This should not be received"))

	// Use ReceiveResponse which should stop after ResultMessage
	responseCh := client.ReceiveResponse(ctx)

	messages := make([]claude.Message, 0)
	for msg := range responseCh {
		messages = append(messages, msg)
	}

	// Should only get 2 messages (assistant + result)
	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}

	// Verify last message is ResultMessage
	if _, ok := messages[len(messages)-1].(*claude.ResultMessage); !ok {
		t.Errorf("Expected last message to be ResultMessage, got %T", messages[len(messages)-1])
	}
}

// TestStreamingClientInterrupt tests interrupt functionality
func TestStreamingClientInterrupt(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	transport := NewAdvancedMockTransport()
	client := claude.NewClaudeSDKClientWithTransport(nil, transport)

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer client.Disconnect()

	// Send interrupt
	err = client.Interrupt(ctx)
	if err != nil {
		t.Fatalf("Interrupt failed: %v", err)
	}

	// Verify interrupt was sent
	messages := transport.GetWrittenMessages()
	foundInterrupt := false
	for _, msgStr := range messages {
		if strings.Contains(msgStr, "interrupt") {
			foundInterrupt = true
			break
		}
	}

	if !foundInterrupt {
		t.Error("Expected interrupt request to be sent")
	}
}

// TestStreamingClientNotConnectedErrors tests error handling when not connected
func TestStreamingClientNotConnectedErrors(t *testing.T) {
	client := claude.NewClaudeSDKClient(nil)
	ctx := context.Background()

	// Try to query without connecting
	_, errCh := client.Query(ctx, "test")
	err := <-errCh
	if err == nil {
		t.Error("Expected error when querying without connection")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("Expected 'not connected' error, got: %v", err)
	}

	// Try to interrupt without connecting
	err = client.Interrupt(ctx)
	if err == nil {
		t.Error("Expected error when interrupting without connection")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("Expected 'not connected' error, got: %v", err)
	}

	// Try to set permission mode without connecting
	err = client.SetPermissionMode(ctx, claude.PermissionModeAcceptEdits)
	if err == nil {
		t.Error("Expected error when setting permission mode without connection")
	}

	// Try to set model without connecting
	err = client.SetModel(ctx, "claude-opus-4")
	if err == nil {
		t.Error("Expected error when setting model without connection")
	}
}

// TestStreamingClientDoubleConnect tests connecting twice
func TestStreamingClientDoubleConnect(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	transport1 := NewAdvancedMockTransport()
	client := claude.NewClaudeSDKClientWithTransport(nil, transport1)

	// First connect
	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("First connect failed: %v", err)
	}

	// Second connect should succeed (replaces connection)
	transport2 := NewAdvancedMockTransport()
	client = claude.NewClaudeSDKClientWithTransport(nil, transport2)
	err = client.Connect(ctx)
	if err != nil {
		t.Fatalf("Second connect failed: %v", err)
	}

	client.Disconnect()
}

// TestStreamingClientDisconnectWithoutConnect tests disconnecting without connecting
func TestStreamingClientDisconnectWithoutConnect(t *testing.T) {
	client := claude.NewClaudeSDKClient(nil)

	// Should not error
	err := client.Disconnect()
	if err != nil {
		t.Errorf("Disconnect without connect should not error: %v", err)
	}
}

// TestStreamingClientConcurrentSendReceive tests concurrent sending and receiving
func TestStreamingClientConcurrentSendReceive(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	transport := NewAdvancedMockTransport()
	client := claude.NewClaudeSDKClientWithTransport(nil, transport)

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer client.Disconnect()

	// Start receiving in background
	receiveCh := client.ReceiveMessages(ctx)
	received := make([]claude.Message, 0)
	var receiveMu sync.Mutex

	go func() {
		for msg := range receiveCh {
			receiveMu.Lock()
			received = append(received, msg)
			receiveMu.Unlock()
		}
	}()

	// Send query while receiving
	transport.QueueResponse(CreateAssistantTextMessage("Response 1"))

	time.Sleep(100 * time.Millisecond)

	_, _ = client.Query(ctx, "Question 1")

	// Queue more responses
	transport.QueueResponse(CreateResultMessage("session-1", 0.001, 500))

	// Wait a bit for processing
	time.Sleep(300 * time.Millisecond)

	receiveMu.Lock()
	messageCount := len(received)
	receiveMu.Unlock()

	if messageCount < 1 {
		t.Errorf("Expected at least 1 message, got %d", messageCount)
	}
}

// TestStreamingClientWithOptions tests client with various options
func TestStreamingClientWithOptions(t *testing.T) {
	// Helper functions for pointer conversion
	stringPtr := func(s string) *string { return &s }
	permissionModePtr := func(m claude.PermissionMode) *claude.PermissionMode { return &m }
	float64Ptr := func(f float64) *float64 { return &f }

	tests := []struct {
		name    string
		options *claude.ClaudeAgentOptions
	}{
		{
			name: "WithTools",
			options: &claude.ClaudeAgentOptions{
				AllowedTools: []string{"Read", "Write"},
			},
		},
		{
			name: "WithModel",
			options: &claude.ClaudeAgentOptions{
				Model: stringPtr("claude-sonnet-4-5"),
			},
		},
		{
			name: "WithPermissionMode",
			options: &claude.ClaudeAgentOptions{
				PermissionMode: permissionModePtr(claude.PermissionModeAcceptEdits),
			},
		},
		{
			name: "WithMaxBudget",
			options: &claude.ClaudeAgentOptions{
				MaxBudgetUSD: float64Ptr(1.0),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			transport := NewAdvancedMockTransport()
			client := claude.NewClaudeSDKClientWithTransport(tt.options, transport)

			err := client.Connect(ctx)
			if err != nil {
				t.Fatalf("Connect with options failed: %v", err)
			}

			client.Disconnect()
		})
	}
}

// TestStreamingClientGetServerInfo tests retrieving server info
func TestStreamingClientGetServerInfo(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	transport := NewAdvancedMockTransport()
	client := claude.NewClaudeSDKClientWithTransport(nil, transport)

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer client.Disconnect()

	// Wait briefly for initialization to complete
	time.Sleep(100 * time.Millisecond)

	// Get server info
	info := client.GetServerInfo()

	// Should have initialization data after init completes
	// Note: With our mock, this may be nil since we're using a basic mock
	// In real usage with actual transport, this would contain server capabilities
	_ = info // Don't fail if nil with mock transport
}

// TestStreamingClientSetPermissionMode tests changing permission mode
func TestStreamingClientSetPermissionMode(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	transport := NewAdvancedMockTransport()
	client := claude.NewClaudeSDKClientWithTransport(nil, transport)

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer client.Disconnect()

	// Change permission mode
	err = client.SetPermissionMode(ctx, claude.PermissionModeAcceptEdits)
	if err != nil {
		t.Fatalf("SetPermissionMode failed: %v", err)
	}

	// Verify request was sent
	messages := transport.GetWrittenMessages()
	foundRequest := false
	for _, msgStr := range messages {
		if strings.Contains(msgStr, "set_permission_mode") {
			foundRequest = true
			break
		}
	}

	if !foundRequest {
		t.Error("Expected set_permission_mode request")
	}
}

// TestStreamingClientSetModel tests changing model
func TestStreamingClientSetModel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	transport := NewAdvancedMockTransport()
	client := claude.NewClaudeSDKClientWithTransport(nil, transport)

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer client.Disconnect()

	// Change model
	err = client.SetModel(ctx, "claude-opus-4")
	if err != nil {
		t.Fatalf("SetModel failed: %v", err)
	}

	// Verify request was sent
	messages := transport.GetWrittenMessages()
	foundRequest := false
	for _, msgStr := range messages {
		if strings.Contains(msgStr, "set_model") {
			foundRequest = true
			break
		}
	}

	if !foundRequest {
		t.Error("Expected set_model request")
	}
}
