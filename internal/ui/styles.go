package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	Green  = lipgloss.Color("#00CC88")
	Red    = lipgloss.Color("#FF4444")
	Yellow = lipgloss.Color("#FFAA00")
	Blue   = lipgloss.Color("#00AAFF")
	Dim    = lipgloss.Color("#666666")
	White  = lipgloss.Color("#FFFFFF")

	// Adaptive colors (auto-adjust for light/dark terminals)
	SpinnerColor = lipgloss.AdaptiveColor{Light: "#555555", Dark: "#AAAAAA"}

	// Text styles
	Bold         = lipgloss.NewStyle().Bold(true)
	DimStyle     = lipgloss.NewStyle().Foreground(Dim)
	SpinnerStyle = lipgloss.NewStyle().Foreground(SpinnerColor)
	Label        = lipgloss.NewStyle().Foreground(Blue).Bold(true)
	Success   = lipgloss.NewStyle().Foreground(Green)
	Error     = lipgloss.NewStyle().Foreground(Red)
	Warning   = lipgloss.NewStyle().Foreground(Yellow)
	InfoStyle = lipgloss.NewStyle().Foreground(Blue)

	// Status banners
	SuccessBanner = lipgloss.NewStyle().
			Foreground(White).
			Background(Green).
			Bold(true).
			Padding(0, 1)

	ErrorBanner = lipgloss.NewStyle().
			Foreground(White).
			Background(Red).
			Bold(true).
			Padding(0, 1)

	WarningBanner = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(Yellow).
			Bold(true).
			Padding(0, 1)

	// Table styles
	TableHeader = lipgloss.NewStyle().Bold(true).Foreground(Blue)
	TableCell   = lipgloss.NewStyle().PaddingRight(2)
)
