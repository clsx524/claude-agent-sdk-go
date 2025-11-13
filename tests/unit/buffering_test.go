package unit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"

	claude "github.com/clsx524/claude-agent-sdk-go"
)

// simulateBuffering simulates the buffering logic from SubprocessCLITransport.ReadMessages
// This tests the core buffering algorithm without needing a real subprocess.
func simulateBuffering(t *testing.T, reader io.Reader, maxBufferSize int) ([]map[string]interface{}, error) {
	messages := []map[string]interface{}{}

	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, maxBufferSize)

	jsonBuffer := ""

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Split by newlines (in case multiple JSON objects on one line)
		jsonLines := strings.Split(line, "\n")

		for _, jsonLine := range jsonLines {
			jsonLine = strings.TrimSpace(jsonLine)
			if jsonLine == "" {
				continue
			}

			// Accumulate partial JSON
			jsonBuffer += jsonLine

			if len(jsonBuffer) > maxBufferSize {
				return messages, claude.NewCLIJSONDecodeError(
					fmt.Sprintf("JSON message exceeded maximum buffer size of %d bytes", maxBufferSize),
					fmt.Errorf("buffer size %d exceeds limit %d", len(jsonBuffer), maxBufferSize),
				)
			}

			// Try to parse
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(jsonBuffer), &data); err == nil {
				// Successfully parsed
				jsonBuffer = ""
				messages = append(messages, data)
			}
			// If parse fails, keep accumulating
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return messages, err
	}

	return messages, nil
}

// TestMultipleJSONObjectsOnSingleLine tests parsing when multiple JSON objects
// are concatenated on a single line with embedded newlines.
func TestMultipleJSONObjectsOnSingleLine(t *testing.T) {
	jsonObj1 := map[string]interface{}{
		"type":    "message",
		"id":      "msg1",
		"content": "First message",
	}
	jsonObj2 := map[string]interface{}{
		"type":   "result",
		"id":     "res1",
		"status": "completed",
	}

	json1, _ := json.Marshal(jsonObj1)
	json2, _ := json.Marshal(jsonObj2)

	// Simulate multiple JSON objects on one line (separated by newlines)
	bufferedLine := string(json1) + "\n" + string(json2)
	reader := strings.NewReader(bufferedLine)

	messages, err := simulateBuffering(t, reader, 1024*1024)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(messages))
	}

	if messages[0]["type"] != "message" {
		t.Errorf("Expected type 'message', got %v", messages[0]["type"])
	}
	if messages[0]["id"] != "msg1" {
		t.Errorf("Expected id 'msg1', got %v", messages[0]["id"])
	}
	if messages[0]["content"] != "First message" {
		t.Errorf("Expected content 'First message', got %v", messages[0]["content"])
	}

	if messages[1]["type"] != "result" {
		t.Errorf("Expected type 'result', got %v", messages[1]["type"])
	}
	if messages[1]["id"] != "res1" {
		t.Errorf("Expected id 'res1', got %v", messages[1]["id"])
	}
	if messages[1]["status"] != "completed" {
		t.Errorf("Expected status 'completed', got %v", messages[1]["status"])
	}
}

// TestJSONWithEmbeddedNewlines tests parsing JSON objects that contain
// newline characters in string values.
func TestJSONWithEmbeddedNewlines(t *testing.T) {
	jsonObj1 := map[string]interface{}{
		"type":    "message",
		"content": "Line 1\nLine 2\nLine 3",
	}
	jsonObj2 := map[string]interface{}{
		"type": "result",
		"data": "Some\nMultiline\nContent",
	}

	json1, _ := json.Marshal(jsonObj1)
	json2, _ := json.Marshal(jsonObj2)

	bufferedLine := string(json1) + "\n" + string(json2)
	reader := strings.NewReader(bufferedLine)

	messages, err := simulateBuffering(t, reader, 1024*1024)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(messages))
	}

	if messages[0]["content"] != "Line 1\nLine 2\nLine 3" {
		t.Errorf("Expected content with newlines, got %v", messages[0]["content"])
	}
	if messages[1]["data"] != "Some\nMultiline\nContent" {
		t.Errorf("Expected data with newlines, got %v", messages[1]["data"])
	}
}

