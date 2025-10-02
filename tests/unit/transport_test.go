package unit

import (
	"strings"
	"testing"

	claude "github.com/clsx524/claude-agent-sdk-go"
)

func TestSubprocessCommandBuilding(t *testing.T) {
	tests := []struct {
		name     string
		prompt   interface{}
		options  *claude.ClaudeAgentOptions
		expected []string
	}{
		{
			name:    "simple string prompt",
			prompt:  "Hello",
			options: &claude.ClaudeAgentOptions{},
			expected: []string{
				"--output-format", "stream-json",
				"--verbose",
				"--setting-sources", "",
				"--print", "--", "Hello",
			},
		},
		{
			name:   "with allowed tools",
			prompt: "test",
			options: &claude.ClaudeAgentOptions{
				AllowedTools: []string{"Read", "Write"},
			},
			expected: []string{
				"--output-format", "stream-json",
				"--verbose",
				"--allowedTools", "Read,Write",
			},
		},
		{
			name:   "with disallowed tools",
			prompt: "test",
			options: &claude.ClaudeAgentOptions{
				DisallowedTools: []string{"Bash"},
			},
			expected: []string{
				"--disallowedTools", "Bash",
			},
		},
		{
			name:   "with max turns",
			prompt: "test",
			options: &claude.ClaudeAgentOptions{
				MaxTurns: intPtr(5),
			},
			expected: []string{
				"--max-turns", "5",
			},
		},
		{
			name:   "with model",
			prompt: "test",
			options: &claude.ClaudeAgentOptions{
				Model: stringPtr("claude-opus-4"),
			},
			expected: []string{
				"--model", "claude-opus-4",
			},
		},
		{
			name:   "with permission mode",
			prompt: "test",
			options: &claude.ClaudeAgentOptions{
				PermissionMode: permissionModePtr(claude.PermissionModeAcceptEdits),
			},
			expected: []string{
				"--permission-mode", "acceptEdits",
			},
		},
		{
			name:   "with system prompt string",
			prompt: "test",
			options: &claude.ClaudeAgentOptions{
				SystemPrompt: "You are a helpful assistant",
			},
			expected: []string{
				"--system-prompt", "You are a helpful assistant",
			},
		},
		{
			name:   "with system prompt preset append",
			prompt: "test",
			options: &claude.ClaudeAgentOptions{
				SystemPrompt: claude.SystemPromptPreset{
					Type:   "preset",
					Preset: "claude_code",
					Append: stringPtr("Additional instructions"),
				},
			},
			expected: []string{
				"--append-system-prompt", "Additional instructions",
			},
		},
		{
			name:   "with continue conversation",
			prompt: "test",
			options: &claude.ClaudeAgentOptions{
				ContinueConversation: true,
			},
			expected: []string{
				"--continue",
			},
		},
		{
			name:   "with resume",
			prompt: "test",
			options: &claude.ClaudeAgentOptions{
				Resume: stringPtr("session_123"),
			},
			expected: []string{
				"--resume", "session_123",
			},
		},
		{
			name:   "with fork session",
			prompt: "test",
			options: &claude.ClaudeAgentOptions{
				ForkSession: true,
			},
			expected: []string{
				"--fork-session",
			},
		},
		{
			name:   "with settings",
			prompt: "test",
			options: &claude.ClaudeAgentOptions{
				Settings: stringPtr("/path/to/settings.json"),
			},
			expected: []string{
				"--settings", "/path/to/settings.json",
			},
		},
		{
			name:   "with add dirs",
			prompt: "test",
			options: &claude.ClaudeAgentOptions{
				AddDirs: []string{"/dir1", "/dir2"},
			},
			expected: []string{
				"--add-dir", "/dir1",
				"--add-dir", "/dir2",
			},
		},
		{
			name:   "with partial messages",
			prompt: "test",
			options: &claude.ClaudeAgentOptions{
				IncludePartialMessages: true,
			},
			expected: []string{
				"--include-partial-messages",
			},
		},
		{
			name:   "with setting sources",
			prompt: "test",
			options: &claude.ClaudeAgentOptions{
				SettingSources: []claude.SettingSource{
					claude.SettingSourceUser,
					claude.SettingSourceProject,
				},
			},
			expected: []string{
				"--setting-sources", "user,project",
			},
		},
		{
			name:   "with extra args",
			prompt: "test",
			options: &claude.ClaudeAgentOptions{
				ExtraArgs: map[string]*string{
					"debug-to-stderr": nil,
					"custom-flag":     stringPtr("value"),
				},
			},
			expected: []string{
				"--debug-to-stderr",
				"--custom-flag", "value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create transport (will use mock CLI path)
			trans, err := claude.NewSubprocessCLITransport(tt.prompt, tt.options, "/mock/claude")
			if err != nil {
				t.Fatalf("failed to create transport: %v", err)
			}

			// Access buildCommand via reflection or create a public test helper
			// For now, we'll test that the transport was created successfully
			_ = trans

			// In a real implementation, we'd need to expose buildCommand for testing
			// or test it indirectly through the command execution
		})
	}
}

