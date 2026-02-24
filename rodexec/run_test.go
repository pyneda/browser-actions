package rodexec

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-rod/rod"
	browseractions "github.com/pyneda/browser-actions"
)

func TestRunValidationFailureReturnsActionCountWithoutLaunching(t *testing.T) {
	origOpenPage := runOpenPage
	origExecute := runExecute
	t.Cleanup(func() {
		runOpenPage = origOpenPage
		runExecute = origExecute
	})

	openCalled := false
	executeCalled := false
	runOpenPage = func(opts RunOptions) (*rod.Page, func(), error) {
		openCalled = true
		return nil, func() {}, nil
	}
	runExecute = func(ctx context.Context, page *rod.Page, actions []browseractions.Action, opts *ExecuteOptions) (ExecutionResult, error) {
		executeCalled = true
		return ExecutionResult{}, nil
	}

	script := browseractions.BrowserActions{
		Title: "invalid",
		Actions: []browseractions.Action{
			{Type: browseractions.ActionWait, For: browseractions.WaitVisible}, // missing selector
			{Type: browseractions.ActionSleep, Duration: 10},
		},
	}

	result, err := Run(context.Background(), script, &RunOptions{
		ExecuteOptions: &ExecuteOptions{
			Validate:          true,
			ValidationProfile: browseractions.ValidationProfileStrict,
		},
		// Intentionally invalid script should fail before browser launch.
	})
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if result.TotalActions != 2 {
		t.Fatalf("expected total actions 2, got %d", result.TotalActions)
	}
	if result.CompletedActions != 0 {
		t.Fatalf("expected no completed actions, got %d", result.CompletedActions)
	}
	if openCalled {
		t.Fatalf("expected browser launch to be skipped on validation failure")
	}
	if executeCalled {
		t.Fatalf("expected execute to be skipped on validation failure")
	}
}

func TestRunPropagatesTimeoutAndExecuteOptions(t *testing.T) {
	origOpenPage := runOpenPage
	origExecute := runExecute
	t.Cleanup(func() {
		runOpenPage = origOpenPage
		runExecute = origExecute
	})

	var openCount int
	var cleanupCount int
	runOpenPage = func(opts RunOptions) (*rod.Page, func(), error) {
		openCount++
		return nil, func() { cleanupCount++ }, nil
	}

	var (
		gotActions []browseractions.Action
		gotOpts    *ExecuteOptions
		deadlineOK bool
	)
	runExecute = func(ctx context.Context, page *rod.Page, actions []browseractions.Action, opts *ExecuteOptions) (ExecutionResult, error) {
		if page != nil {
			t.Fatalf("expected stubbed page to be nil in this test")
		}
		_, deadlineOK = ctx.Deadline()
		gotActions = append([]browseractions.Action(nil), actions...)
		copied := *opts
		gotOpts = &copied
		return ExecutionResult{Succeeded: true, TotalActions: len(actions), CompletedActions: len(actions)}, nil
	}

	script := browseractions.BrowserActions{
		Title: "ok",
		Actions: []browseractions.Action{
			{Type: browseractions.ActionSleep, Duration: 1},
		},
	}
	result, err := Run(context.Background(), script, &RunOptions{
		Timeout: 200 * time.Millisecond,
		ExecuteOptions: &ExecuteOptions{
			Validate:              true,
			ValidationProfile:     browseractions.ValidationProfileLenient,
			IncludeScreenshotData: true,
			WriteFiles:            false,
			OutputDir:             "artifacts",
		},
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !result.Succeeded || result.CompletedActions != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if openCount != 1 {
		t.Fatalf("expected open page once, got %d", openCount)
	}
	if cleanupCount != 1 {
		t.Fatalf("expected cleanup once, got %d", cleanupCount)
	}
	if !deadlineOK {
		t.Fatalf("expected timeout deadline to be applied to execution context")
	}
	if len(gotActions) != 1 || gotActions[0].Type != browseractions.ActionSleep {
		t.Fatalf("unexpected actions passed to execute: %+v", gotActions)
	}
	if gotOpts == nil {
		t.Fatalf("expected execute options")
	}
	if gotOpts.Validate {
		t.Fatalf("expected execute validation disabled because Run validates first")
	}
	if gotOpts.ValidationProfile != browseractions.ValidationProfileLenient {
		t.Fatalf("unexpected validation profile: %q", gotOpts.ValidationProfile)
	}
	if !gotOpts.IncludeScreenshotData {
		t.Fatalf("expected screenshot inline flag to propagate")
	}
	if gotOpts.WriteFiles {
		t.Fatalf("expected WriteFiles=false to propagate")
	}
	if gotOpts.OutputDir != "artifacts" {
		t.Fatalf("unexpected OutputDir: %q", gotOpts.OutputDir)
	}
}

func TestRunRecoversFromExecutePanic(t *testing.T) {
	origOpenPage := runOpenPage
	origExecute := runExecute
	t.Cleanup(func() {
		runOpenPage = origOpenPage
		runExecute = origExecute
	})

	runOpenPage = func(opts RunOptions) (*rod.Page, func(), error) {
		return nil, func() {}, nil
	}
	runExecute = func(ctx context.Context, page *rod.Page, actions []browseractions.Action, opts *ExecuteOptions) (ExecutionResult, error) {
		panic("boom")
	}

	_, err := Run(context.Background(), browseractions.BrowserActions{
		Title: "ok",
		Actions: []browseractions.Action{
			{Type: browseractions.ActionSleep, Duration: 1},
		},
	}, &RunOptions{ExecuteOptions: &ExecuteOptions{Validate: true}})
	if err == nil {
		t.Fatalf("expected panic recovery error")
	}
	if !strings.Contains(err.Error(), "rod runner panic: boom") {
		t.Fatalf("unexpected error: %v", err)
	}
}
