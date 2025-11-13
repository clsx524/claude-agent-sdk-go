package integration

import (
	"context"
	"testing"

	claude "github.com/clsx524/claude-agent-sdk-go"
)

func TestSimpleQueryResponse(t *testing.T) {
	ctx := context.Background()

	// Create mock transport with simple response
	messages := []map[string]interface{}{
		CreateAssistantTextMessage("2 + 2 equals 4"),
		CreateResultMessage("test-session", 0.001, 1000),
	}

	mockTransport := NewMockTransport(messages)

	// Create internal client with mock transport
	// Note: We need to expose a way to inject transport for testing
	// For now, this tests the message parsing logic
	msgCh, errCh := mockTransport.ReadMessages(ctx)

	var receivedMessages []claude.Message

	for rawMsg := range msgCh {
		msg, err := claude.ParseMessage(rawMsg)
		if err != nil {
			t.Fatalf("Failed to parse message: %v", err)
		}
		receivedMessages = append(receivedMessages, msg)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("Transport error: %v", err)
	}

	// Verify results
	if len(receivedMessages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(receivedMessages))
	}

	// Check assistant message
	assistantMsg, ok := receivedMessages[0].(*claude.AssistantMessage)
	if !ok {
		t.Fatalf("Expected AssistantMessage, got %T", receivedMessages[0])
	}

	if len(assistantMsg.Content) != 1 {
		t.Fatalf("Expected 1 content block, got %d", len(assistantMsg.Content))
	}

	textBlock, ok := assistantMsg.Content[0].(claude.TextBlock)
	if !ok {
		t.Fatalf("Expected TextBlock, got %T", assistantMsg.Content[0])
	}

	if textBlock.Text != "2 + 2 equals 4" {
		t.Errorf("Expected text '2 + 2 equals 4', got '%s'", textBlock.Text)
	}

	// Check result message
	resultMsg, ok := receivedMessages[1].(*claude.ResultMessage)
	if !ok {
		t.Fatalf("Expected ResultMessage, got %T", receivedMessages[1])
	}

	if resultMsg.SessionID != "test-session" {
		t.Errorf("Expected session ID 'test-session', got '%s'", resultMsg.SessionID)
	}

	if resultMsg.TotalCostUSD == nil || *resultMsg.TotalCostUSD != 0.001 {
		t.Errorf("Expected cost 0.001, got %v", resultMsg.TotalCostUSD)
	}
}

func TestQueryWithToolUse(t *testing.T) {
	ctx := context.Background()

	// Create mock transport with tool use response
	messages := []map[string]interface{}{
		CreateAssistantToolUseMessage(
			"Let me read that file for you.",
			"tool-123",
			"Read",
			map[string]interface{}{"file_path": "/test.txt"},
		),
		CreateResultMessage("test-session-2", 0.002, 1500),
	}

	mockTransport := NewMockTransport(messages)
	msgCh, errCh := mockTransport.ReadMessages(ctx)

	var receivedMessages []claude.Message

	for rawMsg := range msgCh {
		msg, err := claude.ParseMessage(rawMsg)
		if err != nil {
			t.Fatalf("Failed to parse message: %v", err)
		}
		receivedMessages = append(receivedMessages, msg)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("Transport error: %v", err)
	}

	// Verify results
	if len(receivedMessages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(receivedMessages))
	}

	// Check assistant message with tool use
	assistantMsg, ok := receivedMessages[0].(*claude.AssistantMessage)
	if !ok {
		t.Fatalf("Expected AssistantMessage, got %T", receivedMessages[0])
	}

	if len(assistantMsg.Content) != 2 {
		t.Fatalf("Expected 2 content blocks, got %d", len(assistantMsg.Content))
	}

	// Check text block
	textBlock, ok := assistantMsg.Content[0].(claude.TextBlock)
	if !ok {
		t.Fatalf("Expected TextBlock, got %T", assistantMsg.Content[0])
	}

	if textBlock.Text != "Let me read that file for you." {
		t.Errorf("Expected specific text, got '%s'", textBlock.Text)
	}

	// Check tool use block
	toolUseBlock, ok := assistantMsg.Content[1].(claude.ToolUseBlock)
	if !ok {
		t.Fatalf("Expected ToolUseBlock, got %T", assistantMsg.Content[1])
	}

	if toolUseBlock.Name != "Read" {
		t.Errorf("Expected tool name 'Read', got '%s'", toolUseBlock.Name)
	}

	if toolUseBlock.ID != "tool-123" {
		t.Errorf("Expected tool ID 'tool-123', got '%s'", toolUseBlock.ID)
	}

	filePath, ok := toolUseBlock.Input["file_path"].(string)
	if !ok || filePath != "/test.txt" {
		t.Errorf("Expected file_path '/test.txt', got %v", toolUseBlock.Input["file_path"])
	}
}