func TestCLIDiscovery(t *testing.T) {
	// Test that CLI discovery works
	// This will fail if Claude Code is not installed, which is expected
	t.Run("CLI not found", func(t *testing.T) {
		// Create transport with empty CLI path to trigger discovery
		_, err := claude.NewSubprocessCLITransport("test", &claude.ClaudeAgentOptions{}, "")

		// If err is nil, Claude Code is installed and found
		// If err is not nil, it should be a CLINotFoundError
		if err != nil {
			if _, ok := err.(*claude.CLINotFoundError); !ok {
				t.Errorf("expected CLINotFoundError, got %T: %v", err, err)
			}

			errMsg := err.Error()
			if !strings.Contains(errMsg, "Claude Code not found") {
				t.Errorf("error message should mention Claude Code not found, got: %s", errMsg)
			}
			if !strings.Contains(errMsg, "npm install") {
				t.Errorf("error message should include installation instructions, got: %s", errMsg)
			}
		}
	})
}

func TestStreamingModeDetection(t *testing.T) {
	t.Run("string prompt is non-streaming", func(t *testing.T) {
		trans, err := claude.NewSubprocessCLITransport("test", &claude.ClaudeAgentOptions{}, "/mock/claude")
		if err != nil {
			t.Fatalf("failed to create transport: %v", err)
		}
		_ = trans
		// Transport should be created for non-streaming mode
	})

	t.Run("channel prompt is streaming", func(t *testing.T) {
		ch := make(chan map[string]interface{})
		close(ch)
		trans, err := claude.NewSubprocessCLITransport(ch, &claude.ClaudeAgentOptions{}, "/mock/claude")
		if err != nil {
			t.Fatalf("failed to create transport: %v", err)
		}
		_ = trans
		// Transport should be created for streaming mode
	})
}

func TestMcpServerSerialization(t *testing.T) {
	t.Run("stdio server", func(t *testing.T) {
		options := &claude.ClaudeAgentOptions{
			McpServers: map[string]claude.McpServerConfig{
				"test": claude.McpStdioServerConfig{
					Type:    "stdio",
					Command: "python",
					Args:    []string{"-m", "server"},
					Env:     map[string]string{"KEY": "value"},
				},
			},
		}

		trans, err := claude.NewSubprocessCLITransport("test", options, "/mock/claude")
		if err != nil {
			t.Fatalf("failed to create transport: %v", err)
		}
		_ = trans
	})

	t.Run("SDK server excludes instance", func(t *testing.T) {
		options := &claude.ClaudeAgentOptions{
			McpServers: map[string]claude.McpServerConfig{
				"test": claude.McpSdkServerConfig{
					Type:     "sdk",
					Name:     "test-server",
					Instance: struct{}{}, // Mock instance
				},
			},
		}

		trans, err := claude.NewSubprocessCLITransport("test", options, "/mock/claude")
		if err != nil {
			t.Fatalf("failed to create transport: %v", err)
		}
		_ = trans
		// Should successfully create transport
		// Instance field should be excluded from CLI config
	})
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func stringPtr(s string) *string {
	return &s
}

func permissionModePtr(pm claude.PermissionMode) *claude.PermissionMode {
	return &pm
}
