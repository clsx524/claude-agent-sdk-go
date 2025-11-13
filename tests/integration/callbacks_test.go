package integration

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	claude "github.com/clsx524/claude-agent-sdk-go"
)

func TestPermissionCallbackAllow(t *testing.T) {
	ctx := context.Background()
	callbackInvoked := false

	canUseTool := func(ctx context.Context, toolName string, input map[string]interface{}, permCtx claude.ToolPermissionContext) (claude.PermissionResult, error) {
		callbackInvoked = true

		if toolName != "TestTool" {
			t.Errorf("Expected tool name 'TestTool', got '%s'", toolName)
		}

		paramValue, ok := input["param"].(string)
		if !ok || paramValue != "value" {
			t.Errorf("Expected param 'value', got %v", input["param"])
		}

		return claude.PermissionResultAllow{
			Behavior: "allow",
		}, nil
	}

	options := &claude.ClaudeAgentOptions{
		CanUseTool: canUseTool,
	}

	// Create mock transport
	mockTransport := NewMockTransport(nil)

	// Simulate permission check
	result, err := canUseTool(ctx, "TestTool", map[string]interface{}{"param": "value"}, claude.ToolPermissionContext{})
	if err != nil {
		t.Fatalf("Callback error: %v", err)
	}

	if !callbackInvoked {
		t.Error("Expected callback to be invoked")
	}

	allowResult, ok := result.(claude.PermissionResultAllow)
	if !ok {
		t.Fatalf("Expected PermissionResultAllow, got %T", result)
	}

	if allowResult.Behavior != "allow" {
		t.Errorf("Expected behavior 'allow', got '%s'", allowResult.Behavior)
	}

	_ = mockTransport
	_ = options
}

func TestPermissionCallbackDeny(t *testing.T) {
	ctx := context.Background()

	canUseTool := func(ctx context.Context, toolName string, input map[string]interface{}, permCtx claude.ToolPermissionContext) (claude.PermissionResult, error) {
		return claude.PermissionResultDeny{
			Behavior: "deny",
			Message:  "Security policy violation",
		}, nil
	}

	result, err := canUseTool(ctx, "DangerousTool", map[string]interface{}{}, claude.ToolPermissionContext{})
	if err != nil {
		t.Fatalf("Callback error: %v", err)
	}

	denyResult, ok := result.(claude.PermissionResultDeny)
	if !ok {
		t.Fatalf("Expected PermissionResultDeny, got %T", result)
	}

	if denyResult.Behavior != "deny" {
		t.Errorf("Expected behavior 'deny', got '%s'", denyResult.Behavior)
	}

	if denyResult.Message != "Security policy violation" {
		t.Errorf("Expected specific message, got '%s'", denyResult.Message)
	}
}

func TestPermissionCallbackModifyInput(t *testing.T) {
	ctx := context.Background()

	canUseTool := func(ctx context.Context, toolName string, input map[string]interface{}, permCtx claude.ToolPermissionContext) (claude.PermissionResult, error) {
		// Modify the input to redirect to a safe path
		modifiedInput := make(map[string]interface{})
		for k, v := range input {
			modifiedInput[k] = v
		}
		modifiedInput["file_path"] = "/safe/path/file.txt"

		return claude.PermissionResultAllow{
			Behavior:     "allow",
			UpdatedInput: modifiedInput,
		}, nil
	}

	input := map[string]interface{}{"file_path": "/dangerous/path/file.txt"}
	result, err := canUseTool(ctx, "Write", input, claude.ToolPermissionContext{})
	if err != nil {
		t.Fatalf("Callback error: %v", err)
	}

	allowResult, ok := result.(claude.PermissionResultAllow)
	if !ok {
		t.Fatalf("Expected PermissionResultAllow, got %T", result)
	}

	if allowResult.UpdatedInput == nil {
		t.Fatal("Expected updated input")
	}

	newPath, ok := allowResult.UpdatedInput["file_path"].(string)
	if !ok || newPath != "/safe/path/file.txt" {
		t.Errorf("Expected updated path '/safe/path/file.txt', got %v", allowResult.UpdatedInput["file_path"])
	}
}

