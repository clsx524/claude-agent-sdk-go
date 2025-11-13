package unit

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestChangelogFormat validates that CHANGELOG.md follows the Keep a Changelog format.
// See: https://keepachangelog.com/en/1.0.0/
func TestChangelogFormat(t *testing.T) {
	// Find project root
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Navigate up to find CHANGELOG.md
	changelogPath := ""
	for dir := cwd; dir != "/"; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "CHANGELOG.md")
		if _, err := os.Stat(candidate); err == nil {
			changelogPath = candidate
			break
		}
	}

	if changelogPath == "" {
		t.Skip("CHANGELOG.md not found - skipping validation")
	}

	file, err := os.Open(changelogPath)
	if err != nil {
		t.Fatalf("Failed to open CHANGELOG.md: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	hasTitle := false
	hasVersionSection := false
	validSectionHeaders := map[string]bool{
		"Added":      true,
		"Changed":    true,
		"Deprecated": true,
		"Removed":    true,
		"Fixed":      true,
		"Security":   true,
	}

	// Regex patterns
	titlePattern := regexp.MustCompile(`^#\s+[Cc]hangelog`)
	versionPattern := regexp.MustCompile(`^##\s+\[?(\d+\.\d+\.\d+|Unreleased)\]?`)
	sectionPattern := regexp.MustCompile(`^###\s+(\w+)`)
	linkPattern := regexp.MustCompile(`^\[.*\]:\s+https?://`)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check for title
		if titlePattern.MatchString(line) {
			hasTitle = true
			continue
		}

		// Check for version headers
		if versionPattern.MatchString(line) {
			hasVersionSection = true
			continue
		}

		// Check for section headers
		if matches := sectionPattern.FindStringSubmatch(line); matches != nil {
			sectionName := matches[1]
			if !validSectionHeaders[sectionName] {
				t.Errorf("Invalid section header '%s' at line %d. Must be one of: Added, Changed, Deprecated, Removed, Fixed, Security", sectionName, lineNum)
			}
			continue
		}

		// Allow link definitions at the bottom
		if linkPattern.MatchString(line) {
			continue
		}

		// Allow list items
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			continue
		}

		// Allow description text (not starting with #)
		if !strings.HasPrefix(line, "#") {
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("Error reading CHANGELOG.md: %v", err)
	}

	if !hasTitle {
		t.Error("CHANGELOG.md should start with '# Changelog' header")
	}

	if !hasVersionSection {
		t.Error("CHANGELOG.md should have at least one version section (e.g., '## [Unreleased]' or '## [1.0.0]')")
	}
}

// TestChangelogVersionFormat validates version numbers follow semantic versioning.
func TestChangelogVersionFormat(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	changelogPath := ""
	for dir := cwd; dir != "/"; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "CHANGELOG.md")
		if _, err := os.Stat(candidate); err == nil {
			changelogPath = candidate
			break
		}
	}

	if changelogPath == "" {
		t.Skip("CHANGELOG.md not found - skipping validation")
	}

	file, err := os.Open(changelogPath)
	if err != nil {
		t.Fatalf("Failed to open CHANGELOG.md: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	semverPattern := regexp.MustCompile(`^##\s+\[?(\d+\.\d+\.\d+)\]?`)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if matches := semverPattern.FindStringSubmatch(line); matches != nil {
			version := matches[1]
			// Validate semantic versioning format (MAJOR.MINOR.PATCH)
			parts := strings.Split(version, ".")
			if len(parts) != 3 {
				t.Errorf("Version '%s' at line %d does not follow semantic versioning (MAJOR.MINOR.PATCH)", version, lineNum)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("Error reading CHANGELOG.md: %v", err)
	}
}

// TestChangelogNotEmpty validates that CHANGELOG.md has content.
func TestChangelogNotEmpty(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	changelogPath := ""
	for dir := cwd; dir != "/"; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "CHANGELOG.md")
		if _, err := os.Stat(candidate); err == nil {
			changelogPath = candidate
			break
		}
	}

	if changelogPath == "" {
		t.Skip("CHANGELOG.md not found - skipping validation")
	}

	info, err := os.Stat(changelogPath)
	if err != nil {
		t.Fatalf("Failed to stat CHANGELOG.md: %v", err)
	}

	if info.Size() == 0 {
		t.Error("CHANGELOG.md is empty - should contain version history")
	}

	// Read content
	content, err := os.ReadFile(changelogPath)
	if err != nil {
		t.Fatalf("Failed to read CHANGELOG.md: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	nonEmptyLines := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmptyLines++
		}
	}

	if nonEmptyLines < 3 {
		t.Error("CHANGELOG.md should have at least a title, version section, and some content")
	}
}
