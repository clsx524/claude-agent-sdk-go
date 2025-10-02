package main

import (
	"context"
	"fmt"
	"log"
	"math"

	claude "github.com/clsx524/claude-agent-sdk-go"
	"github.com/clsx524/claude-agent-sdk-go/mcp"
)

func main() {
	fmt.Println("=== SDK MCP Tools Example ===")

	// Create custom tools
	addTool := mcp.Tool(
		"add",
		"Add two numbers together",
		map[string]string{"a": "number", "b": "number"},
		func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			a := args["a"].(float64)
			b := args["b"].(float64)
			sum := a + b
			return mcp.TextContent(fmt.Sprintf("The sum of %.2f and %.2f is %.2f", a, b, sum)), nil
		},
	)

	multiplyTool := mcp.Tool(
		"multiply",
		"Multiply two numbers",
		map[string]string{"a": "number", "b": "number"},
		func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			a := args["a"].(float64)
			b := args["b"].(float64)
			product := a * b
			return mcp.TextContent(fmt.Sprintf("The product of %.2f and %.2f is %.2f", a, b, product)), nil
		},
	)

	powerTool := mcp.Tool(
		"power",
		"Raise a number to a power",
		map[string]string{"base": "number", "exponent": "number"},
		func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			base := args["base"].(float64)
			exponent := args["exponent"].(float64)
			result := math.Pow(base, exponent)
			return mcp.TextContent(fmt.Sprintf("%.2f raised to the power of %.2f is %.2f", base, exponent, result)), nil
		},
	)

	// Create SDK MCP server
	calculatorServer := mcp.CreateSdkMcpServer(
		"calculator",
		"1.0.0",
		[]*mcp.SdkMcpTool{addTool, multiplyTool, powerTool},
	)

	// Configure Claude options
	options := &claude.ClaudeAgentOptions{
		McpServers: map[string]claude.McpServerConfig{
			"calc": calculatorServer.ToConfig(),
		},
		AllowedTools: []string{
			"mcp__calc__add",
			"mcp__calc__multiply",
			"mcp__calc__power",
		},
		SystemPrompt: "You are a math assistant. Use the calculator tools to help with calculations.",
	}

	// Create client
	ctx := context.Background()
	client := claude.NewClaudeSDKClient(options)

	if err := client.Connect(ctx, nil); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	// Test queries
	queries := []string{
		"What is 15 + 27?",
		"Calculate 8 times 9",
		"What is 2 to the power of 10?",
		"Calculate (5 + 3) * 4",
	}

	for i, query := range queries {
		fmt.Printf("\n--- Query %d: %s ---\n", i+1, query)

		if err := client.Query(ctx, query, "default"); err != nil {
			log.Printf("Failed to send query: %v", err)
			continue
		}

		for msg := range client.ReceiveResponse(ctx) {
			switch m := msg.(type) {
			case *claude.AssistantMessage:
				for _, block := range m.Content {
					switch b := block.(type) {
					case claude.TextBlock:
						fmt.Printf("Claude: %s\n", b.Text)
					case claude.ToolUseBlock:
						fmt.Printf("Using tool: %s\n", b.Name)
						fmt.Printf("  Input: %v\n", b.Input)
					}
				}
			case *claude.UserMessage:
				// Tool results
				if blocks, ok := m.Content.([]claude.ContentBlock); ok {
					for _, block := range blocks {
						if toolResult, ok := block.(claude.ToolResultBlock); ok {
							fmt.Printf("Tool result: %v\n", toolResult.Content)
						}
					}
				}
			case *claude.ResultMessage:
				fmt.Printf("Completed in %dms\n", m.DurationMS)
			}
		}
	}

	fmt.Println("\n=== Example with Error Handling ===")
	exampleWithErrorHandling()
}

func exampleWithErrorHandling() {
	// Create a tool that can fail
	divideTool := mcp.Tool(
		"divide",
		"Divide two numbers",
		map[string]string{"a": "number", "b": "number"},
		func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			a := args["a"].(float64)
			b := args["b"].(float64)

			if b == 0 {
				return mcp.ErrorContent("Error: Division by zero"), nil
			}

			result := a / b
			return mcp.TextContent(fmt.Sprintf("%.2f divided by %.2f is %.2f", a, b, result)), nil
		},
	)

	server := mcp.CreateSdkMcpServer("math", "1.0.0", []*mcp.SdkMcpTool{divideTool})

	maxTurns := 2
	options := &claude.ClaudeAgentOptions{
		McpServers: map[string]claude.McpServerConfig{
			"math": server.ToConfig(),
		},
		AllowedTools: []string{"mcp__math__divide"},
		MaxTurns:     &maxTurns,
	}

	ctx := context.Background()
	client := claude.NewClaudeSDKClient(options)

	if err := client.Connect(ctx, nil); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	// Test division by zero
	fmt.Println("Testing division by zero:")
	if err := client.Query(ctx, "What is 10 divided by 0?", "default"); err != nil {
		log.Printf("Failed to send query: %v", err)
		return
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