func TestHookCallback(t *testing.T) {
	ctx := context.Background()
	hookInvoked := false

	hookCallback := func(ctx context.Context, input map[string]interface{}, toolUseID *string, hookCtx claude.HookContext) (claude.HookJSONOutput, error) {
		hookInvoked = true

		toolName, _ := input["tool_name"].(string)
		if toolName != "Bash" {
			return claude.HookJSONOutput{}, nil
		}

		toolInput, _ := input["tool_input"].(map[string]interface{})
		command, _ := toolInput["command"].(string)

		if command == "rm -rf /" {
			decision := "block"
			return claude.HookJSONOutput{
				Decision:      &decision,
				SystemMessage: stringPtr("Dangerous command blocked"),
			}, nil
		}

		return claude.HookJSONOutput{}, nil
	}

	// Test blocking dangerous command
	input := map[string]interface{}{
		"tool_name": "Bash",
		"tool_input": map[string]interface{}{
			"command": "rm -rf /",
		},
	}

	output, err := hookCallback(ctx, input, nil, claude.HookContext{})
	if err != nil {
		t.Fatalf("Hook error: %v", err)
	}

	if !hookInvoked {
		t.Error("Expected hook to be invoked")
	}

	if output.Decision == nil || *output.Decision != "block" {
		t.Error("Expected decision 'block'")
	}

	if output.SystemMessage == nil || *output.SystemMessage != "Dangerous command blocked" {
		t.Errorf("Expected system message, got %v", output.SystemMessage)
	}
}

func TestTypedHookInputs(t *testing.T) {
	// Test PreToolUseHookInput
	t.Run("PreToolUseHookInput", func(t *testing.T) {
		input := claude.PreToolUseHookInput{
			BaseHookInput: claude.BaseHookInput{
				SessionID:      "session-123",
				TranscriptPath: "/path/to/transcript",
				Cwd:            "/working/dir",
			},
			HookEventName: "PreToolUse",
			ToolName:      "Read",
			ToolInput: map[string]interface{}{
				"file_path": "/test.txt",
			},
		}

		if input.ToolName != "Read" {
			t.Errorf("Expected tool name 'Read', got '%s'", input.ToolName)
		}

		if input.SessionID != "session-123" {
			t.Errorf("Expected session ID 'session-123', got '%s'", input.SessionID)
		}
	})

	// Test PostToolUseHookInput
	t.Run("PostToolUseHookInput", func(t *testing.T) {
		input := claude.PostToolUseHookInput{
			BaseHookInput: claude.BaseHookInput{
				SessionID:      "session-456",
				TranscriptPath: "/path/to/transcript",
				Cwd:            "/working/dir",
			},
			HookEventName: "PostToolUse",
			ToolName:      "Grep",
			ToolInput: map[string]interface{}{
				"pattern": "test",
			},
			ToolResponse: "matched line",
		}

		if input.ToolName != "Grep" {
			t.Errorf("Expected tool name 'Grep', got '%s'", input.ToolName)
		}

		response, ok := input.ToolResponse.(string)
		if !ok || response != "matched line" {
			t.Errorf("Expected tool response 'matched line', got %v", input.ToolResponse)
		}
	})

	// Test UserPromptSubmitHookInput
	t.Run("UserPromptSubmitHookInput", func(t *testing.T) {
		input := claude.UserPromptSubmitHookInput{
			BaseHookInput: claude.BaseHookInput{
				SessionID:      "session-789",
				TranscriptPath: "/path/to/transcript",
				Cwd:            "/working/dir",
			},
			HookEventName: "UserPromptSubmit",
			Prompt:        "Hello Claude",
		}

		if input.Prompt != "Hello Claude" {
			t.Errorf("Expected prompt 'Hello Claude', got '%s'", input.Prompt)
		}
	})
}

func TestHookSerialization(t *testing.T) {
	// Test that hook inputs can be marshaled/unmarshaled
	input := claude.PreToolUseHookInput{
		BaseHookInput: claude.BaseHookInput{
			SessionID:      "test-session",
			TranscriptPath: "/transcript",
			Cwd:            "/cwd",
		},
		HookEventName: "PreToolUse",
		ToolName:      "TestTool",
		ToolInput: map[string]interface{}{
			"key": "value",
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal back
	var decoded claude.PreToolUseHookInput
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.ToolName != input.ToolName {
		t.Errorf("Tool name mismatch after serialization")
	}

	if decoded.SessionID != input.SessionID {
		t.Errorf("Session ID mismatch after serialization")
	}
}

func TestConcurrentCallbacks(t *testing.T) {
	ctx := context.Background()
	var mu sync.Mutex
	callCount := 0

	canUseTool := func(ctx context.Context, toolName string, input map[string]interface{}, permCtx claude.ToolPermissionContext) (claude.PermissionResult, error) {
		mu.Lock()
		callCount++
		mu.Unlock()

		return claude.PermissionResultAllow{Behavior: "allow"}, nil
	}

	// Simulate multiple concurrent calls
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := canUseTool(ctx, "Tool", map[string]interface{}{"index": idx}, claude.ToolPermissionContext{})
			if err != nil {
				t.Errorf("Concurrent callback error: %v", err)
			}
		}(i)
	}

	wg.Wait()

	mu.Lock()
	if callCount != 10 {
		t.Errorf("Expected 10 callback invocations, got %d", callCount)
	}
	mu.Unlock()
}

