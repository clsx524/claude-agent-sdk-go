# End-to-End Tests for Claude Agent SDK (Go)

This directory contains end-to-end tests that run against the actual Claude API to verify real-world functionality.

## Requirements

### API Key (REQUIRED)

These tests require a valid Anthropic API key. The tests will **skip** if `ANTHROPIC_API_KEY` is not set.

Set your API key before running tests:

```bash
export ANTHROPIC_API_KEY="your-api-key-here"
```

### Dependencies

Ensure you have Go 1.25+ installed and all dependencies:

```bash
go mod download
```

## Running the Tests

### Run all e2e tests:

```bash
cd tests/e2e
go test -v
```

### Run tests without API key (skips e2e tests):

```bash
go test -v -short
```

### Run a specific test:

```bash
go test -v -run TestAgentDefinition
```

### Run all tests including e2e:

```bash
ANTHROPIC_API_KEY=your-key go test -v ./tests/e2e/...
```

## Cost Considerations

⚠️ **Important**: These tests make actual API calls to Claude, which incur costs based on your Anthropic pricing plan.

- Each test typically uses 1-3 API calls
- Tests use simple prompts to minimize token usage
- The complete test suite should cost less than $0.10 to run
- Tests automatically skip if `ANTHROPIC_API_KEY` is not set

## Test Coverage

### Agents and Settings Tests (`agents_and_settings_test.go`)

Tests agent definition and settings configuration:

- **TestAgentDefinitionAvailable**: Verifies custom agents are loaded correctly
- **TestSettingSourcesDefault**: Tests default settings behavior (no settings loaded)
- **TestSettingSourcesUserOnly**: Validates user-only settings (excludes project)
- **TestSettingSourcesProjectIncluded**: Tests project + local settings loading

### Hooks Tests (`hooks_test.go`)

Tests hook system functionality:

- **TestHooksWithDecisionAndReason**: Validates permissionDecision and reason fields
- **TestHooksWithContinueAndStop**: Tests continue and stopReason control
- **TestHooksWithAdditionalContext**: Verifies hookSpecificOutput additionalContext

### Dynamic Control Tests (`dynamic_control_test.go`)

Tests dynamic control features:

- **TestSetPermissionMode**: Validates changing permission mode mid-conversation
- **TestSetModel**: Tests switching AI model during conversation
- **TestInterrupt**: Verifies interrupt functionality

### Partial Messages Tests (`partial_messages_test.go`)

Tests streaming partial message updates:

- **TestPartialMessagesEnabled**: Validates StreamEvent messages are received
- **TestPartialMessagesDisabled**: Ensures partial messages off by default
- **TestThinkingDeltas**: Tests thinking block streaming

### SDK MCP Tools Tests (`sdk_mcp_tools_test.go`)

Tests in-process MCP server functionality:

- **TestSdkMcpToolExecution**: Verifies SDK MCP tools execute correctly
- **TestSdkMcpPermissionEnforcement**: Tests allowed/disallowed tools work
- **TestSdkMcpMultipleTools**: Validates multiple tools in sequence
- **TestSdkMcpToolError**: Tests error handling in tool execution

### Tool Permissions Tests (`tool_permissions_test.go`)

Tests permission callback system:

- **TestToolPermissionCallback**: Validates CanUseTool callback is invoked

### Stderr Callback Tests (`stderr_callback_test.go`)

Tests stderr capture functionality:

- **TestStderrCallbackCapturesDebug**: Validates stderr callback with --debug-to-stderr
- **TestStderrCallbackWithoutDebug**: Tests stderr callback without debug mode

## CI/CD Integration

These tests can run automatically in CI/CD by:

1. Setting `ANTHROPIC_API_KEY` in your CI environment
2. Running: `go test -v ./tests/e2e/...`

Tests automatically skip if the API key is not available, making them safe for PR checks.

## Troubleshooting

### Tests are skipped

- This is normal if `ANTHROPIC_API_KEY` is not set
- Set the environment variable to run e2e tests
- Tests marked as skipped with: "Skipping test: ANTHROPIC_API_KEY not set"

### Tests timing out

- Check your API key is valid and has quota available
- Ensure network connectivity to api.anthropic.com
- Increase timeout if needed (tests have 30s timeout by default)

### Permission denied errors

- Verify the `AllowedTools` parameter includes the necessary tools
- Check that tool names match the expected format (e.g., `calc__add` for MCP tools)
- Ensure permission mode is set correctly

### "Failed to connect" errors

- Check that Claude Code CLI is installed (version 2.0.0+)
- Verify Claude Code is in your PATH
- Try running `claude-code --version` manually

## Adding New E2E Tests

When adding new e2e tests:

1. Add `RequireAPIKey(t)` at the start of your test function
2. Use context with reasonable timeout (30s recommended)
3. Keep prompts simple to minimize costs
4. Verify actual tool execution, not just mocked responses
5. Clean up resources in defer statements
6. Document any special setup requirements in this README

### Example Test Structure

```go
func TestMyNewFeature(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}
	RequireAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Your test code here
}
```

## Test Helpers

The `e2e_helpers.go` file provides useful utilities:

- `RequireAPIKey(t)` - Skip test if API key not available
- `CreateCalculatorMCP()` - Create a test calculator MCP server
- `HasToolUse()` / `FindToolUse()` - Check for tool usage in messages
- `CollectMessages()` - Collect all messages from a channel
- `GetSystemMessageData()` - Extract system message data
- Type checker functions (IsSystemMessage, IsAssistantMessage, etc.)
- Message extraction helpers (GetResultMessage, GetAssistantMessages, etc.)

## Cost Monitoring

To track API costs during testing:

1. Enable cost reporting in tests
2. Check ResultMessage.TotalCostUSD field
3. Sum costs across all tests
4. Most tests should cost < $0.01 each

Example:
```go
if resultMsg.TotalCostUSD != nil {
    t.Logf("Test cost: $%.4f", *resultMsg.TotalCostUSD)
}
```
