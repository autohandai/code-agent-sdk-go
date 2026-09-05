package autohand

import (
	"context"
	"fmt"
	"time"
)

type stepDecision struct {
	stepID string
	stop   bool
	err    error
}

func evaluateStop(ctx context.Context, conditions []StopCondition, steps []AgentStep, stepID string) (decision stepDecision) {
	decision.stepID = stepID
	defer func() {
		if recovered := recover(); recovered != nil {
			decision.stop = true
			decision.err = fmt.Errorf("stop condition panicked: %v", recovered)
		}
	}()
	for _, condition := range conditions {
		stop, err := condition(ctx, steps)
		if stop || err != nil {
			return stepDecision{stepID: stepID, stop: true, err: err}
		}
	}
	return decision
}

func isTurnTerminal(event Event) bool {
	switch event.(type) {
	case TurnEndEvent, AgentEndEvent:
		return true
	default:
		return false
	}
}

func (s *SDK) streamTurn(ctx context.Context, params *PromptParams, out chan<- Event) (runErr error) {
	// Keep the subscription alive while a cancelled turn is being drained.
	rpcCtx, cancelRPC := context.WithCancel(context.Background())
	defer cancelRPC()
	decisionCtx, cancelDecision := context.WithCancel(ctx)
	defer cancelDecision()
	events := s.client.Events(rpcCtx)
	promptDone := make(chan error, 1)
	go func() { promptDone <- s.client.Prompt(rpcCtx, params) }()
	var completion <-chan error = promptDone
	terminal := false
	defer func() {
		cancelDecision()
		if !terminal || completion != nil {
			if err := s.drainPrompt(events, completion, terminal); runErr == nil {
				runErr = err
			}
		}
	}()
	var steps []AgentStep
	var decisions <-chan stepDecision
	var conditionError error
	for !terminal || completion != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-completion:
			completion = nil
			if err != nil {
				terminal = true
				return err
			}
		case decision := <-decisions:
			decisions = nil
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if terminal {
				continue
			}
			if err := s.client.StepDecision(ctx, decision.stepID, decision.stop); err != nil {
				return err
			}
			conditionError = decision.err
		case event, ok := <-events:
			if !ok {
				if terminal {
					events = nil
					continue
				}
				return ErrTransportClosed
			}
			if failure, ok := event.(ErrorEvent); ok && failure.Err != nil {
				return failure.Err
			}
			if terminal {
				continue
			}
			if unknown, ok := event.(GenericEvent); ok && unknown.Method == "autohand.stepEnd" {
				return fmt.Errorf("CLI emitted a malformed autohand.stepEnd notification")
			}
			if step, ok := event.(StepEndEvent); ok {
				if decisions != nil {
					return fmt.Errorf("CLI emitted a step before acknowledging its predecessor")
				}
				steps = append(steps, step.Step)
				result := make(chan stepDecision, 1)
				decisions = result
				snapshot := append([]AgentStep(nil), steps...)
				go func() { result <- evaluateStop(decisionCtx, params.StopWhen, snapshot, step.StepID) }()
			}
			terminal = isTurnTerminal(event)
			if terminal {
				cancelDecision()
			}
			select {
			case out <- event:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return conditionError
}

func (s *SDK) drainPrompt(events <-chan Event, completion <-chan error, terminal bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if !terminal {
		if err := s.client.Abort(ctx); err != nil {
			_ = s.Stop()
			return fmt.Errorf("abort abandoned prompt: %w", err)
		}
	}
	for !terminal || completion != nil {
		select {
		case <-ctx.Done():
			_ = s.Stop()
			return fmt.Errorf("CLI did not finish the abandoned prompt: %w", ctx.Err())
		case <-completion:
			completion = nil
		case event, ok := <-events:
			if !ok {
				_ = s.Stop()
				return ErrTransportClosed
			}
			terminal = terminal || isTurnTerminal(event)
		}
	}
	return nil
}
