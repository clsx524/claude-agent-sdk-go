package claude

import (
	"context"
	"os"
)

// Query performs a one-shot or unidirectional streaming query to Claude Code.
//
// This function is ideal for simple, stateless queries where you don't need
// bidirectional communication or conversation management. For interactive,
// stateful conversations, use ClaudeSDKClient instead.
//
// Key differences from ClaudeSDKClient:
//   - Unidirectional: Send all messages upfront, receive all responses
//   - Stateless: Each query is independent, no conversation state
//   - Simple: Fire-and-forget style, no connection management
//   - No interrupts: Cannot interrupt or send follow-up messages
//
// Example:
//
//	ctx := context.Background()
//	msgCh, errCh, err := Query(ctx, "What is 2+2?", nil, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for msg := range msgCh {
//	    if assistantMsg, ok := msg.(*AssistantMessage); ok {
//	        for _, block := range assistantMsg.Content {
//	            if textBlock, ok := block.(TextBlock); ok {
//	                fmt.Println(textBlock.Text)
//	            }
//	        }
//	    }
//	}
//
//	if err := <-errCh; err != nil {
//	    log.Fatal(err)
//	}
func Query(
	ctx context.Context,
	prompt string,
	options *ClaudeAgentOptions,
	trans Transport,
) (<-chan Message, <-chan error, error) {
	os.Setenv("CLAUDE_CODE_ENTRYPOINT", "sdk-go")
	return processQuery(ctx, prompt, options, trans)
}

// QueryStream performs a streaming query with multiple input messages.
//
// Example:
//
//	ctx := context.Background()
//	promptCh := make(chan map[string]interface{})
//
//	go func() {
//	    defer close(promptCh)
//	    promptCh <- map[string]interface{}{
//	        "type": "user",
//	        "message": map[string]interface{}{
//	            "role": "user",
//	            "content": "Hello",
//	        },
//	    }
//	}()
//
//	msgCh, errCh, err := QueryStream(ctx, promptCh, nil, nil)
func QueryStream(
	ctx context.Context,
	prompts <-chan map[string]interface{},
	options *ClaudeAgentOptions,
	trans Transport,
) (<-chan Message, <-chan error, error) {
	os.Setenv("CLAUDE_CODE_ENTRYPOINT", "sdk-go")
	return processQuery(ctx, prompts, options, trans)
}

// processQuery is the internal implementation for Query and QueryStream
func processQuery(
	ctx context.Context,
	prompt interface{}, // string or <-chan map[string]interface{}
	options *ClaudeAgentOptions,
	trans Transport,
) (<-chan Message, <-chan error, error) {
	if options == nil {
		options = &ClaudeAgentOptions{}
	}

	// Validate and configure permission settings
	_, isStreaming := prompt.(<-chan map[string]interface{})
	configuredOptions, err := validateAndConfigurePermissions(options, isStreaming)
	if err != nil {
		return nil, nil, err
	}

	// Use provided transport or create subprocess transport
	chosenTransport := trans
	if chosenTransport == nil {
		var err error
		chosenTransport, err = NewSubprocessCLITransport(prompt, configuredOptions, "")
		if err != nil {
			return nil, nil, err
		}
	}

	// Connect transport
	if err := chosenTransport.Connect(ctx); err != nil {
		return nil, nil, err
	}

	// Extract SDK MCP servers using helper function
	sdkMcpServers := extractSdkMcpServers(configuredOptions.McpServers)

	// Determine buffer size
	bufferSize := 100 // default
	if configuredOptions.MessageChannelBufferSize != nil && *configuredOptions.MessageChannelBufferSize > 0 {
		bufferSize = *configuredOptions.MessageChannelBufferSize
	}

	// Create queryHandler to handle control protocol
	q := newQueryHandler(
		chosenTransport,
		isStreaming,
		configuredOptions.CanUseTool,
		configuredOptions.Hooks,
		sdkMcpServers,
		bufferSize,
	)

	// Start reading messages
	if err := q.Start(ctx); err != nil {
		return nil, nil, err
	}

	// Initialize if streaming
	if isStreaming {
		if _, err := q.Initialize(ctx); err != nil {
			return nil, nil, err
		}

		// Stream input in background
		if promptChan, ok := prompt.(<-chan map[string]interface{}); ok {
			go func() {
				q.StreamInput(ctx, promptChan)
			}()
		}
	}

	// Create output channels
	msgCh := make(chan Message, 10)
	errCh := make(chan error, 1)

	// Parse and yield messages
	go func() {
		defer close(msgCh)
		defer close(errCh)
		defer q.Close()

		for {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			case err := <-q.ReceiveErrors():
				if err != nil {
					errCh <- err
					return
				}
			case data, ok := <-q.ReceiveMessages():
				if !ok {
					return
				}
				msg, err := parseMessage(data)
				if err != nil {
					errCh <- err
					return
				}
				select {
				case msgCh <- msg:
				case <-ctx.Done():
					errCh <- ctx.Err()
					return
				}
			}
		}
	}()

	return msgCh, errCh, nil
}
