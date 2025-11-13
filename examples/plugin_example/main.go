package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	claude "github.com/clsx524/claude-agent-sdk-go"
)

func main() {
	fmt.Println("=== Plugin Example ===")
	fmt.Println("This example demonstrates how to use plugins with Claude Code SDK.")
	fmt.Println()

	ctx := context.Background()

	// Get the path to the demo plugin
	// In production, you can use any path to your plugin directory
	pluginPath, err := filepath.Abs("./plugins/demo-plugin")
	if err != nil {
		log.Fatalf("Failed to resolve plugin path: %v", err)
	}

	maxTurns := 1
	options := &claude.ClaudeAgentOptions{
		Plugins: []claude.SdkPluginConfig{
			{
				Type: "local",
				Path: pluginPath,
			},
		},
		MaxTurns: &maxTurns, // Limit to one turn for quick demo
	}

	fmt.Printf("Loading plugin from: %s\n\n", pluginPath)

	msgCh, errCh, err := claude.Query(ctx, "Hello!", options, nil)
	if err != nil {
		log.Fatalf("Failed to create query: %v", err)
	}

	foundPlugins := false

	// Process messages
	for msg := range msgCh {
		switch m := msg.(type) {
		case *claude.SystemMessage:
			if m.Subtype == "init" {
				fmt.Println("System initialized!")
				fmt.Printf("System message data keys: %v\n\n", getKeys(m.Data))

				// Check for plugins in the system message
				if pluginsData, ok := m.Data["plugins"].([]interface{}); ok && len(pluginsData) > 0 {
					fmt.Println("Plugins loaded:")
					for _, plugin := range pluginsData {
						if p, ok := plugin.(map[string]interface{}); ok {
							name := p["name"]
							path := p["path"]
							fmt.Printf("  - %v (path: %v)\n", name, path)
						}
					}
					foundPlugins = true
				} else {
					fmt.Println("Note: Plugin was passed via CLI but may not appear in system message.")
					fmt.Printf("Plugin path configured: %s\n", pluginPath)
					foundPlugins = true
				}
			}

		case *claude.AssistantMessage:
			for _, block := range m.Content {
				if textBlock, ok := block.(claude.TextBlock); ok {
					fmt.Printf("\nClaude: %s\n", textBlock.Text)
				}
			}

		case *claude.ResultMessage:
			if foundPlugins {
				fmt.Println("\nPlugin successfully configured!")
				fmt.Println("\nYou can now use custom commands from the plugin, such as /greet")
			}
		}
	}

	// Check for errors
	if err := <-errCh; err != nil {
		log.Printf("Query error: %v", err)
	}
}

// getKeys returns the keys from a map
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