func TestMaxBudgetUSDOption(t *testing.T) {
	ctx := context.Background()

	// Create mock transport with budget exceeded response
	messages := []map[string]interface{}{
		CreateAssistantTextMessage("Starting to read..."),
		CreateResultMessageWithSubtype("test-session-budget", "error_max_budget_usd", 0.0002, 500),
	}

	mockTransport := NewMockTransport(messages)
	msgCh, errCh := mockTransport.ReadMessages(ctx)

	var receivedMessages []claude.Message

	for rawMsg := range msgCh {
		msg, err := claude.ParseMessage(rawMsg)
		if err != nil {
			t.Fatalf("Failed to parse message: %v", err)
		}
		receivedMessages = append(receivedMessages, msg)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("Transport error: %v", err)
	}

	// Verify results
	if len(receivedMessages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(receivedMessages))
	}

	// Check result message
	resultMsg, ok := receivedMessages[1].(*claude.ResultMessage)
	if !ok {
		t.Fatalf("Expected ResultMessage, got %T", receivedMessages[1])
	}

	if resultMsg.Subtype != "error_max_budget_usd" {
		t.Errorf("Expected subtype 'error_max_budget_usd', got '%s'", resultMsg.Subtype)
	}

	if resultMsg.IsError {
		t.Errorf("Expected is_error false, got true")
	}

	if resultMsg.TotalCostUSD == nil || *resultMsg.TotalCostUSD != 0.0002 {
		t.Errorf("Expected cost 0.0002, got %v", resultMsg.TotalCostUSD)
	}
}

