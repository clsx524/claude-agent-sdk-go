# Claude Agent SDK for Go

Go SDK for building AI agents with Claude Code. See the [Claude Agent SDK documentation](https://docs.anthropic.com/en/docs/claude-code/sdk) for more information.

## Installation

```bash
go get github.com/clsx524/claude-agent-sdk-go
```

**Prerequisites:**
- Go 1.21+
- Node.js
- Claude Code: `npm install -g @anthropic-ai/claude-code`

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    claude "github.com/clsx524/claude-agent-sdk-go"
)

func main() {
    ctx := context.Background()

    // Simple query
    msgCh, errCh, err := claude.Query(ctx, "What is 2 + 2?", nil, nil)
    if err != nil {
        log.Fatal(err)
    }

    for msg := range msgCh {
        if assistantMsg, ok := msg.(*claude.AssistantMessage); ok {
            for _, block := range assistantMsg.Content {
                if textBlock, ok := block.(claude.TextBlock); ok {
                    fmt.Println(textBlock.Text)
                }
            }
        }
    }

    if err := <-errCh; err != nil {
        log.Fatal(err)
    }
}
```

## Basic Usage

### Simple Query API

The `Query()` function is ideal for simple, stateless queries:

```go
ctx := context.Background()

// With options
maxTurns := 1
options := &claude.ClaudeAgentOptions{
    SystemPrompt: "You are a helpful assistant",
    MaxTurns:     &maxTurns,
    AllowedTools: []string{"Read", "Write"},
}

msgCh, errCh, err := claude.Query(ctx, "Create a hello.go file", options, nil)
if err != nil {
    log.Fatal(err)
}

// Process messages
for msg := range msgCh {
    switch m := msg.(type) {
    case *claude.AssistantMessage:
        // Handle assistant response
    case *claude.ResultMessage:
        fmt.Printf("Cost: $%.4f\n", *m.TotalCostUSD)
    }
}
```

### ClaudeSDKClient for Interactive Conversations

For bidirectional, stateful conversations, use `ClaudeSDKClient`:

```go
client := claude.NewClaudeSDKClient(nil)

if err := client.Connect(ctx, nil); err != nil {
    log.Fatal(err)
}
defer client.Disconnect()

// Send query
if err := client.Query(ctx, "Hello Claude", "default"); err != nil {
    log.Fatal(err)
}

// Receive response
for msg := range client.ReceiveResponse(ctx) {
    if assistantMsg, ok := msg.(*claude.AssistantMessage); ok {
        for _, block := range assistantMsg.Content {
            if textBlock, ok := block.(claude.TextBlock); ok {
                fmt.Println(textBlock.Text)
            }
        }
    }
}
```

## Advanced Features

### Custom Tools (SDK MCP Servers)

Create in-process tools that Claude can invoke:

```go
import "github.com/clsx524/claude-agent-sdk-go/mcp"

// Define a tool
addTool := mcp.Tool(
    "add",
    "Add two numbers",
    map[string]string{"a": "number", "b": "number"},
    func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
        a := args["a"].(float64)
        b := args["b"].(float64)
        return mcp.TextContent(fmt.Sprintf("Sum: %.2f", a+b)), nil
    },
)

// Create server
calculator := mcp.CreateSdkMcpServer("calculator", "1.0.0", []*mcp.SdkMcpTool{addTool})

// Use with Claude
options := &claude.ClaudeAgentOptions{
    McpServers: map[string]claude.McpServerConfig{
        "calc": calculator.ToConfig(),
    },
    AllowedTools: []string{"mcp__calc__add"},
}
```

**Benefits:**
- No subprocess overhead
- Direct access to Go application state
- Simpler deployment
- Type-safe with Go

### Hooks

Hooks allow you to intercept and control Claude's behavior:

```go
// Block dangerous bash commands
bashHook := func(ctx context.Context, input map[string]interface{}, toolUseID *string, hookCtx claude.HookContext) (claude.HookJSONOutput, error) {
    toolName := input["tool_name"].(string)
    if toolName != "Bash" {
        return claude.HookJSONOutput{}, nil
    }

    toolInput := input["tool_input"].(map[string]interface{})
    command := toolInput["command"].(string)

    if strings.Contains(command, "rm -rf") {
        decision := "block"
        return claude.HookJSONOutput{
            Decision: &decision,
            HookSpecificOutput: map[string]interface{}{
                "hookEventName":            "PreToolUse",
                "permissionDecision":       "deny",
                "permissionDecisionReason": "Dangerous command blocked",
            },
        }, nil
    }

    return claude.HookJSONOutput{}, nil
}

