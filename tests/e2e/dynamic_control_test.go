package e2e

import (
	"context"
	"testing"

	claude "github.com/clsx524/claude-agent-sdk-go"
)

func TestSetPermissionMode(t *testing.T) {
	RequireClaudeCode(t)

	// Create client with default permission mode
	permissionMode := claude.PermissionModeDefault
	options := &claude.ClaudeAgentOptions{
		PermissionMode: &permissionMode,
	}

	client := claude.NewClaudeSDKClient(options)

	ctx := context.Background()
	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Change permission mode to acceptEdits
	err = client.SetPermissionMode(ctx, claude.PermissionModeAcceptEdits)
	if err != nil {
		t.Fatalf("Failed to set permission mode to acceptEdits: %v", err)
	}
	t.Log("Changed permission mode to acceptEdits")

	// Make a query that would normally require permission
	msgCh1, errCh1 := client.Query(ctx, "What is 2+2? Just respond with the number.")

	// MUST fully consume channels before next Query()
	messages1, err := CollectMessages(msgCh1, errCh1)
	if err != nil {
		t.Fatalf("First query error: %v", err)
	}
	t.Logf("Received %d messages with acceptEdits mode", len(messages1))

	// Change back to default
	err = client.SetPermissionMode(ctx, claude.PermissionModeDefault)
	if err != nil {
		t.Fatalf("Failed to set permission mode to default: %v", err)
	}
	t.Log("Changed permission mode back to default")

	// Make another query - first query channels must be fully consumed
	msgCh2, errCh2 := client.Query(ctx, "What is 3+3? Just respond with the number.")

	// Consume all messages from second query
	messages2, err := CollectMessages(msgCh2, errCh2)
	if err != nil {
		t.Fatalf("Second query error: %v", err)
	}
	t.Logf("Received %d messages with default mode", len(messages2))

	t.Log("Successfully changed permission mode dynamically")
}

func TestSetModel(t *testing.T) {
	RequireClaudeCode(t)

	// Create client with default options
	options := &claude.ClaudeAgentOptions{}

	client := claude.NewClaudeSDKClient(options)

	ctx := context.Background()
	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Start with default model
	msgCh1, errCh1 := client.Query(ctx, "What is 1+1? Just the number.")

	// MUST fully consume channels before next Query()
	messages1, err := CollectMessages(msgCh1, errCh1)
	if err != nil {
		t.Fatalf("First query error: %v", err)
	}
	t.Logf("Default model response: received %d messages", len(messages1))

	// Switch to Haiku model
	err = client.SetModel(ctx, "claude-3-5-haiku-20241022")
	if err != nil {
		t.Fatalf("Failed to set model to Haiku: %v", err)
	}
	t.Log("Changed model to claude-3-5-haiku-20241022")

	msgCh2, errCh2 := client.Query(ctx, "What is 2+2? Just the number.")

	messages2, err := CollectMessages(msgCh2, errCh2)
	if err != nil {
		t.Fatalf("Second query error: %v", err)
	}
	t.Logf("Haiku model response: received %d messages", len(messages2))

	// Switch back to default (empty string means default)
	err = client.SetModel(ctx, "")
	if err != nil {
		t.Fatalf("Failed to set model back to default: %v", err)
	}
	t.Log("Changed model back to default")

	msgCh3, errCh3 := client.Query(ctx, "What is 3+3? Just the number.")

	messages3, err := CollectMessages(msgCh3, errCh3)
	if err != nil {
		t.Fatalf("Third query error: %v", err)
	}
	t.Logf("Back to default model: received %d messages", len(messages3))

	t.Log("Successfully changed model dynamically")
}

func TestInterrupt(t *testing.T) {
	RequireClaudeCode(t)

	// Create client with default options
	options := &claude.ClaudeAgentOptions{}

	client := claude.NewClaudeSDKClient(options)

	ctx := context.Background()
	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Start a query
	msgCh, errCh := client.Query(ctx, "Count from 1 to 100 slowly.")

	// Send interrupt (may or may not stop the response depending on timing)
	err = client.Interrupt(ctx)
	if err != nil {
		t.Logf("Interrupt resulted in error (this is expected): %v", err)
	} else {
		t.Log("Interrupt sent successfully")
	}

	// Consume any remaining messages
	messageCount := 0
	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				t.Logf("Received %d messages before/after interrupt", messageCount)
				return
			}
			messageCount++
			t.Logf("Got message after interrupt: type=%T", msg)

		case err := <-errCh:
			if err != nil {
				// Interrupt may cause an error, which is expected
				t.Logf("Query ended with error (expected after interrupt): %v", err)
				return
			}
		}
	}
}
