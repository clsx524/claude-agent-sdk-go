package main

import (
	"context"
	"fmt"
	"log"
	"time"

	claude "github.com/clsx524/claude-agent-sdk-go"
)

func main() {
	fmt.Println("=== ClaudeSDKClient Streaming Example ===")

	// Example 1: Simple interactive conversation
	example1SimpleConversation()

	// Example 2: Multi-turn conversation
	fmt.Println("\n=== Multi-turn Conversation ===")
	example2MultiTurn()

	// Example 3: Using ReceiveResponse helper
	fmt.Println("\n=== Using ReceiveResponse Helper ===")
	example3ReceiveResponse()
}

func example1SimpleConversation() {
	ctx := context.Background()

	// Create client
	client := claude.NewClaudeSDKClient(nil)

	// Connect (with nil prompt for interactive use)
	if err := client.Connect(ctx, nil); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	// Send a query
	if err := client.Query(ctx, "What is the capital of France?", "default"); err != nil {
		log.Fatalf("Failed to send query: %v", err)
	}

	// Receive response
	for msg := range client.ReceiveResponse(ctx) {
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
}

func example2MultiTurn() {
	ctx := context.Background()

	client := claude.NewClaudeSDKClient(nil)

	if err := client.Connect(ctx, nil); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	// First turn
	fmt.Println("Turn 1:")
	if err := client.Query(ctx, "Hello! What's your name?", "default"); err != nil {
		log.Fatalf("Failed to send query: %v", err)
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

	// Second turn
	fmt.Println("\nTurn 2:")
	if err := client.Query(ctx, "Can you help me with Go programming?", "default"); err != nil {
		log.Fatalf("Failed to send query: %v", err)
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

func example3ReceiveResponse() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Set up options
	maxTurns := 1
	options := &claude.ClaudeAgentOptions{
		SystemPrompt: "You are a helpful assistant. Be concise.",
		MaxTurns:     &maxTurns,
	}

	client := claude.NewClaudeSDKClient(options)

	if err := client.Connect(ctx, nil); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	// Send query
	if err := client.Query(ctx, "Explain goroutines in one sentence", "default"); err != nil {
		log.Fatalf("Failed to send query: %v", err)
	}

	// Use ReceiveResponse which automatically stops after ResultMessage
	var totalCost float64
	for msg := range client.ReceiveResponse(ctx) {
		switch m := msg.(type) {
		case *claude.AssistantMessage:
			fmt.Printf("Model: %s\n", m.Model)
			for _, block := range m.Content {
				if textBlock, ok := block.(claude.TextBlock); ok {
					fmt.Printf("Response: %s\n", textBlock.Text)
				}
			}
		case *claude.ResultMessage:
			if m.TotalCostUSD != nil {
				totalCost = *m.TotalCostUSD
			}
			fmt.Printf("Turns: %d, Duration: %dms\n", m.NumTurns, m.DurationMS)
		}
	}

	if totalCost > 0 {
		fmt.Printf("Total cost: $%.4f\n", totalCost)
	}
}
