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

func TestImageContentHelper(t *testing.T) {
	base64Data := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	result := mcp.ImageContent(base64Data, "image/png")

	content, ok := result["content"].([]map[string]interface{})
	if !ok {
		t.Fatalf("expected content to be []map[string]interface{}, got %T", result["content"])
	}

	if len(content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(content))
	}

	imageBlock := content[0]
	if imageBlock["type"] != "image" {
		t.Errorf("expected type 'image', got %v", imageBlock["type"])
	}

	if imageBlock["data"] != base64Data {
		t.Errorf("expected data '%s', got %v", base64Data, imageBlock["data"])
	}

	if imageBlock["mimeType"] != "image/png" {
		t.Errorf("expected mimeType 'image/png', got %v", imageBlock["mimeType"])
	}
}

func TestMixedContentHelper(t *testing.T) {
	base64Data := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="

	result := mcp.MixedContent(
		map[string]interface{}{"type": "text", "text": "Here's the chart:"},
		map[string]interface{}{"type": "image", "data": base64Data, "mimeType": "image/png"},
	)

	content, ok := result["content"].([]map[string]interface{})
	if !ok {
		t.Fatalf("expected content to be []map[string]interface{}, got %T", result["content"])
	}

	if len(content) != 2 {
		t.Fatalf("expected 2 content blocks, got %d", len(content))
	}

	// Check text block
	if content[0]["type"] != "text" {
		t.Errorf("expected first block type 'text', got %v", content[0]["type"])
	}
	if content[0]["text"] != "Here's the chart:" {
		t.Errorf("expected text 'Here's the chart:', got %v", content[0]["text"])
	}

	// Check image block
	if content[1]["type"] != "image" {
		t.Errorf("expected second block type 'image', got %v", content[1]["type"])
	}
	if content[1]["data"] != base64Data {
		t.Errorf("expected data '%s', got %v", base64Data, content[1]["data"])
	}
}

func TestToolWithImageResponse(t *testing.T) {
	base64Data := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="

	tool := mcp.Tool(
		"generate_chart",
		"Generate a chart",
		map[string]string{"title": "string"},
		func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			return mcp.MixedContent(
				map[string]interface{}{"type": "text", "text": fmt.Sprintf("Generated chart: %s", args["title"])},
				map[string]interface{}{"type": "image", "data": base64Data, "mimeType": "image/png"},
			), nil
		},
	)

	server := mcp.CreateSdkMcpServer("chart-server", "1.0.0", []*mcp.SdkMcpTool{tool})

	// Test tool call
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      "generate_chart",
			"arguments": map[string]interface{}{"title": "Sales Report"},
		},
	}

	response := server.HandleRequest(context.Background(), request)
	result := response["result"].(map[string]interface{})
	content := result["content"].([]map[string]interface{})

	if len(content) != 2 {
		t.Fatalf("expected 2 content blocks, got %d", len(content))
	}

	// Verify text content
	if content[0]["type"] != "text" {
		t.Errorf("expected first block type 'text', got %v", content[0]["type"])
	}
	if content[0]["text"] != "Generated chart: Sales Report" {
		t.Errorf("expected text 'Generated chart: Sales Report', got %v", content[0]["text"])
	}

	// Verify image content
	if content[1]["type"] != "image" {
		t.Errorf("expected second block type 'image', got %v", content[1]["type"])
	}
	if content[1]["data"] != base64Data {
		t.Errorf("expected data '%s', got %v", base64Data, content[1]["data"])
	}
	if content[1]["mimeType"] != "image/png" {
		t.Errorf("expected mimeType 'image/png', got %v", content[1]["mimeType"])
	}
}