options := &claude.ClaudeAgentOptions{
    Hooks: map[claude.HookEvent][]claude.HookMatcher{
        claude.HookEventPreToolUse: {
            {Matcher: "Bash", Hooks: []claude.HookCallback{bashHook}},
        },
    },
}
```

### Permission Callbacks

Control tool execution programmatically:

```go
canUseTool := func(ctx context.Context, toolName string, input map[string]interface{}, permCtx claude.ToolPermissionContext) (claude.PermissionResult, error) {
    // Allow read-only operations
    if toolName == "Read" || toolName == "Grep" {
        return claude.PermissionResultAllow{Behavior: "allow"}, nil
    }

    // Block dangerous commands
    if toolName == "Bash" {
        command := input["command"].(string)
        if strings.Contains(command, "sudo") {
            return claude.PermissionResultDeny{
                Behavior: "deny",
                Message:  "sudo commands not allowed",
            }, nil
        }
    }

    return claude.PermissionResultAllow{Behavior: "allow"}, nil
}

options := &claude.ClaudeAgentOptions{
    CanUseTool: canUseTool,
}
```

### Configuration Options

```go
options := &claude.ClaudeAgentOptions{
    // Tool restrictions
    AllowedTools:    []string{"Read", "Write"},
    DisallowedTools: []string{"Bash"},

    // System prompt
    SystemPrompt: "You are a helpful Go assistant",
    // Or use preset:
    // SystemPrompt: claude.SystemPromptPreset{
    //     Type: "preset",
    //     Preset: "claude_code",
    //     Append: stringPtr("Additional instructions"),
    // },

    // Conversation settings
    MaxTurns:             &maxTurns,
    ContinueConversation: true,
    Resume:               &sessionID,
    ForkSession:          true,

    // Model selection
    Model:         stringPtr("claude-sonnet-4-5"),
    FallbackModel: stringPtr("claude-sonnet-3-5"), // Fallback if primary unavailable

    // Budget and token control
    MaxBudgetUSD:      floatPtr(1.0),  // Maximum spending limit in USD
    MaxThinkingTokens: intPtr(10000),  // Maximum extended thinking tokens

    // Permission mode
    PermissionMode: &permissionMode, // "default", "acceptEdits", "bypassPermissions"

    // Working directory
    Cwd: stringPtr("/path/to/project"),

    // Environment variables
    Env: map[string]string{"KEY": "value"},

    // Additional directories
    AddDirs: []string{"/dir1", "/dir2"},

    // Settings
    Settings:       stringPtr("/path/to/settings.json"),
    SettingSources: []claude.SettingSource{claude.SettingSourceUser, claude.SettingSourceProject},

    // Plugins
    Plugins: []claude.SdkPluginConfig{
        {
            Type: "local",
            Path: "/path/to/plugin",
        },
    },

    // Custom agents
    Agents: map[string]claude.AgentDefinition{
        "code-reviewer": {
            Description: "Reviews code",
            Prompt:      "You are a code reviewer",
            Tools:       []string{"Read", "Grep"},
        },
    },

    // Streaming
    IncludePartialMessages: true,

    // Callbacks
    CanUseTool: canUseToolFunc,
    Hooks:      hooksMap,
    Stderr:     stderrCallback,
}
```

## Message Types

The SDK uses typed messages for type-safe handling:

```go
// Iterate through messages with type assertions
for msg := range msgCh {
    switch m := msg.(type) {
    case *claude.UserMessage:
        // User input
    case *claude.AssistantMessage:
        // Claude's response with content blocks
        for _, block := range m.Content {
            switch b := block.(type) {
            case claude.TextBlock:
                fmt.Println(b.Text)
            case claude.ToolUseBlock:
                fmt.Printf("Tool: %s\n", b.Name)
            case claude.ThinkingBlock:
                fmt.Printf("Thinking: %s\n", b.Thinking)
            }
        }
    case *claude.SystemMessage:
        // System messages
    case *claude.ResultMessage:
        // Final result with metrics
        fmt.Printf("Duration: %dms, Cost: $%.4f\n", m.DurationMS, *m.TotalCostUSD)
    case *claude.StreamEvent:
        // Partial updates (when IncludePartialMessages is true)
    }
}
```

## Error Handling

```go
import "errors"

