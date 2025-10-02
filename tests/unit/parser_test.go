package unit

import (
	"testing"

	claude "github.com/clsx524/claude-agent-sdk-go"
)

func TestParseUserMessage(t *testing.T) {
	t.Run("simple string content", func(t *testing.T) {
		data := map[string]interface{}{
			"type": "user",
			"message": map[string]interface{}{
				"role":    "user",
				"content": "Hello Claude",
			},
		}

		msg, err := claude.ParseMessage(data)
		if err != nil {
			t.Fatalf("ParseMessage failed: %v", err)
		}

		userMsg, ok := msg.(*claude.UserMessage)
		if !ok {
			t.Fatalf("expected *UserMessage, got %T", msg)
		}

		if userMsg.Content != "Hello Claude" {
			t.Errorf("expected content 'Hello Claude', got %v", userMsg.Content)
		}
	})

	t.Run("content blocks", func(t *testing.T) {
		data := map[string]interface{}{
			"type": "user",
			"message": map[string]interface{}{
				"role": "user",
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "Hello",
					},
					map[string]interface{}{
						"type":        "tool_result",
						"tool_use_id": "tool_123",
						"content":     "result",
					},
				},
			},
		}

		msg, err := claude.ParseMessage(data)
		if err != nil {
			t.Fatalf("ParseMessage failed: %v", err)
		}

		userMsg, ok := msg.(*claude.UserMessage)
		if !ok {
			t.Fatalf("expected *UserMessage, got %T", msg)
		}

		blocks, ok := userMsg.Content.([]claude.ContentBlock)
		if !ok {
			t.Fatalf("expected []ContentBlock, got %T", userMsg.Content)
		}

		if len(blocks) != 2 {
			t.Fatalf("expected 2 blocks, got %d", len(blocks))
		}

		textBlock, ok := blocks[0].(claude.TextBlock)
		if !ok {
			t.Errorf("expected TextBlock, got %T", blocks[0])
		}
		if textBlock.Text != "Hello" {
			t.Errorf("expected text 'Hello', got %s", textBlock.Text)
		}

		toolResultBlock, ok := blocks[1].(claude.ToolResultBlock)
		if !ok {
			t.Errorf("expected ToolResultBlock, got %T", blocks[1])
		}
		if toolResultBlock.ToolUseID != "tool_123" {
			t.Errorf("expected tool_use_id 'tool_123', got %s", toolResultBlock.ToolUseID)
		}
	})
}

func TestParseAssistantMessage(t *testing.T) {
	data := map[string]interface{}{
		"type": "assistant",
		"message": map[string]interface{}{
			"role":  "assistant",
			"model": "claude-sonnet-4-5",
			"content": []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Hello!",
				},
				map[string]interface{}{
					"type":      "thinking",
					"thinking":  "Let me think...",
					"signature": "sig123",
				},
				map[string]interface{}{
					"type":  "tool_use",
					"id":    "tool_456",
					"name":  "Read",
					"input": map[string]interface{}{"path": "/test"},
				},
			},
		},
	}

	msg, err := claude.ParseMessage(data)
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}

	assistantMsg, ok := msg.(*claude.AssistantMessage)
	if !ok {
		t.Fatalf("expected *AssistantMessage, got %T", msg)
	}

	if assistantMsg.Model != "claude-sonnet-4-5" {
		t.Errorf("expected model 'claude-sonnet-4-5', got %s", assistantMsg.Model)
	}

	if len(assistantMsg.Content) != 3 {
		t.Fatalf("expected 3 content blocks, got %d", len(assistantMsg.Content))
	}

	// Check text block
	textBlock, ok := assistantMsg.Content[0].(claude.TextBlock)
	if !ok {
		t.Errorf("expected TextBlock, got %T", assistantMsg.Content[0])
	}
	if textBlock.Text != "Hello!" {
		t.Errorf("expected text 'Hello!', got %s", textBlock.Text)
	}

	// Check thinking block
	thinkingBlock, ok := assistantMsg.Content[1].(claude.ThinkingBlock)
	if !ok {
		t.Errorf("expected ThinkingBlock, got %T", assistantMsg.Content[1])
	}
	if thinkingBlock.Thinking != "Let me think..." {
		t.Errorf("expected thinking 'Let me think...', got %s", thinkingBlock.Thinking)
	}

	// Check tool use block
	toolUseBlock, ok := assistantMsg.Content[2].(claude.ToolUseBlock)
	if !ok {
		t.Errorf("expected ToolUseBlock, got %T", assistantMsg.Content[2])
	}
	if toolUseBlock.Name != "Read" {
		t.Errorf("expected tool name 'Read', got %s", toolUseBlock.Name)
	}
}

