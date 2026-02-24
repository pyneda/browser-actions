package rodexec

import (
	"fmt"
	"time"

	browseractions "github.com/pyneda/browser-actions"
)

type LogLevel string

const (
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
)

type LogEntry struct {
	Level     LogLevel  `json:"level"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
}

type ScreenshotResult struct {
	Selector   string `json:"selector"`
	Data       string `json:"data,omitempty"`
	OutputFile string `json:"output_file,omitempty"`
}

type EvaluationResult struct {
	Expression string `json:"expression"`
	Value      string `json:"value,omitempty"`
}

type Failure struct {
	ActionIndex int                       `json:"action_index"`
	ActionType  browseractions.ActionType `json:"action_type"`
	Message     string                    `json:"message"`
}

type ExecutionResult struct {
	Succeeded        bool               `json:"succeeded"`
	TotalActions     int                `json:"total_actions"`
	CompletedActions int                `json:"completed_actions"`
	Screenshots      []ScreenshotResult `json:"screenshots,omitempty"`
	Evaluations      []EvaluationResult `json:"evaluations,omitempty"`
	Logs             []LogEntry         `json:"logs,omitempty"`
	Failure          *Failure           `json:"failure,omitempty"`
	Duration         time.Duration      `json:"duration"`
}

type ExecuteOptions struct {
	Validate              bool
	ValidationProfile     browseractions.ValidationProfile
	IncludeScreenshotData bool
	WriteFiles            bool
	OutputDir             string
	LoggerHook            func(LogEntry)
	ContinueOnError       bool
}

type Viewport struct {
	Width  int
	Height int
}

type RunOptions struct {
	Headless       bool
	Timeout        time.Duration
	Proxy          string
	BinaryPath     string
	Viewport       *Viewport
	ExecuteOptions *ExecuteOptions
}

func (o *ExecuteOptions) withDefaults() ExecuteOptions {
	if o == nil {
		return ExecuteOptions{
			ValidationProfile: browseractions.ValidationProfileStrict,
		}
	}
	out := *o
	if out.ValidationProfile == "" {
		out.ValidationProfile = browseractions.ValidationProfileStrict
	}
	return out
}

func (o *RunOptions) withDefaults() RunOptions {
	if o == nil {
		return RunOptions{
			Headless: true,
			Timeout:  30 * time.Second,
		}
	}
	out := *o
	if out.Timeout <= 0 {
		out.Timeout = 30 * time.Second
	}
	return out
}

func (r *ExecutionResult) appendLog(entry LogEntry, hook func(LogEntry)) {
	r.Logs = append(r.Logs, entry)
	if hook != nil {
		hook(entry)
	}
}

func logf(result *ExecutionResult, hook func(LogEntry), level LogLevel, format string, args ...any) {
	result.appendLog(LogEntry{
		Level:     level,
		Text:      formatText(format, args...),
		Timestamp: time.Now(),
	}, hook)
}

func formatText(format string, args ...any) string {
	if len(args) == 0 {
		return format
	}
	return fmtSprintf(format, args...)
}

var fmtSprintf = func(format string, args ...any) string { return fmt.Sprintf(format, args...) }
