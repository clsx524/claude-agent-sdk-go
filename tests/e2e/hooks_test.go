package e2e

import (
	"context"
	"sync"
	"testing"

	claude "github.com/clsx524/claude-agent-sdk-go"
)

func TestHookWithPermissionDecisionAndReason(t *testing.T) {
	RequireClaudeCode(t)

	hookInvocations := []string{}
	var mu sync.Mutex

	// Hook that uses permissionDecision and reason fields
	testHook := func(ctx context.Context, input map[string]interface{}, toolUseID *string, hookCtx claude.HookContext) (claude.HookJSONOutput, error) {
		toolName := ""
		if name, ok := input["tool_name"].(string); ok {
			toolName = name
		}
		t.Logf("Hook called for tool: %s", toolName)

		mu.Lock()
		hookInvocations = append(hookInvocations, toolName)
		mu.Unlock()

		// Block Bash commands for this test
		if toolName == "Bash" {
			reason := "Bash commands are blocked in this test for safety"
			systemMsg := "⚠️ Command blocked by hook"
			return claude.HookJSONOutput{
				Reason:        &reason,
				SystemMessage: &systemMsg,
				HookSpecificOutput: map[string]interface{}{
					"hookEventName":            "PreToolUse",
					"permissionDecision":       "deny",
					"permissionDecisionReason": "Security policy: Bash blocked",
				},
			}, nil
		}

		reason := "Tool approved by security review"
		return claude.HookJSONOutput{
			Reason: &reason,
			HookSpecificOutput: map[string]interface{}{
				"hookEventName":            "PreToolUse",
				"permissionDecision":       "allow",
				"permissionDecisionReason": "Tool passed security checks",
			},
		}, nil
	}

	options := &claude.ClaudeAgentOptions{
		AllowedTools: []string{"Bash", "Write"},
		Hooks: map[claude.HookEvent][]claude.HookMatcher{
			claude.HookEventPreToolUse: {
				{
					Matcher: "Bash",
					Hooks:   []claude.HookCallback{testHook},
				},
			},
		},
	}

	client := claude.NewClaudeSDKClient(options)

	ctx := context.Background()
	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	msgCh, errCh := client.Query(ctx, "Run this bash command: echo 'hello'")

	messages, err := CollectMessages(msgCh, errCh)
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}
	t.Logf("Received %d messages", len(messages))

	mu.Lock()
	invocations := hookInvocations
	mu.Unlock()

	t.Logf("Hook invocations: %v", invocations)

	// Verify hook was called
	foundBash := false
	for _, inv := range invocations {
		if inv == "Bash" {
			foundBash = true
			break
		}
	}
	if !foundBash {
		t.Errorf("Hook should have been invoked for Bash tool, got: %v", invocations)
	}
}

func TestHookWithContinueAndStopReason(t *testing.T) {
	RequireClaudeCode(t)

	hookInvocations := []string{}
	var mu sync.Mutex

	// PostToolUse hook that stops execution with stopReason
	postToolHook := func(ctx context.Context, input map[string]interface{}, toolUseID *string, hookCtx claude.HookContext) (claude.HookJSONOutput, error) {
		toolName := ""
		if name, ok := input["tool_name"].(string); ok {
			toolName = name
		}

		mu.Lock()
		hookInvocations = append(hookInvocations, toolName)
		mu.Unlock()

		// Test continue_=False and stopReason fields
		continueFalse := false
		stopReason := "Execution halted by test hook for validation"
		reason := "Testing continue and stopReason fields"
		systemMsg := "🛑 Test hook stopped execution"
		return claude.HookJSONOutput{
			Continue:      &continueFalse,
			StopReason:    &stopReason,
			Reason:        &reason,
			SystemMessage: &systemMsg,
		}, nil
	}

	options := &claude.ClaudeAgentOptions{
		AllowedTools: []string{"Bash"},
		Hooks: map[claude.HookEvent][]claude.HookMatcher{
			claude.HookEventPostToolUse: {
				{
					Matcher: "Bash",
					Hooks:   []claude.HookCallback{postToolHook},
				},
			},
		},
	}

	client := claude.NewClaudeSDKClient(options)

	ctx := context.Background()
	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	msgCh, errCh := client.Query(ctx, "Run: echo 'test message'")

	messages, err := CollectMessages(msgCh, errCh)
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}
	t.Logf("Received %d messages", len(messages))

	mu.Lock()
	invocations := hookInvocations
	mu.Unlock()

	t.Logf("Hook invocations: %v", invocations)

	// Verify hook was called
	foundBash := false
	for _, inv := range invocations {
		if inv == "Bash" {
			foundBash = true
			break
		}
	}
	if !foundBash {
		t.Errorf("PostToolUse hook should have been invoked, got: %v", invocations)
	}
}

func TestHookWithAdditionalContext(t *testing.T) {
	RequireClaudeCode(t)

	hookInvocations := []string{}
	var mu sync.Mutex

	// Hook that provides additional context
	contextHook := func(ctx context.Context, input map[string]interface{}, toolUseID *string, hookCtx claude.HookContext) (claude.HookJSONOutput, error) {
		mu.Lock()
		hookInvocations = append(hookInvocations, "context_added")
		mu.Unlock()

		suppressOutput := false
		systemMsg := "Additional context provided by hook"
		reason := "Hook providing monitoring feedback"
		return claude.HookJSONOutput{
			SystemMessage:  &systemMsg,
			Reason:         &reason,
			SuppressOutput: &suppressOutput,
			HookSpecificOutput: map[string]interface{}{
				"hookEventName":     "PostToolUse",
				"additionalContext": "The command executed successfully with hook monitoring",
			},
		}, nil
	}

	options := &claude.ClaudeAgentOptions{
		AllowedTools: []string{"Bash"},
		Hooks: map[claude.HookEvent][]claude.HookMatcher{
			claude.HookEventPostToolUse: {
				{
					Matcher: "Bash",
					Hooks:   []claude.HookCallback{contextHook},
				},
			},
		},
	}

	client := claude.NewClaudeSDKClient(options)

	ctx := context.Background()
	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	msgCh, errCh := client.Query(ctx, "Run: echo 'testing hooks'")

	messages, err := CollectMessages(msgCh, errCh)
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}
	t.Logf("Received %d messages", len(messages))

	mu.Lock()
	invocations := hookInvocations
	mu.Unlock()

	t.Logf("Hook invocations: %v", invocations)

	// Verify hook was called
	foundContext := false
	for _, inv := range invocations {
		if inv == "context_added" {
			foundContext = true
			break
		}
	}
	if !foundContext {
		t.Error("Hook with hookSpecificOutput should have been invoked")
	}
}
