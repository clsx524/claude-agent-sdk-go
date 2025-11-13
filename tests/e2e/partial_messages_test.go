package e2e

import (
	"context"
	"strings"
	"testing"

	claude "github.com/clsx524/claude-agent-sdk-go"
)

func TestIncludePartialMessagesStreamEvents(t *testing.T) {
	RequireClaudeCode(t)

	// Create options with partial messages enabled
	maxTurns := 2
	model := "claude-sonnet-4-5"
	options := &claude.ClaudeAgentOptions{
		IncludePartialMessages: true,
		Model:                  &model,
		MaxTurns:               &maxTurns,
		Env: map[string]string{
			"MAX_THINKING_TOKENS": "8000",
		},
	}

	client := claude.NewClaudeSDKClient(options)

	ctx := context.Background()
	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Send a simple prompt that will generate streaming response with thinking
	msgCh, errCh := client.Query(ctx, "Think of three jokes, then tell one")

	collectedMessages, err := CollectMessages(msgCh, errCh)
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}

	if len(collectedMessages) == 0 {
		t.Fatal("No messages received")
	}

	t.Logf("Received %d messages", len(collectedMessages))

	// Should have SystemMessage(init) at the start
	if !IsSystemMessage(collectedMessages[0]) {
		t.Errorf("First message should be SystemMessage, got: %T", collectedMessages[0])
	}
	if data, ok := GetSystemMessageData(collectedMessages[0]); ok {
		if subtype, ok := data["subtype"].(string); ok && subtype != "init" {
			t.Errorf("First system message should have subtype 'init', got: %s", subtype)
		}
	}

	// Count different message types
	streamEvents := 0
	messageStarts := 0
	contentBlockStarts := 0
	contentBlockDeltas := 0
	contentBlockStops := 0
	messageStops := 0
	hasThinkingBlock := false
	hasTextBlock := false

	for _, msg := range collectedMessages {
		if streamEvent, ok := msg.(*claude.StreamEvent); ok {
			streamEvents++
			eventType, _ := streamEvent.Event["type"].(string)
			switch eventType {
			case "message_start":
				messageStarts++
			case "content_block_start":
				contentBlockStarts++
			case "content_block_delta":
				contentBlockDeltas++
			case "content_block_stop":
				contentBlockStops++
			case "message_stop":
				messageStops++
			}
		}

		if assistantMsg, ok := msg.(*claude.AssistantMessage); ok {
			for _, block := range assistantMsg.Content {
				if _, ok := block.(claude.ThinkingBlock); ok {
					hasThinkingBlock = true
				}
				if _, ok := block.(claude.TextBlock); ok {
					hasTextBlock = true
				}
			}
		}
	}

	t.Logf("Stream events: %d (message_start: %d, content_block_start: %d, content_block_delta: %d, content_block_stop: %d, message_stop: %d)",
		streamEvents, messageStarts, contentBlockStarts, contentBlockDeltas, contentBlockStops, messageStops)

	if streamEvents == 0 {
		t.Error("No StreamEvent messages received")
	}

	if messageStarts == 0 {
		t.Error("No message_start StreamEvent")
	}
	if contentBlockStarts == 0 {
		t.Error("No content_block_start StreamEvent")
	}
	if contentBlockDeltas == 0 {
		t.Error("No content_block_delta StreamEvent")
	}
	if contentBlockStops == 0 {
		t.Error("No content_block_stop StreamEvent")
	}
	if messageStops == 0 {
		t.Error("No message_stop StreamEvent")
	}

	if !hasThinkingBlock {
		t.Log("Warning: No ThinkingBlock found in AssistantMessages (may vary)")
	}
	if !hasTextBlock {
		t.Error("No TextBlock found in AssistantMessages")
	}

	// Should end with ResultMessage
	lastMsg := collectedMessages[len(collectedMessages)-1]
	if !IsResultMessage(lastMsg) {
		t.Errorf("Last message should be ResultMessage, got: %T", lastMsg)
	}
	if resultMsg, ok := lastMsg.(*claude.ResultMessage); ok {
		if resultMsg.Subtype != "success" {
			t.Errorf("Result message should have subtype 'success', got: %s", resultMsg.Subtype)
		}
	}
}