// TestMcpServerContextCancellation tests that context cancellation is respected
func TestMcpServerContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	tool := mcp.Tool(
		"slow_operation",
		"A slow operation",
		map[string]string{},
		func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return mcp.TextContent("Done"), nil
			}
		},
	)

	server := mcp.CreateSdkMcpServer("test", "1.0.0", []*mcp.SdkMcpTool{tool})

	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      "slow_operation",
			"arguments": map[string]interface{}{},
		},
	}

	response := server.HandleRequest(ctx, request)

	result, ok := response["result"].(map[string]interface{})
	if !ok {
		t.Fatal("expected result to be map")
	}

	// Should have error content due to cancellation
	isError, ok := result["isError"].(bool)
	if !ok || !isError {
		t.Error("expected isError to be true due to context cancellation")
	}
}

// TestMcpServerMissingArguments tests calling tool with missing required arguments
func TestMcpServerMissingArguments(t *testing.T) {
	tool := mcp.Tool(
		"require_args",
		"Requires arguments",
		map[string]string{"required_param": "string"},
		func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			// This will panic if required_param is missing and we try to type assert
			param, ok := args["required_param"].(string)
			if !ok || param == "" {
				return nil, fmt.Errorf("required_param is required")
			}
			return mcp.TextContent("Success"), nil
		},
	)

	server := mcp.CreateSdkMcpServer("test", "1.0.0", []*mcp.SdkMcpTool{tool})

	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      "require_args",
			"arguments": map[string]interface{}{}, // Missing required_param
		},
	}

	response := server.HandleRequest(context.Background(), request)

	result, ok := response["result"].(map[string]interface{})
	if !ok {
		t.Fatal("expected result to be map")
	}

	isError, ok := result["isError"].(bool)
	if !ok || !isError {
		t.Error("expected isError to be true due to missing argument")
	}
}

// TestMcpServerInvalidArgumentTypes tests calling tool with wrong argument types
func TestMcpServerInvalidArgumentTypes(t *testing.T) {
	tool := mcp.Tool(
		"expect_number",
		"Expects a number",
		map[string]string{"value": "number"},
		func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			value, ok := args["value"].(float64)
			if !ok {
				return nil, fmt.Errorf("value must be a number")
			}
			return mcp.TextContent(fmt.Sprintf("Got: %f", value)), nil
		},
	)

	server := mcp.CreateSdkMcpServer("test", "1.0.0", []*mcp.SdkMcpTool{tool})

	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "expect_number",
			"arguments": map[string]interface{}{
				"value": "not a number", // Wrong type
			},
		},
	}

	response := server.HandleRequest(context.Background(), request)

	result, ok := response["result"].(map[string]interface{})
	if !ok {
		t.Fatal("expected result to be map")
	}

	isError, ok := result["isError"].(bool)
	if !ok || !isError {
		t.Error("expected isError to be true due to type mismatch")
	}
}

// TestMcpServerComplexSchemaConversion tests complex input schema conversion
func TestMcpServerComplexSchemaConversion(t *testing.T) {
	tool := mcp.Tool(
		"complex",
		"Complex schema",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "User name",
				},
				"age": map[string]interface{}{
					"type":    "number",
					"minimum": 0,
					"maximum": 150,
				},
				"tags": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
			},
			"required": []string{"name"},
		},
		nil,
	)

	server := mcp.CreateSdkMcpServer("test", "1.0.0", []*mcp.SdkMcpTool{tool})

	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
	}

	response := server.HandleRequest(context.Background(), request)
	result := response["result"].(map[string]interface{})
	tools := result["tools"].([]map[string]interface{})

	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}

	schema := tools[0]["inputSchema"].(map[string]interface{})

	// Verify the schema was passed through correctly
	if schema["type"] != "object" {
		t.Errorf("expected type 'object', got %v", schema["type"])
	}

	properties := schema["properties"].(map[string]interface{})
	if properties == nil {
		t.Fatal("expected properties to be defined")
	}

	nameSchema := properties["name"].(map[string]interface{})
	if nameSchema["description"] != "User name" {
		t.Errorf("expected description 'User name', got %v", nameSchema["description"])
	}

	ageSchema := properties["age"].(map[string]interface{})
	if ageSchema["minimum"] != 0 && ageSchema["minimum"] != float64(0) {
		t.Errorf("expected minimum 0, got %v", ageSchema["minimum"])
	}

	// Handle both []string and []interface{} for required field
	requiredRaw := schema["required"]
	var requiredList []string
	switch v := requiredRaw.(type) {
	case []string:
		requiredList = v
	case []interface{}:
		for _, item := range v {
			if str, ok := item.(string); ok {
				requiredList = append(requiredList, str)
			}
		}
	default:
		t.Fatalf("unexpected type for required: %T", requiredRaw)
	}

	if len(requiredList) != 1 || requiredList[0] != "name" {
		t.Errorf("expected required=['name'], got %v", requiredList)
	}
}

