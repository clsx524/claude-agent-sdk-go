package claude

import (
	"fmt"
)

// ParseMessage parses a raw message dictionary into a typed Message object.
// This is exported for testing purposes.
func ParseMessage(data map[string]interface{}) (Message, error) {
	return parseMessage(data)
}

// parseMessage parses a raw message dictionary into a typed Message object.
func parseMessage(data map[string]interface{}) (Message, error) {
	if data == nil {
		return nil, NewMessageParseError("message data is nil", nil)
	}

	msgType, ok := data["type"].(string)
	if !ok {
		return nil, NewMessageParseError("message missing 'type' field", data)
	}

	switch msgType {
	case "user":
		return parseUserMessage(data)
	case "assistant":
		return parseAssistantMessage(data)
	case "system":
		return parseSystemMessage(data)
	case "result":
		return parseResultMessage(data)
	case "stream_event":
		return parseStreamEvent(data)
	default:
		return nil, NewMessageParseError(fmt.Sprintf("unknown message type: %s", msgType), data)
	}
}

func parseUserMessage(data map[string]interface{}) (*UserMessage, error) {
	message, ok := data["message"].(map[string]interface{})
	if !ok {
		return nil, NewMessageParseError("user message missing 'message' field", data)
	}

	content := message["content"]
	var parentToolUseID *string
	if pid, ok := data["parent_tool_use_id"].(string); ok {
		parentToolUseID = &pid
	}

	// Content can be string or []ContentBlock
	if contentStr, ok := content.(string); ok {
		return &UserMessage{
			Content:         contentStr,
			ParentToolUseID: parentToolUseID,
		}, nil
	}

	// Parse content blocks
	contentArray, ok := content.([]interface{})
	if !ok {
		return nil, NewMessageParseError("user message content must be string or array", data)
	}

	blocks := make([]ContentBlock, 0, len(contentArray))
	for _, item := range contentArray {
		block, err := parseContentBlock(item)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}

	return &UserMessage{
		Content:         blocks,
		ParentToolUseID: parentToolUseID,
	}, nil
}

func parseAssistantMessage(data map[string]interface{}) (*AssistantMessage, error) {
	message, ok := data["message"].(map[string]interface{})
	if !ok {
		return nil, NewMessageParseError("assistant message missing 'message' field", data)
	}

	model, ok := message["model"].(string)
	if !ok {
		return nil, NewMessageParseError("assistant message missing 'model' field", data)
	}

	contentArray, ok := message["content"].([]interface{})
	if !ok {
		return nil, NewMessageParseError("assistant message content must be array", data)
	}

	blocks := make([]ContentBlock, 0, len(contentArray))
	for _, item := range contentArray {
		block, err := parseContentBlock(item)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}

	var parentToolUseID *string
	if pid, ok := data["parent_tool_use_id"].(string); ok {
		parentToolUseID = &pid
	}

	return &AssistantMessage{
		Content:         blocks,
		Model:           model,
		ParentToolUseID: parentToolUseID,
	}, nil
}

func parseContentBlock(item interface{}) (ContentBlock, error) {
	block, ok := item.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("content block must be object")
	}

	blockType, ok := block["type"].(string)
	if !ok {
		return nil, fmt.Errorf("content block missing 'type' field")
	}

	switch blockType {
	case "text":
		text, ok := block["text"].(string)
		if !ok {
			return nil, fmt.Errorf("text block missing 'text' field")
		}
		return TextBlock{Text: text}, nil

	case "thinking":
		thinking, ok := block["thinking"].(string)
		if !ok {
			return nil, fmt.Errorf("thinking block missing 'thinking' field")
		}
		signature, ok := block["signature"].(string)
		if !ok {
			return nil, fmt.Errorf("thinking block missing 'signature' field")
		}
		return ThinkingBlock{Thinking: thinking, Signature: signature}, nil

	case "tool_use":
		id, ok := block["id"].(string)
		if !ok {
			return nil, fmt.Errorf("tool_use block missing 'id' field")
		}
		name, ok := block["name"].(string)
		if !ok {
			return nil, fmt.Errorf("tool_use block missing 'name' field")
		}
		input, ok := block["input"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("tool_use block missing 'input' field")
		}
		return ToolUseBlock{ID: id, Name: name, Input: input}, nil

	case "tool_result":
		toolUseID, ok := block["tool_use_id"].(string)
		if !ok {
			return nil, fmt.Errorf("tool_result block missing 'tool_use_id' field")
		}
		result := ToolResultBlock{ToolUseID: toolUseID}
		if content, ok := block["content"]; ok {
			result.Content = content
		}
		if isError, ok := block["is_error"].(bool); ok {
			result.IsError = &isError
		}
		return result, nil

	default:
		return nil, fmt.Errorf("unknown content block type: %s", blockType)
	}
}

func parseSystemMessage(data map[string]interface{}) (*SystemMessage, error) {
	subtype, ok := data["subtype"].(string)
	if !ok {
		return nil, NewMessageParseError("system message missing 'subtype' field", data)
	}

	return &SystemMessage{
		Subtype: subtype,
		Data:    data,
	}, nil
}

func parseResultMessage(data map[string]interface{}) (*ResultMessage, error) {
	subtype, ok := data["subtype"].(string)
	if !ok {
		return nil, NewMessageParseError("result message missing 'subtype' field", data)
	}

	durationMS, ok := data["duration_ms"].(float64)
	if !ok {
		return nil, NewMessageParseError("result message missing 'duration_ms' field", data)
	}

	durationAPIMS, ok := data["duration_api_ms"].(float64)
	if !ok {
		return nil, NewMessageParseError("result message missing 'duration_api_ms' field", data)
	}

	isError, ok := data["is_error"].(bool)
	if !ok {
		return nil, NewMessageParseError("result message missing 'is_error' field", data)
	}

	numTurns, ok := data["num_turns"].(float64)
	if !ok {
		return nil, NewMessageParseError("result message missing 'num_turns' field", data)
	}

	sessionID, ok := data["session_id"].(string)
	if !ok {
		return nil, NewMessageParseError("result message missing 'session_id' field", data)
	}

	result := &ResultMessage{
		Subtype:       subtype,
		DurationMS:    int(durationMS),
		DurationAPIMS: int(durationAPIMS),
		IsError:       isError,
		NumTurns:      int(numTurns),
		SessionID:     sessionID,
	}

	if totalCostUSD, ok := data["total_cost_usd"].(float64); ok {
		result.TotalCostUSD = &totalCostUSD
	}

	if usage, ok := data["usage"].(map[string]interface{}); ok {
		result.Usage = usage
	}

	if resultStr, ok := data["result"].(string); ok {
		result.Result = &resultStr
	}

	return result, nil
}

func parseStreamEvent(data map[string]interface{}) (*StreamEvent, error) {
	uuid, ok := data["uuid"].(string)
	if !ok {
		return nil, NewMessageParseError("stream_event message missing 'uuid' field", data)
	}

	sessionID, ok := data["session_id"].(string)
	if !ok {
		return nil, NewMessageParseError("stream_event message missing 'session_id' field", data)
	}

	event, ok := data["event"].(map[string]interface{})
	if !ok {
		return nil, NewMessageParseError("stream_event message missing 'event' field", data)
	}

	streamEvent := &StreamEvent{
		UUID:      uuid,
		SessionID: sessionID,
		Event:     event,
	}

	if pid, ok := data["parent_tool_use_id"].(string); ok {
		streamEvent.ParentToolUseID = &pid
	}

	return streamEvent, nil
}