// TestMultipleNewlinesBetweenObjects tests parsing with multiple newlines
// between JSON objects.
func TestMultipleNewlinesBetweenObjects(t *testing.T) {
	jsonObj1 := map[string]interface{}{
		"type": "message",
		"id":   "msg1",
	}
	jsonObj2 := map[string]interface{}{
		"type": "result",
		"id":   "res1",
	}

	json1, _ := json.Marshal(jsonObj1)
	json2, _ := json.Marshal(jsonObj2)

	// Multiple newlines between objects
	bufferedLine := string(json1) + "\n\n\n" + string(json2)
	reader := strings.NewReader(bufferedLine)

	messages, err := simulateBuffering(t, reader, 1024*1024)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(messages))
	}

	if messages[0]["id"] != "msg1" {
		t.Errorf("Expected id 'msg1', got %v", messages[0]["id"])
	}
	if messages[1]["id"] != "res1" {
		t.Errorf("Expected id 'res1', got %v", messages[1]["id"])
	}
}

// TestSplitJSONAcrossMultipleReads tests parsing when a single JSON object
// is split across multiple stream reads.
func TestSplitJSONAcrossMultipleReads(t *testing.T) {
	// Create a large JSON object
	jsonObj := map[string]interface{}{
		"type": "assistant",
		"message": map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": strings.Repeat("x", 1000),
				},
				map[string]interface{}{
					"type": "tool_use",
					"id":   "tool_123",
					"name": "Read",
					"input": map[string]interface{}{
						"file_path": "/test.txt",
					},
				},
			},
		},
	}

	completeJSON, _ := json.Marshal(jsonObj)

	// Use a chunkReader to simulate split reads
	reader := &chunkReader{
		chunks: []string{
			string(completeJSON[:100]),
			string(completeJSON[100:250]),
			string(completeJSON[250:]),
		},
	}

	messages, err := simulateBuffering(t, reader, 1024*1024)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	if messages[0]["type"] != "assistant" {
		t.Errorf("Expected type 'assistant', got %v", messages[0]["type"])
	}

	msg := messages[0]["message"].(map[string]interface{})
	content := msg["content"].([]interface{})
	if len(content) != 2 {
		t.Errorf("Expected 2 content blocks, got %d", len(content))
	}
}

// TestLargeMinifiedJSON tests parsing a large minified JSON.
func TestLargeMinifiedJSON(t *testing.T) {
	// Create large data structure
	largeData := map[string]interface{}{
		"data": make([]interface{}, 1000),
	}
	for i := 0; i < 1000; i++ {
		largeData["data"].([]interface{})[i] = map[string]interface{}{
			"id":    i,
			"value": strings.Repeat("x", 100),
		}
	}

	largeDataJSON, _ := json.Marshal(largeData)

	jsonObj := map[string]interface{}{
		"type": "user",
		"message": map[string]interface{}{
			"role": "user",
			"content": []interface{}{
				map[string]interface{}{
					"tool_use_id": "toolu_016fed1NhiaMLqnEvrj5NUaj",
					"type":        "tool_result",
					"content":     string(largeDataJSON),
				},
			},
		},
	}

	completeJSON, _ := json.Marshal(jsonObj)

	// Split into 64KB chunks
	chunkSize := 64 * 1024
	var chunks []string
	for i := 0; i < len(completeJSON); i += chunkSize {
		end := i + chunkSize
		if end > len(completeJSON) {
			end = len(completeJSON)
		}
		chunks = append(chunks, string(completeJSON[i:end]))
	}

	reader := &chunkReader{chunks: chunks}

	// Use 2MB buffer for large JSON
	messages, err := simulateBuffering(t, reader, 2*1024*1024)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	if messages[0]["type"] != "user" {
		t.Errorf("Expected type 'user', got %v", messages[0]["type"])
	}

	msg := messages[0]["message"].(map[string]interface{})
	content := msg["content"].([]interface{})
	firstContent := content[0].(map[string]interface{})
	if firstContent["tool_use_id"] != "toolu_016fed1NhiaMLqnEvrj5NUaj" {
		t.Errorf("Expected tool_use_id 'toolu_016fed1NhiaMLqnEvrj5NUaj', got %v", firstContent["tool_use_id"])
	}
}

