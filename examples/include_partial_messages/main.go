package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	claude "github.com/clsx524/claude-agent-sdk-go"
)

/*
Example of using the "IncludePartialMessages" option to stream partial messages
from Claude Code SDK.

This feature allows you to receive stream events that contain incremental
updates as Claude generates responses. This is useful for:
- Building real-time UIs that show text as it's being generated
- Monitoring tool use progress
- Getting early results before the full response is complete

Note: Partial message streaming requires the CLI to support it, and the
messages will include StreamEvent messages interspersed with regular messages.

Usage:
  go run main.go
*/

func main() {
	fmt.Println("Partial Message Streaming Example")
	fmt.Println(strings.Repeat("=", 50))

	ctx := context.Background()

	// Enable partial message streaming
	model := "claude-sonnet-4-5"
	maxTurns := 2
	options := &claude.ClaudeAgentOptions{
		IncludePartialMessages: true,
		Model:                  &model,
		MaxTurns:               &maxTurns,
		Env: map[string]string{
			"MAX_THINKING_TOKENS": "8000",
		},
	}

	// Send a prompt that will generate a streaming response
	prompt := "Think of three jokes, then tell one"
	fmt.Printf("Prompt: %s\n\n", prompt)
	fmt.Println(strings.Repeat("=", 50))

	msgCh, errCh, err := claude.Query(ctx, prompt, options, nil)
	if err != nil {
		log.Fatalf("Failed to create query: %v", err)
	}

	// Process all messages including stream events
	for msg := range msgCh {
		switch m := msg.(type) {
		case *claude.StreamEvent:
			fmt.Printf("[StreamEvent] UUID: %s, SessionID: %s\n", m.UUID, m.SessionID)
			if eventType, ok := m.Event["type"].(string); ok {
				fmt.Printf("  Type: %s\n", eventType)
			}

		case *claude.AssistantMessage:
			fmt.Println("\n[AssistantMessage]")
			for _, block := range m.Content {
				if textBlock, ok := block.(claude.TextBlock); ok {
					fmt.Printf("  Text: %s\n", textBlock.Text)
				} else if thinkingBlock, ok := block.(claude.ThinkingBlock); ok {
					fmt.Printf("  Thinking: %s\n", thinkingBlock.Thinking[:min(100, len(thinkingBlock.Thinking))])
				}
			}

		case *claude.UserMessage:
			fmt.Println("\n[UserMessage]")
			if content, ok := m.Content.(string); ok {
				fmt.Printf("  Content: %s\n", content)
			}

		case *claude.SystemMessage:
			fmt.Printf("\n[SystemMessage] Subtype: %s\n", m.Subtype)

		case *claude.ResultMessage:
			fmt.Println("\n[ResultMessage]")
			fmt.Printf("  Duration: %dms\n", m.DurationMS)
			if m.TotalCostUSD != nil {
				fmt.Printf("  Cost: $%.4f\n", *m.TotalCostUSD)
			}
			fmt.Printf("  Turns: %d\n", m.NumTurns)

		default:
			fmt.Printf("\n[Unknown Message Type]: %T\n", msg)
		}
	}

	// Check for errors
	if err := <-errCh; err != nil {
		log.Printf("Query error: %v", err)
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("Streaming complete!")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
