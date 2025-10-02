package claude

import (
	"context"
	"encoding/json"
)

// PermissionMode defines the permission handling mode.
type PermissionMode string

const (
	PermissionModeDefault           PermissionMode = "default"
	PermissionModeAcceptEdits       PermissionMode = "acceptEdits"
	PermissionModePlan              PermissionMode = "plan"
	PermissionModeBypassPermissions PermissionMode = "bypassPermissions"
)

// SettingSource defines where settings are loaded from.
type SettingSource string

const (
	SettingSourceUser    SettingSource = "user"
	SettingSourceProject SettingSource = "project"
	SettingSourceLocal   SettingSource = "local"
)

// HookEvent defines supported hook event types.
type HookEvent string

const (
	HookEventPreToolUse       HookEvent = "PreToolUse"
	HookEventPostToolUse      HookEvent = "PostToolUse"
	HookEventUserPromptSubmit HookEvent = "UserPromptSubmit"
	HookEventStop             HookEvent = "Stop"
	HookEventSubagentStop     HookEvent = "SubagentStop"
	HookEventPreCompact       HookEvent = "PreCompact"
)

// Message interface for all message types.
type Message interface {
	isMessage()
}

// ContentBlock interface for all content block types.
type ContentBlock interface {
	isContentBlock()
}

// TextBlock represents text content.
type TextBlock struct {
	Text string `json:"text"`
}

func (TextBlock) isContentBlock() {}

// ThinkingBlock represents thinking content with signature.
type ThinkingBlock struct {
	Thinking  string `json:"thinking"`
	Signature string `json:"signature"`
}

func (ThinkingBlock) isContentBlock() {}

