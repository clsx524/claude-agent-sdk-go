package unit

import (
	"context"
	"testing"

	claude "github.com/clsx524/claude-agent-sdk-go"
)

// TestToolPermissionCallback_Allow tests that allowed tools are permitted
func TestToolPermissionCallback_Allow(t *testing.T) {
	called := false
	callback := func(ctx context.Context, toolName string, input map[string]interface{}, permCtx claude.ToolPermissionContext) (claude.PermissionResult, error) {
		called = true
		if toolName == "Read" {
			return claude.PermissionResultAllow{Behavior: "allow"}, nil
		}
		return claude.PermissionResultDeny{Behavior: "deny", Message: "Not allowed"}, nil
	}

	// Simulate a tool use scenario
	result, err := callback(context.Background(), "Read", map[string]interface{}{"file": "test.txt"}, claude.ToolPermissionContext{})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !called {
		t.Fatal("Callback was not called")
	}
	if allow, ok := result.(claude.PermissionResultAllow); !ok || allow.Behavior != "allow" {
		t.Fatalf("Expected allow result, got %v", result)
	}
}

// TestToolPermissionCallback_Deny tests that denied tools are blocked
func TestToolPermissionCallback_Deny(t *testing.T) {
	callback := func(ctx context.Context, toolName string, input map[string]interface{}, permCtx claude.ToolPermissionContext) (claude.PermissionResult, error) {
		if toolName == "Bash" {
			command, ok := input["command"].(string)
			if ok && command == "rm -rf /" {
				return claude.PermissionResultDeny{
					Behavior: "deny",
					Message:  "Dangerous command blocked",
				}, nil
			}
		}
		return claude.PermissionResultAllow{Behavior: "allow"}, nil
	}

	result, err := callback(context.Background(), "Bash", map[string]interface{}{"command": "rm -rf /"}, claude.ToolPermissionContext{})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if deny, ok := result.(claude.PermissionResultDeny); !ok {
		t.Fatalf("Expected deny result, got %v", result)
	} else if deny.Message != "Dangerous command blocked" {
		t.Fatalf("Expected specific message, got: %s", deny.Message)
	}
}

// TestToolPermissionCallback_Ask tests the ask behavior
func TestToolPermissionCallback_Ask(t *testing.T) {
	callback := func(ctx context.Context, toolName string, input map[string]interface{}, permCtx claude.ToolPermissionContext) (claude.PermissionResult, error) {
		if toolName == "Write" {
			return claude.PermissionResultAsk{
				Behavior: "ask",
				Message:  "Allow writing to this file?",
			}, nil
		}
		return claude.PermissionResultAllow{Behavior: "allow"}, nil
	}

	result, err := callback(context.Background(), "Write", map[string]interface{}{"file": "important.txt"}, claude.ToolPermissionContext{})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if ask, ok := result.(claude.PermissionResultAsk); !ok {
		t.Fatalf("Expected ask result, got %v", result)
	} else if ask.Message != "Allow writing to this file?" {
		t.Fatalf("Expected specific message, got: %s", ask.Message)
	}
}

// TestToolPermissionCallback_ParameterValidation tests parameter validation
func TestToolPermissionCallback_ParameterValidation(t *testing.T) {
	tests := []struct {
		name        string
		toolName    string
		input       map[string]interface{}
		expectAllow bool
	}{
		{
			name:     "Valid parameters",
			toolName: "Read",
			input: map[string]interface{}{
				"file_path": "/valid/path.txt",
			},
			expectAllow: true,
		},
		{
			name:        "Missing parameters",
			toolName:    "Read",
			input:       map[string]interface{}{},
			expectAllow: false,
		},
		{
			name:     "Invalid parameter type",
			toolName: "Read",
			input: map[string]interface{}{
				"file_path": 123, // Should be string
			},
			expectAllow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callback := func(ctx context.Context, toolName string, input map[string]interface{}, permCtx claude.ToolPermissionContext) (claude.PermissionResult, error) {
				if toolName == "Read" {
					filePath, ok := input["file_path"].(string)
					if !ok || filePath == "" {
						return claude.PermissionResultDeny{
							Behavior: "deny",
							Message:  "Invalid or missing file_path parameter",
						}, nil
					}
				}
				return claude.PermissionResultAllow{Behavior: "allow"}, nil
			}

			result, err := callback(context.Background(), tt.toolName, tt.input, claude.ToolPermissionContext{})

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			_, isAllow := result.(claude.PermissionResultAllow)
			if isAllow != tt.expectAllow {
				t.Fatalf("Expected allow=%v, got result: %v", tt.expectAllow, result)
			}
		})
	}
}

// TestToolPermissionCallback_ContextCancellation tests context cancellation
func TestToolPermissionCallback_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	callback := func(ctx context.Context, toolName string, input map[string]interface{}, permCtx claude.ToolPermissionContext) (claude.PermissionResult, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			return claude.PermissionResultAllow{Behavior: "allow"}, nil
		}
	}

	_, err := callback(ctx, "Read", map[string]interface{}{}, claude.ToolPermissionContext{})

	if err != context.Canceled {
		t.Fatalf("Expected context.Canceled error, got: %v", err)
	}
}

