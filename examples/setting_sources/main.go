package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	claude "github.com/clsx524/claude-agent-sdk-go"
)

/*
Example demonstrating setting sources control.

This example shows how to use the SettingSources option to control which
settings are loaded, including custom slash commands, agents, and other
configurations.

Setting sources determine where Claude Code loads configurations from:
- "user": Global user settings (~/.claude/)
- "project": Project-level settings (.claude/ in project)
- "local": Local gitignored settings (.claude-local/)

IMPORTANT: When SettingSources is not provided (nil), NO settings are loaded
by default. This creates an isolated environment. To load settings, explicitly
specify which sources to use.

By controlling which sources are loaded, you can:
- Create isolated environments with no custom settings (default)
- Load only user settings, excluding project-specific configurations
- Combine multiple sources as needed

Usage:
  go run main.go             - List available examples
  go run main.go all         - Run all examples
  go run main.go default     - Run a specific example
*/

func extractSlashCommands(msg *claude.SystemMessage) []string {
	if msg.Subtype == "init" {
		if commands, ok := msg.Data["slash_commands"].([]interface{}); ok {
			result := make([]string, 0, len(commands))
			for _, cmd := range commands {
				if cmdStr, ok := cmd.(string); ok {
					result = append(result, cmdStr)
				}
			}
			return result
		}
	}
	return []string{}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func exampleDefault() {
	fmt.Println("=== Default Behavior Example ===")
	fmt.Println("Setting sources: nil (default)")
	fmt.Println("Expected: No custom slash commands will be available")

	ctx := context.Background()

	// Get SDK directory
	sdkDir, _ := os.Getwd()
	sdkDir = filepath.Join(sdkDir, "../..")

	options := &claude.ClaudeAgentOptions{
		Cwd:      &sdkDir,
		MaxTurns: intPtr(1),
	}

	msgCh, errCh, err := claude.Query(ctx, "What is 2 + 2?", options, nil)
	if err != nil {
		log.Fatalf("Failed to create query: %v", err)
	}

	for msg := range msgCh {
		if sysMsg, ok := msg.(*claude.SystemMessage); ok && sysMsg.Subtype == "init" {
			commands := extractSlashCommands(sysMsg)
			fmt.Printf("Available slash commands: %v\n", commands)
			if contains(commands, "commit") {
				fmt.Println("❌ /commit is available (unexpected)")
			} else {
				fmt.Println("✓ /commit is NOT available (expected - no settings loaded)")
			}
			break
		}
	}

	if err := <-errCh; err != nil {
		log.Printf("Query error: %v", err)
	}

	fmt.Println()
}

func exampleUserOnly() {
	fmt.Println("=== User Settings Only Example ===")
	fmt.Println("Setting sources: [user]")
	fmt.Println("Expected: Project slash commands (like /commit) will NOT be available")

	ctx := context.Background()

	// Get SDK directory
	sdkDir, _ := os.Getwd()
	sdkDir = filepath.Join(sdkDir, "../..")

	options := &claude.ClaudeAgentOptions{
		SettingSources: []claude.SettingSource{claude.SettingSourceUser},
		Cwd:            &sdkDir,
		MaxTurns:       intPtr(1),
	}

	msgCh, errCh, err := claude.Query(ctx, "What is 2 + 2?", options, nil)
	if err != nil {
		log.Fatalf("Failed to create query: %v", err)
	}

	for msg := range msgCh {
		if sysMsg, ok := msg.(*claude.SystemMessage); ok && sysMsg.Subtype == "init" {
			commands := extractSlashCommands(sysMsg)
			fmt.Printf("Available slash commands: %v\n", commands)
			if contains(commands, "commit") {
				fmt.Println("❌ /commit is available (unexpected)")
			} else {
				fmt.Println("✓ /commit is NOT available (expected)")
			}
			break
		}
	}

	if err := <-errCh; err != nil {
		log.Printf("Query error: %v", err)
	}

	fmt.Println()
}

func exampleProjectAndUser() {
	fmt.Println("=== Project + User Settings Example ===")
	fmt.Println("Setting sources: [user, project]")
	fmt.Println("Expected: Project slash commands (like /commit) WILL be available")

	ctx := context.Background()

	// Get SDK directory
	sdkDir, _ := os.Getwd()
	sdkDir = filepath.Join(sdkDir, "../..")

	options := &claude.ClaudeAgentOptions{
		SettingSources: []claude.SettingSource{
			claude.SettingSourceUser,
			claude.SettingSourceProject,
		},
		Cwd:      &sdkDir,
		MaxTurns: intPtr(1),
	}

	msgCh, errCh, err := claude.Query(ctx, "What is 2 + 2?", options, nil)
	if err != nil {
		log.Fatalf("Failed to create query: %v", err)
	}

	for msg := range msgCh {
		if sysMsg, ok := msg.(*claude.SystemMessage); ok && sysMsg.Subtype == "init" {
			commands := extractSlashCommands(sysMsg)
			fmt.Printf("Available slash commands: %v\n", commands)
			if contains(commands, "commit") {
				fmt.Println("✓ /commit is available (expected)")
			} else {
				fmt.Println("❌ /commit is NOT available (unexpected)")
			}
			break
		}
	}

	if err := <-errCh; err != nil {
		log.Printf("Query error: %v", err)
	}

	fmt.Println()
}

func main() {
	examples := map[string]func(){
		"default":          exampleDefault,
		"user_only":        exampleUserOnly,
		"project_and_user": exampleProjectAndUser,
	}

	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <example_name>")
		fmt.Println("\nAvailable examples:")
		fmt.Println("  all - Run all examples")
		for name := range examples {
			fmt.Printf("  %s\n", name)
		}
		os.Exit(0)
	}

	exampleName := os.Args[1]

	if exampleName == "all" {
		for _, example := range examples {
			example()
			fmt.Println(string(make([]byte, 50, 50)))
			fmt.Println()
		}
	} else if example, ok := examples[exampleName]; ok {
		example()
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

func intPtr(i int) *int {
	return &i
}
