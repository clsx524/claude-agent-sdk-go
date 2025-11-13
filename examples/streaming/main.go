package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	claude "github.com/clsx524/claude-agent-sdk-go"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <example_number>")
		fmt.Println("\nAvailable examples:")
		fmt.Println("  1  - Basic streaming")
		fmt.Println("  2  - Multi-turn conversation")
		fmt.Println("  3  - Concurrent send/receive")
		fmt.Println("  4  - With interrupt")
		fmt.Println("  5  - Manual message handling")
		fmt.Println("  6  - With options")
		fmt.Println("  7  - Async iterable prompt (channel-based)")
		fmt.Println("  8  - Bash command")
		fmt.Println("  9  - Control protocol (server info, interrupt)")
		fmt.Println("  10 - Error handling")
		fmt.Println("  all - Run all examples")
		os.Exit(0)
	}

	example := os.Args[1]

	examples := map[string]func(){
		"1":  example1BasicStreaming,
		"2":  example2MultiTurnConversation,
		"3":  example3ConcurrentResponses,
		"4":  example4WithInterrupt,
		"5":  example5ManualMessageHandling,
		"6":  example6WithOptions,
		"7":  example7AsyncIterablePrompt,
		"8":  example8BashCommand,
		"9":  example9ControlProtocol,
		"10": example10ErrorHandling,
	}

	if example == "all" {
		for i := 1; i <= 10; i++ {
			fmt.Printf("\n=== Example %d ===\n", i)
			examples[fmt.Sprintf("%d", i)]()
			fmt.Println(strings.Repeat("-", 50))
		}
	} else if fn, ok := examples[example]; ok {
		fn()
	} else {
		fmt.Printf("Unknown example: %s\n", example)
		os.Exit(1)
	}
}

// Example 1: Basic streaming
func example1BasicStreaming() {
	fmt.Println("=== Basic Streaming Example ===")

	ctx := context.Background()
	client := claude.NewClaudeSDKClient(nil)

	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	msgCh, errCh := client.Query(ctx, "What is the capital of France?")

	for msg := range msgCh {
		switch m := msg.(type) {
		case *claude.AssistantMessage:
			for _, block := range m.Content {
				if textBlock, ok := block.(claude.TextBlock); ok {
					fmt.Printf("Claude: %s\n", textBlock.Text)
				}
			}
		case *claude.ResultMessage:
			fmt.Printf("Completed in %dms\n", m.DurationMS)
		}
	}

	if err := <-errCh; err != nil {
		log.Printf("Error: %v", err)
	}
}

// Example 2: Multi-turn conversation
func example2MultiTurnConversation() {
	fmt.Println("=== Multi-Turn Conversation ===")

	ctx := context.Background()
	client := claude.NewClaudeSDKClient(nil)

	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	// Turn 1
	fmt.Println("Turn 1:")
	msgCh, errCh := client.Query(ctx, "Hello! What's your name?")
	for msg := range msgCh {
		if assistantMsg, ok := msg.(*claude.AssistantMessage); ok {
			for _, block := range assistantMsg.Content {
				if textBlock, ok := block.(claude.TextBlock); ok {
					fmt.Printf("Claude: %s\n", textBlock.Text)
				}
			}
		}
	}
	if err := <-errCh; err != nil {
		log.Printf("Error: %v", err)
	}

	// Turn 2
	fmt.Println("\nTurn 2:")
	msgCh2, errCh2 := client.Query(ctx, "Can you help me with Go programming?")
	for msg := range msgCh2 {
		if assistantMsg, ok := msg.(*claude.AssistantMessage); ok {
			for _, block := range assistantMsg.Content {
				if textBlock, ok := block.(claude.TextBlock); ok {
					fmt.Printf("Claude: %s\n", textBlock.Text)
				}
			}
		}
	}
	if err := <-errCh2; err != nil {
		log.Printf("Error: %v", err)
	}
}

