package unit

import (
	"context"
	"fmt"
	"testing"

	"github.com/clsx524/claude-agent-sdk-go/mcp"
)

func TestToolCreation(t *testing.T) {
	tool := mcp.Tool(
		"greet",
		"Greet a user",
		map[string]string{"name": "string"},
		func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			name := args["name"].(string)
			return map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": fmt.Sprintf("Hello, %s!", name)},
				},
			}, nil
		},
	)

	if tool.Name != "greet" {
		t.Errorf("expected name 'greet', got %s", tool.Name)
	}
	if tool.Description != "Greet a user" {
		t.Errorf("expected description 'Greet a user', got %s", tool.Description)
	}
	if tool.Handler == nil {
		t.Error("expected handler to be set")
	}
}

func TestSdkMcpServerCreation(t *testing.T) {
	tool1 := mcp.Tool("tool1", "First tool", map[string]string{}, nil)
	tool2 := mcp.Tool("tool2", "Second tool", map[string]string{}, nil)

	server := mcp.CreateSdkMcpServer("test-server", "1.0.0", []*mcp.SdkMcpTool{tool1, tool2})

	if server.Name != "test-server" {
		t.Errorf("expected name 'test-server', got %s", server.Name)
	}
	if server.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %s", server.Version)
	}
	if len(server.Tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(server.Tools))
	}
}

func TestMcpServerHandleInitialize(t *testing.T) {
	server := mcp.CreateSdkMcpServer("test", "1.0.0", nil)

	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
	}

	response := server.HandleRequest(context.Background(), request)

	if response["jsonrpc"] != "2.0" {
		t.Error("response should have jsonrpc 2.0")
	}
	if response["id"] != 1 {
		t.Error("response should have matching id")
	}

	result, ok := response["result"].(map[string]interface{})
	if !ok {
		t.Fatal("expected result to be map")
	}

	if result["protocolVersion"] != "2024-11-05" {
		t.Errorf("expected protocol version '2024-11-05', got %v", result["protocolVersion"])
	}

	serverInfo, ok := result["serverInfo"].(map[string]interface{})
	if !ok {
		t.Fatal("expected serverInfo to be map")
	}
	if serverInfo["name"] != "test" {
		t.Errorf("expected server name 'test', got %v", serverInfo["name"])
	}
}

func TestMcpServerHandleListTools(t *testing.T) {
	addTool := mcp.Tool(
		"add",
		"Add two numbers",
		map[string]string{"a": "number", "b": "number"},
		func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			a := args["a"].(float64)
			b := args["b"].(float64)
			return mcp.TextContent(fmt.Sprintf("Sum: %f", a+b)), nil
		},
	)

	server := mcp.CreateSdkMcpServer("calculator", "1.0.0", []*mcp.SdkMcpTool{addTool})

	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	}

	response := server.HandleRequest(context.Background(), request)

	result, ok := response["result"].(map[string]interface{})
	if !ok {
		t.Fatal("expected result to be map")
	}

	tools, ok := result["tools"].([]map[string]interface{})
	if !ok {
		t.Fatal("expected tools to be array")
	}

	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}

	if tools[0]["name"] != "add" {
		t.Errorf("expected tool name 'add', got %v", tools[0]["name"])
	}
	if tools[0]["description"] != "Add two numbers" {
		t.Errorf("expected tool description 'Add two numbers', got %v", tools[0]["description"])
	}

	schema, ok := tools[0]["inputSchema"].(map[string]interface{})
	if !ok {
		t.Fatal("expected inputSchema to be map")
	}
	if schema["type"] != "object" {
		t.Errorf("expected schema type 'object', got %v", schema["type"])
	}
}

func TestMcpServerHandleCallTool(t *testing.T) {
	called := false
	greetTool := mcp.Tool(
		"greet",
		"Greet a user",
		map[string]string{"name": "string"},
		func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			called = true
			name := args["name"].(string)
			return map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": fmt.Sprintf("Hello, %s!", name)},
				},
			}, nil
		},
	)

	server := mcp.CreateSdkMcpServer("test", "1.0.0", []*mcp.SdkMcpTool{greetTool})

	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      "greet",
			"arguments": map[string]interface{}{"name": "Alice"},
		},
	}

	response := server.HandleRequest(context.Background(), request)

	if !called {
		t.Error("tool handler should have been called")
	}

	result, ok := response["result"].(map[string]interface{})
	if !ok {
		t.Fatal("expected result to be map")
	}

	content, ok := result["content"].([]map[string]interface{})
	if !ok {
		t.Fatal("expected content to be array")
	}

	if len(content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(content))
	}

	if content[0]["type"] != "text" {
		t.Errorf("expected content type 'text', got %v", content[0]["type"])
	}
	if content[0]["text"] != "Hello, Alice!" {
		t.Errorf("expected text 'Hello, Alice!', got %v", content[0]["text"])
	}
}

