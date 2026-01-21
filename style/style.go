package style

// ANSI escape codes for terminal styling
const (
	reset     = "\033[0m"
	bold      = "\033[1m"
	dim       = "\033[2m"
	italic    = "\033[3m"
	underline = "\033[4m"

	// Foreground colors
	black   = "\033[30m"
	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	blue    = "\033[34m"
	magenta = "\033[35m"
	cyan    = "\033[36m"
	white   = "\033[37m"
	gray    = "\033[90m"

	// Background colors
	bgBlack   = "\033[40m"
	bgRed     = "\033[41m"
	bgGreen   = "\033[42m"
	bgYellow  = "\033[43m"
	bgBlue    = "\033[44m"
	bgMagenta = "\033[45m"
	bgCyan    = "\033[46m"
	bgWhite   = "\033[47m"
)

// Style represents a combination of ANSI styling codes
type Style struct {
	codes string
}

// Pre-computed Style variables
var (
	Bold      = Style{codes: bold}
	Dim       = Style{codes: dim}
	Italic    = Style{codes: italic}
	Underline = Style{codes: underline}

	Black   = Style{codes: black}
	Red     = Style{codes: red}
	Green   = Style{codes: green}
	Yellow  = Style{codes: yellow}
	Blue    = Style{codes: blue}
	Magenta = Style{codes: magenta}
	Cyan    = Style{codes: cyan}
	White   = Style{codes: white}
	Gray    = Style{codes: gray}

	BoldBlack   = Style{codes: bold + black}
	BoldRed     = Style{codes: bold + red}
	BoldGreen   = Style{codes: bold + green}
	BoldYellow  = Style{codes: bold + yellow}
	BoldBlue    = Style{codes: bold + blue}
	BoldMagenta = Style{codes: bold + magenta}
	BoldCyan    = Style{codes: bold + cyan}
	BoldWhite   = Style{codes: bold + white}

	DimGray = Style{codes: dim + gray}
)

// New creates a new Style with the given codes
func New(codes ...string) Style {
	combined := ""
	for _, code := range codes {
		combined += code
	}
	return Style{codes: combined}
}

// Apply applies the style to the given text
func (s Style) Apply(text string) string {
	if s.codes == "" {
		return text
	}
	return s.codes + text + reset
}

// String returns the styled text (for use with variadic args)
func (s Style) String() string {
	return s.codes
}

// Builder pattern for dynamic styling
type Builder struct {
	codes string
}

// NewBuilder creates a new style builder
func NewBuilder() *Builder {
	return &Builder{}
}

// Bold adds bold styling
func (b *Builder) Bold() *Builder {
	b.codes += bold
	return b
}

// Dim adds dim styling
func (b *Builder) Dim() *Builder {
	b.codes += dim
	return b
}

// Italic adds italic styling
func (b *Builder) Italic() *Builder {
	b.codes += italic
	return b
}

// Underline adds underline styling
func (b *Builder) Underline() *Builder {
	b.codes += underline
	return b
}

// Color adds a foreground color
func (b *Builder) Color(color string) *Builder {
	colorMap := map[string]string{
		"black":   black,
		"red":     red,
		"green":   green,
		"yellow":  yellow,
		"blue":    blue,
		"magenta": magenta,
		"cyan":    cyan,
		"white":   white,
		"gray":    gray,
	}
	if c, ok := colorMap[color]; ok {
		b.codes += c
	}
	return b
}

// Background adds a background color
func (b *Builder) Background(color string) *Builder {
	colorMap := map[string]string{
		"black":   bgBlack,
		"red":     bgRed,
		"green":   bgGreen,
		"yellow":  bgYellow,
		"blue":    bgBlue,
		"magenta": bgMagenta,
		"cyan":    bgCyan,
		"white":   bgWhite,
	}
	if c, ok := colorMap[color]; ok {
		b.codes += c
	}
	return b
}

// Build returns the constructed Style
func (b *Builder) Build() Style {
	return Style{codes: b.codes}
}

// Apply applies the built style to text
func (b *Builder) Apply(text string) string {
	return b.Build().Apply(text)
}