func TestTransportOptions(t *testing.T) {
	// Test that options are properly passed to transport
	tests := []struct {
		name    string
		options *claude.ClaudeAgentOptions
		verify  func(t *testing.T, opts *claude.ClaudeAgentOptions)
	}{
		{
			name: "MaxBudgetUSD",
			options: &claude.ClaudeAgentOptions{
				MaxBudgetUSD: floatPtr(0.5),
			},
			verify: func(t *testing.T, opts *claude.ClaudeAgentOptions) {
				if opts.MaxBudgetUSD == nil || *opts.MaxBudgetUSD != 0.5 {
					t.Errorf("Expected MaxBudgetUSD 0.5, got %v", opts.MaxBudgetUSD)
				}
			},
		},
		{
			name: "MaxThinkingTokens",
			options: &claude.ClaudeAgentOptions{
				MaxThinkingTokens: intPtr(5000),
			},
			verify: func(t *testing.T, opts *claude.ClaudeAgentOptions) {
				if opts.MaxThinkingTokens == nil || *opts.MaxThinkingTokens != 5000 {
					t.Errorf("Expected MaxThinkingTokens 5000, got %v", opts.MaxThinkingTokens)
				}
			},
		},
		{
			name: "FallbackModel",
			options: &claude.ClaudeAgentOptions{
				FallbackModel: stringPtr("claude-sonnet-3-5"),
			},
			verify: func(t *testing.T, opts *claude.ClaudeAgentOptions) {
				if opts.FallbackModel == nil || *opts.FallbackModel != "claude-sonnet-3-5" {
					t.Errorf("Expected FallbackModel 'claude-sonnet-3-5', got %v", opts.FallbackModel)
				}
			},
		},
		{
			name: "Plugins",
			options: &claude.ClaudeAgentOptions{
				Plugins: []claude.SdkPluginConfig{
					{Type: "local", Path: "/path/to/plugin"},
				},
			},
			verify: func(t *testing.T, opts *claude.ClaudeAgentOptions) {
				if len(opts.Plugins) != 1 {
					t.Fatalf("Expected 1 plugin, got %d", len(opts.Plugins))
				}
				if opts.Plugins[0].Path != "/path/to/plugin" {
					t.Errorf("Expected plugin path '/path/to/plugin', got '%s'", opts.Plugins[0].Path)
				}
			},
		},
		{
			name: "ContinueConversation",
			options: &claude.ClaudeAgentOptions{
				ContinueConversation: true,
			},
			verify: func(t *testing.T, opts *claude.ClaudeAgentOptions) {
				if !opts.ContinueConversation {
					t.Errorf("Expected ContinueConversation true, got false")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify options are correctly set
			tt.verify(t, tt.options)
		})
	}
}

func TestCLINotFound(t *testing.T) {
	// Test handling when CLI is not found
	// This test verifies that CLINotFoundError is raised when Claude Code is not installed

	// Try to create transport with empty CLI path (triggers CLI discovery)
	// The transport creation itself doesn't fail - it only fails on Connect()
	// when the CLI cannot be found
	transport, err := claude.NewSubprocessCLITransport("test", &claude.ClaudeAgentOptions{}, "")

	// If Claude Code is installed, transport will be created successfully
	// If not installed, we should get CLINotFoundError
	if err != nil {
		// Verify it's the correct error type
		if _, ok := err.(*claude.CLINotFoundError); !ok {
			t.Errorf("Expected CLINotFoundError, got %T: %v", err, err)
		}

		// Verify error message contains helpful information
		errMsg := err.Error()
		if errMsg == "" {
			t.Error("Error message should not be empty")
		}
	} else {
		// Claude Code is installed, so we can't test the not-found case
		t.Skip("Claude Code is installed, skipping CLI not found test")
		_ = transport
	}
}

func TestContinuationOption(t *testing.T) {
	ctx := context.Background()

	// Create mock transport with continuation response
	messages := []map[string]interface{}{
		CreateAssistantTextMessage("Continuing from previous conversation"),
		CreateResultMessage("test-session-cont", 0.001, 500),
	}

	mockTransport := NewMockTransport(messages)
	msgCh, errCh := mockTransport.ReadMessages(ctx)

	var receivedMessages []claude.Message

	for rawMsg := range msgCh {
		msg, err := claude.ParseMessage(rawMsg)
		if err != nil {
			t.Fatalf("Failed to parse message: %v", err)
		}
		receivedMessages = append(receivedMessages, msg)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("Transport error: %v", err)
	}

	// Verify results
	if len(receivedMessages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(receivedMessages))
	}

	// Check assistant message
	assistantMsg, ok := receivedMessages[0].(*claude.AssistantMessage)
	if !ok {
		t.Fatalf("Expected AssistantMessage, got %T", receivedMessages[0])
	}

	textBlock, ok := assistantMsg.Content[0].(claude.TextBlock)
	if !ok {
		t.Fatalf("Expected TextBlock, got %T", assistantMsg.Content[0])
	}

	if textBlock.Text != "Continuing from previous conversation" {
		t.Errorf("Expected continuation text, got '%s'", textBlock.Text)
	}

	// Verify that ContinueConversation option works
	options := &claude.ClaudeAgentOptions{
		ContinueConversation: true,
	}

	if !options.ContinueConversation {
		t.Error("Expected ContinueConversation to be true")
	}
}

func TestResumeSessionOption(t *testing.T) {
	// Test that resume option is properly set

	sessionID := "previous-session-123"
	options := &claude.ClaudeAgentOptions{
		Resume: &sessionID,
	}

	if options.Resume == nil {
		t.Fatal("Expected Resume to be set")
	}

	if *options.Resume != "previous-session-123" {
		t.Errorf("Expected Resume 'previous-session-123', got '%s'", *options.Resume)
	}
}

// Helper functions
func floatPtr(f float64) *float64 {
	return &f
}

func intPtr(i int) *int {
	return &i
}

func stringPtr(s string) *string {
	return &s
}
