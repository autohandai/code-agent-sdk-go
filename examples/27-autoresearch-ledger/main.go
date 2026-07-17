// 27-autoresearch-ledger demonstrates typed autoresearch lifecycle and ledger operations.
//
// Usage:
//
//	AUTOHAND_TARGET_REPO=/path/to/project go run ./examples/27-autoresearch-ledger
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	autohand "github.com/autohandai/code-agent-sdk-go"
)

func main() {
	ctx := context.Background()
	targetRepo := os.Getenv("AUTOHAND_TARGET_REPO")
	if targetRepo == "" {
		targetRepo = "."
	}

	agent, err := autohand.NewAgent(ctx, &autohand.Config{
		CWD:     targetRepo,
		CLIPath: os.Getenv("AUTOHAND_CLI_PATH"),
		Model:   os.Getenv("AUTOHAND_MODEL"),
	})
	if err != nil {
		log.Fatalf("create agent: %v", err)
	}
	defer agent.Close()

	started, err := agent.StartAutoresearch(ctx, &autohand.AutoresearchStartParams{
		Objective:      "Improve the benchmark while preserving correctness",
		MaxIterations:  10,
		MetricName:     "duration_ms",
		MetricUnit:     "ms",
		Direction:      autohand.AutoresearchLower,
		MeasureCommand: "go test ./...",
		ChecksCommand:  "go test ./...",
		FilesInScope:   []string{"."},
		Sampling: &autohand.AutoresearchSamplingOptions{
			MinSamples:          2,
			MaxSamples:          5,
			ConfidenceThreshold: 0.9,
		},
	})
	if err != nil {
		log.Fatalf("start autoresearch: %v", err)
	}
	if !started.Success {
		log.Fatalf("start autoresearch: %s", started.Error)
	}
	if started.Instruction == "" {
		log.Fatal("start autoresearch returned no loop instruction")
	}

	run, err := agent.Send(ctx, started.Instruction, nil)
	if err != nil {
		log.Fatalf("send loop instruction: %v", err)
	}
	events, err := run.Stream(ctx)
	if err != nil {
		log.Fatalf("stream loop: %v", err)
	}
	for event := range events {
		switch event := event.(type) {
		case autohand.MessageUpdateEvent:
			fmt.Print(event.Delta)
		case autohand.AutoresearchLifecycleEvent:
			fmt.Printf("\n[autoresearch:%s] %s\n", event.Phase, event.StatusText)
		case autohand.AutoresearchOperationEvent:
			fmt.Printf("\n[autoresearch:%s:%s] success=%t\n", event.Operation, event.Phase, event.Success)
		}
	}

	if _, err := run.Wait(ctx); err != nil {
		log.Fatalf("wait: %v", err)
	}
	history, err := agent.GetAutoresearchHistory(ctx)
	if err != nil {
		log.Fatalf("history: %v", err)
	}
	fmt.Printf("\n%d persisted attempts\n", len(history.Attempts))
	status, err := agent.GetAutoresearchStatus(ctx)
	if err != nil {
		log.Fatalf("status: %v", err)
	}
	fmt.Printf("Autoresearch active=%t runs=%d: %s\n", status.Active, status.RunsLogged, status.StatusText)

	replayable := make([]autohand.AutoresearchHistoryAttempt, 0, len(history.Attempts))
	for _, attempt := range history.Attempts {
		if attempt.Replayable {
			replayable = append(replayable, attempt)
		}
	}
	if len(replayable) > 0 {
		attempt := replayable[0]
		if _, err := agent.ReplayAutoresearch(ctx, &autohand.AutoresearchReplayParams{
			AttemptID: attempt.AttemptID,
			Evaluator: autohand.AutoresearchEvaluatorOriginal,
		}); err != nil {
			log.Printf("replay %s: %v", attempt.AttemptID, err)
		}
		if _, err := agent.RescoreAutoresearch(ctx, autohand.AutoresearchRescoreAttempt(attempt.AttemptID)); err != nil {
			log.Printf("rescore %s: %v", attempt.AttemptID, err)
		}
		if _, err := agent.ReplayAutoresearch(ctx, &autohand.AutoresearchReplayParams{
			AttemptID: attempt.AttemptID,
			Evaluator: autohand.AutoresearchEvaluatorCurrent,
		}); err != nil {
			log.Printf("current replay %s: %v", attempt.AttemptID, err)
		}
		if _, err := agent.PinAutoresearch(ctx, &autohand.AutoresearchPinParams{
			AttemptID: attempt.AttemptID,
			Pinned:    true,
		}); err != nil {
			log.Printf("pin %s: %v", attempt.AttemptID, err)
		}
	}
	if len(replayable) > 1 {
		if _, err := agent.CompareAutoresearch(ctx, &autohand.AutoresearchCompareParams{
			LeftAttemptID:  replayable[0].AttemptID,
			RightAttemptID: replayable[1].AttemptID,
		}); err != nil {
			log.Printf("compare attempts: %v", err)
		}
	}

	pareto, err := agent.GetAutoresearchPareto(ctx)
	if err != nil {
		log.Fatalf("pareto: %v", err)
	}
	fmt.Printf("Pareto attempts: %v\n", pareto.AttemptIDs)

	dryRun := true
	preview, err := agent.PruneAutoresearch(ctx, &autohand.AutoresearchPruneParams{DryRun: &dryRun})
	if err != nil {
		log.Fatalf("prune preview: %v", err)
	}
	fmt.Printf("Prune preview: %d candidates, %d bytes\n", len(preview.Candidates), preview.BytesFreed)
	if os.Getenv("AUTOHAND_APPLY_PRUNE") == "1" {
		dryRun = false
		if _, err := agent.PruneAutoresearch(ctx, &autohand.AutoresearchPruneParams{
			DryRun: &dryRun,
			Yes:    true,
		}); err != nil {
			log.Fatalf("apply prune: %v", err)
		}
	}

	if _, err := agent.StopAutoresearch(ctx); err != nil {
		log.Fatalf("stop autoresearch: %v", err)
	}
}
