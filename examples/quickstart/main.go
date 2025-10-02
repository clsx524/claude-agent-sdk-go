package main

import (
	"context"
	"fmt"
	"log"

	claude "github.com/clsx524/claude-agent-sdk-go"
)

func main() {
	// Simple query example
	fmt.Println("=== Simple Query Example ===")
	simpleQuery()

	fmt.Println("\n=== Query with Options Example ===")
	queryWithOptions()
}

func simpleQuery() {
	ctx := context.Background()

	// Perform a simple query
	msgCh, errCh, err := claude.Query(ctx, "What is 2 + 2?", nil, nil)
	if err != nil {
		log.Fatalf("Failed to create query: %v", err)
	}

	// Process messages
	for msg := range msgCh {
		switch m := msg.(type) {
		case *claude.AssistantMessage:
			for _, block := range m.Content {
				if textBlock, ok := block.(claude.TextBlock); ok {
					fmt.Printf("Claude: %s\n", textBlock.Text)
				}
			}
		case *claude.ResultMessage:
			fmt.Printf("Query completed in %dms\n", m.DurationMS)
			if m.TotalCostUSD != nil {
				fmt.Printf("Cost: $%.4f\n", *m.TotalCostUSD)
			}
		}
	}

	// Check for errors
	if err := <-errCh; err != nil {
		log.Fatalf("Query error: %v", err)
	}
}

func queryWithOptions() {
	ctx := context.Background()

	// Create options
	maxTurns := 1
	options := &claude.ClaudeAgentOptions{
		SystemPrompt: "You are a helpful math assistant. Be concise.",
		MaxTurns:     &maxTurns,
		AllowedTools: []string{}, // No tools for this simple query
	}

	msgCh, errCh, err := claude.Query(ctx, "Calculate 15 * 23", options, nil)
	if err != nil {
		log.Fatalf("Failed to create query: %v", err)
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
		log.Fatalf("Query error: %v", err)
	}
}
