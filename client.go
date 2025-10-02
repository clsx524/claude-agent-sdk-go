package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

)

// ClaudeSDKClient provides bidirectional, interactive conversations with Claude Code.
//
// This client provides full control over the conversation flow with support
// for streaming, interrupts, and dynamic message sending. For simple one-shot
// queries, consider using the Query() function instead.
//
// Key features:
//   - Bidirectional: Send and receive messages at any time
//   - Stateful: Maintains conversation context across messages
//   - Interactive: Send follow-ups based on responses
//   - Control flow: Support for interrupts and session management
//
// Example:
//
//	client := NewClaudeSDKClient(nil)
//	ctx := context.Background()
//
//	if err := client.Connect(ctx, nil); err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Disconnect()
//
//	if err := client.Query(ctx, "Hello Claude", "default"); err != nil {
//	    log.Fatal(err)
//	}
//
//	for msg := range client.ReceiveResponse(ctx) {
//	    fmt.Printf("%+v\n", msg)
//	}
type ClaudeSDKClient struct {
	options         *ClaudeAgentOptions
	customTransport Transport
	transport       Transport
	queryHandler    *queryHandler
	ctx             context.Context
	cancel          context.CancelFunc
}

// NewClaudeSDKClient creates a new Claude SDK client.
func NewClaudeSDKClient(options *ClaudeAgentOptions) *ClaudeSDKClient {
	if options == nil {
		options = &ClaudeAgentOptions{}
	}

	return &ClaudeSDKClient{
		options: options,
	}
}

// NewClaudeSDKClientWithTransport creates a client with a custom transport.
func NewClaudeSDKClientWithTransport(options *ClaudeAgentOptions, trans Transport) *ClaudeSDKClient {
	if options == nil {
		options = &ClaudeAgentOptions{}
	}

	return &ClaudeSDKClient{
		options:         options,
		customTransport: trans,
	}
}

// Connect establishes connection to Claude Code.
//
// The prompt parameter can be:
//   - nil: Empty connection for interactive use
//   - string: Initial prompt message
//   - <-chan map[string]interface{}: Stream of input messages
func (c *ClaudeSDKClient) Connect(ctx context.Context, prompt interface{}) error {
	os.Setenv("CLAUDE_CODE_ENTRYPOINT", "sdk-go-client")

	// Create cancellable context
	c.ctx, c.cancel = context.WithCancel(ctx)

	// Determine actual prompt (empty channel if nil)
	actualPrompt := prompt
	if actualPrompt == nil {
		emptyCh := make(chan map[string]interface{})
		close(emptyCh)
		actualPrompt = emptyCh
	}

	// Validate and configure permission settings
	options := c.options
	if c.options.CanUseTool != nil {
		// canUseTool requires streaming mode
		if _, isString := prompt.(string); isString {
			return fmt.Errorf("can_use_tool callback requires streaming mode")
		}

		if c.options.PermissionPromptToolName != nil {
			return fmt.Errorf("can_use_tool callback cannot be used with permission_prompt_tool_name")
		}

		// Set permission_prompt_tool_name to "stdio"
		stdio := "stdio"
		newOpts := *c.options
		newOpts.PermissionPromptToolName = &stdio
		options = &newOpts
	}

	// Use provided transport or create subprocess transport
	if c.customTransport != nil {
		c.transport = c.customTransport
	} else {
		var err error
		c.transport, err = NewSubprocessCLITransport(actualPrompt, options, "")
		if err != nil {
			return err
		}
	}

	if err := c.transport.Connect(c.ctx); err != nil {
		return err
	}

	// Extract SDK MCP servers
	sdkMcpServers := make(map[string]interface{})
	for name, config := range c.options.McpServers {
		if sdkConfig, ok := config.(McpSdkServerConfig); ok {
			sdkMcpServers[name] = sdkConfig.Instance
		}
	}

	// Create queryHandler - ClaudeSDKClient always uses streaming mode
	c.queryHandler = newQueryHandler(
		c.transport,
		true, // Always streaming mode
		options.CanUseTool,
		options.Hooks,
		sdkMcpServers,
	)

	// Start reading messages
	if err := c.queryHandler.Start(c.ctx); err != nil {
		return err
	}

	// Initialize
	if _, err := c.queryHandler.Initialize(c.ctx); err != nil {
		return err
	}

	// If we have an initial prompt stream, start streaming it
	if prompt != nil {
		if promptChan, ok := prompt.(<-chan map[string]interface{}); ok {
			go c.queryHandler.StreamInput(c.ctx, promptChan)
		}
	}

	return nil
}