func TestMcpServerHandleCallToolError(t *testing.T) {
	errorTool := mcp.Tool(
		"fail",
		"A tool that fails",
		map[string]string{},
		func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			return nil, fmt.Errorf("intentional error")
		},
	)

	server := mcp.CreateSdkMcpServer("test", "1.0.0", []*mcp.SdkMcpTool{errorTool})

	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      "fail",
			"arguments": map[string]interface{}{},
		},
	}

	response := server.HandleRequest(context.Background(), request)

	result, ok := response["result"].(map[string]interface{})
	if !ok {
		t.Fatal("expected result to be map")
	}

	isError, ok := result["isError"].(bool)
	if !ok || !isError {
		t.Error("expected isError to be true")
	}

	content, ok := result["content"].([]map[string]interface{})
	if !ok {
		t.Fatal("expected content to be array")
	}

	if len(content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(content))
	}

	text, ok := content[0]["text"].(string)
	if !ok {
		t.Fatal("expected text field")
	}
	if text != "Error: intentional error" {
		t.Errorf("expected error message, got %s", text)
	}
}

func TestMcpServerHandleCallToolNotFound(t *testing.T) {
	server := mcp.CreateSdkMcpServer("test", "1.0.0", nil)

	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      5,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      "nonexistent",
			"arguments": map[string]interface{}{},
		},
	}

	response := server.HandleRequest(context.Background(), request)

	errorObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("expected error to be map")
	}

	if errorObj["code"] != -32602 {
		t.Errorf("expected error code -32602, got %v", errorObj["code"])
	}
}

func TestMcpServerUnknownMethod(t *testing.T) {
	server := mcp.CreateSdkMcpServer("test", "1.0.0", nil)

	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      6,
		"method":  "unknown/method",
	}

	response := server.HandleRequest(context.Background(), request)

	errorObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("expected error to be map")
	}

	if errorObj["code"] != -32601 {
		t.Errorf("expected error code -32601 (method not found), got %v", errorObj["code"])
	}
}

func TestTextContentHelper(t *testing.T) {
	content := mcp.TextContent("test message")

	contentArray, ok := content["content"].([]map[string]interface{})
	if !ok {
		t.Fatal("expected content to be array")
	}

	if len(contentArray) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(contentArray))
	}

	if contentArray[0]["type"] != "text" {
		t.Errorf("expected type 'text', got %v", contentArray[0]["type"])
	}
	if contentArray[0]["text"] != "test message" {
		t.Errorf("expected text 'test message', got %v", contentArray[0]["text"])
	}
}

func TestErrorContentHelper(t *testing.T) {
	content := mcp.ErrorContent("error message")

	isError, ok := content["isError"].(bool)
	if !ok || !isError {
		t.Error("expected isError to be true")
	}

	contentArray, ok := content["content"].([]map[string]interface{})
	if !ok {
		t.Fatal("expected content to be array")
	}

	if contentArray[0]["text"] != "error message" {
		t.Errorf("expected text 'error message', got %v", contentArray[0]["text"])
	}
}

func TestSchemaConversion(t *testing.T) {
	t.Run("simple map schema", func(t *testing.T) {
		server := mcp.CreateSdkMcpServer("test", "1.0.0", []*mcp.SdkMcpTool{
			mcp.Tool("test", "test", map[string]string{
				"name": "string",
				"age":  "number",
			}, nil),
		})

		request := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "tools/list",
		}

		response := server.HandleRequest(context.Background(), request)
		result := response["result"].(map[string]interface{})
		tools := result["tools"].([]map[string]interface{})
		schema := tools[0]["inputSchema"].(map[string]interface{})

		if schema["type"] != "object" {
			t.Errorf("expected type 'object', got %v", schema["type"])
		}

		properties := schema["properties"].(map[string]interface{})
		if properties["name"].(map[string]interface{})["type"] != "string" {
			t.Error("expected name to be string type")
		}
		if properties["age"].(map[string]interface{})["type"] != "number" {
			t.Error("expected age to be number type")
		}
	})
}
