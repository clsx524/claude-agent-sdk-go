package e2e

import (
	"context"
	"sync"
	"testing"

	claude "github.com/clsx524/claude-agent-sdk-go"
)

func TestPermissionCallbackGetsCalled(t *testing.T) {
	RequireClaudeCode(t)

	callbackInvocations := []string{}
	var mu sync.Mutex

	// Track callback invocations
	permissionCallback := func(ctx context.Context, toolName string, inputData map[string]interface{}, permCtx claude.ToolPermissionContext) (claude.PermissionResult, error) {
		t.Logf("Permission callback called for: %s, input: %v", toolName, inputData)

		mu.Lock()
		callbackInvocations = append(callbackInvocations, toolName)
		mu.Unlock()

		return claude.PermissionResultAllow{}, nil
	}

	options := &claude.ClaudeAgentOptions{
		CanUseTool: permissionCallback,
	}

	client := claude.NewClaudeSDKClient(options)

	ctx := context.Background()
	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	msgCh, errCh := client.Query(ctx, "Use the Write tool to create a new file at /tmp/test_permissions.txt with the content 'test'. You must use the Write tool.")

	// Consume messages
	messages, err := CollectMessages(msgCh, errCh)
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}
	t.Logf("Received %d messages", len(messages))

	mu.Lock()
	invocations := callbackInvocations
	mu.Unlock()

	t.Logf("Callback invocations: %v", invocations)

	// Verify callback was invoked for at least one tool
	if len(invocations) == 0 {
		t.Errorf("can_use_tool callback should have been invoked for at least one tool, but got none")
	}

	// Check if Write was called (it should be for this prompt)
	foundWrite := false
	for _, inv := range invocations {
		if inv == "Write" || inv == "mcp__acp__Write" {
			foundWrite = true
			break
		}
	}

	if !foundWrite {
		t.Logf("Warning: Write tool was not used, but callback was invoked for: %v", invocations)
		// Don't fail the test - just verify the callback mechanism works
		if len(invocations) == 0 {
			t.Errorf("Callback mechanism is broken - no tools triggered the callback")
		}
	}
}