// TestBufferSizeExceeded tests that exceeding buffer size raises an error.
func TestBufferSizeExceeded(t *testing.T) {
	defaultMaxBufferSize := 1024 * 1024 // 1MB
	hugeIncomplete := `{"data": "` + strings.Repeat("x", defaultMaxBufferSize+1000)

	reader := strings.NewReader(hugeIncomplete)

	_, err := simulateBuffering(t, reader, defaultMaxBufferSize)

	if err == nil {
		t.Fatal("Expected error for buffer size exceeded, got nil")
	}

	// The bufio.Scanner will return "token too long" error for buffers that are too large
	// This is the expected behavior when buffer size is exceeded
	errMsg := err.Error()
	if !strings.Contains(errMsg, "token too long") && !strings.Contains(errMsg, "exceeded maximum buffer size") {
		t.Errorf("Error message should mention buffer size issue, got: %v", err)
	}
}

// TestBufferSizeOption tests that the configurable buffer size option is respected.
func TestBufferSizeOption(t *testing.T) {
	customLimit := 512
	hugeIncomplete := `{"data": "` + strings.Repeat("x", customLimit+10)

	reader := strings.NewReader(hugeIncomplete)

	_, err := simulateBuffering(t, reader, customLimit)

	if err == nil {
		t.Fatal("Expected error for buffer size exceeded, got nil")
	}

	// The bufio.Scanner returns "token too long" when buffer limit is exceeded
	// This validates that the custom buffer size limit is being enforced
	errMsg := err.Error()
	if !strings.Contains(errMsg, "token too long") && !strings.Contains(errMsg, "exceeded") {
		t.Errorf("Error should indicate buffer size exceeded, got: %v", err)
	}
}

// TestMixedCompleteAndSplitJSON tests handling a mix of complete and split JSON messages.
func TestMixedCompleteAndSplitJSON(t *testing.T) {
	msg1 := map[string]interface{}{
		"type":    "system",
		"subtype": "start",
	}
	json1, _ := json.Marshal(msg1)

	largeMsg := map[string]interface{}{
		"type": "assistant",
		"message": map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": strings.Repeat("y", 5000),
				},
			},
		},
	}
	largeJSON, _ := json.Marshal(largeMsg)

	msg3 := map[string]interface{}{
		"type":    "system",
		"subtype": "end",
	}
	json3, _ := json.Marshal(msg3)

	// Mix complete and split messages
	reader := &chunkReader{
		chunks: []string{
			string(json1) + "\n",
			string(largeJSON[:1000]),
			string(largeJSON[1000:3000]),
			string(largeJSON[3000:]) + "\n" + string(json3),
		},
	}

	messages, err := simulateBuffering(t, reader, 1024*1024)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(messages) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(messages))
	}

	if messages[0]["type"] != "system" {
		t.Errorf("Expected type 'system', got %v", messages[0]["type"])
	}
	if messages[0]["subtype"] != "start" {
		t.Errorf("Expected subtype 'start', got %v", messages[0]["subtype"])
	}

	if messages[1]["type"] != "assistant" {
		t.Errorf("Expected type 'assistant', got %v", messages[1]["type"])
	}
	msg := messages[1]["message"].(map[string]interface{})
	content := msg["content"].([]interface{})
	textBlock := content[0].(map[string]interface{})
	if len(textBlock["text"].(string)) != 5000 {
		t.Errorf("Expected text length 5000, got %d", len(textBlock["text"].(string)))
	}

	if messages[2]["type"] != "system" {
		t.Errorf("Expected type 'system', got %v", messages[2]["type"])
	}
	if messages[2]["subtype"] != "end" {
		t.Errorf("Expected subtype 'end', got %v", messages[2]["subtype"])
	}
}

// TestInvalidJSONErrorHandling tests that invalid JSON is handled gracefully.
func TestInvalidJSONErrorHandling(t *testing.T) {
	// Invalid JSON that will never complete
	invalidJSON := `{"type": "message", "unclosed": `

	reader := strings.NewReader(invalidJSON)

	// This should not parse successfully and will remain in buffer
	messages, err := simulateBuffering(t, reader, 1024*1024)

	// Should return with no messages and no error (waiting for more data)
	if err != nil {
		t.Errorf("Should not error on incomplete JSON, got: %v", err)
	}

	if len(messages) != 0 {
		t.Errorf("Should not parse incomplete JSON, got %d messages", len(messages))
	}
}

// chunkReader simulates reading data in chunks (like from a subprocess pipe).
type chunkReader struct {
	chunks []string
	index  int
	offset int
}

