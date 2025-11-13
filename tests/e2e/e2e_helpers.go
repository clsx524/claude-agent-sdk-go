package e2e

import (
	"context"
	"fmt"
	"math"
	"os/exec"
	"testing"

	claude "github.com/clsx524/claude-agent-sdk-go"
	"github.com/clsx524/claude-agent-sdk-go/mcp"
)

// RequireClaudeCode checks if Claude Code CLI is available and skips the test if not.
// E2E tests use the Claude Code CLI which must be installed and authenticated.
func RequireClaudeCode(t *testing.T) {
	t.Helper()

	// Skip if running in short mode
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode (use -short to skip)")
	}

	// Check if Claude Code is installed
	_, err := exec.LookPath("claude")
	if err != nil {
		t.Skip("Claude Code CLI not found. Install it with: npm install -g @anthropic-ai/claude-code")
	}
}

// CreateCalculatorMCP creates a test calculator MCP server with basic math tools.
func CreateCalculatorMCP() claude.McpServerConfig {
	// Define calculator tools
	addTool := mcp.Tool(
		"add",
		"Add two numbers together",
		map[string]string{
			"a": "number",
			"b": "number",
		},
		func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			a, ok1 := args["a"].(float64)
			b, ok2 := args["b"].(float64)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("arguments must be numbers")
			}
			return mcp.TextContent(fmt.Sprintf("%.2f", a+b)), nil
		},
	)

	divideTool := mcp.Tool(
		"divide",
		"Divide two numbers",
		map[string]string{
			"a": "number",
			"b": "number",
		},
		func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			a, ok1 := args["a"].(float64)
			b, ok2 := args["b"].(float64)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("arguments must be numbers")
			}
			if b == 0 {
				return nil, fmt.Errorf("division by zero")
			}
			return mcp.TextContent(fmt.Sprintf("%.2f", a/b)), nil
		},
	)

	sqrtTool := mcp.Tool(
		"sqrt",
		"Calculate square root of a number",
		map[string]string{
			"x": "number",
		},
		func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			x, ok := args["x"].(float64)
			if !ok {
				return nil, fmt.Errorf("argument must be a number")
			}
			if x < 0 {
				return nil, fmt.Errorf("cannot take square root of negative number")
			}
			return mcp.TextContent(fmt.Sprintf("%.2f", math.Sqrt(x))), nil
		},
	)

	powerTool := mcp.Tool(
		"power",
		"Raise a number to a power",
		map[string]string{
			"base":     "number",
			"exponent": "number",
		},
		func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			base, ok1 := args["base"].(float64)
			exponent, ok2 := args["exponent"].(float64)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("arguments must be numbers")
			}
			return mcp.TextContent(fmt.Sprintf("%.2f", math.Pow(base, exponent))), nil
		},
	)

	// Create MCP server
	calculator := mcp.CreateSdkMcpServer(
		"calculator",
		"1.0.0",
		[]*mcp.SdkMcpTool{addTool, divideTool, sqrtTool, powerTool},
	)

	return calculator.ToConfig()
}

// HasToolUse checks if any message in the slice contains a ToolUseBlock.
func HasToolUse(messages []claude.Message) bool {
	for _, msg := range messages {
		if assistantMsg, ok := msg.(*claude.AssistantMessage); ok {
			for _, block := range assistantMsg.Content {
				if _, ok := block.(claude.ToolUseBlock); ok {
					return true
				}
			}
		}
	}
	return false
}

// FindToolUse finds the first ToolUseBlock with the given tool name.
func FindToolUse(messages []claude.Message, toolName string) *claude.ToolUseBlock {
	for _, msg := range messages {
		if assistantMsg, ok := msg.(*claude.AssistantMessage); ok {
			for _, block := range assistantMsg.Content {
				if toolUse, ok := block.(claude.ToolUseBlock); ok {
					if toolUse.Name == toolName {
						return &toolUse
					}
				}
			}
		}
	}
	return nil
}

// CollectMessages collects all messages from Query channels.
func CollectMessages(msgCh <-chan claude.Message, errCh <-chan error) ([]claude.Message, error) {
	var messages []claude.Message

	for msg := range msgCh {
		messages = append(messages, msg)
	}

	if err := <-errCh; err != nil {
		return messages, err
	}

	return messages, nil
}

// GetSystemMessageData returns the Data field from a SystemMessage if the message is one.
func GetSystemMessageData(msg claude.Message) (map[string]interface{}, bool) {
	if sysMsg, ok := msg.(*claude.SystemMessage); ok {
		return sysMsg.Data, true
	}
	return nil, false
}

// IsSystemMessage checks if a message is a SystemMessage.
func IsSystemMessage(msg claude.Message) bool {
	_, ok := msg.(*claude.SystemMessage)
	return ok
}

// IsAssistantMessage checks if a message is an AssistantMessage.
func IsAssistantMessage(msg claude.Message) bool {
	_, ok := msg.(*claude.AssistantMessage)
	return ok
}

// IsResultMessage checks if a message is a ResultMessage.
func IsResultMessage(msg claude.Message) bool {
	_, ok := msg.(*claude.ResultMessage)
	return ok
}

// IsStreamEvent checks if a message is a StreamEvent.
func IsStreamEvent(msg claude.Message) bool {
	_, ok := msg.(*claude.StreamEvent)
	return ok
}

// GetResultMessage finds the ResultMessage from a slice of messages.
func GetResultMessage(messages []claude.Message) *claude.ResultMessage {
	for _, msg := range messages {
		if resultMsg, ok := msg.(*claude.ResultMessage); ok {
			return resultMsg
		}
	}
	return nil
}

// GetAssistantMessages returns all AssistantMessage instances.
func GetAssistantMessages(messages []claude.Message) []*claude.AssistantMessage {
	var assistantMsgs []*claude.AssistantMessage
	for _, msg := range messages {
		if assistantMsg, ok := msg.(*claude.AssistantMessage); ok {
			assistantMsgs = append(assistantMsgs, assistantMsg)
		}
	}
	return assistantMsgs
}

// GetTextContent extracts all text content from messages.
func GetTextContent(messages []claude.Message) []string {
	var texts []string
	for _, msg := range messages {
		if assistantMsg, ok := msg.(*claude.AssistantMessage); ok {
			for _, block := range assistantMsg.Content {
				if textBlock, ok := block.(claude.TextBlock); ok {
					texts = append(texts, textBlock.Text)
				}
			}
		}
	}
	return texts
}
