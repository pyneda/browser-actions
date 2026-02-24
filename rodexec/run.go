package rodexec

import (
	"context"
	"fmt"

	"github.com/go-rod/rod"
	browseractions "github.com/pyneda/browser-actions"
)

var runOpenPage = defaultRunOpenPage
var runExecute = Execute

func Run(ctx context.Context, script browseractions.BrowserActions, opts *RunOptions) (result ExecutionResult, err error) {
	o := opts.withDefaults()
	result.TotalActions = len(script.Actions)

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("rod runner panic: %v", r)
		}
	}()

	if ctx == nil {
		ctx = context.Background()
	}
	if o.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, o.Timeout)
		defer cancel()
	}

	if o.Validate {
		if err := browseractions.ValidateScript(script, o.ValidationProfile); err != nil {
			return result, err
		}
	}

	page, cleanup, err := runOpenPage(o)
	if err != nil {
		return result, err
	}
	defer cleanup()

	execOpts := &ExecuteOptions{
		Validate:              false, // already validated above when enabled
		ValidationProfile:     o.ValidationProfile,
		IncludeScreenshotData: o.IncludeScreenshotData,
		WriteFiles:            o.WriteFiles,
		OutputDir:             o.OutputDir,
	}

	return runExecute(ctx, page, script.Actions, execOpts)
}

func defaultRunOpenPage(opts RunOptions) (*rod.Page, func(), error) {
	controlURL, err := newLauncher(opts).Launch()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	browser := rod.New().ControlURL(controlURL).MustConnect()
	page := browser.MustPage("")
	applyViewport(page, opts.Viewport)

	cleanup := func() {
		page.MustClose()
		browser.MustClose()
	}
	return page, cleanup, nil
}
