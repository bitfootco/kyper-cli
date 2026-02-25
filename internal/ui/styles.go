package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Semantic signal colors — used only for ✓ ✗ ! status symbols
	colorSuccess = lipgloss.AdaptiveColor{Light: "#1A7F37", Dark: "#3FB950"}
	colorError   = lipgloss.AdaptiveColor{Light: "#CF222E", Dark: "#F85149"}
	colorWarning = lipgloss.AdaptiveColor{Light: "#9A6700", Dark: "#D29922"}
	colorDim     = lipgloss.AdaptiveColor{Light: "#57606A", Dark: "#8B949E"}

	SpinnerColor = lipgloss.AdaptiveColor{Light: "#555555", Dark: "#AAAAAA"}

	// Text styles
	Bold         = lipgloss.NewStyle().Bold(true)
	DimStyle     = lipgloss.NewStyle().Foreground(colorDim)
	SpinnerStyle = lipgloss.NewStyle().Foreground(SpinnerColor)

	// InfoStyle and Label are plain bold — no color
	InfoStyle   = lipgloss.NewStyle().Bold(true)
	Label       = lipgloss.NewStyle().Bold(true)
	TableHeader = lipgloss.NewStyle().Bold(true)
	TableCell   = lipgloss.NewStyle()

	// Status signal styles — color reserved for ✓ ✗ ! symbols and status text
	Success = lipgloss.NewStyle().Foreground(colorSuccess)
	Error   = lipgloss.NewStyle().Foreground(colorError)
	Warning = lipgloss.NewStyle().Foreground(colorWarning)

	// Build result banners — same color as signals, no background
	SuccessBanner = lipgloss.NewStyle().Foreground(colorSuccess).Bold(true)
	ErrorBanner   = lipgloss.NewStyle().Foreground(colorError).Bold(true)
	WarningBanner = lipgloss.NewStyle().Foreground(colorWarning).Bold(true)
)
