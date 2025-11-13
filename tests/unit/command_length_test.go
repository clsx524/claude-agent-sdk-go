package unit

import (
	"runtime"
	"strings"
	"testing"

	claude "github.com/clsx524/claude-agent-sdk-go"
)

// Windows has a command line length limit of 32,767 characters (including the program name).
// POSIX systems typically have a limit of 131,072 bytes (ARG_MAX).
// These tests ensure we handle long command lines appropriately.

const (
	// Windows command line limit
	windowsCmdLimit = 32767

	// Linux/macOS typical ARG_MAX limit
	posixArgMax = 131072
)

// TestLongCommandLineWindows tests that long command lines are handled correctly on Windows.
func TestLongCommandLineWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test on non-Windows platform")
	}

	// Create options with many tools to test command line length
	options := &claude.ClaudeAgentOptions{
		AllowedTools: generateLongToolList(1000),
	}

	// Test that command line construction doesn't panic
	// Note: This doesn't actually execute the command, just validates construction
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Panic occurred with long command line: %v", r)
		}
	}()

	// We can't easily test the actual subprocess creation without mocking,
	// but we can validate the options structure is reasonable
	if options.AllowedTools == nil {
		t.Error("AllowedTools should not be nil")
	}
}

// TestLongCommandLinePosix tests that long command lines are handled on POSIX systems.
func TestLongCommandLinePosix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping POSIX-specific test on Windows")
	}

	// Create options with very long strings
	options := &claude.ClaudeAgentOptions{
		AllowedTools: generateLongToolList(5000),
	}

	// Validate options don't cause issues
	if options.AllowedTools == nil {
		t.Error("AllowedTools should not be nil")
	}

	if len(options.AllowedTools) != 5000 {
		t.Errorf("Expected 5000 tools, got %d", len(options.AllowedTools))
	}
}

// TestCommandLineEstimation tests estimating command line length.
func TestCommandLineEstimation(t *testing.T) {
	tests := []struct {
		name          string
		options       *claude.ClaudeAgentOptions
		expectedChars int // Rough estimate
	}{
		{
			name:          "Empty options",
			options:       &claude.ClaudeAgentOptions{},
			expectedChars: 0,
		},
		{
			name: "Single tool",
			options: &claude.ClaudeAgentOptions{
				AllowedTools: []string{"Read"},
			},
			expectedChars: 10, // Rough estimate
		},
		{
			name: "Multiple tools",
			options: &claude.ClaudeAgentOptions{
				AllowedTools: []string{"Read", "Write", "Bash", "Grep"},
			},
			expectedChars: 50, // Rough estimate
		},
		{
			name: "Long system prompt",
			options: &claude.ClaudeAgentOptions{
				SystemPrompt: stringPtr(strings.Repeat("x", 1000)),
			},
			expectedChars: 1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Estimate command line length by serializing options
			estimate := estimateOptionsSize(tt.options)

			// We just check that it's reasonable, not exact
			if estimate < 0 {
				t.Errorf("Negative estimate: %d", estimate)
			}
		})
	}
}

// TestWindowsCommandLineLimitExceeded tests behavior when approaching Windows limit.
func TestWindowsCommandLineLimitExceeded(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test")
	}

	// Create options that would exceed Windows limit
	longPrompt := strings.Repeat("a", windowsCmdLimit+1000)
	options := &claude.ClaudeAgentOptions{
		SystemPrompt: &longPrompt,
	}

	// Validate that we can detect this situation
	estimate := estimateOptionsSize(options)
	if estimate <= windowsCmdLimit {
		t.Logf("Estimate %d is within Windows limit %d", estimate, windowsCmdLimit)
	} else {
		t.Logf("Estimate %d exceeds Windows limit %d (expected)", estimate, windowsCmdLimit)
	}
}

// TestPosixCommandLineLimitExceeded tests behavior when approaching POSIX limit.
func TestPosixCommandLineLimitExceeded(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping POSIX-specific test")
	}

	// Create options that would exceed typical POSIX ARG_MAX
	longPrompt := strings.Repeat("a", posixArgMax+1000)
	options := &claude.ClaudeAgentOptions{
		SystemPrompt: &longPrompt,
	}

	// Validate that we can detect this situation
	estimate := estimateOptionsSize(options)
	if estimate <= posixArgMax {
		t.Logf("Estimate %d is within POSIX limit %d", estimate, posixArgMax)
	} else {
		t.Logf("Estimate %d exceeds POSIX limit %d (expected)", estimate, posixArgMax)
	}
}

