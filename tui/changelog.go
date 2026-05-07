package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// ChangelogEntry holds one version section from the CHANGELOG.md.
type ChangelogEntry struct {
	Version string
	Date    string
	Body    string
}

func loadChangelog() ([]ChangelogEntry, error) {
	paths := []string{"CHANGELOG.md"}

	if execPath, err := os.Executable(); err == nil {
		if dir := filepath.Dir(execPath); dir != "" {
			paths = append(paths, filepath.Join(dir, "CHANGELOG.md"))
		}
	}

	if runtime.GOOS == "darwin" {
		for _, p := range paths {
			dir := filepath.Dir(p)
			for i := 0; i < 3 && dir != "/" && dir != "."; i++ {
				paths = append(paths, filepath.Join(dir, "CHANGELOG.md"))
				dir = filepath.Dir(dir)
			}
		}
	}

	var data []byte
	var err error
	for _, p := range paths {
		data, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("CHANGELOG.md not found")
	}

	return parseChangelog(string(data)), nil
}

// parseChangelog splits the markdown by version headers like:
// ## [1.2.3] - 2026-05-06
// ## [1.2.3]
// ## 1.2.3 - 2026-05-06
func parseChangelog(text string) []ChangelogEntry {
	// Header pattern: ## [version] - date   or   ## [version]   or   ## version - date
	headerRe := regexp.MustCompile(`^##\s+\[?([^\]]+)\]?(?:\s+-\s+(\d{4}-\d{2}-\d{2}))?\s*$`)

	lines := strings.Split(text, "\n")
	var entries []ChangelogEntry
	var current *ChangelogEntry

	for _, line := range lines {
		line = strings.TrimRight(line, " \r")
		if m := headerRe.FindStringSubmatch(line); m != nil {
			if current != nil {
				entries = append(entries, *current)
			}
			current = &ChangelogEntry{
				Version: strings.TrimSpace(m[1]),
				Date:    strings.TrimSpace(m[2]),
			}
			continue
		}
		if current != nil {
			current.Body += line + "\n"
		}
	}
	if current != nil {
		entries = append(entries, *current)
	}

	for i := range entries {
		entries[i].Body = strings.TrimSpace(entries[i].Body)
	}

	return entries
}

func versionIndex(entries []ChangelogEntry, version string) int {
	version = strings.TrimPrefix(version, "v")
	for i, e := range entries {
		v := strings.TrimPrefix(e.Version, "v")
		if v == version {
			return i
		}
	}
	return -1
}

// getChangesSince returns entries from oldVersion up to and including currentVersion.
// The changelog is expected newest-first (Unreleased at top).
func getChangesSince(entries []ChangelogEntry, oldVersion, currentVersion string) []ChangelogEntry {
	oldIdx := versionIndex(entries, oldVersion)
	curIdx := versionIndex(entries, currentVersion)

	if curIdx < 0 {
		return nil
	}
	if oldIdx < 0 {
		return []ChangelogEntry{entries[curIdx]}
	}

	var result []ChangelogEntry
	for i := oldIdx - 1; i >= curIdx; i-- {
		result = append(result, entries[i])
	}
	return result
}

func renderChangelogBody(body string) string {
	lines := strings.Split(body, "\n")
	var out []string
	for _, line := range lines {
		line = strings.TrimRight(line, " \r")
		trimmed := strings.TrimLeft(line, " ")
		if strings.HasPrefix(trimmed, "### ") {
			cat := strings.TrimPrefix(trimmed, "### ")
			out = append(out, "\n"+labelStyle.Render(cat))
		} else if strings.HasPrefix(trimmed, "- ") {
			item := strings.TrimPrefix(trimmed, "- ")
			out = append(out, "  • "+item)
		} else if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return strings.Join(out, "\n")
}

func renderChangelogEntries(entries []ChangelogEntry) string {
	if len(entries) == 0 {
		return "No hay cambios registrados."
	}

	var parts []string
	for _, e := range entries {
		header := e.Version
		if e.Date != "" {
			header = fmt.Sprintf("%s  (%s)", e.Version, e.Date)
		}
		parts = append(parts, titleStyle.Render(header), renderChangelogBody(e.Body), "")
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}
