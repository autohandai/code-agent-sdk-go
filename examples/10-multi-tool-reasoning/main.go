// 10-multi-tool-reasoning demonstrates using multiple tools across turns.
//
// The agent uses READ_FILE and BASH together across multiple turns to
// understand code, run tests, and report a summary.
//
// Usage:
//
//	go run ./examples/10-multi-tool-reasoning
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	autohand "github.com/autohandai/code-agent-sdk-go"
)

func main() {
	ctx := context.Background()

	// Create a temporary project with a module and a test
	tmpDir, err := os.MkdirTemp("", "multi-tool-example-")
	if err != nil {
		log.Fatalf("create temp dir: %v", err)
	}
	defer func() {
		os.RemoveAll(tmpDir)
		fmt.Printf("\nCleaned up test directory: %s\n", tmpDir)
	}()

	mathUtils := `package math

func Fibonacci(n int) int {
	if n <= 0 { return 0 }
	if n == 1 { return 1 }
	a, b := 0, 1
	for i := 2; i <= n; i++ {
		a, b = b, a+b
	}
	return b
}

func Factorial(n int) int {
	if n < 0 { panic("negative") }
	result := 1
	for i := 2; i <= n; i++ {
		result *= i
	}
	return result
}
`
	testFile := `package math

import "testing"

func TestFibonacci(t *testing.T) {
	if Fibonacci(0) != 0 { t.Error() }
	if Fibonacci(1) != 1 { t.Error() }
	if Fibonacci(5) != 5 { t.Error() }
	if Fibonacci(10) != 55 { t.Error() }
}

func TestFactorial(t *testing.T) {
	if Factorial(0) != 1 { t.Error() }
	if Factorial(5) != 120 { t.Error() }
	if Factorial(10) != 3628800 { t.Error() }
}
`
	goMod := `module test-project

go 1.22
`

	_ = os.WriteFile(filepath.Join(tmpDir, "math.go"), []byte(mathUtils), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "math_test.go"), []byte(testFile), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)

	fmt.Println("=== Multi-Tool Reasoning Demo ===")
	fmt.Printf("Created test project in: %s\n\n", tmpDir)

	sdk := autohand.NewSDK(&autohand.Config{})

	if err := sdk.Start(ctx); err != nil {
		log.Fatalf("start SDK: %v", err)
	}
	fmt.Println("SDK started")

	oldCwd, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	prompt := "First, list all Go files in this directory. Then read each Go file. Finally, run `go test` and report the test results. Summarize the codebase."
	fmt.Printf("Sending prompt: %q\n\n", prompt)

	var fullResponse string
	events, err := sdk.StreamPrompt(ctx, &autohand.PromptParams{Message: prompt})
	if err != nil {
		log.Fatalf("stream prompt: %v", err)
	}

	for event := range events {
		switch e := event.(type) {
		case autohand.ToolStartEvent:
			fmt.Printf("[Tool called: %s]\n", e.ToolName)
		case autohand.ToolEndEvent:
			fmt.Printf("[Tool completed: %s]\n", e.ToolName)
			if e.Output != "" {
				truncated := e.Output
				if len(truncated) > 1000 {
					truncated = truncated[:1000] + "..."
				}
				fmt.Println("Output:")
				fmt.Println(truncated)
			}
		case autohand.PermissionRequestEvent:
			fmt.Printf("[Permission request: %s]\n", e.Tool)
			fmt.Printf("  Description: %s\n", e.Description)
			if err := sdk.AllowPermission(ctx, e.RequestID, autohand.ScopeOnce); err != nil {
				log.Printf("allow: %v", err)
			}
		case autohand.MessageUpdateEvent:
			fmt.Print(e.Delta)
			fullResponse += e.Delta
		case autohand.MessageEndEvent:
			fullResponse = e.Content
		}
	}

	fmt.Println("\n=== Agent Response ===")
	fmt.Println(fullResponse)

	if err := sdk.Close(); err != nil {
		log.Fatalf("close SDK: %v", err)
	}
	fmt.Println("\nSDK stopped")
}
