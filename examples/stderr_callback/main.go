package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	claude "github.com/clsx524/claude-agent-sdk-go"
)

/*
Simple example demonstrating stderr callback for capturing CLI debug output.

Usage:
  go run main.go
*/

func main() {
	fmt.Println("Running query with stderr capture...")

	ctx := context.Background()

	// Collect stderr messages
	var stderrMessages []string

	// Callback that receives each line of stderr output
	stderrCallback := func(message string) {
		stderrMessages = append(stderrMessages, message)
		// Optionally print specific messages
		if strings.Contains(message, "[ERROR]") {
			fmt.Printf("Error detected: %s\n", message)
		}
	}

	// Create options with stderr callback and enable debug mode
	options := &claude.ClaudeAgentOptions{
		Stderr: stderrCallback,
		ExtraArgs: map[string]*string{
			"debug-to-stderr": nil, // Enable debug output (flag without value)
		},
	}

	msgCh, errCh, err := claude.Query(ctx, "What is 2+2?", options, nil)
	if err != nil {
		log.Fatalf("Failed to create query: %v", err)
	}

	// Process messages
	for msg := range msgCh {
		switch m := msg.(type) {
		case *claude.AssistantMessage:
			for _, block := range m.Content {
				if textBlock, ok := block.(claude.TextBlock); ok {
					fmt.Printf("Response: %s\n", textBlock.Text)
				}
			}
		case *claude.UserMessage:
			if content, ok := m.Content.(string); ok {
				fmt.Printf("Response: %s\n", content)
			}
		}
	}

	// Check for errors
	if err := <-errCh; err != nil {
		log.Printf("Query error: %v", err)
	}

	// Show what we captured
	fmt.Printf("\nCaptured %d stderr lines\n", len(stderrMessages))
	if len(stderrMessages) > 0 {
		firstLine := stderrMessages[0]
		if len(firstLine) > 100 {
			firstLine = firstLine[:100]
		}
		fmt.Printf("First stderr line: %s\n", firstLine)
	}
}
