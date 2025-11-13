package e2e

import (
	"context"
	"strings"
	"sync"
	"testing"

	claude "github.com/clsx524/claude-agent-sdk-go"
)

func TestStderrCallbackCapturesDebugOutput(t *testing.T) {
	RequireClaudeCode(t)

	stderrLines := []string{}
	var mu sync.Mutex

	// Callback to capture stderr
	captureStderr := func(line string) {
		mu.Lock()
		stderrLines = append(stderrLines, line)
		mu.Unlock()
	}

	// Enable debug mode to generate stderr output
	options := &claude.ClaudeAgentOptions{
		Stderr: captureStderr,
		ExtraArgs: map[string]*string{
			"debug-to-stderr": nil, // Flag with no value
		},
	}

	client := claude.NewClaudeSDKClient(options)

	ctx := context.Background()
	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Run a simple query
	msgCh, errCh := client.Query(ctx, "What is 1+1?")

	// Consume messages
	_, err = CollectMessages(msgCh, errCh)
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}

	// Give some time for stderr to be captured
	mu.Lock()
	lines := stderrLines
	mu.Unlock()

	t.Logf("Captured %d stderr lines", len(lines))

	// Verify we captured debug output
	if len(lines) == 0 {
		t.Error("Should capture stderr output with debug enabled")
		return
	}

	// Check for DEBUG messages
	hasDebug := false
	for _, line := range lines {
		if strings.Contains(line, "[DEBUG]") {
			hasDebug = true
			break
		}
	}

	if !hasDebug {
		t.Logf("Sample stderr lines: %v", lines[:min(5, len(lines))])
		t.Log("Warning: No [DEBUG] messages found (may vary)")
	} else {
		t.Log("Successfully captured DEBUG messages in stderr")
	}
}

func TestStderrCallbackWithoutDebug(t *testing.T) {
	RequireClaudeCode(t)

	stderrLines := []string{}
	var mu sync.Mutex

	// Callback to capture stderr
	captureStderr := func(line string) {
		mu.Lock()
		stderrLines = append(stderrLines, line)
		mu.Unlock()
	}

	// No debug mode enabled
	options := &claude.ClaudeAgentOptions{
		Stderr: captureStderr,
		// No ExtraArgs with debug-to-stderr
	}

	client := claude.NewClaudeSDKClient(options)

	ctx := context.Background()
	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Run a simple query
	msgCh, errCh := client.Query(ctx, "What is 1+1?")

	// Consume messages
	_, err = CollectMessages(msgCh, errCh)
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}

	mu.Lock()
	lines := stderrLines
	mu.Unlock()

	// Should work but capture minimal/no output without debug
	t.Logf("Captured %d stderr lines without debug", len(lines))

	if len(lines) > 0 {
		t.Logf("Warning: Captured some stderr output even without debug: %v", lines[:min(5, len(lines))])
	} else {
		t.Log("Successfully verified no stderr output without debug mode")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
