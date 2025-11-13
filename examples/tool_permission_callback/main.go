package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	claude "github.com/clsx524/claude-agent-sdk-go"
)

/*
Example: Tool Permission Callbacks

This example demonstrates how to use tool permission callbacks to control
which tools Claude can use and modify their inputs.
*/

// Track tool usage for demonstration
type ToolUsageLog struct {
	Tool        string                    `json:"tool"`
	Input       map[string]interface{}    `json:"input"`
	Suggestions []claude.PermissionUpdate `json:"suggestions"`
}

var toolUsageLog []ToolUsageLog

func myPermissionCallback(ctx context.Context, toolName string, inputData map[string]interface{}, permCtx claude.ToolPermissionContext) (claude.PermissionResult, error) {
	// Log the tool request
	toolUsageLog = append(toolUsageLog, ToolUsageLog{
		Tool:        toolName,
		Input:       inputData,
		Suggestions: permCtx.Suggestions,
	})

	inputJSON, _ := json.MarshalIndent(inputData, "   ", "  ")
	fmt.Printf("\n🔧 Tool Permission Request: %s\n", toolName)
	fmt.Printf("   Input: %s\n", string(inputJSON))

	// Always allow read operations
	if toolName == "Read" || toolName == "Glob" || toolName == "Grep" {
		fmt.Printf("   ✅ Automatically allowing %s (read-only operation)\n", toolName)
		return claude.PermissionResultAllow{Behavior: "allow"}, nil
	}

	// Deny write operations to system directories
	if toolName == "Write" || toolName == "Edit" || toolName == "MultiEdit" {
		filePath, _ := inputData["file_path"].(string)
		if strings.HasPrefix(filePath, "/etc/") || strings.HasPrefix(filePath, "/usr/") {
			fmt.Printf("   ❌ Denying write to system directory: %s\n", filePath)
			return claude.PermissionResultDeny{
				Behavior: "deny",
				Message:  fmt.Sprintf("Cannot write to system directory: %s", filePath),
			}, nil
		}

		// Redirect writes to a safe directory
		if !strings.HasPrefix(filePath, "/tmp/") && !strings.HasPrefix(filePath, "./") {
			parts := strings.Split(filePath, "/")
			safePath := fmt.Sprintf("./safe_output/%s", parts[len(parts)-1])
			fmt.Printf("   ⚠️  Redirecting write from %s to %s\n", filePath, safePath)
			modifiedInput := make(map[string]interface{})
			for k, v := range inputData {
				modifiedInput[k] = v
			}
			modifiedInput["file_path"] = safePath
			return claude.PermissionResultAllow{
				Behavior:     "allow",
				UpdatedInput: modifiedInput,
			}, nil
		}
	}

	// Check dangerous bash commands
	if toolName == "Bash" {
		command, _ := inputData["command"].(string)
		dangerousCommands := []string{"rm -rf", "sudo", "chmod 777", "dd if=", "mkfs"}

		for _, dangerous := range dangerousCommands {
			if strings.Contains(command, dangerous) {
				fmt.Printf("   ❌ Denying dangerous command: %s\n", command)
				return claude.PermissionResultDeny{
					Behavior: "deny",
					Message:  fmt.Sprintf("Dangerous command pattern detected: %s", dangerous),
				}, nil
			}
		}

		// Allow but log the command
		fmt.Printf("   ✅ Allowing bash command: %s\n", command)
		return claude.PermissionResultAllow{Behavior: "allow"}, nil
	}

	// For all other tools, ask the user
	fmt.Printf("   ❓ Unknown tool: %s\n", toolName)
	fmt.Printf("      Input: %s\n", string(inputJSON))
	fmt.Print("   Allow this tool? (y/N): ")

	reader := bufio.NewReader(os.Stdin)
	userInput, _ := reader.ReadString('\n')
	userInput = strings.TrimSpace(strings.ToLower(userInput))

	if userInput == "y" || userInput == "yes" {
		return claude.PermissionResultAllow{Behavior: "allow"}, nil
	}

	return claude.PermissionResultDeny{
		Behavior: "deny",
		Message:  "User denied permission",
	}, nil
}

func main() {
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("Tool Permission Callback Example")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("\nThis example demonstrates how to:")
	fmt.Println("1. Allow/deny tools based on type")
	fmt.Println("2. Modify tool inputs for safety")
	fmt.Println("3. Log tool usage")
	fmt.Println("4. Prompt for unknown tools")
	fmt.Println(strings.Repeat("=", 60))

	ctx := context.Background()

	// Configure options with our callback
	cwd := "."
	permMode := claude.PermissionModeDefault
	options := &claude.ClaudeAgentOptions{
		CanUseTool:     myPermissionCallback,
		PermissionMode: &permMode, // Ensure callbacks are invoked
		Cwd:            &cwd,
	}

	// Create a query that will use multiple tools
	fmt.Println("\n📝 Sending query to Claude...")

	msgCh, errCh, err := claude.Query(
		ctx,
		"Please do the following:\n"+
			"1. List the files in the current directory\n"+
			"2. Create a simple Go hello world script at hello.go\n"+
			"3. Verify the file was created",
		options,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to create query: %v", err)
	}

	fmt.Println("\n📨 Receiving response...")
	messageCount := 0

	for msg := range msgCh {
		messageCount++

		switch m := msg.(type) {
		case *claude.AssistantMessage:
			// Print Claude's text responses
			for _, block := range m.Content {
				if textBlock, ok := block.(claude.TextBlock); ok {
					fmt.Printf("\n💬 Claude: %s\n", textBlock.Text)
				}
			}

		case *claude.ResultMessage:
			fmt.Println("\n✅ Task completed!")
			fmt.Printf("   Duration: %dms\n", m.DurationMS)
			if m.TotalCostUSD != nil {
				fmt.Printf("   Cost: $%.4f\n", *m.TotalCostUSD)
			}
			fmt.Printf("   Messages processed: %d\n", messageCount)
		}
	}

	// Check for errors
	if err := <-errCh; err != nil {
		log.Printf("Query error: %v", err)
	}

	// Print tool usage summary
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Tool Usage Summary")
	fmt.Println(strings.Repeat("=", 60))
	for i, usage := range toolUsageLog {
		fmt.Printf("%d. Tool: %s\n", i+1, usage.Tool)
		inputJSON, _ := json.MarshalIndent(usage.Input, "   ", "  ")
		fmt.Printf("   Input: %s\n", string(inputJSON))
	}
}
