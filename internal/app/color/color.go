package color

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	reset = "\033[0m"
	bold  = "\033[1m"
)

// Color palette
var (
	Primary     = Purple
	Purple      = NewColor("#a78bfa")
	Blue        = NewColor("#60a5fa")
	Green       = NewColor("#34d399")
	GreenLight  = NewColor("#6ee7b7")
	Yellow      = NewColor("#f5a623")
	YellowLight = NewColor("#f7c86a")
	Amber       = NewColor("#fbbf24")
	Error       = NewColor("#f87171")
	Danger      = NewColor("#e63570")
	ErrorLight  = NewColor("#fca5a5")
	Muted       = NewColor("#6b7280")
	Border      = NewColor("#374151")
	Text        = NewColor("#e5e7eb")
)

// Color represents an RGB color
type Color struct {
	R, G, B uint8
	hex     string
}

// NewColor creates a Color from a hex string (e.g., "#a78bfa" or "a78bfa")
func NewColor(hexColor string) Color {
	hexColor = strings.ToLower(strings.TrimPrefix(hexColor, "#"))
	if len(hexColor) != 6 {
		return Color{R: 0, G: 0, B: 0, hex: "#000000"}
	}

	r, _ := strconv.ParseInt(hexColor[0:2], 16, 32)
	g, _ := strconv.ParseInt(hexColor[2:4], 16, 32)
	b, _ := strconv.ParseInt(hexColor[4:6], 16, 32)

	return Color{
		R:   uint8(r),
		G:   uint8(g),
		B:   uint8(b),
		hex: "#" + hexColor,
	}
}

// NewColorRGB creates a Color from RGB values
func NewColorRGB(r, g, b uint8) Color {
	return Color{
		R:   r,
		G:   g,
		B:   b,
		hex: fmt.Sprintf("#%02X%02X%02X", r, g, b),
	}
}

// Hex returns the hex representation of the color (e.g., "#a78bfa")
func (c Color) Hex() string {
	return c.hex
}

// ToRGB returns the RGB values as separate integers
func (c Color) ToRGB() (uint8, uint8, uint8) {
	return c.R, c.G, c.B
}

// ToANSI returns the ANSI escape code for the color
func (c Color) ToANSI() string {
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", c.R, c.G, c.B)
}

// RGB creates a string with an RGB color applied
func RGB(color, text string) string {
	return color + text + reset
}

// BoldText creates a bolded string
func BoldText(text string) string {
	return bold + text + reset
}
