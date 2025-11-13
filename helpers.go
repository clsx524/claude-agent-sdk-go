package claude

import "fmt"

// hookMatcherInternal represents the internal format of hook matchers
type hookMatcherInternal struct {
	Matcher string
	Hooks   []HookCallback
}

// convertHooksToInternal converts public hooks to internal format used by queryHandler
func convertHooksToInternal(hooks map[HookEvent][]HookMatcher) map[string][]hookMatcherInternal {
	if hooks == nil {
		return nil
	}

	internalHooks := make(map[string][]hookMatcherInternal)
	for event, matchers := range hooks {
		if len(matchers) == 0 {
			continue
		}

		internal := make([]hookMatcherInternal, len(matchers))
		for i, m := range matchers {
			internal[i] = hookMatcherInternal{
				Matcher: m.Matcher,
				Hooks:   m.Hooks,
			}
		}
		internalHooks[string(event)] = internal
	}
	return internalHooks
}

// extractSdkMcpServers extracts SDK MCP servers from the McpServers map
func extractSdkMcpServers(servers map[string]McpServerConfig) map[string]interface{} {
	if servers == nil {
		return nil
	}

	sdkMcpServers := make(map[string]interface{})
	for name, config := range servers {
		if sdkConfig, ok := config.(McpSdkServerConfig); ok {
			sdkMcpServers[name] = sdkConfig.Instance
		}
	}

	if len(sdkMcpServers) == 0 {
		return nil
	}

	return sdkMcpServers
}

// validateAndConfigurePermissions validates permission settings and returns configured options.
// If CanUseTool is set, it validates compatibility and sets PermissionPromptToolName to "stdio".
// For streaming mode validation, pass isStreaming=true.
// Returns the configured options and an error if validation fails.
func validateAndConfigurePermissions(options *ClaudeAgentOptions, isStreaming bool) (*ClaudeAgentOptions, error) {
	if options == nil {
		return &ClaudeAgentOptions{}, nil
	}

	if options.CanUseTool != nil {
		// canUseTool requires streaming mode
		if !isStreaming {
			return nil, fmt.Errorf("can_use_tool callback requires streaming mode")
		}

		// canUseTool and permission_prompt_tool_name are mutually exclusive
		if options.PermissionPromptToolName != nil {
			return nil, fmt.Errorf("can_use_tool callback cannot be used with permission_prompt_tool_name")
		}

		// Set permission_prompt_tool_name to "stdio" for control protocol
		stdio := "stdio"
		newOpts := *options
		newOpts.PermissionPromptToolName = &stdio
		return &newOpts, nil
	}

	return options, nil
}
