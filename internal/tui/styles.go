package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Color palette per spec §9.2, adapted to light/dark backgrounds.
type palettes struct {
	success string
	error   string
	warning string
	info    string
	muted   string
	accent  string
}

var darkPalette = palettes{
	success: "#00D26A",
	error:   "#FF4757",
	warning: "#FFA502",
	info:    "#1E90FF",
	muted:   "#636E72",
	accent:  "#A855F7",
}

var lightPalette = palettes{
	success: "#00A854",
	error:   "#E63946",
	warning: "#E67E22",
	info:    "#2980B9",
	muted:   "#95A5A6",
	accent:  "#7C3AED",
}

// Styles holds the themed lipgloss styles used across the CLI.
type Styles struct {
	palette palettes

	Success lipgloss.Style
	Error   lipgloss.Style
	Warning lipgloss.Style
	Info    lipgloss.Style
	Muted   lipgloss.Style
	Accent  lipgloss.Style

	Header       lipgloss.Style
	SubHeader    lipgloss.Style
	Item         lipgloss.Style
	SelectedItem lipgloss.Style
	DimmedItem   lipgloss.Style
	Help         lipgloss.Style
	Divider      lipgloss.Style
	TableHeader  lipgloss.Style
	TableRow     lipgloss.Style
	Focus        lipgloss.Style
	ErrorBar     lipgloss.Style
}

// NewStyles builds the style set for the current terminal background.
func NewStyles() *Styles {
	p := darkPalette
	if lipgloss.HasDarkBackground() {
		p = darkPalette
	} else {
		p = lightPalette
	}

	muted := p.muted
	accent := p.accent

	return &Styles{
		palette: p,

		Success: lipgloss.NewStyle().Foreground(lipgloss.Color(p.success)).Bold(true),
		Error:   lipgloss.NewStyle().Foreground(lipgloss.Color(p.error)).Bold(true),
		Warning: lipgloss.NewStyle().Foreground(lipgloss.Color(p.warning)).Bold(true),
		Info:    lipgloss.NewStyle().Foreground(lipgloss.Color(p.info)),
		Muted:   lipgloss.NewStyle().Foreground(lipgloss.Color(muted)),
		Accent:  lipgloss.NewStyle().Foreground(lipgloss.Color(accent)).Bold(true),

		Header:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(accent)).MarginBottom(1),
		SubHeader:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(muted)).MarginBottom(1),
		Item:         lipgloss.NewStyle().PaddingLeft(2),
		SelectedItem: lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color(accent)).Bold(true),
		DimmedItem:   lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color(muted)),
		Help:         lipgloss.NewStyle().Foreground(lipgloss.Color(muted)).MarginTop(1),
		Divider:      lipgloss.NewStyle().Foreground(lipgloss.Color(muted)).Faint(true),
		TableHeader:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(accent)),
		TableRow:     lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")),
		Focus:        lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(accent)),
		ErrorBar:     lipgloss.NewStyle().Background(lipgloss.Color(p.error)).Foreground(lipgloss.Color("#FFFFFF")).Padding(0, 1),
	}
}

// ColorSeverity returns a styled ✓ / ⚠ / ✗ prefix for a level.
func (s *Styles) ColorSeverity(severity string) lipgloss.Style {
	switch severity {
	case "OK":
		return s.Success
	case "WARNING":
		return s.Warning
	default:
		return s.Error
	}
}