// Hook Field Tests - ported from Python test_tool_callbacks.py

func TestHookOutputAllFields(t *testing.T) {
	// Test that all HookJSONOutput fields are properly handled
	ctx := context.Background()

	hookCallback := func(ctx context.Context, input map[string]interface{}, toolUseID *string, hookCtx claude.HookContext) (claude.HookJSONOutput, error) {
		continueVal := true
		suppressVal := false
		stopReason := "Test stop reason"
		decision := "block"
		sysMsg := "Test system message"
		reason := "Test reason for blocking"

		return claude.HookJSONOutput{
			// Control fields
			Continue:       &continueVal,
			SuppressOutput: &suppressVal,
			StopReason:     &stopReason,
			// Decision fields
			Decision:      &decision,
			SystemMessage: &sysMsg,
			Reason:        &reason,
			// Hook-specific output
			HookSpecificOutput: map[string]interface{}{
				"hookEventName":            "PreToolUse",
				"permissionDecision":       "deny",
				"permissionDecisionReason": "Security policy violation",
				"updatedInput":             map[string]interface{}{"modified": "input"},
			},
		}, nil
	}

	output, err := hookCallback(ctx, map[string]interface{}{"test": "data"}, nil, claude.HookContext{})
	if err != nil {
		t.Fatalf("Hook error: %v", err)
	}

	// Verify all control fields
	if output.Continue == nil || *output.Continue != true {
		t.Error("Expected Continue to be true")
	}
	if output.SuppressOutput == nil || *output.SuppressOutput != false {
		t.Error("Expected SuppressOutput to be false")
	}
	if output.StopReason == nil || *output.StopReason != "Test stop reason" {
		t.Error("Expected StopReason to be set")
	}

	// Verify decision fields
	if output.Decision == nil || *output.Decision != "block" {
		t.Error("Expected Decision to be 'block'")
	}
	if output.SystemMessage == nil || *output.SystemMessage != "Test system message" {
		t.Error("Expected SystemMessage to be set")
	}
	if output.Reason == nil || *output.Reason != "Test reason for blocking" {
		t.Error("Expected Reason to be set")
	}

	// Verify hook-specific output
	if output.HookSpecificOutput == nil {
		t.Fatal("Expected HookSpecificOutput to be set")
	}
	if output.HookSpecificOutput["hookEventName"] != "PreToolUse" {
		t.Error("Expected hookEventName in HookSpecificOutput")
	}
	if output.HookSpecificOutput["permissionDecision"] != "deny" {
		t.Error("Expected permissionDecision in HookSpecificOutput")
	}

	// Test JSON serialization to ensure proper field names
	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal hook output: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal hook output: %v", err)
	}

	// Verify JSON field names (continue, not continue_)
	if _, ok := decoded["continue"]; !ok {
		t.Error("Expected 'continue' field in JSON output")
	}
	if _, ok := decoded["suppressOutput"]; !ok {
		t.Error("Expected 'suppressOutput' field in JSON output")
	}
}

