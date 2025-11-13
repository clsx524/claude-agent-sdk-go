package integration

import (
	"context"
	"sync"

	claude "github.com/clsx524/claude-agent-sdk-go"
)

// MockTransport implements the Transport interface for testing
type MockTransport struct {
	ConnectFunc      func(ctx context.Context) error
	WriteFunc        func(ctx context.Context, data string) error
	ReadMessagesFunc func(ctx context.Context) (<-chan map[string]interface{}, <-chan error)
	EndInputFunc     func() error
	IsReadyFunc      func() bool
	CloseFunc        func() error

	Messages []map[string]interface{}
	mu       sync.Mutex
}

func NewMockTransport(messages []map[string]interface{}) *MockTransport {
	return &MockTransport{
		Messages: messages,
	}
}

func (m *MockTransport) Connect(ctx context.Context) error {
	if m.ConnectFunc != nil {
		return m.ConnectFunc(ctx)
	}
	return nil
}

func (m *MockTransport) Write(ctx context.Context, data string) error {
	if m.WriteFunc != nil {
		return m.WriteFunc(ctx, data)
	}
	return nil
}

func (m *MockTransport) ReadMessages(ctx context.Context) (<-chan map[string]interface{}, <-chan error) {
	if m.ReadMessagesFunc != nil {
		return m.ReadMessagesFunc(ctx)
	}

	msgCh := make(chan map[string]interface{}, len(m.Messages))
	errCh := make(chan error, 1)

	go func() {
		defer close(msgCh)
		defer close(errCh)

		for _, msg := range m.Messages {
			select {
			case <-ctx.Done():
				return
			case msgCh <- msg:
			}
		}
	}()

	return msgCh, errCh
}

func (m *MockTransport) EndInput() error {
	if m.EndInputFunc != nil {
		return m.EndInputFunc()
	}
	return nil
}

func (m *MockTransport) IsReady() bool {
	if m.IsReadyFunc != nil {
		return m.IsReadyFunc()
	}
	return true
}

func (m *MockTransport) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

// Helper functions for creating test messages

func CreateAssistantTextMessage(text string) map[string]interface{} {
	return map[string]interface{}{
		"type": "assistant",
		"message": map[string]interface{}{
			"role": "assistant",
			"content": []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": text,
				},
			},
			"model": "claude-sonnet-4-5",
		},
	}
}

func CreateAssistantToolUseMessage(text string, toolID string, toolName string, toolInput map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"type": "assistant",
		"message": map[string]interface{}{
			"role": "assistant",
			"content": []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": text,
				},
				map[string]interface{}{
					"type":  "tool_use",
					"id":    toolID,
					"name":  toolName,
					"input": toolInput,
				},
			},
			"model": "claude-sonnet-4-5",
		},
	}
}

func CreateResultMessage(sessionID string, costUSD float64, durationMS int) map[string]interface{} {
	return map[string]interface{}{
		"type":            "result",
		"subtype":         "success",
		"duration_ms":     float64(durationMS),
		"duration_api_ms": float64(durationMS - 200),
		"is_error":        false,
		"num_turns":       float64(1),
		"session_id":      sessionID,
		"total_cost_usd":  costUSD,
	}
}

func CreateResultMessageWithSubtype(sessionID string, subtype string, costUSD float64, durationMS int) map[string]interface{} {
	msg := CreateResultMessage(sessionID, costUSD, durationMS)
	msg["subtype"] = subtype
	return msg
}

// CollectMessages is a helper to collect all messages from a query
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
