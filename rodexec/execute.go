package rodexec

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	browseractions "github.com/pyneda/browser-actions"
)

func Execute(ctx context.Context, page *rod.Page, actions []browseractions.Action, opts *ExecuteOptions) (ExecutionResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	o := opts.withDefaults()

	result := ExecutionResult{
		TotalActions: len(actions),
	}
	start := time.Now()
	defer func() {
		result.Duration = time.Since(start)
	}()

	if o.Validate {
		if err := browseractions.ValidateActions(actions, o.ValidationProfile); err != nil {
			logf(&result, o.LoggerHook, LogLevelError, "validation failed: %v", err)
			return result, err
		}
	}

	for i, action := range actions {
		select {
		case <-ctx.Done():
			logf(&result, o.LoggerHook, LogLevelInfo, "context cancelled")
			result.Failure = &Failure{
				ActionIndex: i,
				ActionType:  action.Type,
				Message:     ctx.Err().Error(),
			}
			return result, ctx.Err()
		default:
		}

		if err := executeAction(ctx, page, action, i, &result, o); err != nil {
			if result.Failure == nil {
				result.Failure = &Failure{
					ActionIndex: i,
					ActionType:  action.Type,
					Message:     err.Error(),
				}
			}
			if o.ContinueOnError {
				logf(&result, o.LoggerHook, LogLevelWarn, "continuing after action %d (%s) failure: %v", i, action.Type, err)
				continue
			}
			return result, err
		}
		result.CompletedActions++
	}

	result.Succeeded = true
	logf(&result, o.LoggerHook, LogLevelInfo, "all actions completed successfully")
	return result, nil
}