// TestMcpServerMultipleToolsHandling tests server with multiple tools
func TestMcpServerMultipleToolsHandling(t *testing.T) {
	tool1 := mcp.Tool("add", "Add numbers", map[string]string{"a": "number", "b": "number"},
		func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			a := args["a"].(float64)
			b := args["b"].(float64)
			return mcp.TextContent(fmt.Sprintf("%.0f", a+b)), nil
		})

	tool2 := mcp.Tool("multiply", "Multiply numbers", map[string]string{"a": "number", "b": "number"},
		func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			a := args["a"].(float64)
			b := args["b"].(float64)
			return mcp.TextContent(fmt.Sprintf("%.0f", a*b)), nil
		})

	tool3 := mcp.Tool("greet", "Greet user", map[string]string{"name": "string"},
		func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			name := args["name"].(string)
			return mcp.TextContent("Hello, " + name), nil
		})

	server := mcp.CreateSdkMcpServer("multi-tool", "1.0.0", []*mcp.SdkMcpTool{tool1, tool2, tool3})

	// Test tools/list
	listRequest := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
	}

	response := server.HandleRequest(context.Background(), listRequest)
	result := response["result"].(map[string]interface{})
	tools := result["tools"].([]map[string]interface{})

	if len(tools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(tools))
	}

	// Test calling each tool
	testCases := []struct {
		toolName    string
		args        map[string]interface{}
		expectedOut string
	}{
		{"add", map[string]interface{}{"a": 5.0, "b": 3.0}, "8"},
		{"multiply", map[string]interface{}{"a": 4.0, "b": 7.0}, "28"},
		{"greet", map[string]interface{}{"name": "Alice"}, "Hello, Alice"},
	}

	for _, tc := range testCases {
		t.Run(tc.toolName, func(t *testing.T) {
			callRequest := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"method":  "tools/call",
				"params": map[string]interface{}{
					"name":      tc.toolName,
					"arguments": tc.args,
				},
			}

			response := server.HandleRequest(context.Background(), callRequest)
			result := response["result"].(map[string]interface{})
			content := result["content"].([]map[string]interface{})

			if content[0]["text"] != tc.expectedOut {
				t.Errorf("expected '%s', got '%v'", tc.expectedOut, content[0]["text"])
			}
		})
	}
}

// TestMcpServerEmptyToolsList tests server with no tools
func TestMcpServerEmptyToolsList(t *testing.T) {
	server := mcp.CreateSdkMcpServer("empty-server", "1.0.0", nil)

	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
	}

	response := server.HandleRequest(context.Background(), request)
	result := response["result"].(map[string]interface{})
	tools := result["tools"].([]map[string]interface{})

	if len(tools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(tools))
	}
}

// TestMcpServerNilResponse tests handling when tool returns nil
func TestMcpServerNilResponse(t *testing.T) {
	tool := mcp.Tool(
		"nil_tool",
		"Returns nil",
		map[string]string{},
		func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			return nil, nil
		},
	)

	server := mcp.CreateSdkMcpServer("test", "1.0.0", []*mcp.SdkMcpTool{tool})

	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      "nil_tool",
			"arguments": map[string]interface{}{},
		},
	}

	response := server.HandleRequest(context.Background(), request)

	// Should handle nil gracefully
	if response["error"] != nil {
		t.Error("should not return error for nil result")
	}
}