msgCh, errCh, err := claude.Query(ctx, "Hello", nil, nil)
if err != nil {
    var cliNotFound *claude.CLINotFoundError
    if errors.As(err, &cliNotFound) {
        fmt.Println("Please install Claude Code")
    }
    
    var processErr *claude.ProcessError
    if errors.As(err, &processErr) {
        fmt.Printf("Exit code: %d\n", processErr.ExitCode)
    }
    
    log.Fatal(err)
}

// Process messages...

// Check for runtime errors
if err := <-errCh; err != nil {
    log.Fatal(err)
}
```

Error types:
- `ClaudeSDKError` - Base error
- `CLINotFoundError` - Claude Code not installed
- `CLIConnectionError` - Connection issues
- `ProcessError` - Process failures
- `CLIJSONDecodeError` - JSON parsing errors
- `MessageParseError` - Message parsing errors

## Examples

See the [examples](examples/) directory for complete working examples:

### Getting Started
- **[quickstart](examples/quickstart/main.go)** - Basic query usage
- **[streaming](examples/streaming/main.go)** - Multi-turn conversations with ClaudeSDKClient

### Advanced Features
- **[mcp_tools](examples/mcp_tools/main.go)** - Custom in-process tools
- **[hooks](examples/hooks/main.go)** - Hook system for control and interception
- **[tool_permission_callback](examples/tool_permission_callback/main.go)** - Control tool permissions
- **[agents](examples/agents/main.go)** - Define and use custom agents

### Configuration
- **[max_budget_usd](examples/max_budget_usd/main.go)** - Set maximum spending limits
- **[plugin_example](examples/plugin_example/main.go)** - Load custom plugins
- **[setting_sources](examples/setting_sources/main.go)** - Control which settings are loaded
- **[system_prompt](examples/system_prompt/main.go)** - Different system prompt configurations

### Streaming & Callbacks
- **[include_partial_messages](examples/include_partial_messages/main.go)** - Stream partial updates
- **[stderr_callback](examples/stderr_callback/main.go)** - Capture debug output

Run examples:
```bash
cd examples/quickstart
go run main.go
```

## Concurrency and Context

The SDK is designed for Go's concurrency model:

```go
// Use context for cancellation
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// Channels for message passing
msgCh, errCh, err := claude.Query(ctx, "Hello", nil, nil)

// Use select for concurrent operations
for {
    select {
    case msg, ok := <-msgCh:
        if !ok {
            return
        }
        // Process message
    case err := <-errCh:
        if err != nil {
            log.Fatal(err)
        }
        return
    case <-ctx.Done():
        log.Println("Timeout or cancellation")
        return
    }
}
```

## Testing

Run tests:
```bash
# Unit tests
go test ./tests/unit/...

# All tests
go test ./...

# With coverage
go test -cover ./...

# With race detection
go test -race ./...
```

## Comparison with Python SDK

| Feature | Python SDK | Go SDK |
|---------|-----------|--------|
| Simple queries | `query()` | `Query()` |
| Streaming client | `ClaudeSDKClient` | `ClaudeSDKClient` |
| Async iteration | `async for` | channels (`<-chan`) |
| Callbacks | async functions | functions with `context.Context` |
| Error handling | exceptions | error returns + channels |
| Type safety | runtime type checking | compile-time type safety |
| Concurrency | asyncio/trio | goroutines + channels |

## Architecture

```
claude-agent-sdk-go/
├── types.go           # Core types and interfaces
├── errors.go          # Error types
├── query.go           # Simple Query API
├── client.go          # ClaudeSDKClient
├── transport/         # Transport layer
│   ├── transport.go   # Interface
│   └── subprocess.go  # CLI subprocess transport
├── internal/
│   ├── parser/        # Message parser
│   ├── query/         # Control protocol handler
│   └── client/        # Internal client
├── mcp/               # SDK MCP server support
│   └── sdk_server.go
├── examples/          # Example applications
└── tests/             # Unit and integration tests
```

## Contributing

Contributions are welcome! Please:
1. Write tests for new features
2. Follow Go conventions (`go fmt`, `go vet`)
3. Update documentation
4. Add examples for significant features

## License

MIT

## Resources

- [Claude Agent SDK Documentation](https://docs.anthropic.com/en/docs/claude-code/sdk)
- [Go API Reference](https://pkg.go.dev/github.com/clsx524/claude-agent-sdk-go)
- [Examples](examples/)
- [Issues](https://github.com/clsx524/claude-agent-sdk-go/issues)
