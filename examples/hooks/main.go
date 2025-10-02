package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	claude "github.com/clsx524/claude-agent-sdk-go"
)

func main() {
	fmt.Println("=== Hooks Example ===")

	// Example 1: PreToolUse hook to block dangerous commands
	example1BlockDangerousCommands()

	// Example 2: UserPromptSubmit hook to add context
	fmt.Println("\n=== UserPromptSubmit Hook Example ===")
	example2AddContext()
}

func example1BlockDangerousCommands() {
	// Create a hook that blocks certain bash commands
	bashHook := func(ctx context.Context, input map[string]interface{}, toolUseID *string, hookCtx claude.HookContext) (claude.HookJSONOutput, error) {
		toolName, _ := input["tool_name"].(string)
		toolInput, _ := input["tool_input"].(map[string]interface{})

		// Only process Bash tool
		if toolName != "Bash" {
			return claude.HookJSONOutput{}, nil
		}

		command, _ := toolInput["command"].(string)

		// Block patterns
		blockPatterns := []string{
			"rm -rf",
			"sudo",
			"foo.sh", // From Python SDK example
		}

		for _, pattern := range blockPatterns {
			if strings.Contains(command, pattern) {
				decision := "block"
				reason := fmt.Sprintf("Command contains forbidden pattern: %s", pattern)

				return claude.HookJSONOutput{
					Decision: &decision,
					HookSpecificOutput: map[string]interface{}{
						"hookEventName":            "PreToolUse",
						"permissionDecision":       "deny",
						"permissionDecisionReason": reason,
					},
				}, nil
			}
		}

		return claude.HookJSONOutput{}, nil
	}

	options := &claude.ClaudeAgentOptions{
		AllowedTools: []string{"Bash"},
		Hooks: map[claude.HookEvent][]claude.HookMatcher{
			claude.HookEventPreToolUse: {
				{
					Matcher: "Bash",
					Hooks:   []claude.HookCallback{bashHook},
				},
			},
		},
	}

	ctx := context.Background()
	client := claude.NewClaudeSDKClient(options)

	if err := client.Connect(ctx, nil); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	// Test 1: Blocked command
	fmt.Println("Test 1: Trying to run blocked command (foo.sh)")
	if err := client.Query(ctx, "Run the bash command: ./foo.sh --help", "default"); err != nil {
		log.Printf("Failed to send query: %v", err)
	} else {
		for msg := range client.ReceiveResponse(ctx) {
			if assistantMsg, ok := msg.(*claude.AssistantMessage); ok {
				for _, block := range assistantMsg.Content {
					if textBlock, ok := block.(claude.TextBlock); ok {
						fmt.Printf("Claude: %s\n", textBlock.Text)
					}
				}
			}
		}
	}

	// Test 2: Allowed command
	fmt.Println("\nTest 2: Trying to run allowed command")
	if err := client.Query(ctx, "Run the bash command: echo 'Hello from hooks example!'", "default"); err != nil {
		log.Printf("Failed to send query: %v", err)
	} else {
		for msg := range client.ReceiveResponse(ctx) {
			if assistantMsg, ok := msg.(*claude.AssistantMessage); ok {
				for _, block := range assistantMsg.Content {
					if textBlock, ok := block.(claude.TextBlock); ok {
						fmt.Printf("Claude: %s\n", textBlock.Text)
					}
				}
			}
		}
	}
}

func example2AddContext() {
	// Create a hook that adds additional context to user prompts
	userPromptHook := func(ctx context.Context, input map[string]interface{}, toolUseID *string, hookCtx claude.HookContext) (claude.HookJSONOutput, error) {
		// Add timestamp and additional context
		additionalContext := "Current time: 2025-10-02. You are running in a Go SDK example."

		return claude.HookJSONOutput{
			HookSpecificOutput: map[string]interface{}{
				"additionalContext": additionalContext,
			},
		}, nil
	}

	maxTurns := 1
	options := &claude.ClaudeAgentOptions{
		MaxTurns: &maxTurns,
		Hooks: map[claude.HookEvent][]claude.HookMatcher{
			claude.HookEventUserPromptSubmit: {
				{
					Matcher: "", // Empty matcher matches all
					Hooks:   []claude.HookCallback{userPromptHook},
				},
			},
		},
	}

	ctx := context.Background()
	client := claude.NewClaudeSDKClient(options)

	if err := client.Connect(ctx, nil); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	if err := client.Query(ctx, "What time is it?", "default"); err != nil {
		log.Printf("Failed to send query: %v", err)
		return
	}

	for msg := range client.ReceiveResponse(ctx) {
		if assistantMsg, ok := msg.(*claude.AssistantMessage); ok {
			for _, block := range assistantMsg.Content {
				if textBlock, ok := block.(claude.TextBlock); ok {
					fmt.Printf("Claude: %s\n", textBlock.Text)
				}
			}
		}
	}
}

func examplePermissionCallback() {
	// Example with canUseTool callback
	canUseTool := func(ctx context.Context, toolName string, input map[string]interface{}, permCtx claude.ToolPermissionContext) (claude.PermissionResult, error) {
		fmt.Printf("Permission request for tool: %s\n", toolName)
		fmt.Printf("Input: %v\n", input)

		// Allow read-only operations
		readOnlyTools := []string{"Read", "Glob", "Grep"}
		for _, tool := range readOnlyTools {
			if tool == toolName {
				return claude.PermissionResultAllow{
					Behavior: "allow",
				}, nil
			}
		}

		// Block dangerous operations
		if toolName == "Bash" {
			command, _ := input["command"].(string)
			if strings.Contains(command, "rm") {
				return claude.PermissionResultDeny{
					Behavior: "deny",
					Message:  "Destructive commands are not allowed",
				}, nil
			}
		}

		// Allow everything else
		return claude.PermissionResultAllow{
			Behavior: "allow",
		}, nil
	}

	options := &claude.ClaudeAgentOptions{
		CanUseTool:   canUseTool,
		AllowedTools: []string{"Read", "Write", "Bash"},
	}

	ctx := context.Background()
	client := claude.NewClaudeSDKClient(options)

	if err := client.Connect(ctx, nil); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	fmt.Println("\n=== Permission Callback Example ===")

	if err := client.Query(ctx, "List the files in the current directory", "default"); err != nil {
		log.Printf("Failed to send query: %v", err)
		return
	}

	for msg := range client.ReceiveResponse(ctx) {
		if assistantMsg, ok := msg.(*claude.AssistantMessage); ok {
			for _, block := range assistantMsg.Content {
				if textBlock, ok := block.(claude.TextBlock); ok {
					fmt.Printf("Claude: %s\n", textBlock.Text)
				}
			}
		}
	}
}