// ToolUseBlock represents a tool invocation.
type ToolUseBlock struct {
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

func (ToolUseBlock) isContentBlock() {}

// ToolResultBlock represents a tool execution result.
type ToolResultBlock struct {
	ToolUseID string      `json:"tool_use_id"`
	Content   interface{} `json:"content,omitempty"` // Can be string or []map[string]interface{}
	IsError   *bool       `json:"is_error,omitempty"`
}

func (ToolResultBlock) isContentBlock() {}

// UserMessage represents a user message.
type UserMessage struct {
	Content         interface{} `json:"content"` // Can be string or []ContentBlock
	ParentToolUseID *string     `json:"parent_tool_use_id,omitempty"`
}

func (UserMessage) isMessage() {}

// AssistantMessage represents an assistant message.
type AssistantMessage struct {
	Content         []ContentBlock `json:"content"`
	Model           string         `json:"model"`
	ParentToolUseID *string        `json:"parent_tool_use_id,omitempty"`
}

func (AssistantMessage) isMessage() {}

// SystemMessage represents a system message with metadata.
type SystemMessage struct {
	Subtype string                 `json:"subtype"`
	Data    map[string]interface{} `json:"data"`
}

func (SystemMessage) isMessage() {}

// ResultMessage represents the final result of a query with cost and usage information.
type ResultMessage struct {
	Subtype       string                 `json:"subtype"`
	DurationMS    int                    `json:"duration_ms"`
	DurationAPIMS int                    `json:"duration_api_ms"`
	IsError       bool                   `json:"is_error"`
	NumTurns      int                    `json:"num_turns"`
	SessionID     string                 `json:"session_id"`
	TotalCostUSD  *float64               `json:"total_cost_usd,omitempty"`
	Usage         map[string]interface{} `json:"usage,omitempty"`
	Result        *string                `json:"result,omitempty"`
}

func (ResultMessage) isMessage() {}

// StreamEvent represents a partial message update during streaming.
type StreamEvent struct {
	UUID            string                 `json:"uuid"`
	SessionID       string                 `json:"session_id"`
	Event           map[string]interface{} `json:"event"`
	ParentToolUseID *string                `json:"parent_tool_use_id,omitempty"`
}

func (StreamEvent) isMessage() {}

// SystemPromptPreset represents a system prompt preset configuration.
type SystemPromptPreset struct {
	Type   string  `json:"type"`
	Preset string  `json:"preset"`
	Append *string `json:"append,omitempty"`
}

// AgentDefinition represents a custom agent configuration.
type AgentDefinition struct {
	Description string   `json:"description"`
	Prompt      string   `json:"prompt"`
	Tools       []string `json:"tools,omitempty"`
	Model       *string  `json:"model,omitempty"` // "sonnet", "opus", "haiku", "inherit"
}

// PermissionRuleValue represents a permission rule.
type PermissionRuleValue struct {
	ToolName    string  `json:"tool_name"`
	RuleContent *string `json:"rule_content,omitempty"`
}

// PermissionBehavior defines permission behavior.
type PermissionBehavior string

const (
	PermissionBehaviorAllow PermissionBehavior = "allow"
	PermissionBehaviorDeny  PermissionBehavior = "deny"
	PermissionBehaviorAsk   PermissionBehavior = "ask"
)

// PermissionUpdateDestination defines where permission updates are stored.
type PermissionUpdateDestination string

const (
	PermissionUpdateDestinationUserSettings    PermissionUpdateDestination = "userSettings"
	PermissionUpdateDestinationProjectSettings PermissionUpdateDestination = "projectSettings"
	PermissionUpdateDestinationLocalSettings   PermissionUpdateDestination = "localSettings"
	PermissionUpdateDestinationSession         PermissionUpdateDestination = "session"
)

// PermissionUpdate represents a permission update configuration.
type PermissionUpdate struct {
	Type        string                       `json:"type"` // "addRules", "replaceRules", "removeRules", "setMode", "addDirectories", "removeDirectories"
	Rules       []PermissionRuleValue        `json:"rules,omitempty"`
	Behavior    *PermissionBehavior          `json:"behavior,omitempty"`
	Mode        *PermissionMode              `json:"mode,omitempty"`
	Directories []string                     `json:"directories,omitempty"`
	Destination *PermissionUpdateDestination `json:"destination,omitempty"`
}

// ToolPermissionContext provides context for tool permission callbacks.
type ToolPermissionContext struct {
	Suggestions []PermissionUpdate `json:"suggestions,omitempty"`
}

// PermissionResult is the interface for permission callback results.
type PermissionResult interface {
	isPermissionResult()
}

// PermissionResultAllow indicates the tool is allowed.
type PermissionResultAllow struct {
	Behavior           string                 `json:"behavior"` // Always "allow"
	UpdatedInput       map[string]interface{} `json:"updated_input,omitempty"`
	UpdatedPermissions []PermissionUpdate     `json:"updated_permissions,omitempty"`
}

func (PermissionResultAllow) isPermissionResult() {}

// PermissionResultDeny indicates the tool is denied.
type PermissionResultDeny struct {
	Behavior  string `json:"behavior"` // Always "deny"
	Message   string `json:"message,omitempty"`
	Interrupt bool   `json:"interrupt,omitempty"`
}

func (PermissionResultDeny) isPermissionResult() {}

// CanUseTool is the function type for tool permission callbacks.
type CanUseTool func(ctx context.Context, toolName string, input map[string]interface{}, permCtx ToolPermissionContext) (PermissionResult, error)

// HookJSONOutput represents the output from a hook callback.
type HookJSONOutput struct {
	Decision           *string                `json:"decision,omitempty"` // "block"
	SystemMessage      *string                `json:"systemMessage,omitempty"`
	HookSpecificOutput map[string]interface{} `json:"hookSpecificOutput,omitempty"`
}

// HookContext provides context information for hook callbacks.
type HookContext struct {
	// Future: abort signal support
}

// HookCallback is the function type for hook callbacks.
type HookCallback func(ctx context.Context, input map[string]interface{}, toolUseID *string, hookCtx HookContext) (HookJSONOutput, error)

// HookMatcher configures hook matching and callbacks.
type HookMatcher struct {
	Matcher string         // Tool name pattern or nil for all
	Hooks   []HookCallback // List of hook callbacks
}

// StderrCallback is called for each line of stderr output.
type StderrCallback func(line string)

// McpServerConfig represents MCP server configuration (various types).
type McpServerConfig interface {
	isMcpServerConfig()
}

// McpStdioServerConfig represents a stdio-based MCP server.
type McpStdioServerConfig struct {
	Type    string            `json:"type,omitempty"` // "stdio" or omitted
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

func (McpStdioServerConfig) isMcpServerConfig() {}

// McpSSEServerConfig represents a SSE-based MCP server.
type McpSSEServerConfig struct {
	Type    string            `json:"type"` // "sse"
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

func (McpSSEServerConfig) isMcpServerConfig() {}

// McpHTTPServerConfig represents an HTTP-based MCP server.
type McpHTTPServerConfig struct {
	Type    string            `json:"type"` // "http"
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

func (McpHTTPServerConfig) isMcpServerConfig() {}

// McpSdkServerConfig represents an SDK MCP server (in-process).
type McpSdkServerConfig struct {
	Type     string      `json:"type"` // "sdk"
	Name     string      `json:"name"`
	Instance interface{} `json:"-"` // MCP Server instance (not serialized)
}

func (McpSdkServerConfig) isMcpServerConfig() {}

// MarshalJSON allows JSON serialization (excluding Instance field).
func (c McpSdkServerConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type": c.Type,
		"name": c.Name,
	})
}

// ClaudeAgentOptions contains all configuration options for Claude SDK.
type ClaudeAgentOptions struct {
	// Tool restrictions
	AllowedTools    []string `json:"allowed_tools,omitempty"`
	DisallowedTools []string `json:"disallowed_tools,omitempty"`

	// System prompt configuration
	SystemPrompt interface{} `json:"system_prompt,omitempty"` // Can be string or SystemPromptPreset

	// MCP servers
	McpServers map[string]McpServerConfig `json:"mcp_servers,omitempty"`

	// Permission settings
	PermissionMode           *PermissionMode `json:"permission_mode,omitempty"`
	PermissionPromptToolName *string         `json:"permission_prompt_tool_name,omitempty"`

	// Conversation settings
	ContinueConversation bool    `json:"continue_conversation,omitempty"`
	Resume               *string `json:"resume,omitempty"`
	MaxTurns             *int    `json:"max_turns,omitempty"`
	ForkSession          bool    `json:"fork_session,omitempty"`

	// Model
	Model *string `json:"model,omitempty"`

	// Working directory and environment
	Cwd     *string           `json:"cwd,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	User    *string           `json:"user,omitempty"`
	AddDirs []string          `json:"add_dirs,omitempty"`

	// Settings
	Settings       *string         `json:"settings,omitempty"`
	SettingSources []SettingSource `json:"setting_sources,omitempty"`

	// Callbacks
	CanUseTool CanUseTool                  `json:"-"` // Function, not serialized
	Hooks      map[HookEvent][]HookMatcher `json:"-"` // Functions, not serialized
	Stderr     StderrCallback              `json:"-"` // Function, not serialized

	// Agents
	Agents map[string]AgentDefinition `json:"agents,omitempty"`

	// Advanced options
	IncludePartialMessages bool               `json:"include_partial_messages,omitempty"`
	MaxBufferSize          *int               `json:"max_buffer_size,omitempty"`
	ExtraArgs              map[string]*string `json:"extra_args,omitempty"` // nil value = flag without value
}
