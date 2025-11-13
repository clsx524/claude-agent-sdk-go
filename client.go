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
//	if err := client.Connect(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	msgCh, errCh := client.Query(ctx, "Hello Claude")
//	for msg := range msgCh {
//	    fmt.Printf("%+v\n", msg)
//	}
//	if err := <-errCh; err != nil {
//	    log.Fatal(err)
//	}
type ClaudeSDKClient struct {
	options         *ClaudeAgentOptions
	customTransport Transport
	transport       Transport
	queryHandler    *queryHandler
	ctx             context.Context
	cancel          context.CancelFunc
	currentSession  string // Auto-managed session ID
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
// This method initializes the connection without sending any prompt.
// Use Query() to send messages after connecting.
//
// Example:
//
//	client := claude.NewClaudeSDKClient(nil)
//	ctx := context.Background()
//
//	if err := client.Connect(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	// Now ready to send queries
//	msgCh, errCh := client.Query(ctx, "Hello!")
//	for msg := range msgCh {
//	    // Process messages
//	}
func (c *ClaudeSDKClient) Connect(ctx context.Context) error {
	return c.ConnectWithPrompt(ctx, nil)
}

// ConnectWithPrompt establishes connection to Claude Code with an initial prompt.
//
// The prompt parameter can be:
//   - nil: Empty connection for interactive use
//   - string: Initial prompt message
//   - <-chan map[string]interface{}: Stream of input messages
//
// For most cases, use Connect() and then Query() instead.
func (c *ClaudeSDKClient) ConnectWithPrompt(ctx context.Context, prompt interface{}) error {
	os.Setenv("CLAUDE_CODE_ENTRYPOINT", "sdk-go-client")

	// Create cancellable context
	c.ctx, c.cancel = context.WithCancel(ctx)

	// Determine actual prompt (empty channel if nil)
	actualPrompt := prompt
	if actualPrompt == nil {
		ch := make(chan map[string]interface{})
		close(ch)
		var emptyCh <-chan map[string]interface{} = ch
		actualPrompt = emptyCh
	}

	// Validate and configure permission settings
	_, isString := prompt.(string)
	options, err := validateAndConfigurePermissions(c.options, !isString)
	if err != nil {
		return err
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

	// Extract SDK MCP servers using helper function
	sdkMcpServers := extractSdkMcpServers(c.options.McpServers)

	// Determine buffer size
	bufferSize := 100 // default
	if options.MessageChannelBufferSize != nil && *options.MessageChannelBufferSize > 0 {
		bufferSize = *options.MessageChannelBufferSize
	}

	// Create queryHandler - ClaudeSDKClient always uses streaming mode
	c.queryHandler = newQueryHandler(
		c.transport,
		true, // Always streaming mode
		options.CanUseTool,
		options.Hooks,
		sdkMcpServers,
		bufferSize,
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
//
// IMPORTANT: Only ONE goroutine should call ReceiveMessages() to avoid competing
// readers on the underlying queryHandler channel. For multi-query workflows,
// use Query() which properly manages message distribution.
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

// Query sends a new user message and returns channels for receiving responses.
//
// IMPORTANT: For multi-query workflows (calling Query() multiple times), you MUST
// fully consume the returned channels before calling Query() again. Otherwise, use
// the Python-style pattern: call QueryWithSession() to send, then call ReceiveResponse()
// or ReceiveMessages() directly to receive.
//
// The returned channels will receive ALL messages (not just for this query) until a
// ResultMessage is received. If you need finer control, use QueryWithSession() +
// ReceiveResponse() separately.
//
// Returns:
//   - Message channel: Receives messages until ResultMessage
//   - Error channel: Receives any errors that occur
//
// Example - Single query:
//
//	msgCh, errCh := client.Query(ctx, "Hello Claude")
//	for msg := range msgCh {
//	    // Process message
//	}
//	if err := <-errCh; err != nil {
//	    // Handle error
//	}
//
// Example - Multiple queries (Python-style):
//
//	// First query
//	client.QueryWithSession(ctx, "What is 2+2?", "default")
//	for msg := range client.ReceiveResponse(ctx) {
//	    // Process first response
//	}
//	// Second query
//	client.QueryWithSession(ctx, "What is 3+3?", "default")
//	for msg := range client.ReceiveResponse(ctx) {
//	    // Process second response
//	}
func (c *ClaudeSDKClient) Query(ctx context.Context, prompt string) (<-chan Message, <-chan error) {
	// Auto-generate session ID if not set
	if c.currentSession == "" {
		c.currentSession = "default"
	}

	// Send the query
	err := c.QueryWithSession(ctx, prompt, c.currentSession)
	if err != nil {
		// Return channels with error
		msgCh := make(chan Message)
		errCh := make(chan error, 1)
		close(msgCh)
		errCh <- err
		close(errCh)
		return msgCh, errCh
	}

	// Return the shared response channel directly
	// This matches Python's behavior where multiple query() calls share the same receive_response()
	return c.wrapReceiveResponseWithError(ctx)
}

// wrapReceiveResponseWithError wraps ReceiveResponse to also return an error channel
func (c *ClaudeSDKClient) wrapReceiveResponseWithError(ctx context.Context) (<-chan Message, <-chan error) {
	msgCh := make(chan Message, 10)
	errCh := make(chan error, 1)

	go func() {
		defer close(msgCh)
		defer close(errCh)

		for msg := range c.ReceiveResponse(ctx) {
			select {
			case msgCh <- msg:
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			}
		}
	}()

	return msgCh, errCh
}

// QueryWithSession sends a new user message with an explicit session ID.
//
// For most cases, use Query() which auto-manages session IDs.
// The prompt can be either a string or <-chan map[string]interface{}.
func (c *ClaudeSDKClient) QueryWithSession(ctx context.Context, prompt interface{}, sessionID string) error {
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
//
// Example:
//
//	client := claude.NewClaudeSDKClient(nil)
//	ctx := context.Background()
//	client.Connect(ctx)
//	defer client.Close()
//
//	// Start a long-running query
//	go client.Query(ctx, "Analyze this entire codebase...")
//
//	// User decides to cancel
//	if err := client.Interrupt(ctx); err != nil {
//	    log.Printf("Failed to interrupt: %v", err)
//	}
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

		// Create a cancellable context so we can stop ReceiveMessages when done
		receiveCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		for msg := range c.ReceiveMessages(receiveCtx) {
			select {
			case msgCh <- msg:
			case <-ctx.Done():
				return
			}

			// Stop after ResultMessage and cancel the context to stop ReceiveMessages
			if _, ok := msg.(*ResultMessage); ok {
				cancel()
				return
			}
		}
	}()

	return msgCh
}

// Close closes the connection to Claude Code.
//
// This is the preferred method name for Python API compatibility.
// It's an alias for Disconnect().
func (c *ClaudeSDKClient) Close() error {
	return c.Disconnect()
}

// Disconnect closes the connection to Claude Code.
//
// Prefer using Close() for consistency with Python SDK.
func (c *ClaudeSDKClient) Disconnect() error {
	if c.cancel != nil {
		c.cancel()
	}

	if c.queryHandler != nil {
		return c.queryHandler.Close()
	}

	return nil
}
