package component

import (
	"app/color"
	"fmt"
	"math"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

type LoaderMode int

const (
	LoaderBounce LoaderMode = iota
	LoaderWrap
)

// Loader represents an animated loader bar with configurable mode, color, and behavior
type Loader struct {
	pos       float64
	vel       float64
	mode      LoaderMode
	baseColor color.Color
	glowColor color.Color
	radius    float64
	lastFrame time.Time
}

// NewLoader creates a new loader with specified velocity, mode, and colors
func NewLoader(velocity float64, mode LoaderMode) *Loader {

	base, glow := generateLoaderColors(color.Primary.Hex())

	return &Loader{
		pos:       0,
		vel:       velocity,
		mode:      mode,
		baseColor: base,
		glowColor: glow,
		radius:    12.0,
		lastFrame: time.Now(),
	}
}

// Update updates the loader position based on delta time
func (l *Loader) Update(dt float64, width int) {
	if width == 0 {
		return
	}

	l.pos += l.vel * dt

	if l.mode == LoaderBounce {
		mx := float64(width - 1)

		if l.pos < 0 {
			l.pos = 0
			l.vel = math.Abs(l.vel)
		}
		if l.pos > mx {
			l.pos = mx
			l.vel = -math.Abs(l.vel)
		}
	} else {
		// Wrap mode - add extra space for glow radius
		mx := float64(width) + l.radius*2

		for l.pos < -l.radius {
			l.pos += mx
		}
		for l.pos >= float64(width)+l.radius {
			l.pos -= mx
		}
	}
}

// SetMode changes the loader mode
func (l *Loader) SetMode(mode LoaderMode) {
	l.mode = mode
}

// Render renders the loader bar
func (l *Loader) Render(width int) string {
	if width == 0 {
		return ""
	}

	var b strings.Builder
	b.Grow(width * 10)

	for i := 0; i < width; i++ {
		x := float64(i)
		dist := math.Abs(x - l.pos)

		// In wrap mode, check if loader is off-screen
		if l.mode == LoaderWrap {
			// Check wrapped distance (from the other side)
			wrapDist := math.Abs(x - (l.pos - float64(width) - l.radius*2))
			dist = math.Min(dist, wrapDist)
		}

		// Calculate glow intensity
		t := math.Max(0, 1-(dist/l.radius))
		t = smoothstep(t)

		// Interpolate between base and glow colors
		baseR, baseG, baseB := l.baseColor.ToRGB()
		glowR, glowG, glowB := l.glowColor.ToRGB()
		r := int(float64(baseR) + float64(glowR-baseR)*t)
		g := int(float64(baseG) + float64(glowG-baseG)*t)
		bl := int(float64(baseB) + float64(glowB-baseB)*t)

		c := lipgloss.Color(fmt.Sprintf("#%02X%02X%02X", r, g, bl))
		b.WriteString(lipgloss.NewStyle().Foreground(c).Render("─"))
	}

	return b.String()
}

func generateLoaderColors(hex string) (color.Color, color.Color) {
	base := adjustColor(color.NewColor(hex), 0.85) // slightly darker
	glow := adjustColor(color.NewColor(hex), 1.25) // brighter glow
	return base, glow
}

func adjustChannel(v uint8, factor float64) uint8 {
	newVal := float64(v) * factor
	if newVal > 255 {
		newVal = 255
	}
	if newVal < 0 {
		newVal = 0
	}
	return uint8(newVal)
}

func adjustColor(c color.Color, factor float64) color.Color {
	return color.Color{
		R: adjustChannel(c.R, factor),
		G: adjustChannel(c.G, factor),
		B: adjustChannel(c.B, factor),
	}
}

func smoothstep(t float64) float64 {
	return t * t * (3 - 2*t)
}