func TestIncludePartialMessagesThinkingDeltas(t *testing.T) {
	RequireClaudeCode(t)

	// Create options with partial messages enabled
	maxTurns := 2
	model := "claude-sonnet-4-5"
	options := &claude.ClaudeAgentOptions{
		IncludePartialMessages: true,
		Model:                  &model,
		MaxTurns:               &maxTurns,
		Env: map[string]string{
			"MAX_THINKING_TOKENS": "8000",
		},
	}

	client := claude.NewClaudeSDKClient(options)

	ctx := context.Background()
	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	msgCh, errCh := client.Query(ctx, "Think step by step about what 2 + 2 equals")

	thinkingDeltas := []string{}

	for msg := range msgCh {
		if streamEvent, ok := msg.(*claude.StreamEvent); ok {
			if eventType, _ := streamEvent.Event["type"].(string); eventType == "content_block_delta" {
				if delta, ok := streamEvent.Event["delta"].(map[string]interface{}); ok {
					if deltaType, _ := delta["type"].(string); deltaType == "thinking_delta" {
						if thinking, ok := delta["thinking"].(string); ok {
							thinkingDeltas = append(thinkingDeltas, thinking)
						}
					}
				}
			}
		}
	}

	if err := <-errCh; err != nil {
		t.Fatalf("Query error: %v", err)
	}

	t.Logf("Received %d thinking deltas", len(thinkingDeltas))

	// Should have received multiple thinking deltas
	if len(thinkingDeltas) == 0 {
		t.Error("No thinking deltas received")
		return
	}

	// Combined thinking should form coherent text
	combinedThinking := strings.Join(thinkingDeltas, "")
	if len(combinedThinking) <= 10 {
		t.Errorf("Thinking content too short: %d chars", len(combinedThinking))
	}

	// Should contain some reasoning about the calculation
	if !strings.Contains(strings.ToLower(combinedThinking), "2") {
		t.Error("Thinking doesn't mention the numbers")
	}

	t.Logf("Combined thinking: %s", combinedThinking)
}

func TestPartialMessagesDisabledByDefault(t *testing.T) {
	RequireClaudeCode(t)

	// Create options WITHOUT partial messages (defaults to false)
	maxTurns := 2
	model := "claude-sonnet-4-5"
	options := &claude.ClaudeAgentOptions{
		Model:    &model,
		MaxTurns: &maxTurns,
		// IncludePartialMessages not set - defaults to false
	}

	client := claude.NewClaudeSDKClient(options)

	ctx := context.Background()
	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	msgCh, errCh := client.Query(ctx, "Say hello")

	collectedMessages, err := CollectMessages(msgCh, errCh)
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}

	// Should NOT have any StreamEvent messages
	streamEvents := 0
	hasSystem := false
	hasAssistant := false
	hasResult := false

	for _, msg := range collectedMessages {
		if IsStreamEvent(msg) {
			streamEvents++
		}
		if IsSystemMessage(msg) {
			hasSystem = true
		}
		if IsAssistantMessage(msg) {
			hasAssistant = true
		}
		if IsResultMessage(msg) {
			hasResult = true
		}
	}

	if streamEvents > 0 {
		t.Errorf("StreamEvent messages present when partial messages disabled: %d events", streamEvents)
	}

	// Should still have the regular messages
	if !hasSystem {
		t.Error("No SystemMessage received")
	}
	if !hasAssistant {
		t.Error("No AssistantMessage received")
	}
	if !hasResult {
		t.Error("No ResultMessage received")
	}

	t.Logf("Successfully verified partial messages disabled by default (%d messages, 0 stream events)", len(collectedMessages))
}
