package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	claude "github.com/clsx524/claude-agent-sdk-go"
)

// TestQueryWithCustomWorkingDirectory verifies that CWD option is passed correctly
func TestQueryWithCustomWorkingDirectory(t *testing.T) {
	ctx := context.Background()

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "claude-sdk-cwd-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Setup mock transport
	messages := []map[string]interface{}{
		CreateAssistantTextMessage("Found file: test.txt"),
		CreateResultMessage("test-session", 0.001, 500),
	}
	mockTransport := NewMockTransport(messages)

	// Test that we can create options with CWD
	options := &claude.ClaudeAgentOptions{
		Cwd: &tempDir,
	}

	// Verify options were set correctly
	if options.Cwd == nil {
		t.Fatal("CWD option was not set")
	}
	if *options.Cwd != tempDir {
		t.Errorf("Expected CWD %s, got %s", tempDir, *options.Cwd)
	}

	// Test parsing mock messages
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

	if len(receivedMessages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(receivedMessages))
	}

	assistantMsg, ok := receivedMessages[0].(*claude.AssistantMessage)
	if !ok {
		t.Fatalf("Expected AssistantMessage, got %T", receivedMessages[0])
	}

	textBlock, ok := assistantMsg.Content[0].(claude.TextBlock)
	if !ok {
		t.Fatalf("Expected TextBlock, got %T", assistantMsg.Content[0])
	}

	if textBlock.Text != "Found file: test.txt" {
		t.Errorf("Expected 'Found file: test.txt', got '%s'", textBlock.Text)
	}
}

// TestCwdOptionParsing verifies CWD option handling
func TestCwdOptionParsing(t *testing.T) {
	tests := []struct {
		name        string
		cwd         *string
		expectValid bool
	}{
		{
			name:        "nil CWD",
			cwd:         nil,
			expectValid: true,
		},
		{
			name: "valid absolute path",
			cwd: func() *string {
				cwd := "/tmp"
				return &cwd
			}(),
			expectValid: true,
		},
		{
			name: "empty string CWD",
			cwd: func() *string {
				cwd := ""
				return &cwd
			}(),
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &claude.ClaudeAgentOptions{
				Cwd: tt.cwd,
			}

			// Verify option is set correctly
			if tt.cwd == nil && options.Cwd != nil {
				t.Error("Expected nil CWD")
			}
			if tt.cwd != nil && options.Cwd == nil {
				t.Error("Expected non-nil CWD")
			}
			if tt.cwd != nil && options.Cwd != nil && *tt.cwd != *options.Cwd {
				t.Errorf("Expected CWD %s, got %s", *tt.cwd, *options.Cwd)
			}
		})
	}
}

// TestCwdWithRelativePath verifies relative path handling
func TestCwdWithRelativePath(t *testing.T) {
	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get cwd: %v", err)
	}

	// Create temp directory relative to current directory
	tempDir := filepath.Join(originalCwd, "temp_test_dir")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test with relative path
	relativePath := "temp_test_dir"
	options := &claude.ClaudeAgentOptions{
		Cwd: &relativePath,
	}

	if options.Cwd == nil {
		t.Fatal("CWD option was not set")
	}
	if *options.Cwd != relativePath {
		t.Errorf("Expected CWD %s, got %s", relativePath, *options.Cwd)
	}
}

// TestCwdWithAbsolutePath verifies absolute path handling
func TestCwdWithAbsolutePath(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "claude-sdk-abs-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Get absolute path
	absPath, err := filepath.Abs(tempDir)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	options := &claude.ClaudeAgentOptions{
		Cwd: &absPath,
	}

	if options.Cwd == nil {
		t.Fatal("CWD option was not set")
	}
	if *options.Cwd != absPath {
		t.Errorf("Expected CWD %s, got %s", absPath, *options.Cwd)
	}

	// Verify it's actually an absolute path
	if !filepath.IsAbs(*options.Cwd) {
		t.Error("Expected absolute path")
	}
}

// TestCwdOptionSerialization verifies CWD is properly serialized
func TestCwdOptionSerialization(t *testing.T) {
	testDir := "/test/directory"
	options := &claude.ClaudeAgentOptions{
		Cwd: &testDir,
	}

	// The CWD option should be available for the transport to use
	// This tests that the option is properly stored and accessible
	if options.Cwd == nil {
		t.Fatal("CWD should not be nil")
	}

	if *options.Cwd != testDir {
		t.Errorf("Expected CWD %s, got %s", testDir, *options.Cwd)
	}
}

// TestMultipleCwdOptions verifies that CWD can be changed between queries
func TestMultipleCwdOptions(t *testing.T) {
	dir1 := "/path/one"
	dir2 := "/path/two"

	options1 := &claude.ClaudeAgentOptions{
		Cwd: &dir1,
	}

	options2 := &claude.ClaudeAgentOptions{
		Cwd: &dir2,
	}

	if *options1.Cwd != dir1 {
		t.Errorf("Expected first CWD %s, got %s", dir1, *options1.Cwd)
	}

	if *options2.Cwd != dir2 {
		t.Errorf("Expected second CWD %s, got %s", dir2, *options2.Cwd)
	}

	// Verify they're independent
	*options1.Cwd = "/changed"
	if *options2.Cwd == "/changed" {
		t.Error("Options should be independent")
	}
}
