package e2e

import (
	"context"
	"sync"
	"testing"
	"time"

	claude "github.com/clsx524/claude-agent-sdk-go"
	"github.com/clsx524/claude-agent-sdk-go/mcp"
)

// TestSDKMCPToolExecution tests that SDK MCP tools can be called and executed.
func TestSDKMCPToolExecution(t *testing.T) {
	RequireClaudeCode(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var mu sync.Mutex
	executions := []string{}

	// Create echo tool
	echoTool := mcp.Tool(
		"echo",
		"Echo back the input text",
		map[string]string{"text": "string"},
		func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			mu.Lock()
			executions = append(executions, "echo")
			mu.Unlock()

			text, _ := args["text"].(string)
			return mcp.TextContent("Echo: " + text), nil
		},
	)

	server := mcp.CreateSdkMcpServer("test", "1.0.0", []*mcp.SdkMcpTool{echoTool})

	options := &claude.ClaudeAgentOptions{
		McpServers: map[string]claude.McpServerConfig{
			"test": server.ToConfig(),
		},
		AllowedTools: []string{"mcp__test__echo"},
	}

	client := claude.NewClaudeSDKClient(options)
	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	msgCh, errCh := client.Query(ctx, "Call the mcp__test__echo tool with any text")

	// Collect messages
	_, err = CollectMessages(msgCh, errCh)
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}

	// Check if the actual Go function was called
	mu.Lock()
	hasEcho := false
	for _, exec := range executions {
		if exec == "echo" {
			hasEcho = true
			break
		}
	}
	mu.Unlock()

	if !hasEcho {
		t.Error("Echo tool function was not executed")
	}
}

// TestSDKMCPPermissionEnforcement tests that disallowed_tools prevents SDK MCP tool execution.
func TestSDKMCPPermissionEnforcement(t *testing.T) {
	RequireClaudeCode(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var mu sync.Mutex
	executions := []string{}

	// Create echo tool
	echoTool := mcp.Tool(
		"echo",
		"Echo back the input text",
		map[string]string{"text": "string"},
		func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			mu.Lock()
			executions = append(executions, "echo")
			mu.Unlock()

			text, _ := args["text"].(string)
			return mcp.TextContent("Echo: " + text), nil
		},
	)

	// Create greet tool
	greetTool := mcp.Tool(
		"greet",
		"Greet a person by name",
		map[string]string{"name": "string"},
		func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			mu.Lock()
			executions = append(executions, "greet")
			mu.Unlock()

			name, _ := args["name"].(string)
			return mcp.TextContent("Hello, " + name + "!"), nil
		},
	)

	server := mcp.CreateSdkMcpServer("test", "1.0.0", []*mcp.SdkMcpTool{echoTool, greetTool})

	options := &claude.ClaudeAgentOptions{
		McpServers: map[string]claude.McpServerConfig{
			"test": server.ToConfig(),
		},
		DisallowedTools: []string{"mcp__test__echo"},  // Block echo tool
		AllowedTools:    []string{"mcp__test__greet"}, // But allow greet
	}

	client := claude.NewClaudeSDKClient(options)
	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	msgCh, errCh := client.Query(ctx, "Use the echo tool to echo 'test' and use greet tool to greet 'Alice'")

	// Collect messages
	_, err = CollectMessages(msgCh, errCh)
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}

	// Check actual function executions
	mu.Lock()
	hasEcho := false
	hasGreet := false
	for _, exec := range executions {
		if exec == "echo" {
			hasEcho = true
		}
		if exec == "greet" {
			hasGreet = true
		}
	}
	mu.Unlock()

	if hasEcho {
		t.Error("Disallowed echo tool was executed")
	}
	if !hasGreet {
		t.Error("Allowed greet tool was not executed")
	}
}

// TestSDKMCPMultipleTools tests that multiple SDK MCP tools can be called in sequence.
func TestSDKMCPMultipleTools(t *testing.T) {
	RequireClaudeCode(t)

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	var mu sync.Mutex
	executions := []string{}

	// Create add tool
	addTool := mcp.Tool(
		"add",
		"Add two numbers",
		map[string]string{"a": "number", "b": "number"},
		func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			mu.Lock()
			executions = append(executions, "add")
			mu.Unlock()

			a, _ := args["a"].(float64)
			b, _ := args["b"].(float64)
			return mcp.TextContent("Result: " + string(rune(int(a+b)))), nil
		},
	)

	// Create multiply tool
	multiplyTool := mcp.Tool(
		"multiply",
		"Multiply two numbers",
		map[string]string{"a": "number", "b": "number"},
		func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			mu.Lock()
			executions = append(executions, "multiply")
			mu.Unlock()

			a, _ := args["a"].(float64)
			b, _ := args["b"].(float64)
			return mcp.TextContent("Result: " + string(rune(int(a*b)))), nil
		},
	)

	server := mcp.CreateSdkMcpServer("math", "1.0.0", []*mcp.SdkMcpTool{addTool, multiplyTool})

	options := &claude.ClaudeAgentOptions{
		McpServers: map[string]claude.McpServerConfig{
			"math": server.ToConfig(),
		},
		AllowedTools: []string{"mcp__math__add", "mcp__math__multiply"},
	}

	client := claude.NewClaudeSDKClient(options)
	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	msgCh, errCh := client.Query(ctx, "First use add tool with 5 and 3, then multiply tool with 4 and 2")

	// Collect messages
	_, err = CollectMessages(msgCh, errCh)
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}

	// Check that both tools were executed
	mu.Lock()
	hasAdd := false
	hasMultiply := false
	for _, exec := range executions {
		if exec == "add" {
			hasAdd = true
		}
		if exec == "multiply" {
			hasMultiply = true
		}
	}
	mu.Unlock()

	if !hasAdd {
		t.Error("Add tool was not executed")
	}
	if !hasMultiply {
		t.Error("Multiply tool was not executed")
	}
}

// TestSDKMCPToolWithError tests error handling in SDK MCP tools.
func TestSDKMCPToolWithError(t *testing.T) {
	RequireClaudeCode(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a tool that returns an error
	errorTool := mcp.Tool(
		"faulty",
		"A tool that always fails",
		map[string]string{},
		func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			return nil, context.DeadlineExceeded // Return an error
		},
	)

	server := mcp.CreateSdkMcpServer("test", "1.0.0", []*mcp.SdkMcpTool{errorTool})

	options := &claude.ClaudeAgentOptions{
		McpServers: map[string]claude.McpServerConfig{
			"test": server.ToConfig(),
		},
		AllowedTools: []string{"mcp__test__faulty"},
	}

	client := claude.NewClaudeSDKClient(options)
	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	msgCh, errCh := client.Query(ctx, "Call the faulty tool")

	// Collect messages - should complete without crashing
	messages, err := CollectMessages(msgCh, errCh)
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}

	gotResult := false
	for _, msg := range messages {
		if _, ok := msg.(*claude.ResultMessage); ok {
			gotResult = true
			break
		}
	}

	if !gotResult {
		t.Error("Did not receive ResultMessage (query may have crashed)")
	}
}
