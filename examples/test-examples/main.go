// test-examples validates that all Go examples have correct structure.
//
// Usage:
//
//	go run ./examples/test-examples
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	examplesDir := filepath.Join("..")

	entries, err := os.ReadDir(examplesDir)
	if err != nil {
		log.Fatalf("read examples dir: %v", err)
	}

	var exampleDirs []string
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			exampleDirs = append(exampleDirs, entry.Name())
		}
	}

	fmt.Println("=== Validating Go Examples ===\n")
	fmt.Printf("Found %d example directories\n\n", len(exampleDirs))

	passed := 0
	failed := 0

	for _, dir := range exampleDirs {
		if dir == "test-examples" {
			continue
		}
		mainPath := filepath.Join(examplesDir, dir, "main.go")
		modPath := filepath.Join(examplesDir, dir, "go.mod")

		content, err := os.ReadFile(mainPath)
		if err != nil {
			fmt.Printf("  %s: missing main.go\n", dir)
			failed++
			continue
		}
		contentStr := string(content)
		usesSDK := strings.Contains(contentStr, "autohand.NewSDK")
		usesAgent := strings.Contains(contentStr, "autohand.NewAgent")
		reflectsSDKSurface := strings.Contains(contentStr, "reflect.TypeOf") &&
			strings.Contains(contentStr, "MethodByName")
		startsLifecycle := strings.Contains(contentStr, ".Start(") || usesAgent || reflectsSDKSurface
		closesLifecycle := strings.Contains(contentStr, ".Close()") || reflectsSDKSurface

		_, err = os.ReadFile(modPath)
		if err != nil {
			fmt.Printf("  %s: missing go.mod\n", dir)
			failed++
			continue
		}

		checks := map[string]bool{
			"hasPackageMain":    strings.Contains(contentStr, "package main"),
			"hasMainFunc":       strings.Contains(contentStr, "func main()"),
			"hasSdkImport":      strings.Contains(contentStr, "autohand \"github.com/autohandai/code-agent-sdk-go\""),
			"hasSupportedApi":   usesSDK || usesAgent,
			"hasErrorHandling":  strings.Contains(contentStr, "log.Fatalf"),
			"hasLifecycleStart": startsLifecycle,
			"hasLifecycleClose": closesLifecycle,
		}

		allPassed := true
		for check, result := range checks {
			if !result {
				if allPassed {
					fmt.Printf("  %s: failed checks:\n", dir)
					allPassed = false
				}
				fmt.Printf("    - %s\n", check)
			}
		}

		if allPassed {
			fmt.Printf("  %s: all checks passed\n", dir)
			passed++
		} else {
			failed++
		}
	}

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Passed: %d/%d\n", passed, len(exampleDirs)-1)
	fmt.Printf("Failed: %d/%d\n", failed, len(exampleDirs)-1)

	if failed > 0 {
		os.Exit(1)
	}
}