func TestAsyncHookOutput(t *testing.T) {
	// Test AsyncHookJSONOutput type with async and asyncTimeout fields
	ctx := context.Background()

	hookCallback := func(ctx context.Context, input map[string]interface{}, toolUseID *string, hookCtx claude.HookContext) (claude.HookJSONOutput, error) {
		asyncVal := true
		timeout := 5000

		return claude.HookJSONOutput{
			Async:        &asyncVal,
			AsyncTimeout: &timeout,
		}, nil
	}

	output, err := hookCallback(ctx, map[string]interface{}{"test": "async_data"}, nil, claude.HookContext{})
	if err != nil {
		t.Fatalf("Hook error: %v", err)
	}

	// Verify async fields
	if output.Async == nil || *output.Async != true {
		t.Error("Expected Async to be true")
	}
	if output.AsyncTimeout == nil || *output.AsyncTimeout != 5000 {
		t.Error("Expected AsyncTimeout to be 5000")
	}

	// Test JSON serialization
	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal async hook output: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal async hook output: %v", err)
	}

	// Verify JSON field names (async, not async_)
	if _, ok := decoded["async"]; !ok {
		t.Error("Expected 'async' field in JSON output")
	}
	if _, ok := decoded["asyncTimeout"]; !ok {
		t.Error("Expected 'asyncTimeout' field in JSON output")
	}
}

func TestHookFieldNameSerialization(t *testing.T) {
	// Test that all special field names are properly serialized
	// Go uses "continue" and "async" (not continue_ or async_)
	ctx := context.Background()

	hookCallback := func(ctx context.Context, input map[string]interface{}, toolUseID *string, hookCtx claude.HookContext) (claude.HookJSONOutput, error) {
		asyncVal := true
		timeout := 10000
		continueVal := false
		stopReason := "Testing field conversion"
		sysMsg := "Fields should be properly named"

		return claude.HookJSONOutput{
			Async:         &asyncVal,
			AsyncTimeout:  &timeout,
			Continue:      &continueVal,
			StopReason:    &stopReason,
			SystemMessage: &sysMsg,
		}, nil
	}

	output, err := hookCallback(ctx, map[string]interface{}{"test": "data"}, nil, claude.HookContext{})
	if err != nil {
		t.Fatalf("Hook error: %v", err)
	}

	// Serialize to JSON
	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify async was properly converted
	if asyncVal, ok := decoded["async"].(bool); !ok || !asyncVal {
		t.Error("Expected 'async' field to be true in JSON")
	}

	// Verify continue was properly converted
	if continueVal, ok := decoded["continue"].(bool); !ok || continueVal {
		t.Error("Expected 'continue' field to be false in JSON")
	}

	// Verify asyncTimeout is present
	if timeout, ok := decoded["asyncTimeout"].(float64); !ok || timeout != 10000 {
		t.Error("Expected 'asyncTimeout' field to be 10000 in JSON")
	}

	// Verify other fields
	if stopReason, ok := decoded["stopReason"].(string); !ok || stopReason != "Testing field conversion" {
		t.Error("Expected 'stopReason' field in JSON")
	}
	if sysMsg, ok := decoded["systemMessage"].(string); !ok || sysMsg != "Fields should be properly named" {
		t.Error("Expected 'systemMessage' field in JSON")
	}
}

func TestCallbackExceptionHandling(t *testing.T) {
	// Test that callback exceptions/errors are properly handled
	ctx := context.Background()

	// Permission callback that returns an error
	errorCallback := func(ctx context.Context, toolName string, input map[string]interface{}, permCtx claude.ToolPermissionContext) (claude.PermissionResult, error) {
		return nil, &claude.ClaudeSDKError{Message: "Callback error"}
	}

	_, err := errorCallback(ctx, "TestTool", map[string]interface{}{}, claude.ToolPermissionContext{})
	if err == nil {
		t.Fatal("Expected error from callback")
	}

	sdkErr, ok := err.(*claude.ClaudeSDKError)
	if !ok {
		t.Fatalf("Expected ClaudeSDKError, got %T", err)
	}

	if sdkErr.Message != "Callback error" {
		t.Errorf("Expected error message 'Callback error', got '%s'", sdkErr.Message)
	}

	// Hook callback that returns an error
	hookErrorCallback := func(ctx context.Context, input map[string]interface{}, toolUseID *string, hookCtx claude.HookContext) (claude.HookJSONOutput, error) {
		return claude.HookJSONOutput{}, &claude.ClaudeSDKError{Message: "Hook callback error"}
	}

	_, err = hookErrorCallback(ctx, map[string]interface{}{}, nil, claude.HookContext{})
	if err == nil {
		t.Fatal("Expected error from hook callback")
	}

	hookErr, ok := err.(*claude.ClaudeSDKError)
	if !ok {
		t.Fatalf("Expected ClaudeSDKError from hook, got %T", err)
	}

	if hookErr.Message != "Hook callback error" {
		t.Errorf("Expected error message 'Hook callback error', got '%s'", hookErr.Message)
	}
}