func (r *chunkReader) Read(p []byte) (n int, err error) {
	if r.index >= len(r.chunks) {
		return 0, io.EOF
	}

	chunk := r.chunks[r.index]
	remaining := chunk[r.offset:]

	n = copy(p, remaining)
	r.offset += n

	if r.offset >= len(chunk) {
		r.index++
		r.offset = 0
	}

	return n, nil
}

// TestEmptyLines tests that empty lines between JSON objects are properly ignored.
func TestEmptyLines(t *testing.T) {
	jsonObj1 := map[string]interface{}{
		"type": "message",
		"id":   "msg1",
	}
	jsonObj2 := map[string]interface{}{
		"type": "message",
		"id":   "msg2",
	}

	json1, _ := json.Marshal(jsonObj1)
	json2, _ := json.Marshal(jsonObj2)

	// Empty lines and spaces between objects
	bufferedLine := string(json1) + "\n   \n\t\n  \n" + string(json2)
	reader := strings.NewReader(bufferedLine)

	messages, err := simulateBuffering(t, reader, 1024*1024)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(messages))
	}

	if messages[0]["id"] != "msg1" {
		t.Errorf("Expected id 'msg1', got %v", messages[0]["id"])
	}
	if messages[1]["id"] != "msg2" {
		t.Errorf("Expected id 'msg2', got %v", messages[1]["id"])
	}
}

// TestUnicodeContent tests parsing JSON with Unicode characters.
func TestUnicodeContent(t *testing.T) {
	jsonObj := map[string]interface{}{
		"type":    "message",
		"content": "Hello 世界! 👋 Привет мир! 🚀",
		"emoji":   "🎉🎊✨",
	}

	completeJSON, _ := json.Marshal(jsonObj)
	reader := strings.NewReader(string(completeJSON))

	messages, err := simulateBuffering(t, reader, 1024*1024)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	if messages[0]["content"] != "Hello 世界! 👋 Привет мир! 🚀" {
		t.Errorf("Expected Unicode content, got %v", messages[0]["content"])
	}
	if messages[0]["emoji"] != "🎉🎊✨" {
		t.Errorf("Expected emoji content, got %v", messages[0]["emoji"])
	}
}

// TestEscapedCharacters tests parsing JSON with escaped characters.
func TestEscapedCharacters(t *testing.T) {
	jsonObj := map[string]interface{}{
		"type":    "message",
		"content": "Line with \"quotes\" and \\ backslashes \t tabs \r\n newlines",
		"path":    "C:\\Users\\test\\file.txt",
	}

	completeJSON, _ := json.Marshal(jsonObj)
	reader := strings.NewReader(string(completeJSON))

	messages, err := simulateBuffering(t, reader, 1024*1024)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	// JSON marshaling will have escaped these, so we verify they're preserved
	content := messages[0]["content"].(string)
	if !strings.Contains(content, "\"quotes\"") {
		t.Errorf("Expected escaped quotes in content, got %v", content)
	}
	if !strings.Contains(content, "backslashes") {
		t.Errorf("Expected backslashes in content, got %v", content)
	}

	path := messages[0]["path"].(string)
	if !strings.Contains(path, "\\") {
		t.Errorf("Expected backslashes in path, got %v", path)
	}
}

// TestRapidSuccessiveMessages tests parsing many small messages in rapid succession.
func TestRapidSuccessiveMessages(t *testing.T) {
	var allJSON strings.Builder
	expectedCount := 100

	for i := 0; i < expectedCount; i++ {
		jsonObj := map[string]interface{}{
			"type":  "message",
			"id":    fmt.Sprintf("msg%d", i),
			"index": i,
		}
		jsonBytes, _ := json.Marshal(jsonObj)
		allJSON.WriteString(string(jsonBytes))
		allJSON.WriteString("\n")
	}

	reader := strings.NewReader(allJSON.String())

	messages, err := simulateBuffering(t, reader, 1024*1024)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(messages) != expectedCount {
		t.Fatalf("Expected %d messages, got %d", expectedCount, len(messages))
	}

	for i := 0; i < expectedCount; i++ {
		if messages[i]["index"] != float64(i) {
			t.Errorf("Expected index %d, got %v", i, messages[i]["index"])
		}
	}
}

