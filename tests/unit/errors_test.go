package unit

import (
	"errors"
	"strings"
	"testing"

	claude "github.com/clsx524/claude-agent-sdk-go"
)

func TestCLINotFoundError(t *testing.T) {
	err := claude.NewCLINotFoundError("Claude Code not found", "/usr/bin/claude")

	if err.CLIPath != "/usr/bin/claude" {
		t.Errorf("expected CLIPath '/usr/bin/claude', got %s", err.CLIPath)
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "Claude Code not found") {
		t.Errorf("error message should contain 'Claude Code not found', got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "/usr/bin/claude") {
		t.Errorf("error message should contain path, got: %s", errMsg)
	}
}

func TestCLIConnectionError(t *testing.T) {
	wrappedErr := errors.New("connection refused")
	err := claude.NewCLIConnectionError("failed to connect", wrappedErr)

	if !strings.Contains(err.Error(), "failed to connect") {
		t.Errorf("error message should contain 'failed to connect', got: %s", err.Error())
	}

	if !errors.Is(err, wrappedErr) {
		t.Error("CLIConnectionError should wrap the original error")
	}
}

func TestProcessError(t *testing.T) {
	err := claude.NewProcessError("command failed", 1, "stderr output")

	if err.ExitCode != 1 {
		t.Errorf("expected ExitCode 1, got %d", err.ExitCode)
	}
	if err.Stderr != "stderr output" {
		t.Errorf("expected Stderr 'stderr output', got %s", err.Stderr)
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "exit code: 1") {
		t.Errorf("error message should contain exit code, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "stderr output") {
		t.Errorf("error message should contain stderr, got: %s", errMsg)
	}
}

func TestCLIJSONDecodeError(t *testing.T) {
	originalErr := errors.New("invalid JSON")
	longLine := strings.Repeat("a", 200)
	err := claude.NewCLIJSONDecodeError(longLine, originalErr)

	if err.Line != longLine {
		t.Error("Line field should store full line")
	}
	if err.OriginalError != originalErr {
		t.Error("OriginalError should be stored")
	}

	errMsg := err.Error()
	// Should truncate long lines in message
	if len(errMsg) > 150 && !strings.Contains(errMsg, "...") {
		t.Error("Long lines should be truncated in error message")
	}
}

func TestMessageParseError(t *testing.T) {
	data := map[string]interface{}{
		"type": "unknown",
		"foo":  "bar",
	}
	err := claude.NewMessageParseError("unknown message type", data)

	if err.Data == nil {
		t.Error("Data should be stored")
	}
	if err.Data["type"] != "unknown" {
		t.Errorf("expected Data[type] = 'unknown', got %v", err.Data["type"])
	}
}

func TestErrorWrapping(t *testing.T) {
	innerErr := errors.New("inner error")
	outerErr := claude.NewCLIConnectionError("outer error", innerErr)

	if !errors.Is(outerErr, innerErr) {
		t.Error("errors.Is should find wrapped error")
	}

	unwrapped := errors.Unwrap(outerErr)
	if unwrapped == nil {
		t.Error("errors.Unwrap should return wrapped error")
	}
}
