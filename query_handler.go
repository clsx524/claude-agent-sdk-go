package claude

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// queryHandler handles bidirectional control protocol on top of Transport.
type queryHandler struct {
	transport       Transport
	isStreamingMode bool
	canUseTool      CanUseTool
	hooks           map[string][]hookMatcherInternal
	sdkMcpServers   map[string]interface{} // Map of server name to MCP server instance

	// Control protocol state
	pendingControlResponses map[string]chan controlResult
	hookCallbacks           map[string]HookCallback
	nextCallbackID          int
	requestCounter          int
	mu                      sync.Mutex

	// Message streaming
	messageChan chan map[string]interface{}
	errorChan   chan error
	cancelFunc  context.CancelFunc
	initialized bool
	initResult  map[string]interface{}
}

type controlResult struct {
	response map[string]interface{}
	err      error
}

// New creates a new queryHandler instance.
func newQueryHandler(
	transport Transport,
	isStreamingMode bool,
	canUseTool CanUseTool,
	hooks map[HookEvent][]HookMatcher,
	sdkMcpServers map[string]interface{},
	bufferSize int,
) *queryHandler {
	// Convert hooks to internal format using helper function
	internalHooks := convertHooksToInternal(hooks)

	// Use default buffer size if not specified or invalid
	if bufferSize <= 0 {
		bufferSize = 100
	}

	return &queryHandler{
		transport:               transport,
		isStreamingMode:         isStreamingMode,
		canUseTool:              canUseTool,
		hooks:                   internalHooks,
		sdkMcpServers:           sdkMcpServers,
		pendingControlResponses: make(map[string]chan controlResult),
		hookCallbacks:           make(map[string]HookCallback),
		messageChan:             make(chan map[string]interface{}, bufferSize),
		errorChan:               make(chan error, 1),
	}
}

// Start begins reading messages from transport.
func (q *queryHandler) Start(ctx context.Context) error {
	msgCh, errCh := q.transport.ReadMessages(ctx)

	ctx, cancel := context.WithCancel(ctx)
	q.cancelFunc = cancel

	// Start message router
	go q.routeMessages(ctx, msgCh, errCh)

	return nil
}