func executeAction(ctx context.Context, page *rod.Page, action browseractions.Action, idx int, result *ExecutionResult, opts ExecuteOptions) error {
	switch action.Type {
	case browseractions.ActionNavigate:
		if err := page.Navigate(action.URL); err != nil {
			logf(result, opts.LoggerHook, LogLevelError, "failed to navigate to %s: %s", action.URL, err)
			return actionErr(idx, action.Type, fmt.Sprintf("failed to navigate to %s", action.URL), err)
		}
		logf(result, opts.LoggerHook, LogLevelInfo, "navigated to %s", action.URL)
		if err := page.WaitLoad(); err != nil {
			logf(result, opts.LoggerHook, LogLevelError, "failed to wait for page load: %s", err)
			return actionErr(idx, action.Type, "failed to wait for page load", err)
		}
		logf(result, opts.LoggerHook, LogLevelInfo, "page loaded")
		return nil

	case browseractions.ActionWait:
		if action.For == browseractions.WaitLoad {
			if err := page.WaitLoad(); err != nil {
				logf(result, opts.LoggerHook, LogLevelError, "failed to wait for page load: %s", err)
				return actionErr(idx, action.Type, "failed to wait for page load", err)
			}
			logf(result, opts.LoggerHook, LogLevelInfo, "waited for page load")
			return nil
		}

		el, err := page.Element(action.Selector)
		if err != nil {
			logf(result, opts.LoggerHook, LogLevelError, "failed to find element %s: %s", action.Selector, err)
			return actionErr(idx, action.Type, fmt.Sprintf("failed to find element %s", action.Selector), err)
		}

		switch action.For {
		case browseractions.WaitVisible:
			err = el.WaitVisible()
		case browseractions.WaitHidden:
			err = el.WaitInvisible()
		case browseractions.WaitEnabled:
			err = el.WaitEnabled()
		default:
			err = nil
		}
		if err != nil {
			logf(result, opts.LoggerHook, LogLevelError, "failed to wait for element %s: %s", action.Selector, err)
			return actionErr(idx, action.Type, fmt.Sprintf("failed to wait for element %s", action.Selector), err)
		}
		logf(result, opts.LoggerHook, LogLevelInfo, "waited for element %s", action.Selector)
		return nil

	case browseractions.ActionClick:
		el, err := page.Element(action.Selector)
		if err != nil {
			logf(result, opts.LoggerHook, LogLevelError, "failed to find element %s: %s", action.Selector, err)
			return actionErr(idx, action.Type, fmt.Sprintf("failed to find element %s", action.Selector), err)
		}
		logf(result, opts.LoggerHook, LogLevelInfo, "clicking element %s", action.Selector)
		if err := el.Click(proto.InputMouseButtonLeft, 1); err != nil {
			logf(result, opts.LoggerHook, LogLevelError, "failed to click element %s: %s", action.Selector, err)
			return actionErr(idx, action.Type, fmt.Sprintf("failed to click element %s", action.Selector), err)
		}
		logf(result, opts.LoggerHook, LogLevelInfo, "clicked element %s", action.Selector)
		return nil

	case browseractions.ActionFill:
		el, err := page.Element(action.Selector)
		if err != nil {
			logf(result, opts.LoggerHook, LogLevelError, "failed to find element %s: %s", action.Selector, err)
			return actionErr(idx, action.Type, fmt.Sprintf("failed to find element %s", action.Selector), err)
		}
		logf(result, opts.LoggerHook, LogLevelInfo, "filling element %s", action.Selector)
		if err := el.Input(action.Value); err != nil {
			logf(result, opts.LoggerHook, LogLevelError, "failed to fill element %s: %s", action.Selector, err)
			return actionErr(idx, action.Type, fmt.Sprintf("failed to fill element %s", action.Selector), err)
		}
		logf(result, opts.LoggerHook, LogLevelInfo, "filled element %s", action.Selector)
		return nil

	case browseractions.ActionAssert:
		el, err := page.Element(action.Selector)
		if err != nil {
			logf(result, opts.LoggerHook, LogLevelError, "failed to find element %s: %s", action.Selector, err)
			return actionErr(idx, action.Type, fmt.Sprintf("failed to find element %s", action.Selector), err)
		}
		logf(result, opts.LoggerHook, LogLevelInfo, "asserting element %s", action.Selector)

		switch action.Condition {
		case browseractions.AssertVisible, browseractions.AssertHidden:
			isVisible, err := el.Visible()
			if err != nil {
				logf(result, opts.LoggerHook, LogLevelError, "failed to check visibility of element %s: %s", action.Selector, err)
				return actionErr(idx, action.Type, fmt.Sprintf("failed to check visibility of element %s", action.Selector), err)
			}
			if action.Condition == browseractions.AssertVisible && !isVisible {
				msg := fmt.Sprintf("assertion failed: element %s is not visible", action.Selector)
				logf(result, opts.LoggerHook, LogLevelError, msg)
				return actionErr(idx, action.Type, msg, nil)
			}
			if action.Condition == browseractions.AssertHidden && isVisible {
				msg := fmt.Sprintf("assertion failed: element %s is visible, expected hidden", action.Selector)
				logf(result, opts.LoggerHook, LogLevelError, msg)
				return actionErr(idx, action.Type, msg, nil)
			}
			logf(result, opts.LoggerHook, LogLevelInfo, "assertion passed for visibility on %s", action.Selector)
			return nil
		}

		text, err := el.Text()
		if err != nil {
			logf(result, opts.LoggerHook, LogLevelError, "failed to get text of element %s: %s", action.Selector, err)
			return actionErr(idx, action.Type, fmt.Sprintf("failed to get text of element %s", action.Selector), err)
		}
		switch action.Condition {
		case browseractions.AssertContains:
			if !strings.Contains(text, action.Value) {
				msg := fmt.Sprintf("assertion failed: element text does not contain '%s'", action.Value)
				logf(result, opts.LoggerHook, LogLevelError, msg)
				return actionErr(idx, action.Type, msg, nil)
			}
		case browseractions.AssertEquals:
			if text != action.Value {
				msg := fmt.Sprintf("assertion failed: element text is not equal to '%s'", action.Value)
				logf(result, opts.LoggerHook, LogLevelError, msg)
				return actionErr(idx, action.Type, msg, nil)
			}
		default:
			// no-op for unknown condition when validation is disabled
		}
		logf(result, opts.LoggerHook, LogLevelInfo, "assertion passed for element %s", action.Selector)
		return nil

	case browseractions.ActionScroll:
		if action.Selector == "" {
			logf(result, opts.LoggerHook, LogLevelInfo, "scrolling page to %s", defaultScrollPosition(action.Position))
			if err := scrollPage(page, action.Position); err != nil {
				logf(result, opts.LoggerHook, LogLevelError, "failed to scroll page: %s", err)
				return actionErr(idx, action.Type, "failed to scroll page", err)
			}
			logf(result, opts.LoggerHook, LogLevelInfo, "scrolled page to %s", defaultScrollPosition(action.Position))
			return nil
		}

		el, err := page.Element(action.Selector)
		logf(result, opts.LoggerHook, LogLevelInfo, "scrolling element %s into view", action.Selector)
		if err != nil {
			logf(result, opts.LoggerHook, LogLevelError, "failed to find element %s: %s", action.Selector, err)
			return actionErr(idx, action.Type, fmt.Sprintf("failed to find element %s", action.Selector), err)
		}
		if err := scrollElementIntoView(el, action.Position); err != nil {
			logf(result, opts.LoggerHook, LogLevelError, "failed to scroll element %s into view: %s", action.Selector, err)
			return actionErr(idx, action.Type, fmt.Sprintf("failed to scroll element %s into view", action.Selector), err)
		}
		logf(result, opts.LoggerHook, LogLevelInfo, "scrolled element %s into view", action.Selector)
		return nil

	case browseractions.ActionScreenshot:
		sr, err := takeScreenshot(page, action, opts)
		if err != nil {
			logf(result, opts.LoggerHook, LogLevelError, "failed to take screenshot: %s", err)
			return actionErr(idx, action.Type, "failed to take screenshot", err)
		}
		result.Screenshots = append(result.Screenshots, sr)
		if action.Selector != "" {
			logf(result, opts.LoggerHook, LogLevelInfo, "took screenshot of element %s", action.Selector)
		} else {
			logf(result, opts.LoggerHook, LogLevelInfo, "took screenshot of page")
		}
		return nil

	case browseractions.ActionSleep:
		logf(result, opts.LoggerHook, LogLevelInfo, "sleeping for %d milliseconds", action.Duration)
		select {
		case <-time.After(time.Duration(action.Duration) * time.Millisecond):
		case <-ctx.Done():
			logf(result, opts.LoggerHook, LogLevelInfo, "sleep cancelled")
			return ctx.Err()
		}
		logf(result, opts.LoggerHook, LogLevelInfo, "slept for %d milliseconds", action.Duration)
		return nil

	case browseractions.ActionEvaluate:
		logf(result, opts.LoggerHook, LogLevelInfo, "evaluating JavaScript")
		evalResult, err := page.Eval(action.Expression)
		if err != nil {
			logf(result, opts.LoggerHook, LogLevelError, "failed to evaluate JavaScript: %s", err)
			return actionErr(idx, action.Type, "error evaluating JavaScript", err)
		}
		value := ""
		if evalResult != nil {
			value = evalResult.Value.String()
		}
		result.Evaluations = append(result.Evaluations, EvaluationResult{
			Expression: action.Expression,
			Value:      value,
		})
		logf(result, opts.LoggerHook, LogLevelInfo, "evaluated JavaScript")
		return nil

	default:
		msg := fmt.Sprintf("unsupported action type %q", action.Type)
		logf(result, opts.LoggerHook, LogLevelError, msg)
		return actionErr(idx, action.Type, msg, nil)
	}
}