// Example 3: Concurrent send/receive
func example3ConcurrentResponses() {
	fmt.Println("=== Concurrent Send/Receive ===")

	ctx := context.Background()
	client := claude.NewClaudeSDKClient(nil)

	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	// Start receiving in background
	receiveCh := client.ReceiveMessages(ctx)
	go func() {
		for msg := range receiveCh {
			if assistantMsg, ok := msg.(*claude.AssistantMessage); ok {
				for _, block := range assistantMsg.Content {
					if textBlock, ok := block.(claude.TextBlock); ok {
						fmt.Printf("Claude: %s\n", textBlock.Text)
					}
				}
			}
		}
	}()

	// Send multiple queries
	msgCh1, errCh1 := client.Query(ctx, "What is 2+2?")
	for range msgCh1 {
	}
	if err := <-errCh1; err != nil {
		log.Printf("Error: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
}

// Example 4: With interrupt
func example4WithInterrupt() {
	fmt.Println("=== With Interrupt ===")

	ctx := context.Background()
	client := claude.NewClaudeSDKClient(nil)

	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	// Start a long-running query
	msgCh, errCh := client.Query(ctx, "Count to 1000 slowly")

	// Start receiving
	go func() {
		for msg := range msgCh {
			if assistantMsg, ok := msg.(*claude.AssistantMessage); ok {
				for _, block := range assistantMsg.Content {
					if textBlock, ok := block.(claude.TextBlock); ok {
						fmt.Printf("Claude: %s\n", textBlock.Text)
					}
				}
			}
		}
	}()

	// Interrupt after 1 second
	time.Sleep(1 * time.Second)
	fmt.Println("\nSending interrupt...")
	if err := client.Interrupt(ctx); err != nil {
		log.Printf("Interrupt error: %v", err)
	}

	if err := <-errCh; err != nil {
		log.Printf("Error: %v", err)
	}
}

// Example 5: Manual message handling
func example5ManualMessageHandling() {
	fmt.Println("=== Manual Message Handling ===")

	ctx := context.Background()
	client := claude.NewClaudeSDKClient(nil)

	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	msgCh, errCh := client.Query(ctx, "What is Go?")

	var assistantMessages []*claude.AssistantMessage
	var resultMsg *claude.ResultMessage

	for msg := range msgCh {
		switch m := msg.(type) {
		case *claude.AssistantMessage:
			assistantMessages = append(assistantMessages, m)
			fmt.Printf("Received assistant message from model: %s\n", m.Model)
		case *claude.ResultMessage:
			resultMsg = m
			fmt.Printf("Result: %s, Duration: %dms\n", m.Subtype, m.DurationMS)
		case *claude.SystemMessage:
			fmt.Printf("System message: %s\n", m.Subtype)
		}
	}

	if err := <-errCh; err != nil {
		log.Printf("Error: %v", err)
	}

	fmt.Printf("\nTotal assistant messages: %d\n", len(assistantMessages))
	if resultMsg != nil && resultMsg.TotalCostUSD != nil {
		fmt.Printf("Total cost: $%.4f\n", *resultMsg.TotalCostUSD)
	}
}

// Example 6: With options
func example6WithOptions() {
	fmt.Println("=== With Options ===")

	maxTurns := 1
	systemPrompt := "You are a helpful assistant. Be concise."
	options := &claude.ClaudeAgentOptions{
		SystemPrompt: &systemPrompt,
		MaxTurns:     &maxTurns,
		AllowedTools: []string{"Read", "Write"},
	}

	ctx := context.Background()
	client := claude.NewClaudeSDKClient(options)

	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	msgCh, errCh := client.Query(ctx, "Explain goroutines in one sentence")
	for msg := range msgCh {
		if assistantMsg, ok := msg.(*claude.AssistantMessage); ok {
			for _, block := range assistantMsg.Content {
				if textBlock, ok := block.(claude.TextBlock); ok {
					fmt.Printf("Claude: %s\n", textBlock.Text)
				}
			}
		}
	}

	if err := <-errCh; err != nil {
		log.Printf("Error: %v", err)
	}
}

// Example 7: Async iterable prompt (channel-based)
func example7AsyncIterablePrompt() {
	fmt.Println("=== Async Iterable Prompt (Channel-Based) ===")

	ctx := context.Background()

	// Create a channel of messages
	promptCh := make(chan map[string]interface{}, 2)
	go func() {
		defer close(promptCh)
		promptCh <- map[string]interface{}{
			"type": "user",
			"message": map[string]interface{}{
				"role":    "user",
				"content": "Hello",
			},
		}
		promptCh <- map[string]interface{}{
			"type": "user",
			"message": map[string]interface{}{
				"role":    "user",
				"content": "What is Go?",
			},
		}
	}()

	msgCh, errCh, err := claude.QueryStream(ctx, promptCh, nil, nil)
	if err != nil {
		log.Fatalf("Failed to create query stream: %v", err)
	}

	for msg := range msgCh {
		if assistantMsg, ok := msg.(*claude.AssistantMessage); ok {
			for _, block := range assistantMsg.Content {
				if textBlock, ok := block.(claude.TextBlock); ok {
					fmt.Printf("Claude: %s\n", textBlock.Text)
				}
			}
		}
	}

	if err := <-errCh; err != nil {
		log.Printf("Error: %v", err)
	}
}

// Example 8: Bash command
func example8BashCommand() {
	fmt.Println("=== Bash Command ===")

	ctx := context.Background()
	options := &claude.ClaudeAgentOptions{
		AllowedTools: []string{"Bash"},
	}

	client := claude.NewClaudeSDKClient(options)

	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	msgCh, errCh := client.Query(ctx, "Run the command: echo 'Hello from Claude SDK'")

	for msg := range msgCh {
		switch m := msg.(type) {
		case *claude.AssistantMessage:
			for _, block := range m.Content {
				switch b := block.(type) {
				case claude.TextBlock:
					fmt.Printf("Text: %s\n", b.Text)
				case claude.ToolUseBlock:
					fmt.Printf("Tool: %s (ID: %s)\n", b.Name, b.ID)
				}
			}
		case *claude.ResultMessage:
			fmt.Printf("Command completed in %dms\n", m.DurationMS)
		}
	}

	if err := <-errCh; err != nil {
		log.Printf("Error: %v", err)
	}
}

// Example 9: Control protocol (server info, interrupt, model switching)
func example9ControlProtocol() {
	fmt.Println("=== Control Protocol ===")

	ctx := context.Background()
	client := claude.NewClaudeSDKClient(nil)

	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	// Get server info
	info := client.GetServerInfo()
	fmt.Printf("Server info: %+v\n", info)

	// Change permission mode
	if err := client.SetPermissionMode(ctx, claude.PermissionModeAcceptEdits); err != nil {
		log.Printf("SetPermissionMode error: %v", err)
	} else {
		fmt.Println("Permission mode changed to acceptEdits")
	}

	// Change model
	if err := client.SetModel(ctx, "claude-opus-4"); err != nil {
		log.Printf("SetModel error: %v", err)
	} else {
		fmt.Println("Model changed to claude-opus-4")
	}

	// Send a query
	msgCh, errCh := client.Query(ctx, "Say hello")
	for msg := range msgCh {
		if assistantMsg, ok := msg.(*claude.AssistantMessage); ok {
			fmt.Printf("Model used: %s\n", assistantMsg.Model)
		}
	}

	if err := <-errCh; err != nil {
		log.Printf("Error: %v", err)
	}
}

// Example 10: Error handling
func example10ErrorHandling() {
	fmt.Println("=== Error Handling ===")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := claude.NewClaudeSDKClient(nil)

	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	msgCh, errCh := client.Query(ctx, "What is Go?")

	done := false
	for !done {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				done = true
				break
			}
			if assistantMsg, ok := msg.(*claude.AssistantMessage); ok {
				for _, block := range assistantMsg.Content {
					if textBlock, ok := block.(claude.TextBlock); ok {
						fmt.Printf("Claude: %s\n", textBlock.Text)
					}
				}
			}
		case err := <-errCh:
			if err != nil {
				fmt.Printf("Error occurred: %v\n", err)
				done = true
			}
		case <-ctx.Done():
			fmt.Println("Context timeout")
			done = true
		}
	}
}