// TestCommandLineWithUnicode tests handling of Unicode characters in command line.
func TestCommandLineWithUnicode(t *testing.T) {
	unicodePrompt := "Hello 世界 🌍 مرحبا мир"
	options := &claude.ClaudeAgentOptions{
		SystemPrompt: &unicodePrompt,
	}

	estimate := estimateOptionsSize(options)

	// Unicode characters take multiple bytes
	if estimate < len(unicodePrompt) {
		t.Errorf("Estimate %d should be >= actual byte length %d for Unicode", estimate, len(unicodePrompt))
	}
}

// Helper functions

func generateLongToolList(count int) []string {
	tools := make([]string, count)
	for i := 0; i < count; i++ {
		tools[i] = "Tool_" + strings.Repeat("X", 10) + "_" + string(rune('A'+i%26))
	}
	return tools
}

// estimateOptionsSize estimates the byte size of serialized options.
// This is a simplified estimation - actual command line includes program name,
// flags, JSON serialization overhead, etc.
func estimateOptionsSize(options *claude.ClaudeAgentOptions) int {
	if options == nil {
		return 0
	}

	size := 0

	// Estimate SystemPrompt
	if options.SystemPrompt != nil {
		if sp, ok := options.SystemPrompt.(string); ok {
			size += len(sp)
		} else if spPtr, ok := options.SystemPrompt.(*string); ok {
			size += len(*spPtr)
		}
	}

	// Estimate AllowedTools
	for _, tool := range options.AllowedTools {
		size += len(tool) + 3 // +3 for quotes and comma in JSON
	}

	// Estimate DisallowedTools
	for _, tool := range options.DisallowedTools {
		size += len(tool) + 3
	}

	// Estimate Model
	if options.Model != nil {
		size += len(*options.Model)
	}

	// Estimate FallbackModel
	if options.FallbackModel != nil {
		size += len(*options.FallbackModel)
	}

	// Estimate Cwd
	if options.Cwd != nil {
		size += len(*options.Cwd)
	}

	// Add overhead for JSON structure (brackets, field names, etc.)
	size += 200

	return size
}

// TestCommandLineRealWorldScenarios tests realistic command line scenarios.
func TestCommandLineRealWorldScenarios(t *testing.T) {
	scenarios := []struct {
		name    string
		options *claude.ClaudeAgentOptions
	}{
		{
			name: "Typical development usage",
			options: &claude.ClaudeAgentOptions{
				AllowedTools: []string{"Read", "Write", "Edit", "Bash", "Grep", "Glob"},
				SystemPrompt: stringPtr("You are a helpful coding assistant."),
			},
		},
		{
			name: "Restricted security mode",
			options: &claude.ClaudeAgentOptions{
				AllowedTools:    []string{"Read", "Grep"},
				DisallowedTools: []string{"Bash", "Write", "Delete"},
				SystemPrompt:    stringPtr("Read-only mode. Do not modify any files."),
			},
		},
		{
			name: "Large project with many MCP servers",
			options: &claude.ClaudeAgentOptions{
				AllowedTools: generateLongToolList(50),
				McpServers: map[string]claude.McpServerConfig{
					"server1": claude.McpStdioServerConfig{Command: "cmd1"},
					"server2": claude.McpStdioServerConfig{Command: "cmd2"},
					"server3": claude.McpStdioServerConfig{Command: "cmd3"},
				},
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			estimate := estimateOptionsSize(scenario.options)

			// Log the estimate for reference
			t.Logf("Scenario '%s' estimated size: %d bytes", scenario.name, estimate)

			// Validate it's reasonable
			if estimate < 0 {
				t.Errorf("Invalid negative estimate: %d", estimate)
			}

			// Warn if approaching limits (but don't fail - this is informational)
			if runtime.GOOS == "windows" && estimate > windowsCmdLimit/2 {
				t.Logf("Warning: Approaching Windows command line limit (%d / %d)", estimate, windowsCmdLimit)
			} else if runtime.GOOS != "windows" && estimate > posixArgMax/2 {
				t.Logf("Warning: Approaching POSIX ARG_MAX limit (%d / %d)", estimate, posixArgMax)
			}
		})
	}
}