// TestToolPermissionCallback_AllToolNames tests various tool names
func TestToolPermissionCallback_AllToolNames(t *testing.T) {
	toolNames := []string{"Read", "Write", "Edit", "Bash", "Grep", "Glob", "WebFetch", "Task"}

	for _, toolName := range toolNames {
		t.Run(toolName, func(t *testing.T) {
			callback := func(ctx context.Context, name string, input map[string]interface{}, permCtx claude.ToolPermissionContext) (claude.PermissionResult, error) {
				// Allow all read-only tools
				readOnlyTools := map[string]bool{
					"Read": true, "Grep": true, "Glob": true, "WebFetch": true,
				}
				if readOnlyTools[name] {
					return claude.PermissionResultAllow{Behavior: "allow"}, nil
				}
				return claude.PermissionResultAsk{
					Behavior: "ask",
					Message:  "Allow " + name + "?",
				}, nil
			}

			result, err := callback(context.Background(), toolName, map[string]interface{}{}, claude.ToolPermissionContext{})

			if err != nil {
				t.Fatalf("Unexpected error for %s: %v", toolName, err)
			}
			if result == nil {
				t.Fatalf("Expected non-nil result for %s", toolName)
			}
		})
	}
}

// TestToolPermissionCallback_ComplexInput tests complex input validation
func TestToolPermissionCallback_ComplexInput(t *testing.T) {
	callback := func(ctx context.Context, toolName string, input map[string]interface{}, permCtx claude.ToolPermissionContext) (claude.PermissionResult, error) {
		if toolName == "Edit" {
			// Validate nested structure
			oldString, hasOld := input["old_string"].(string)
			newString, hasNew := input["new_string"].(string)

			if !hasOld || !hasNew || oldString == "" || newString == "" {
				return claude.PermissionResultDeny{
					Behavior: "deny",
					Message:  "Both old_string and new_string required",
				}, nil
			}

			// Don't allow replacing important patterns
			if oldString == "password" {
				return claude.PermissionResultDeny{
					Behavior: "deny",
					Message:  "Cannot edit password fields",
				}, nil
			}
		}
		return claude.PermissionResultAllow{Behavior: "allow"}, nil
	}

	tests := []struct {
		name        string
		input       map[string]interface{}
		expectAllow bool
		expectMsg   string
	}{
		{
			name: "Valid edit",
			input: map[string]interface{}{
				"old_string": "foo",
				"new_string": "bar",
			},
			expectAllow: true,
		},
		{
			name: "Missing new_string",
			input: map[string]interface{}{
				"old_string": "foo",
			},
			expectAllow: false,
			expectMsg:   "Both old_string and new_string required",
		},
		{
			name: "Password replacement blocked",
			input: map[string]interface{}{
				"old_string": "password",
				"new_string": "newpass",
			},
			expectAllow: false,
			expectMsg:   "Cannot edit password fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := callback(context.Background(), "Edit", tt.input, claude.ToolPermissionContext{})

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			_, isAllow := result.(claude.PermissionResultAllow)
			if isAllow != tt.expectAllow {
				t.Fatalf("Expected allow=%v, got result: %v", tt.expectAllow, result)
			}

			if !tt.expectAllow && tt.expectMsg != "" {
				deny, ok := result.(claude.PermissionResultDeny)
				if !ok {
					t.Fatalf("Expected deny result with message")
				}
				if deny.Message != tt.expectMsg {
					t.Fatalf("Expected message '%s', got '%s'", tt.expectMsg, deny.Message)
				}
			}
		})
	}
}

// TestToolPermissionCallback_ChainedCallbacks tests multiple callbacks
func TestToolPermissionCallback_ChainedCallbacks(t *testing.T) {
	// Simulate two different permission strategies
	callback1 := func(ctx context.Context, toolName string, input map[string]interface{}, permCtx claude.ToolPermissionContext) (claude.PermissionResult, error) {
		// First layer: block dangerous tools
		if toolName == "Bash" {
			return claude.PermissionResultDeny{Behavior: "deny", Message: "Bash blocked"}, nil
		}
		return claude.PermissionResultAllow{Behavior: "allow"}, nil
	}

	callback2 := func(ctx context.Context, toolName string, input map[string]interface{}, permCtx claude.ToolPermissionContext) (claude.PermissionResult, error) {
		// Second layer: ask for writes
		if toolName == "Write" {
			return claude.PermissionResultAsk{Behavior: "ask", Message: "Allow write?"}, nil
		}
		return claude.PermissionResultAllow{Behavior: "allow"}, nil
	}

	// Test callback1
	result1, err := callback1(context.Background(), "Bash", map[string]interface{}{}, claude.ToolPermissionContext{})
	if err != nil {
		t.Fatalf("callback1 error: %v", err)
	}
	if _, ok := result1.(claude.PermissionResultDeny); !ok {
		t.Fatal("callback1 should deny Bash")
	}

	// Test callback2
	result2, err := callback2(context.Background(), "Write", map[string]interface{}{}, claude.ToolPermissionContext{})
	if err != nil {
		t.Fatalf("callback2 error: %v", err)
	}
	if _, ok := result2.(claude.PermissionResultAsk); !ok {
		t.Fatal("callback2 should ask for Write")
	}

	// Test callback2 allows Read
	result3, err := callback2(context.Background(), "Read", map[string]interface{}{}, claude.ToolPermissionContext{})
	if err != nil {
		t.Fatalf("callback2 error: %v", err)
	}
	if _, ok := result3.(claude.PermissionResultAllow); !ok {
		t.Fatal("callback2 should allow Read")
	}
}
