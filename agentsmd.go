package autohand

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LoadAgentsMd loads AGENTS.md content from a file path, URL, or inline string.
func LoadAgentsMd(source string) (string, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return "", fmt.Errorf("empty AGENTS.md source")
	}

	// URL
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Get(source)
		if err != nil {
			return "", fmt.Errorf("fetch AGENTS.md: %w", err)
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("read AGENTS.md response: %w", err)
		}
		return string(data), nil
	}

	// File path (absolute or relative)
	path := source
	if !filepath.IsAbs(path) {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("get working directory: %w", err)
		}
		path = filepath.Join(cwd, path)
	}

	if _, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read AGENTS.md file: %w", err)
		}
		return string(data), nil
	}

	// Inline content
	return source, nil
}

// CreateDefaultAgentsMd generates a default AGENTS.md template.
func CreateDefaultAgentsMd(projectName string) string {
	name := projectName
	if name == "" {
		name = "Project"
	}
	return fmt.Sprintf(`# %s Autopilot

This file helps AI assistants understand your project structure, conventions, and workflows.

## Overview

- **Language**: (e.g., Go, TypeScript, Python)
- **Framework**: (e.g., React, Gin, FastAPI)
- **Package Manager**: (e.g., go modules, npm, poetry)

## Architecture

Describe the high-level architecture and design patterns used.

## Conventions

- Code style and formatting rules
- Naming conventions
- Testing patterns
- Documentation standards

## Commands

- Build: (e.g., go build, npm run build)
- Test: (e.g., go test, npm test)
- Lint: (e.g., golangci-lint, eslint)

## Skills

List any domain-specific skills or knowledge required.
`, name)
}
