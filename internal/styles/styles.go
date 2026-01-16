package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	Red    = lipgloss.Color("#FF5555")
	Green  = lipgloss.Color("#50FA7B")
	Yellow = lipgloss.Color("#F1FA8C")
	Blue   = lipgloss.Color("#8BE9FD")
	Purple = lipgloss.Color("#BD93F9")

	// Styles
	ErrorStyle = lipgloss.NewStyle().
			Foreground(Red).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(Green)

	InfoStyle = lipgloss.NewStyle().
			Foreground(Yellow)

	HeaderStyle = lipgloss.NewStyle().
			Foreground(Blue).
			Bold(true)

	DimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4"))
)

func Error(msg string) string {
	return ErrorStyle.Render("Error: " + msg)
}

func Success(msg string) string {
	return SuccessStyle.Render(msg)
}

func Info(msg string) string {
	return InfoStyle.Render(msg)
}

func Header(msg string) string {
	return HeaderStyle.Render("=== " + msg + " ===")
}

func Dim(msg string) string {
	return DimStyle.Render(msg)
}
