package claude

import "fmt"

// ClaudeSDKError is the base error type for all Claude SDK errors.
type ClaudeSDKError struct {
	Message string
	Err     error
}

func (e *ClaudeSDKError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *ClaudeSDKError) Unwrap() error {
	return e.Err
}

// CLINotFoundError is returned when Claude Code CLI is not found or not installed.
type CLINotFoundError struct {
	*ClaudeSDKError
	CLIPath string
}

// NewCLINotFoundError creates a new CLINotFoundError.
func NewCLINotFoundError(message string, cliPath string) *CLINotFoundError {
	if cliPath != "" {
		message = fmt.Sprintf("%s: %s", message, cliPath)
	}
	return &CLINotFoundError{
		ClaudeSDKError: &ClaudeSDKError{Message: message},
		CLIPath:        cliPath,
	}
}

// CLIConnectionError is returned when unable to connect to Claude Code.
type CLIConnectionError struct {
	*ClaudeSDKError
}

// NewCLIConnectionError creates a new CLIConnectionError.
func NewCLIConnectionError(message string, err error) *CLIConnectionError {
	return &CLIConnectionError{
		ClaudeSDKError: &ClaudeSDKError{Message: message, Err: err},
	}
}

// ProcessError is returned when the CLI process fails.
type ProcessError struct {
	*ClaudeSDKError
	ExitCode int
	Stderr   string
}

// NewProcessError creates a new ProcessError.
func NewProcessError(message string, exitCode int, stderr string) *ProcessError {
	fullMessage := message
	if exitCode != 0 {
		fullMessage = fmt.Sprintf("%s (exit code: %d)", fullMessage, exitCode)
	}
	if stderr != "" {
		fullMessage = fmt.Sprintf("%s\nError output: %s", fullMessage, stderr)
	}
	return &ProcessError{
		ClaudeSDKError: &ClaudeSDKError{Message: fullMessage},
		ExitCode:       exitCode,
		Stderr:         stderr,
	}
}

// CLIJSONDecodeError is returned when unable to decode JSON from CLI output.
type CLIJSONDecodeError struct {
	*ClaudeSDKError
	Line          string
	OriginalError error
}

// NewCLIJSONDecodeError creates a new CLIJSONDecodeError.
func NewCLIJSONDecodeError(line string, err error) *CLIJSONDecodeError {
	truncated := line
	if len(line) > 100 {
		truncated = line[:100] + "..."
	}
	return &CLIJSONDecodeError{
		ClaudeSDKError: &ClaudeSDKError{
			Message: fmt.Sprintf("Failed to decode JSON: %s", truncated),
			Err:     err,
		},
		Line:          line,
		OriginalError: err,
	}
}

// MessageParseError is returned when unable to parse a message from CLI output.
type MessageParseError struct {
	*ClaudeSDKError
	Data map[string]interface{}
}

// NewMessageParseError creates a new MessageParseError.
func NewMessageParseError(message string, data map[string]interface{}) *MessageParseError {
	return &MessageParseError{
		ClaudeSDKError: &ClaudeSDKError{Message: message},
		Data:           data,
	}
}
