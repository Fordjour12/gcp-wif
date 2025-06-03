// Package ui provides shared styling for Bubble Tea UI components
package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Shared UI styles used across different components
var (
	TitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			MarginBottom(1)

	HelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))

	InputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF75B7")).
			Bold(true)

	ContinueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF7DB")).
			Background(lipgloss.Color("#FF5F87")).
			Padding(0, 1).
			MarginTop(1)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5F87")).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFB000")).
			Bold(true)

	SectionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true).
			MarginTop(1)

	ProgressStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4"))

	StatusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575"))
)
