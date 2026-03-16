package style

import (
	"app/color"

	"charm.land/lipgloss/v2"
)

var (
	Primary = lipgloss.NewStyle().Foreground(lipgloss.Color(color.Primary.Hex()))
	Header  = Primary.Bold(true)
	Running = Primary.Italic(true)

	Divider = lipgloss.NewStyle().Foreground(lipgloss.Color(color.Border.Hex()))
	Muted   = lipgloss.NewStyle().Foreground(lipgloss.Color(color.Muted.Hex()))
	Text    = lipgloss.NewStyle().Foreground(lipgloss.Color(color.Text.Hex()))

	SuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(color.Green.Hex()))
	ErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(color.Error.Hex()))
	WarningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(color.Yellow.Hex()))
	InfoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(color.Blue.Hex()))
	DebugStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(color.Muted.Hex()))
	DangerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color(color.Danger.Hex()))

	ErrorLabel   = ErrorStyle.Bold(true)
	WarningLabel = WarningStyle.Bold(true)
	SuccessLabel = SuccessStyle.Bold(true)
	DebugLabel   = DebugStyle.Bold(true)
	InfoLabel    = InfoStyle.Bold(true)
	DangerLabel  = DangerStyle.Bold(true)

	ErrorText   = lipgloss.NewStyle().Foreground(lipgloss.Color(color.ErrorLight.Hex()))
	WarningText = lipgloss.NewStyle().Foreground(lipgloss.Color(color.YellowLight.Hex()))
	SuccessText = lipgloss.NewStyle().Foreground(lipgloss.Color(color.GreenLight.Hex()))
)
