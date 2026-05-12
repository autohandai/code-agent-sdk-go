// 26-runtime-error-to-pr demonstrates how to turn an application runtime error
// into an automated repair pull request.
//
// Usage:
//
//	AUTOHAND_TARGET_REPO=/path/to/app go run ./examples/26-runtime-error-to-pr
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"strings"

	autohand "github.com/autohandai/code-agent-sdk-go"
)

type Cart struct {
	Subtotal float64
	Customer *Customer
}

type Customer struct {
	LoyaltyTier string
}

func checkoutDiscount(cart Cart) (discount float64) {
	defer func() {
		if recovered := recover(); recovered != nil {
			panic(fmt.Sprintf("checkout discount failed: %v", recovered))
		}
	}()

	if cart.Customer.LoyaltyTier == "gold" {
		return cart.Subtotal * 0.15
	}
	return cart.Subtotal * 0.05
}

func captureRuntimeError() (report string) {
	defer func() {
		if recovered := recover(); recovered != nil {
			report = strings.Join([]string{
				fmt.Sprintf("panic: %v", recovered),
				string(debug.Stack()),
				"Request: POST /checkout",
				`Payload: {"subtotal":129,"customer":null}`,
			}, "\n")
		}
	}()

	checkoutDiscount(Cart{Subtotal: 129})
	return strings.Join([]string{
		"panic: checkout discount failed: runtime error: invalid memory address or nil pointer dereference",
		"goroutine 88 [running]:",
		"checkout/discounts.checkoutDiscount({Subtotal:129, Customer:nil})",
		"    src/checkout/discounts.go:42",
		"checkout/session.CreateCheckoutSession(...)",
		"    src/checkout/session.go:88",
		"Request: POST /checkout",
		`Payload: {"subtotal":129,"customer":null}`,
	}, "\n")
}

func main() {
	ctx := context.Background()
	targetRepo := os.Getenv("AUTOHAND_TARGET_REPO")
	if targetRepo == "" {
		targetRepo = "."
	}

	cfg := &autohand.Config{
		CWD:     targetRepo,
		CLIPath: os.Getenv("AUTOHAND_CLI_PATH"),
		Model:   os.Getenv("AUTOHAND_MODEL"),
	}

	agent, err := autohand.NewAgent(ctx, cfg)
	if err != nil {
		log.Fatalf("create agent: %v", err)
	}
	defer agent.Close()

	capturedError := captureRuntimeError()

	prompt := strings.Join([]string{
		"You are a QA engineering agent that turns production error reports into small repair pull requests.",
		"Reproduce the failure when the repository makes that possible.",
		"Fix the root cause, add or update a focused regression test, run the relevant validation command, commit the fix, push a branch, and create a pull request.",
		"Keep the pull request description concise and include the error signature, the fix summary, and the validation result.",
		"",
		"A runtime error was captured by the application error boundary.",
		"",
		"Captured error:",
		"```text",
		capturedError,
		"```",
		"",
		"Expected user impact:",
		"A checkout session should still calculate a safe default discount when the customer object is missing.",
		"",
		"Please create a pull request with the fix.",
	}, "\n")

	run, err := agent.Send(ctx, prompt, nil)
	if err != nil {
		log.Fatalf("send: %v", err)
	}

	events, err := run.Stream(ctx)
	if err != nil {
		log.Fatalf("stream: %v", err)
	}

	for event := range events {
		switch e := event.(type) {
		case autohand.MessageUpdateEvent:
			fmt.Print(e.Delta)
		case autohand.ToolStartEvent:
			fmt.Printf("\n[tool] %s\n", e.ToolName)
		case autohand.PermissionRequestEvent:
			fmt.Printf("\n[permission] %s: %s\n", e.Tool, e.Description)
		case autohand.ErrorEvent:
			fmt.Printf("\n[error] %s\n", e.Message)
		}
	}

	result, err := run.Wait(ctx)
	if err != nil {
		log.Fatalf("wait: %v", err)
	}
	fmt.Printf("\n\nRun %s %s.\n", result.ID, result.Status)
}
