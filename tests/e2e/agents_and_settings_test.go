package e2e

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	claude "github.com/clsx524/claude-agent-sdk-go"
)

func TestAgentDefinition(t *testing.T) {
	RequireClaudeCode(t)

	// Create options with custom agent definition
	maxTurns := 1
	model := "sonnet"
	options := &claude.ClaudeAgentOptions{
		MaxTurns: &maxTurns,
		Agents: map[string]claude.AgentDefinition{
			"test-agent": {
				Description: "A test agent for verification",
				Prompt:      "You are a test agent. Always respond with 'Test agent activated'",
				Tools:       []string{"Read"},
				Model:       &model,
			},
		},
	}

	// Create client with custom agent
	client := claude.NewClaudeSDKClient(options)

	ctx := context.Background()
	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Start query
	msgCh, errCh := client.Query(ctx, "What is 2 + 2?")

	// Check for init message with agent definition
	foundAgent := false
	for msg := range msgCh {
		if data, ok := GetSystemMessageData(msg); ok {
			if subtype, ok := data["subtype"].(string); ok && subtype == "init" {
				if agents, ok := data["agents"].([]interface{}); ok {
					for _, agent := range agents {
						if agentStr, ok := agent.(string); ok && agentStr == "test-agent" {
							foundAgent = true
							t.Logf("Found test-agent in init message")
							break
						}
					}
				}
			}
		}
	}

	if err := <-errCh; err != nil {
		t.Fatalf("Query error: %v", err)
	}

	if !foundAgent {
		t.Fatal("Did not receive init message with agent definition")
	}
}

func TestSettingSourcesDefault(t *testing.T) {
	RequireClaudeCode(t)

	// Create temporary project directory with local settings
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	err := os.MkdirAll(claudeDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create .claude dir: %v", err)
	}

	// Create local settings with custom outputStyle
	settingsFile := filepath.Join(claudeDir, "settings.local.json")
	settingsContent := `{"outputStyle": "local-test-style"}`
	err = os.WriteFile(settingsFile, []byte(settingsContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write settings file: %v", err)
	}

	// Don't provide setting_sources - should default to no settings
	maxTurns := 1
	options := &claude.ClaudeAgentOptions{
		Cwd:      &tmpDir,
		MaxTurns: &maxTurns,
		// No SettingSources - should default to no settings
	}

	client := claude.NewClaudeSDKClient(options)

	ctx := context.Background()
	err = client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Start query
	msgCh, errCh := client.Query(ctx, "What is 2 + 2?")

	// Check that settings were NOT loaded (should be default)
	foundInit := false
	for msg := range msgCh {
		if data, ok := GetSystemMessageData(msg); ok {
			if subtype, ok := data["subtype"].(string); ok && subtype == "init" {
				foundInit = true
				outputStyle, ok := data["output_style"].(string)
				if !ok {
					t.Log("output_style not found in init message, treating as default")
					break
				}
				if outputStyle == "local-test-style" {
					t.Errorf("outputStyle should NOT be from local settings (default is no settings), got: %s", outputStyle)
				}
				if outputStyle != "default" {
					t.Logf("outputStyle is %s (expected 'default', but may vary)", outputStyle)
				} else {
					t.Logf("outputStyle correctly set to 'default'")
				}
				break
			}
		}
	}

	if err := <-errCh; err != nil {
		t.Fatalf("Query error: %v", err)
	}

	if !foundInit {
		t.Fatal("Did not receive init message")
	}
}

func TestSettingSourcesUserOnly(t *testing.T) {
	RequireClaudeCode(t)

	// Create temporary project directory with a slash command
	tmpDir := t.TempDir()
	commandsDir := filepath.Join(tmpDir, ".claude", "commands")
	err := os.MkdirAll(commandsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create commands dir: %v", err)
	}

	// Create test command
	testCommand := filepath.Join(commandsDir, "testcmd.md")
	commandContent := `---
description: Test command
---

This is a test command.
`
	err = os.WriteFile(testCommand, []byte(commandContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write command file: %v", err)
	}

	// Use setting_sources=["user"] to exclude project settings
	maxTurns := 1
	options := &claude.ClaudeAgentOptions{
		SettingSources: []claude.SettingSource{claude.SettingSourceUser},
		Cwd:            &tmpDir,
		MaxTurns:       &maxTurns,
	}

	client := claude.NewClaudeSDKClient(options)

	ctx := context.Background()
	err = client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Start query
	msgCh, errCh := client.Query(ctx, "What is 2 + 2?")

	// Check that project command is NOT available
	foundInit := false
	for msg := range msgCh {
		if data, ok := GetSystemMessageData(msg); ok {
			if subtype, ok := data["subtype"].(string); ok && subtype == "init" {
				foundInit = true
				if commands, ok := data["slash_commands"].([]interface{}); ok {
					for _, cmd := range commands {
						if cmdStr, ok := cmd.(string); ok && cmdStr == "testcmd" {
							t.Errorf("testcmd should NOT be available with user-only sources, but found it in: %v", commands)
						}
					}
					t.Logf("Verified testcmd not in slash_commands: %v", commands)
				}
				break
			}
		}
	}

	if err := <-errCh; err != nil {
		t.Fatalf("Query error: %v", err)
	}

	if !foundInit {
		t.Fatal("Did not receive init message")
	}
}

func TestSettingSourcesProjectIncluded(t *testing.T) {
	RequireClaudeCode(t)

	// Create temporary project directory with local settings
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	err := os.MkdirAll(claudeDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create .claude dir: %v", err)
	}

	// Create local settings with custom outputStyle
	settingsFile := filepath.Join(claudeDir, "settings.local.json")
	settingsContent := `{"outputStyle": "local-test-style"}`
	err = os.WriteFile(settingsFile, []byte(settingsContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write settings file: %v", err)
	}

	// Use setting_sources=["user", "project", "local"] to include local settings
	maxTurns := 1
	options := &claude.ClaudeAgentOptions{
		SettingSources: []claude.SettingSource{
			claude.SettingSourceUser,
			claude.SettingSourceProject,
			claude.SettingSourceLocal,
		},
		Cwd:      &tmpDir,
		MaxTurns: &maxTurns,
	}

	client := claude.NewClaudeSDKClient(options)

	ctx := context.Background()
	err = client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Start query
	msgCh, errCh := client.Query(ctx, "What is 2 + 2?")

	// Check that settings WERE loaded
	foundInit := false
	for msg := range msgCh {
		if data, ok := GetSystemMessageData(msg); ok {
			if subtype, ok := data["subtype"].(string); ok && subtype == "init" {
				foundInit = true
				outputStyle, ok := data["output_style"].(string)
				if !ok {
					t.Fatal("output_style not found in init message")
				}
				if outputStyle != "local-test-style" {
					t.Errorf("outputStyle should be from local settings, got: %s", outputStyle)
				} else {
					t.Logf("Successfully loaded local settings: outputStyle=%s", outputStyle)
				}
				// On Windows, wait for file handles to be released before cleanup
				if runtime.GOOS == "windows" {
					time.Sleep(500 * time.Millisecond)
				}
				break
			}
		}
	}

	if err := <-errCh; err != nil {
		t.Fatalf("Query error: %v", err)
	}

	if !foundInit {
		t.Fatal("Did not receive init message")
	}

	// On Windows, wait for file handles to be released before cleanup
	if runtime.GOOS == "windows" {
		time.Sleep(500 * time.Millisecond)
	}
}
