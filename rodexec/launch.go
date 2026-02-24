package rodexec

import (
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

func newLauncher(opts RunOptions) *launcher.Launcher {
	l := launcher.New().
		Headless(opts.Headless).
		Set("no-sandbox")

	if opts.Proxy != "" {
		l = l.Proxy(opts.Proxy)
	}
	if opts.BinaryPath != "" {
		l = l.Bin(opts.BinaryPath)
	}
	return l
}

func applyViewport(page *rod.Page, vp *Viewport) {
	if page == nil || vp == nil {
		return
	}
	if vp.Width <= 0 || vp.Height <= 0 {
		return
	}
	page.MustSetViewport(vp.Width, vp.Height, 1.0, false)
}
