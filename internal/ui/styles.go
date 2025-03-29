package ui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Vibrant color palette
	colorPrimary    = lipgloss.Color("#FF5F87") // Vibrant pink
	colorSecondary  = lipgloss.Color("#5FAFFF") // Bright blue
	colorAccent     = lipgloss.Color("#FFAF00") // Bold orange
	colorSuccess    = lipgloss.Color("#50FA7B") // Neon green
	colorDanger     = lipgloss.Color("#FF5555") // Bright red
	colorWarning    = lipgloss.Color("#F1FA8C") // Vibrant yellow
	colorBackground = lipgloss.Color("#282A36") // Dark background
	colorText       = lipgloss.Color("#F8F8F2") // Light text
	colorSubtext    = lipgloss.Color("#BFBFBF") // Grey text
	colorHighlight  = lipgloss.Color("#BD93F9") // Purple highlight

	// Updated styles with more vibrant colors
	titleStyle       = lipgloss.NewStyle().Foreground(colorHighlight).Bold(true).Padding(1, 2)
	selectedStyle    = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true).PaddingLeft(2)
	unselectedStyle  = lipgloss.NewStyle().Foreground(colorText).PaddingLeft(2)
	logStyle         = lipgloss.NewStyle().Padding(1, 2).Background(colorBackground).Foreground(colorText)
	instructionStyle = lipgloss.NewStyle().Foreground(colorSubtext).Padding(1, 2)
	debugStyle       = lipgloss.NewStyle().Foreground(colorDanger)

	// Redesigned UI components with vibrant borders and backgrounds
	headerStyle     = lipgloss.NewStyle().Foreground(colorText).Background(colorPrimary).Bold(true).Padding(0, 1).Width(100)
	stackPanelStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorSecondary).Padding(1, 2).Background(colorBackground)
	containerStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorSuccess).Padding(1, 2).Background(colorBackground)
	logPanelStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorAccent).Padding(1, 2).Background(colorBackground)
	helpPanelStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorHighlight).Padding(1, 2).Background(colorBackground)
	actionMenuStyle = lipgloss.NewStyle().Padding(1, 2).Border(lipgloss.RoundedBorder()).BorderForeground(colorPrimary).Background(colorBackground).Foreground(colorText)

	// Status indicators
	statusRunning = lipgloss.NewStyle().Foreground(colorSuccess).Bold(true)
	statusStopped = lipgloss.NewStyle().Foreground(colorDanger).Bold(true)
	statusOther   = lipgloss.NewStyle().Foreground(colorWarning).Bold(true)
)
