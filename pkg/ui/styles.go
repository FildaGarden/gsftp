package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Design System & Palette (Modern Cyber/Neon Dark Theme)
var (
	// Colors
	ColorPrimary   = lipgloss.Color("#7D56F4") // Vibrant Purple
	ColorSecondary = lipgloss.Color("#00F5D4") // Cyber Cyan / Teal
	ColorAccent    = lipgloss.Color("#FF007F") // Neon Pink
	ColorBgDark    = lipgloss.Color("#1A1B26") // Deep Night Dark
	ColorSurface   = lipgloss.Color("#24283B") // Surface Dark
	ColorText      = lipgloss.Color("#C0CAF5") // Soft Ice White
	ColorMuted     = lipgloss.Color("#565F89") // Muted Slate
	ColorSuccess   = lipgloss.Color("#9ECE6A") // Soft Lime
	ColorWarning   = lipgloss.Color("#E0AF68") // Warm Amber
	ColorDanger    = lipgloss.Color("#F7768E") // Soft Coral Red
	ColorSelected  = lipgloss.Color("#BB9AF7") // Soft Violet

	// Header Styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ColorPrimary).
			Padding(0, 1)

	BadgeLocalStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#111111")).
			Background(ColorSecondary).
			Padding(0, 1)

	BadgeRemoteStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ColorAccent).
			Padding(0, 1)

	// Dual-Pane Borders
	ActivePaneBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorSecondary).
				Padding(0)

	InactivePaneBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorMuted).
				Padding(0)

	// File List Row Styles
	CursorRowStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#333A56"))

	SelectedRowStyle = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true)

	DirItemStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary)

	SymlinkItemStyle = lipgloss.NewStyle().
				Italic(true).
				Foreground(ColorWarning)

	MutedTextStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// Dialog & Modal Styles
	ModalStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(ColorPrimary).
			Background(ColorBgDark).
			Padding(1, 2)

	// Progress Bar & Status Styles
	StatusSuccessStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess).
				Bold(true)

	StatusErrorStyle = lipgloss.NewStyle().
				Foreground(ColorDanger).
				Bold(true)

	// Keybindings Footer
	HelpKeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary)

	HelpDescStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)
)
