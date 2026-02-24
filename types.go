package browseractions

type ActionType string
type WaitCondition string
type AssertCondition string
type ScrollPosition string
type InputFormat string

const (
	ActionNavigate   ActionType = "navigate"
	ActionClick      ActionType = "click"
	ActionFill       ActionType = "fill"
	ActionWait       ActionType = "wait"
	ActionAssert     ActionType = "assert"
	ActionScroll     ActionType = "scroll"
	ActionScreenshot ActionType = "screenshot"
	ActionSleep      ActionType = "sleep"
	ActionEvaluate   ActionType = "evaluate"

	WaitVisible WaitCondition = "visible"
	WaitHidden  WaitCondition = "hidden"
	WaitEnabled WaitCondition = "enabled"
	WaitLoad    WaitCondition = "load"

	AssertContains AssertCondition = "contains"
	AssertEquals   AssertCondition = "equals"
	AssertVisible  AssertCondition = "visible"
	AssertHidden   AssertCondition = "hidden"

	ScrollTop    ScrollPosition = "top"
	ScrollBottom ScrollPosition = "bottom"

	InputFormatAuto InputFormat = "auto"
	InputFormatJSON InputFormat = "json"
	InputFormatYAML InputFormat = "yaml"
)

type Action struct {
	Type       ActionType      `yaml:"type" json:"type"`
	Selector   string          `yaml:"selector,omitempty" json:"selector,omitempty"`
	Value      string          `yaml:"value,omitempty" json:"value,omitempty"`
	URL        string          `yaml:"url,omitempty" json:"url,omitempty"`
	For        WaitCondition   `yaml:"for,omitempty" json:"for,omitempty"`
	Condition  AssertCondition `yaml:"condition,omitempty" json:"condition,omitempty"`
	Position   ScrollPosition  `yaml:"position,omitempty" json:"position,omitempty"`
	File       string          `yaml:"file,omitempty" json:"file,omitempty"`
	Duration   int             `yaml:"duration,omitempty" json:"duration,omitempty"`
	Expression string          `yaml:"expression,omitempty" json:"expression,omitempty"`
}

type BrowserActions struct {
	Title   string   `yaml:"title" json:"title"`
	Actions []Action `yaml:"actions" json:"actions"`
}

type ParseOptions struct {
	Format     InputFormat
	ArrayTitle string
}
