package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	claude "github.com/clsx524/claude-agent-sdk-go"
)

func displayMessage(msg claude.Message) {
	if assistantMsg, ok := msg.(*claude.AssistantMessage); ok {
		for _, block := range assistantMsg.Content {
			if textBlock, ok := block.(claude.TextBlock); ok {
				fmt.Printf("Claude: %s\n", textBlock.Text)
			}
		}
	} else if _, ok := msg.(*claude.ResultMessage); ok {
		fmt.Println("Result ended")
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <example_name>")
		fmt.Println("\nAvailable examples:")
		fmt.Println("  all              - Run all examples")
		fmt.Println("  PreToolUse       - Block commands using PreToolUse hook")
		fmt.Println("  UserPromptSubmit - Add context at prompt submission")
		fmt.Println("  PostToolUse      - Review tool output with reason and systemMessage")
		fmt.Println("  DecisionFields   - Use permissionDecision='allow'/'deny' with reason")
		fmt.Println("  ContinueControl  - Control execution with continue and stopReason")
		os.Exit(0)
	}

	exampleName := os.Args[1]

	examples := map[string]func(){
		"PreToolUse":       examplePreToolUse,
		"UserPromptSubmit": exampleUserPromptSubmit,
		"PostToolUse":      examplePostToolUse,
		"DecisionFields":   exampleDecisionFields,
		"ContinueControl":  exampleContinueControl,
	}

	if exampleName == "all" {
		for name, fn := range examples {
			fmt.Printf("\n=== Running %s ===\n", name)
			fn()
			fmt.Println(strings.Repeat("-", 50))
		}
	} else if fn, ok := examples[exampleName]; ok {
		fn()
	} else {
		fmt.Printf("Error: Unknown example '%s'\n", exampleName)
		fmt.Println("\nAvailable examples:")
		fmt.Println("  all - Run all examples")
		for name := range examples {
			fmt.Printf("  %s\n", name)
		}
		os.Exit(1)
	}
}