// ReceiveMessages receives all messages from Claude.
//
// Returns a channel that yields messages until the client is disconnected
// or an error occurs.
func (c *ClaudeSDKClient) ReceiveMessages(ctx context.Context) <-chan Message {
	msgCh := make(chan Message, 10)

	go func() {
		defer close(msgCh)

		for {
			select {
			case <-ctx.Done():
				return
			case err := <-c.queryHandler.ReceiveErrors():
				if err != nil {
					// Errors are logged but we continue to receive messages
					return
				}
			case data, ok := <-c.queryHandler.ReceiveMessages():
				if !ok {
					return
				}

				msg, err := parseMessage(data)
				if err != nil {
					return
				}

				select {
				case msgCh <- msg:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return msgCh
}

// Query sends a new user message in streaming mode.
//
// The prompt can be either a string or <-chan map[string]interface{}.
func (c *ClaudeSDKClient) Query(ctx context.Context, prompt interface{}, sessionID string) error {
	if c.queryHandler == nil || c.transport == nil {
		return NewCLIConnectionError("not connected. Call Connect() first", nil)
	}

	// Handle string prompts
	if promptStr, ok := prompt.(string); ok {
		message := map[string]interface{}{
			"type": "user",
			"message": map[string]interface{}{
				"role":    "user",
				"content": promptStr,
			},
			"parent_tool_use_id": nil,
			"session_id":         sessionID,
		}
		data, _ := json.Marshal(message)
		return c.transport.Write(ctx, string(data)+"\n")
	}

	// Handle channel prompts
	if promptChan, ok := prompt.(<-chan map[string]interface{}); ok {
		go func() {
			for msg := range promptChan {
				if msg["session_id"] == nil {
					msg["session_id"] = sessionID
				}
				data, _ := json.Marshal(msg)
				c.transport.Write(ctx, string(data)+"\n")
			}
		}()
		return nil
	}

	return fmt.Errorf("prompt must be string or <-chan map[string]interface{}")
}

// Interrupt sends interrupt signal (only works with streaming mode).
func (c *ClaudeSDKClient) Interrupt(ctx context.Context) error {
	if c.queryHandler == nil {
		return NewCLIConnectionError("not connected. Call Connect() first", nil)
	}
	return c.queryHandler.Interrupt(ctx)
}

// SetPermissionMode changes permission mode during conversation.
//
// Valid modes:
//   - "default": CLI prompts for dangerous tools
//   - "acceptEdits": Auto-accept file edits
//   - "bypassPermissions": Allow all tools (use with caution)
func (c *ClaudeSDKClient) SetPermissionMode(ctx context.Context, mode PermissionMode) error {
	if c.queryHandler == nil {
		return NewCLIConnectionError("not connected. Call Connect() first", nil)
	}
	return c.queryHandler.SetPermissionMode(ctx, mode)
}

// SetModel changes the AI model during conversation.
//
// Examples: "claude-sonnet-4-5", "claude-opus-4-20250514"
func (c *ClaudeSDKClient) SetModel(ctx context.Context, model string) error {
	if c.queryHandler == nil {
		return NewCLIConnectionError("not connected. Call Connect() first", nil)
	}
	return c.queryHandler.SetModel(ctx, model)
}

// GetServerInfo retrieves server initialization info including available commands.
//
// Returns initialization information from the Claude Code server including:
//   - Available commands (slash commands, system commands, etc.)
//   - Current and available output styles
//   - Server capabilities
func (c *ClaudeSDKClient) GetServerInfo() map[string]interface{} {
	if c.queryHandler == nil {
		return nil
	}
	return c.queryHandler.GetInitResult()
}

// ReceiveResponse receives messages until and including a ResultMessage.
//
// This is a convenience method over ReceiveMessages() for single-response workflows.
// The channel will close after yielding a ResultMessage.
func (c *ClaudeSDKClient) ReceiveResponse(ctx context.Context) <-chan Message {
	msgCh := make(chan Message, 10)

	go func() {
		defer close(msgCh)

		for msg := range c.ReceiveMessages(ctx) {
			select {
			case msgCh <- msg:
			case <-ctx.Done():
				return
			}

			// Stop after ResultMessage
			if _, ok := msg.(*ResultMessage); ok {
				return
			}
		}
	}()

	return msgCh
}

// Disconnect closes the connection to Claude Code.
func (c *ClaudeSDKClient) Disconnect() error {
	if c.cancel != nil {
		c.cancel()
	}

	if c.queryHandler != nil {
		return c.queryHandler.Close()
	}

	return nil
}