func takeScreenshot(page *rod.Page, action browseractions.Action, opts ExecuteOptions) (ScreenshotResult, error) {
	var (
		data []byte
		err  error
	)
	if action.Selector != "" {
		el, elErr := page.Element(action.Selector)
		if elErr != nil {
			return ScreenshotResult{}, fmt.Errorf("failed to find element %s: %w", action.Selector, elErr)
		}
		data, err = el.Screenshot(proto.PageCaptureScreenshotFormatPng, 90)
	} else {
		data, err = page.Screenshot(true, nil)
	}
	if err != nil {
		return ScreenshotResult{}, err
	}

	outFile := action.File
	if strings.TrimSpace(action.File) != "" && opts.WriteFiles {
		resolved := resolveOutputPath(opts.OutputDir, action.File)
		if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
			return ScreenshotResult{}, fmt.Errorf("failed to create screenshot directory: %w", err)
		}
		if err := os.WriteFile(resolved, data, 0o644); err != nil {
			return ScreenshotResult{}, fmt.Errorf("failed to save screenshot to %s: %w", resolved, err)
		}
		outFile = resolved
	}

	sr := ScreenshotResult{
		Selector:   action.Selector,
		OutputFile: outFile,
	}
	if opts.IncludeScreenshotData {
		sr.Data = base64.StdEncoding.EncodeToString(data)
	}
	return sr, nil
}

func resolveOutputPath(outputDir, file string) string {
	if file == "" {
		return ""
	}
	if filepath.IsAbs(file) || outputDir == "" {
		return file
	}
	return filepath.Join(outputDir, file)
}

func scrollElementIntoView(el *rod.Element, position browseractions.ScrollPosition) error {
	block := "start"
	if position == browseractions.ScrollBottom {
		block = "end"
	}
	if _, err := el.Eval(fmt.Sprintf(`() => this.scrollIntoView({behavior:"auto", block:"%s", inline:"nearest"})`, block)); err == nil {
		return nil
	}
	return el.ScrollIntoView()
}

func scrollPage(page *rod.Page, position browseractions.ScrollPosition) error {
	switch defaultScrollPosition(position) {
	case browseractions.ScrollBottom:
		_, err := page.Eval(`() => {
			const h = Math.max(
				document.body ? document.body.scrollHeight : 0,
				document.documentElement ? document.documentElement.scrollHeight : 0
			);
			window.scrollTo(0, h);
			return window.scrollY;
		}`)
		return err
	default:
		_, err := page.Eval(`() => { window.scrollTo(0, 0); return window.scrollY; }`)
		return err
	}
}

func defaultScrollPosition(position browseractions.ScrollPosition) browseractions.ScrollPosition {
	if position == browseractions.ScrollBottom {
		return browseractions.ScrollBottom
	}
	return browseractions.ScrollTop
}

func actionErr(index int, actionType browseractions.ActionType, msg string, cause error) error {
	prefix := fmt.Sprintf("action %d (%s): %s", index, actionType, msg)
	if cause != nil {
		return fmt.Errorf("%s: %w", prefix, cause)
	}
	return fmt.Errorf("%s", prefix)
}