func TestParseResultMessage(t *testing.T) {
	data := map[string]interface{}{
		"type":            "result",
		"subtype":         "success",
		"duration_ms":     1000.0,
		"duration_api_ms": 800.0,
		"is_error":        false,
		"num_turns":       5.0,
		"session_id":      "session_123",
		"total_cost_usd":  0.05,
		"usage": map[string]interface{}{
			"input_tokens":  100.0,
			"output_tokens": 50.0,
		},
		"result": "completed",
	}

	msg, err := claude.ParseMessage(data)
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}

	resultMsg, ok := msg.(*claude.ResultMessage)
	if !ok {
		t.Fatalf("expected *ResultMessage, got %T", msg)
	}

	if resultMsg.Subtype != "success" {
		t.Errorf("expected subtype 'success', got %s", resultMsg.Subtype)
	}
	if resultMsg.DurationMS != 1000 {
		t.Errorf("expected duration_ms 1000, got %d", resultMsg.DurationMS)
	}
	if resultMsg.SessionID != "session_123" {
		t.Errorf("expected session_id 'session_123', got %s", resultMsg.SessionID)
	}
	if resultMsg.TotalCostUSD == nil || *resultMsg.TotalCostUSD != 0.05 {
		t.Errorf("expected total_cost_usd 0.05, got %v", resultMsg.TotalCostUSD)
	}
}

func TestParseSystemMessage(t *testing.T) {
	data := map[string]interface{}{
		"type":    "system",
		"subtype": "init",
		"data": map[string]interface{}{
			"commands": []interface{}{"help", "exit"},
		},
	}

	msg, err := claude.ParseMessage(data)
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}

	systemMsg, ok := msg.(*claude.SystemMessage)
	if !ok {
		t.Fatalf("expected *SystemMessage, got %T", msg)
	}

	if systemMsg.Subtype != "init" {
		t.Errorf("expected subtype 'init', got %s", systemMsg.Subtype)
	}
}

func TestParseStreamEvent(t *testing.T) {
	data := map[string]interface{}{
		"type":       "stream_event",
		"uuid":       "event_123",
		"session_id": "session_456",
		"event": map[string]interface{}{
			"type":  "content_block_delta",
			"delta": map[string]interface{}{"text": "Hello"},
		},
	}

	msg, err := claude.ParseMessage(data)
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}

	streamEvent, ok := msg.(*claude.StreamEvent)
	if !ok {
		t.Fatalf("expected *StreamEvent, got %T", msg)
	}

	if streamEvent.UUID != "event_123" {
		t.Errorf("expected uuid 'event_123', got %s", streamEvent.UUID)
	}
	if streamEvent.SessionID != "session_456" {
		t.Errorf("expected session_id 'session_456', got %s", streamEvent.SessionID)
	}
}

func TestParseMessageErrors(t *testing.T) {
	tests := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "nil data",
			data: nil,
		},
		{
			name: "missing type",
			data: map[string]interface{}{
				"message": "test",
			},
		},
		{
			name: "unknown type",
			data: map[string]interface{}{
				"type": "unknown",
			},
		},
		{
			name: "user message missing message field",
			data: map[string]interface{}{
				"type": "user",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := claude.ParseMessage(tt.data)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}
