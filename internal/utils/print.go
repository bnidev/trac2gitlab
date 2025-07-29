package utils

import (
    "fmt"

    "github.com/charmbracelet/lipgloss"
)

// printStyled is an unexported helper function that prints a styled icon and message
// with the specified color code. Used internally by the public print functions.
func printStyled(icon string, colorCode string, msg string) {
    iconStyled := lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color(colorCode)).
        Render(icon)

    textStyled := lipgloss.NewStyle().
        Foreground(lipgloss.Color(colorCode)).
        Render(msg)

    fmt.Println(iconStyled, textStyled)
}

// PrintSuccess prints a success message with a checkmark icon.
func PrintSuccess(msg string) {
    printStyled("✓", "10", msg)
}

// PrintError prints an error message with an 'x' icon.
func PrintError(msg string) {
    printStyled("x", "9", msg)
}

// PrintWarning prints a warning message with an '!' icon.
func PrintWarning(msg string) {
    printStyled("!", "11", msg)
}

// PrintInProgress prints a message indicating an ongoing process with a '>' icon.
func PrintInProgress(msg string) {
    printStyled(">", "14", msg)
}
