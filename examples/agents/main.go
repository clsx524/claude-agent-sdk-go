package main

import (
	"context"
	"fmt"
	"log"

	claude "github.com/clsx524/claude-agent-sdk-go"
)

/*
Example of using custom agents with Claude Code SDK.

This example demonstrates how to define and use custom agents with specific
tools, prompts, and models.

Usage:
  go run main.go
*/

func codeReviewerExample() {
	fmt.Println("=== Code Reviewer Agent Example ===")

	ctx := context.Background()

	model := "sonnet"
	options := &claude.ClaudeAgentOptions{
		Agents: map[string]claude.AgentDefinition{
			"code-reviewer": {
				Description: "Reviews code for best practices and potential issues",
				Prompt: "You are a code reviewer. Analyze code for bugs, performance issues, " +
					"security vulnerabilities, and adherence to best practices. " +
					"Provide constructive feedback.",
				Tools: []string{"Read", "Grep"},
				Model: &model,
			},
		},
	}

	msgCh, errCh, err := claude.Query(
		ctx,
		"Use the code-reviewer agent to review the code in types.go",
		options,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to create query: %v", err)
	}

	for msg := range msgCh {
		switch m := msg.(type) {
		case *claude.AssistantMessage:
			for _, block := range m.Content {
				if textBlock, ok := block.(claude.TextBlock); ok {
					fmt.Printf("Claude: %s\n", textBlock.Text)
				}
			}
		case *claude.ResultMessage:
			if m.TotalCostUSD != nil && *m.TotalCostUSD > 0 {
				fmt.Printf("\nCost: $%.4f\n", *m.TotalCostUSD)
			}
		}
	}

	if err := <-errCh; err != nil {
		log.Printf("Query error: %v", err)
	}

	fmt.Println()
}

func documentationWriterExample() {
	fmt.Println("=== Documentation Writer Agent Example ===")

	ctx := context.Background()

	model := "sonnet"
	options := &claude.ClaudeAgentOptions{
		Agents: map[string]claude.AgentDefinition{
			"doc-writer": {
				Description: "Writes comprehensive documentation",
				Prompt: "You are a technical documentation expert. Write clear, comprehensive " +
					"documentation with examples. Focus on clarity and completeness.",
				Tools: []string{"Read", "Write", "Edit"},
				Model: &model,
			},
		},
	}

	msgCh, errCh, err := claude.Query(
		ctx,
		"Use the doc-writer agent to explain what AgentDefinition is used for",
		options,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to create query: %v", err)
	}

	for msg := range msgCh {
		switch m := msg.(type) {
		case *claude.AssistantMessage:
			for _, block := range m.Content {
				if textBlock, ok := block.(claude.TextBlock); ok {
					fmt.Printf("Claude: %s\n", textBlock.Text)
				}
			}
		case *claude.ResultMessage:
			if m.TotalCostUSD != nil && *m.TotalCostUSD > 0 {
				fmt.Printf("\nCost: $%.4f\n", *m.TotalCostUSD)
			}
		}
	}

	if err := <-errCh; err != nil {
		log.Printf("Query error: %v", err)
	}

	fmt.Println()
}

func multipleAgentsExample() {
	fmt.Println("=== Multiple Agents Example ===")

	ctx := context.Background()

	model := "sonnet"
	options := &claude.ClaudeAgentOptions{
		Agents: map[string]claude.AgentDefinition{
			"analyzer": {
				Description: "Analyzes code structure and patterns",
				Prompt:      "You are a code analyzer. Examine code structure, patterns, and architecture.",
				Tools:       []string{"Read", "Grep", "Glob"},
			},
			"tester": {
				Description: "Creates and runs tests",
				Prompt:      "You are a testing expert. Write comprehensive tests and ensure code quality.",
				Tools:       []string{"Read", "Write", "Bash"},
				Model:       &model,
			},
		},
		SettingSources: []claude.SettingSource{claude.SettingSourceUser, claude.SettingSourceProject},
	}

	msgCh, errCh, err := claude.Query(
		ctx,
		"Use the analyzer agent to find all Go files in the examples/ directory",
		options,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to create query: %v", err)
	}

	for msg := range msgCh {
		switch m := msg.(type) {
		case *claude.AssistantMessage:
			for _, block := range m.Content {
				if textBlock, ok := block.(claude.TextBlock); ok {
					fmt.Printf("Claude: %s\n", textBlock.Text)
				}
			}
		case *claude.ResultMessage:
			if m.TotalCostUSD != nil && *m.TotalCostUSD > 0 {
				fmt.Printf("\nCost: $%.4f\n", *m.TotalCostUSD)
			}
		}
	}

	if err := <-errCh; err != nil {
		log.Printf("Query error: %v", err)
	}

	fmt.Println()
}

func main() {
	fmt.Println("Running all agent examples...")
	codeReviewerExample()
	documentationWriterExample()
	multipleAgentsExample()
}