// Example 1: PreToolUse hook to block dangerous commands
func examplePreToolUse() {
	fmt.Println("=== PreToolUse Example ===")
	fmt.Println("This example demonstrates how PreToolUse can block some bash commands but not others.")

	// Create a hook that blocks certain bash commands
	checkBashCommand := func(ctx context.Context, input map[string]interface{}, toolUseID *string, hookCtx claude.HookContext) (claude.HookJSONOutput, error) {
		toolName, _ := input["tool_name"].(string)
		toolInput, _ := input["tool_input"].(map[string]interface{})

		if toolName != "Bash" {
			return claude.HookJSONOutput{}, nil
		}

		command, _ := toolInput["command"].(string)
		blockPatterns := []string{"foo.sh"}

		for _, pattern := range blockPatterns {
			if strings.Contains(command, pattern) {
				log.Printf("Blocked command: %s", command)
				return claude.HookJSONOutput{
					HookSpecificOutput: map[string]interface{}{
						"hookEventName":            "PreToolUse",
						"permissionDecision":       "deny",
						"permissionDecisionReason": fmt.Sprintf("Command contains invalid pattern: %s", pattern),
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
				{Matcher: "Bash", Hooks: []claude.HookCallback{checkBashCommand}},
			},
		},
	}

	ctx := context.Background()
	client := claude.NewClaudeSDKClient(options)

	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	// Test 1: Command with forbidden pattern (will be blocked)
	fmt.Println("Test 1: Trying a command that our PreToolUse hook should block...")
	fmt.Println("User: Run the bash command: ./foo.sh --help")
	msgCh, errCh := client.Query(ctx, "Run the bash command: ./foo.sh --help")
	for msg := range msgCh {
		displayMessage(msg)
	}
	if err := <-errCh; err != nil {
		log.Printf("Error: %v", err)
	}

	fmt.Println("\n" + strings.Repeat("=", 50) + "\n")

	// Test 2: Safe command that should work
	fmt.Println("Test 2: Trying a command that our PreToolUse hook should allow...")
	fmt.Println("User: Run the bash command: echo 'Hello from hooks example!'")
	msgCh2, errCh2 := client.Query(ctx, "Run the bash command: echo 'Hello from hooks example!'")
	for msg := range msgCh2 {
		displayMessage(msg)
	}
	if err := <-errCh2; err != nil {
		log.Printf("Error: %v", err)
	}

	fmt.Println()
}

// Example 2: UserPromptSubmit hook to add context
func exampleUserPromptSubmit() {
	fmt.Println("=== UserPromptSubmit Example ===")
	fmt.Println("This example shows how a UserPromptSubmit hook can add context.")

	addCustomInstructions := func(ctx context.Context, input map[string]interface{}, toolUseID *string, hookCtx claude.HookContext) (claude.HookJSONOutput, error) {
		return claude.HookJSONOutput{
			HookSpecificOutput: map[string]interface{}{
				"hookEventName":     "SessionStart",
				"additionalContext": "My favorite color is hot pink",
			},
		}, nil
	}

	options := &claude.ClaudeAgentOptions{
		Hooks: map[claude.HookEvent][]claude.HookMatcher{
			claude.HookEventUserPromptSubmit: {
				{Matcher: "", Hooks: []claude.HookCallback{addCustomInstructions}},
			},
		},
	}

	ctx := context.Background()
	client := claude.NewClaudeSDKClient(options)

	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	fmt.Println("User: What's my favorite color?")
	msgCh, errCh := client.Query(ctx, "What's my favorite color?")
	for msg := range msgCh {
		displayMessage(msg)
	}
	if err := <-errCh; err != nil {
		log.Printf("Error: %v", err)
	}

	fmt.Println()
}

// Example 3: PostToolUse hook to review tool output
func examplePostToolUse() {
	fmt.Println("=== PostToolUse Example ===")
	fmt.Println("This example shows how PostToolUse can provide feedback with reason and systemMessage.")

	reviewToolOutput := func(ctx context.Context, input map[string]interface{}, toolUseID *string, hookCtx claude.HookContext) (claude.HookJSONOutput, error) {
		toolResponse, _ := input["tool_response"]

		// If the tool produced an error, add helpful context
		if strings.Contains(strings.ToLower(fmt.Sprint(toolResponse)), "error") {
			reason := "Tool execution failed - consider checking the command syntax"
			systemMessage := "⚠️ The command produced an error"

			return claude.HookJSONOutput{
				SystemMessage: &systemMessage,
				Reason:        &reason,
				HookSpecificOutput: map[string]interface{}{
					"hookEventName":     "PostToolUse",
					"additionalContext": "The command encountered an error. You may want to try a different approach.",
				},
			}, nil
		}

		return claude.HookJSONOutput{}, nil
	}

	options := &claude.ClaudeAgentOptions{
		AllowedTools: []string{"Bash"},
		Hooks: map[claude.HookEvent][]claude.HookMatcher{
			claude.HookEventPostToolUse: {
				{Matcher: "Bash", Hooks: []claude.HookCallback{reviewToolOutput}},
			},
		},
	}

	ctx := context.Background()
	client := claude.NewClaudeSDKClient(options)

	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	fmt.Println("User: Run a command that will produce an error: ls /nonexistent_directory")
	msgCh, errCh := client.Query(ctx, "Run this command: ls /nonexistent_directory")
	for msg := range msgCh {
		displayMessage(msg)
	}
	if err := <-errCh; err != nil {
		log.Printf("Error: %v", err)
	}

	fmt.Println()
}

// Example 4: Using permissionDecision with reason and systemMessage
func exampleDecisionFields() {
	fmt.Println("=== Permission Decision Example ===")
	fmt.Println("This example shows how to use permissionDecision='allow'/'deny' with reason and systemMessage.")

	strictApprovalHook := func(ctx context.Context, input map[string]interface{}, toolUseID *string, hookCtx claude.HookContext) (claude.HookJSONOutput, error) {
		toolName, _ := input["tool_name"].(string)
		toolInput, _ := input["tool_input"].(map[string]interface{})

		// Block any Write operations to specific files
		if toolName == "Write" {
			filePath, _ := toolInput["file_path"].(string)
			if strings.Contains(strings.ToLower(filePath), "important") {
				log.Printf("Blocked Write to: %s", filePath)
				reason := "Writes to files containing 'important' in the name are not allowed for safety"
				systemMessage := "🚫 Write operation blocked by security policy"

				return claude.HookJSONOutput{
					Reason:        &reason,
					SystemMessage: &systemMessage,
					HookSpecificOutput: map[string]interface{}{
						"hookEventName":            "PreToolUse",
						"permissionDecision":       "deny",
						"permissionDecisionReason": "Security policy blocks writes to important files",
					},
				}, nil
			}
		}

		// Allow everything else explicitly
		reason := "Tool use approved after security review"
		return claude.HookJSONOutput{
			Reason: &reason,
			HookSpecificOutput: map[string]interface{}{
				"hookEventName":            "PreToolUse",
				"permissionDecision":       "allow",
				"permissionDecisionReason": "Tool passed security checks",
			},
		}, nil
	}

	model := "claude-sonnet-4-5-20250929"
	options := &claude.ClaudeAgentOptions{
		AllowedTools: []string{"Write", "Bash"},
		Model:        &model,
		Hooks: map[claude.HookEvent][]claude.HookMatcher{
			claude.HookEventPreToolUse: {
				{Matcher: "Write", Hooks: []claude.HookCallback{strictApprovalHook}},
			},
		},
	}

	ctx := context.Background()
	client := claude.NewClaudeSDKClient(options)

	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	// Test 1: Try to write to a file with "important" in the name (should be blocked)
	fmt.Println("Test 1: Trying to write to important_config.txt (should be blocked)...")
	fmt.Println("User: Write 'test' to important_config.txt")
	msgCh, errCh := client.Query(ctx, "Write the text 'test data' to a file called important_config.txt")
	for msg := range msgCh {
		displayMessage(msg)
	}
	if err := <-errCh; err != nil {
		log.Printf("Error: %v", err)
	}

	fmt.Println("\n" + strings.Repeat("=", 50) + "\n")

	// Test 2: Write to a regular file (should be approved)
	fmt.Println("Test 2: Trying to write to regular_file.txt (should be approved)...")
	fmt.Println("User: Write 'test' to regular_file.txt")
	msgCh2, errCh2 := client.Query(ctx, "Write the text 'test data' to a file called regular_file.txt")
	for msg := range msgCh2 {
		displayMessage(msg)
	}
	if err := <-errCh2; err != nil {
		log.Printf("Error: %v", err)
	}

	fmt.Println()
}

// Example 5: Using continue and stopReason for execution control
func exampleContinueControl() {
	fmt.Println("=== Continue/Stop Control Example ===")
	fmt.Println("This example shows how to use Continue=false with StopReason to halt execution.")

	stopOnErrorHook := func(ctx context.Context, input map[string]interface{}, toolUseID *string, hookCtx claude.HookContext) (claude.HookJSONOutput, error) {
		toolResponse, _ := input["tool_response"]

		// Stop execution if we see a critical error
		if strings.Contains(strings.ToLower(fmt.Sprint(toolResponse)), "critical") {
			log.Println("Critical error detected - stopping execution")
			continueExec := false
			stopReason := "Critical error detected in tool output - execution halted for safety"
			systemMessage := "🛑 Execution stopped due to critical error"

			return claude.HookJSONOutput{
				Continue:      &continueExec,
				StopReason:    &stopReason,
				SystemMessage: &systemMessage,
			}, nil
		}

		continueExec := true
		return claude.HookJSONOutput{
			Continue: &continueExec,
		}, nil
	}

	options := &claude.ClaudeAgentOptions{
		AllowedTools: []string{"Bash"},
		Hooks: map[claude.HookEvent][]claude.HookMatcher{
			claude.HookEventPostToolUse: {
				{Matcher: "Bash", Hooks: []claude.HookCallback{stopOnErrorHook}},
			},
		},
	}

	ctx := context.Background()
	client := claude.NewClaudeSDKClient(options)

	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	fmt.Println("User: Run a command that outputs 'CRITICAL ERROR'")
	msgCh, errCh := client.Query(ctx, "Run this bash command: echo 'CRITICAL ERROR: system failure'")
	for msg := range msgCh {
		displayMessage(msg)
	}
	if err := <-errCh; err != nil {
		log.Printf("Error: %v", err)
	}

	fmt.Println()
}
