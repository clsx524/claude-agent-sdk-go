package main

import (
	"context"
	"fmt"
	"log"

	claude "github.com/clsx524/claude-agent-sdk-go"
)

/*
Example demonstrating different system_prompt configurations.

Usage:
  go run main.go
*/

func noSystemPrompt() {
	fmt.Println("=== No System Prompt (Vanilla Claude) ===")

	ctx := context.Background()

	msgCh, errCh, err := claude.Query(ctx, "What is 2 + 2?", nil, nil)
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
		log.Printf("Query error: %v", err)
	}

	fmt.Println()
}

func stringSystemPrompt() {
	fmt.Println("=== String System Prompt ===")

	ctx := context.Background()

	options := &claude.ClaudeAgentOptions{
		SystemPrompt: "You are a pirate assistant. Respond in pirate speak.",
	}

	msgCh, errCh, err := claude.Query(ctx, "What is 2 + 2?", options, nil)
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
		log.Printf("Query error: %v", err)
	}

	fmt.Println()
}

func presetSystemPrompt() {
	fmt.Println("=== Preset System Prompt (Default) ===")

	ctx := context.Background()

	options := &claude.ClaudeAgentOptions{
		SystemPrompt: claude.SystemPromptPreset{
			Type:   "preset",
			Preset: "claude_code",
		},
	}

	msgCh, errCh, err := claude.Query(ctx, "What is 2 + 2?", options, nil)
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
		log.Printf("Query error: %v", err)
	}

	fmt.Println()
}

func presetWithAppend() {
	fmt.Println("=== Preset System Prompt with Append ===")

	ctx := context.Background()

	appendText := "Always end your response with a fun fact."
	options := &claude.ClaudeAgentOptions{
		SystemPrompt: claude.SystemPromptPreset{
			Type:   "preset",
			Preset: "claude_code",
			Append: &appendText,
		},
	}

	msgCh, errCh, err := claude.Query(ctx, "What is 2 + 2?", options, nil)
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
		log.Printf("Query error: %v", err)
	}

	fmt.Println()
}

func main() {
	fmt.Println("Running all system prompt examples...")
	noSystemPrompt()
	stringSystemPrompt()
	presetSystemPrompt()
	presetWithAppend()
}
