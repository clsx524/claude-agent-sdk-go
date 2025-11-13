package main

import (
	"context"
	"fmt"
	"log"
	"os"

	claude "github.com/clsx524/claude-agent-sdk-go"
)

func displayMessage(msg claude.Message) {
	switch m := msg.(type) {
	case *claude.AssistantMessage:
		for _, block := range m.Content {
			if textBlock, ok := block.(claude.TextBlock); ok {
				fmt.Printf("Claude: %s\n", textBlock.Text)
			}
		}
	case *claude.ResultMessage:
		if m.TotalCostUSD != nil {
			fmt.Printf("Total cost: $%.4f\n", *m.TotalCostUSD)
		}
		fmt.Printf("Status: %s\n", m.Subtype)
	}
}

func withoutBudget() {
	fmt.Println("=== Without Budget Limit ===")

	ctx := context.Background()
	msgCh, errCh, err := claude.Query(ctx, "What is 2 + 2?", nil, nil)
	if err != nil {
		log.Fatalf("Failed to create query: %v", err)
	}

	for msg := range msgCh {
		displayMessage(msg)
	}

	if err := <-errCh; err != nil {
		log.Printf("Error: %v", err)
	}

	fmt.Println()
}

func withReasonableBudget() {
	fmt.Println("=== With Reasonable Budget ($0.10) ===")

	maxBudget := 0.10 // 10 cents - plenty for a simple query
	options := &claude.ClaudeAgentOptions{
		MaxBudgetUSD: &maxBudget,
	}

	ctx := context.Background()
	msgCh, errCh, err := claude.Query(ctx, "What is 2 + 2?", options, nil)
	if err != nil {
		log.Fatalf("Failed to create query: %v", err)
	}

	for msg := range msgCh {
		displayMessage(msg)
	}

	if err := <-errCh; err != nil {
		log.Printf("Error: %v", err)
	}

	fmt.Println()
}

func withTightBudget() {
	fmt.Println("=== With Tight Budget ($0.0001) ===")

	maxBudget := 0.0001 // Very small budget - will be exceeded quickly
	options := &claude.ClaudeAgentOptions{
		MaxBudgetUSD: &maxBudget,
	}

	ctx := context.Background()
	msgCh, errCh, err := claude.Query(ctx, "Read the README.md file and summarize it", options, nil)
	if err != nil {
		log.Fatalf("Failed to create query: %v", err)
	}

	for msg := range msgCh {
		displayMessage(msg)

		// Check if budget was exceeded
		if resultMsg, ok := msg.(*claude.ResultMessage); ok {
			if resultMsg.Subtype == "error_max_budget_usd" {
				fmt.Println("⚠️  Budget limit exceeded!")
				fmt.Println("Note: The cost may exceed the budget by up to one API call's worth")
			}
		}
	}

	if err := <-errCh; err != nil {
		log.Printf("Error: %v", err)
	}

	fmt.Println()
}

func main() {
	if len(os.Args) > 1 {
		example := os.Args[1]
		switch example {
		case "without":
			withoutBudget()
		case "reasonable":
			withReasonableBudget()
		case "tight":
			withTightBudget()
		case "all":
			fmt.Println("This example demonstrates using max_budget_usd to control API costs.")
			withoutBudget()
			withReasonableBudget()
			withTightBudget()
			fmt.Println("\nNote: Budget checking happens after each API call completes,")
			fmt.Println("so the final cost may slightly exceed the specified budget.")
		default:
			fmt.Printf("Unknown example: %s\n", example)
			fmt.Println("Usage: go run main.go [without|reasonable|tight|all]")
			os.Exit(1)
		}
	} else {
		fmt.Println("Usage: go run main.go [without|reasonable|tight|all]")
		fmt.Println("\nAvailable examples:")
		fmt.Println("  without    - Run query without budget limit")
		fmt.Println("  reasonable - Run query with reasonable budget ($0.10)")
		fmt.Println("  tight      - Run query with tight budget ($0.0001)")
		fmt.Println("  all        - Run all examples")
		os.Exit(0)
	}
}