// routeMessages reads from transport and routes control vs regular messages.
func (q *queryHandler) routeMessages(ctx context.Context, msgCh <-chan map[string]interface{}, errCh <-chan error) {
	defer close(q.messageChan)
	defer close(q.errorChan)

	for {
		select {
		case <-ctx.Done():
			return
		case err := <-errCh:
			if err != nil {
				q.errorChan <- err
			}
			return
		case msg, ok := <-msgCh:
			if !ok {
				return
			}

			msgType, _ := msg["type"].(string)

			switch msgType {
			case "control_response":
				q.handleControlResponse(msg)
			case "control_request":
				go q.handleControlRequest(ctx, msg)
			case "control_cancel_request":
				// TODO: Implement cancellation
			default:
				// Regular SDK message
				select {
				case q.messageChan <- msg:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

// Initialize sends initialization request (streaming mode only).
func (q *queryHandler) Initialize(ctx context.Context) (map[string]interface{}, error) {
	if !q.isStreamingMode {
		return nil, nil
	}

	// Build hooks configuration
	hooksConfig := make(map[string]interface{})
	if len(q.hooks) > 0 {
		for event, matchers := range q.hooks {
			if len(matchers) == 0 {
				continue
			}

			matcherConfigs := make([]map[string]interface{}, len(matchers))
			for i, matcher := range matchers {
				callbackIDs := make([]string, len(matcher.Hooks))
				for j, callback := range matcher.Hooks {
					callbackID := fmt.Sprintf("hook_%d", q.nextCallbackID)
					q.nextCallbackID++
					q.hookCallbacks[callbackID] = callback
					callbackIDs[j] = callbackID
				}

				matcherConfigs[i] = map[string]interface{}{
					"matcher":         matcher.Matcher,
					"hookCallbackIds": callbackIDs,
				}
			}
			hooksConfig[event] = matcherConfigs
		}
	}

	request := map[string]interface{}{
		"subtype": "initialize",
	}
	if len(hooksConfig) > 0 {
		request["hooks"] = hooksConfig
	}

	response, err := q.sendControlRequest(ctx, request)
	if err != nil {
		return nil, err
	}

	q.initialized = true
	q.initResult = response
	return response, nil
}

// sendControlRequest sends a control request and waits for response.
func (q *queryHandler) sendControlRequest(ctx context.Context, request map[string]interface{}) (map[string]interface{}, error) {
	if !q.isStreamingMode {
		return nil, fmt.Errorf("control requests require streaming mode")
	}

	q.mu.Lock()
	q.requestCounter++
	requestID := fmt.Sprintf("req_%d_%s", q.requestCounter, randomHex(4))
	resultChan := make(chan controlResult, 1)
	q.pendingControlResponses[requestID] = resultChan
	q.mu.Unlock()

	defer func() {
		q.mu.Lock()
		delete(q.pendingControlResponses, requestID)
		q.mu.Unlock()
	}()

	// Build and send request
	controlRequest := map[string]interface{}{
		"type":       "control_request",
		"request_id": requestID,
		"request":    request,
	}

	data, err := json.Marshal(controlRequest)
	if err != nil {
		return nil, err
	}

	if err := q.transport.Write(ctx, string(data)+"\n"); err != nil {
		return nil, err
	}

	// Wait for response with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	select {
	case result := <-resultChan:
		if result.err != nil {
			return nil, result.err
		}
		return result.response, nil
	case <-timeoutCtx.Done():
		return nil, fmt.Errorf("control request timeout: %s", request["subtype"])
	}
}

// handleControlResponse routes control responses to waiting requests.
func (q *queryHandler) handleControlResponse(msg map[string]interface{}) {
	response, ok := msg["response"].(map[string]interface{})
	if !ok {
		return
	}

	requestID, ok := response["request_id"].(string)
	if !ok {
		return
	}

	q.mu.Lock()
	resultChan, exists := q.pendingControlResponses[requestID]
	q.mu.Unlock()

	if !exists {
		return
	}

	subtype, _ := response["subtype"].(string)
	if subtype == "error" {
		errorMsg, _ := response["error"].(string)
		resultChan <- controlResult{err: fmt.Errorf("%s", errorMsg)}
	} else {
		responseData, _ := response["response"].(map[string]interface{})
		resultChan <- controlResult{response: responseData}
	}
}

// handleControlRequest processes incoming control requests from CLI.
func (q *queryHandler) handleControlRequest(ctx context.Context, msg map[string]interface{}) {
	requestID, _ := msg["request_id"].(string)
	request, _ := msg["request"].(map[string]interface{})
	subtype, _ := request["subtype"].(string)

	var responseData map[string]interface{}
	var err error

	switch subtype {
	case "can_use_tool":
		responseData, err = q.handleCanUseTool(ctx, request)
	case "hook_callback":
		responseData, err = q.handleHookCallback(ctx, request)
	case "mcp_message":
		responseData, err = q.handleMcpMessage(ctx, request)
	default:
		err = fmt.Errorf("unsupported control request subtype: %s", subtype)
	}

	// Send response
	var controlResponse map[string]interface{}
	if err != nil {
		controlResponse = map[string]interface{}{
			"type": "control_response",
			"response": map[string]interface{}{
				"subtype":    "error",
				"request_id": requestID,
				"error":      err.Error(),
			},
		}
	} else {
		controlResponse = map[string]interface{}{
			"type": "control_response",
			"response": map[string]interface{}{
				"subtype":    "success",
				"request_id": requestID,
				"response":   responseData,
			},
		}
	}

	data, _ := json.Marshal(controlResponse)
	q.transport.Write(ctx, string(data)+"\n")
}

// handleCanUseTool processes tool permission requests.
func (q *queryHandler) handleCanUseTool(ctx context.Context, request map[string]interface{}) (map[string]interface{}, error) {
	if q.canUseTool == nil {
		return nil, fmt.Errorf("canUseTool callback is not provided")
	}

	toolName, _ := request["tool_name"].(string)
	originalInput, _ := request["input"].(map[string]interface{})
	suggestions, _ := request["permission_suggestions"].([]interface{})

	// Convert suggestions
	permSuggestions := make([]PermissionUpdate, 0)
	for _, s := range suggestions {
		if sMap, ok := s.(map[string]interface{}); ok {
			// TODO: Properly unmarshal PermissionUpdate
			_ = sMap
		}
	}

	permCtx := ToolPermissionContext{
		Suggestions: permSuggestions,
	}

	result, err := q.canUseTool(ctx, toolName, originalInput, permCtx)
	if err != nil {
		return nil, err
	}

	// Convert result to response format matching Python SDK
	switch r := result.(type) {
	case PermissionResultAllow:
		// Use JSON marshaling to ensure proper field names (behavior, updatedInput, updatedPermissions)
		response := map[string]interface{}{
			"behavior": "allow",
		}
		// Use updatedInput if provided, otherwise use original input
		if r.UpdatedInput != nil {
			response["updatedInput"] = r.UpdatedInput
		} else {
			response["updatedInput"] = originalInput
		}
		if r.UpdatedPermissions != nil && len(r.UpdatedPermissions) > 0 {
			response["updatedPermissions"] = r.UpdatedPermissions
		}
		return response, nil
	case PermissionResultDeny:
		response := map[string]interface{}{
			"behavior": "deny",
			"message":  r.Message,
		}
		if r.Interrupt {
			response["interrupt"] = r.Interrupt
		}
		return response, nil
	default:
		return nil, fmt.Errorf("invalid permission result type")
	}
}

// handleHookCallback processes hook callback requests.
func (q *queryHandler) handleHookCallback(ctx context.Context, request map[string]interface{}) (map[string]interface{}, error) {
	callbackID, _ := request["callback_id"].(string)
	input, _ := request["input"].(map[string]interface{})

	var toolUseID *string
	if tuid, ok := request["tool_use_id"].(string); ok {
		toolUseID = &tuid
	}

	callback, exists := q.hookCallbacks[callbackID]
	if !exists {
		return nil, fmt.Errorf("no hook callback found for ID: %s", callbackID)
	}

	hookCtx := HookContext{}
	result, err := callback(ctx, input, toolUseID, hookCtx)
	if err != nil {
		return nil, err
	}

	// Convert HookJSONOutput to map using JSON marshaling to ensure all fields
	// are properly serialized with correct JSON tags (e.g., "continue", "async")
	var response map[string]interface{}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal hook result: %w", err)
	}
	if err := json.Unmarshal(resultJSON, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal hook result: %w", err)
	}

	return response, nil
}

// handleMcpMessage handles SDK MCP server requests.
func (q *queryHandler) handleMcpMessage(ctx context.Context, request map[string]interface{}) (map[string]interface{}, error) {
	serverName, _ := request["server_name"].(string)
	message, _ := request["message"].(map[string]interface{})

	if serverName == "" || message == nil {
		return nil, fmt.Errorf("missing server_name or message for MCP request")
	}

	server, exists := q.sdkMcpServers[serverName]
	if !exists {
		return map[string]interface{}{
			"mcp_response": map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      message["id"],
				"error": map[string]interface{}{
					"code":    -32601,
					"message": fmt.Sprintf("Server '%s' not found", serverName),
				},
			},
		}, nil
	}

	// Route MCP request to server
	response := q.routeMcpRequest(ctx, server, message)
	return map[string]interface{}{"mcp_response": response}, nil
}

// routeMcpRequest routes JSONRPC requests to MCP server.
func (q *queryHandler) routeMcpRequest(ctx context.Context, server interface{}, message map[string]interface{}) map[string]interface{} {
	// Check if it's an SDK MCP server
	type McpServerHandler interface {
		HandleRequest(ctx context.Context, message map[string]interface{}) map[string]interface{}
	}

	if handler, ok := server.(McpServerHandler); ok {
		return handler.HandleRequest(ctx, message)
	}

	// Unknown server type
	msgID := message["id"]
	return map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      msgID,
		"error": map[string]interface{}{
			"code":    -32601,
			"message": "Server does not implement MCP protocol",
		},
	}
}

// Interrupt sends interrupt control request.
func (q *queryHandler) Interrupt(ctx context.Context) error {
	request := map[string]interface{}{"subtype": "interrupt"}
	_, err := q.sendControlRequest(ctx, request)
	return err
}

// SetPermissionMode changes permission mode.
func (q *queryHandler) SetPermissionMode(ctx context.Context, mode PermissionMode) error {
	request := map[string]interface{}{
		"subtype": "set_permission_mode",
		"mode":    string(mode),
	}
	_, err := q.sendControlRequest(ctx, request)
	return err
}

// SetModel changes the AI model.
func (q *queryHandler) SetModel(ctx context.Context, model string) error {
	request := map[string]interface{}{
		"subtype": "set_model",
		"model":   model,
	}
	_, err := q.sendControlRequest(ctx, request)
	return err
}

// StreamInput streams input messages to transport.
func (q *queryHandler) StreamInput(ctx context.Context, stream <-chan map[string]interface{}) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-stream:
			if !ok {
				// Stream closed, end input
				return q.transport.EndInput()
			}
			data, err := json.Marshal(msg)
			if err != nil {
				return err
			}
			if err := q.transport.Write(ctx, string(data)+"\n"); err != nil {
				return err
			}
		}
	}
}

// ReceiveMessages returns a channel for receiving SDK messages.
func (q *queryHandler) ReceiveMessages() <-chan map[string]interface{} {
	return q.messageChan
}

// ReceiveErrors returns a channel for receiving errors.
func (q *queryHandler) ReceiveErrors() <-chan error {
	return q.errorChan
}

// GetInitResult returns the initialization result.
func (q *queryHandler) GetInitResult() map[string]interface{} {
	return q.initResult
}

// Close closes the query and transport.
func (q *queryHandler) Close() error {
	if q.cancelFunc != nil {
		q.cancelFunc()
	}
	return q.transport.Close()
}

// randomHex generates random hex string.
func randomHex(n int) string {
	bytes := make([]byte, n)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