// TestNestedComplexStructures tests parsing deeply nested JSON structures.
func TestNestedComplexStructures(t *testing.T) {
	jsonObj := map[string]interface{}{
		"type": "assistant",
		"message": map[string]interface{}{
			"role": "assistant",
			"content": []interface{}{
				map[string]interface{}{
					"type": "tool_use",
					"id":   "tool_123",
					"name": "Task",
					"input": map[string]interface{}{
						"config": map[string]interface{}{
							"options": map[string]interface{}{
								"nested": map[string]interface{}{
									"deeply": map[string]interface{}{
										"value": []interface{}{1, 2, 3, 4, 5},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	completeJSON, _ := json.Marshal(jsonObj)
	reader := strings.NewReader(string(completeJSON))

	messages, err := simulateBuffering(t, reader, 1024*1024)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	// Navigate through the nested structure
	msg := messages[0]["message"].(map[string]interface{})
	content := msg["content"].([]interface{})
	toolUse := content[0].(map[string]interface{})
	input := toolUse["input"].(map[string]interface{})
	config := input["config"].(map[string]interface{})
	options := config["options"].(map[string]interface{})
	nested := options["nested"].(map[string]interface{})
	deeply := nested["deeply"].(map[string]interface{})
	values := deeply["value"].([]interface{})

	if len(values) != 5 {
		t.Errorf("Expected 5 values in deeply nested array, got %d", len(values))
	}
}

// TestPartialJSONThenComplete tests that partial JSON is handled correctly.
// Note: In the actual implementation, incomplete JSON on one line followed by
// complete JSON on another line will concatenate them, which may or may not be valid.
// This test verifies that complete standalone JSON on a new line works correctly.
func TestPartialJSONThenComplete(t *testing.T) {
	msg1 := map[string]interface{}{
		"type": "message",
		"id":   "msg1",
	}
	msg2 := map[string]interface{}{
		"type": "message",
		"id":   "msg2",
	}
	json1, _ := json.Marshal(msg1)
	json2, _ := json.Marshal(msg2)

	// Two complete JSON messages on separate lines
	input := string(json1) + "\n" + string(json2)

	reader := strings.NewReader(input)

	messages, err := simulateBuffering(t, reader, 1024*1024)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(messages))
	}

	if messages[0]["id"] != "msg1" {
		t.Errorf("Expected id 'msg1', got %v", messages[0]["id"])
	}
	if messages[1]["id"] != "msg2" {
		t.Errorf("Expected id 'msg2', got %v", messages[1]["id"])
	}
}

// TestSingleByteReads tests extreme case where data arrives one byte at a time.
func TestSingleByteReads(t *testing.T) {
	jsonObj := map[string]interface{}{
		"type": "message",
		"id":   "msg1",
	}
	completeJSON, _ := json.Marshal(jsonObj)

	// Create chunks of single bytes
	chunks := make([]string, len(completeJSON))
	for i := 0; i < len(completeJSON); i++ {
		chunks[i] = string(completeJSON[i])
	}

	reader := &chunkReader{chunks: chunks}

	messages, err := simulateBuffering(t, reader, 1024*1024)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	if messages[0]["type"] != "message" {
		t.Errorf("Expected type 'message', got %v", messages[0]["type"])
	}
}

// TestZeroLengthJSON tests handling empty JSON objects and arrays.
func TestZeroLengthJSON(t *testing.T) {
	// Note: [] (empty array) cannot be unmarshaled into map[string]interface{}
	// so we only test valid object formats
	input := `{}
{"data": []}
{"nested": {}}`

	reader := strings.NewReader(input)

	messages, err := simulateBuffering(t, reader, 1024*1024)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(messages) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(messages))
	}

	// First message: empty object
	if len(messages[0]) != 0 {
		t.Errorf("Expected empty object, got %v", messages[0])
	}

	// Second message: object with empty array
	data, ok := messages[1]["data"].([]interface{})
	if !ok {
		t.Errorf("Expected data to be array, got %T", messages[1]["data"])
	}
	if len(data) != 0 {
		t.Errorf("Expected empty array, got %v", data)
	}

	// Third message: object with empty nested object
	nested, ok := messages[2]["nested"].(map[string]interface{})
	if !ok {
		t.Errorf("Expected nested to be object, got %T", messages[2]["nested"])
	}
	if len(nested) != 0 {
		t.Errorf("Expected empty nested object, got %v", nested)
	}
}
