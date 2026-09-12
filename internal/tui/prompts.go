package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// inputModel is a Bubble Tea model wrapping a text input.
type inputModel struct {
	title  string
	input  textinput.Model
	secret bool
	focus  bool
	result string
	cancel bool
}

func (m inputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m inputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			m.result = m.input.Value()
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.cancel = true
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m inputModel) View() string {
	if !m.focus {
		m.input.Focus()
		m.focus = true
	}
	if m.secret {
		m.input.EchoMode = textinput.EchoPassword
		m.input.EchoCharacter = '•'
	}
	title := NewStyles().Accent.Render("? ")
	title += NewStyles().Info.Render(m.title)
	return title + " : " + m.input.View() + "\n"
}

// askInput runs an interactive text input prompt.
func askInput(title, placeholder string, secret bool) (string, bool, error) {
	if !IsInteractive() {
		return "", true, RequireInteractive("La saisie")
	}
	s := NewStyles()
	input := textinput.New()
	input.Placeholder = placeholder
	input.CharLimit = 256
	input.Width = 48
	input.Prompt = ""
	input.PromptStyle = s.Accent
	input.PlaceholderStyle = s.Muted
	input.TextStyle = s.Info

	m := inputModel{title: title, input: input, secret: secret}
	p := tea.NewProgram(m)
	final, err := p.Run()
	if err != nil {
		return "", false, err
	}
	fm, ok := final.(inputModel)
	if !ok {
		return "", false, fmt.Errorf("prompt terminé de manière inattendue")
	}
	if fm.cancel {
		return "", true, nil
	}
	return fm.result, false, nil
}

// AskText collects one line of visible text.
func AskText(title, placeholder string) (string, error) {
	value, cancelled, err := askInput(title, placeholder, false)
	if err != nil {
		return "", err
	}
	if cancelled {
		return "", fmt.Errorf("opération annulée")
	}
	return value, nil
}

// AskSecret collects a masked secret.
func AskSecret(title string) (string, error) {
	value, cancelled, err := askInput(title, "••••••••", true)
	if err != nil {
		return "", err
	}
	if cancelled {
		return "", fmt.Errorf("opération annulée")
	}
	return value, nil
}

// selectItem adapts a label to the bubbles list item interface.
type selectItem struct {
	label string
}

func (i selectItem) Title() string       { return i.label }
func (i selectItem) Description() string { return "" }
func (i selectItem) FilterValue() string { return i.label }

// selectModel is a Bubble Tea model wrapping a bubbles list.
type selectModel struct {
	title  string
	list   list.Model
	result string
	cancel bool
}

func (m selectModel) Init() tea.Cmd {
	return nil
}

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if it, ok := m.list.SelectedItem().(selectItem); ok {
				m.result = it.label
			}
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.cancel = true
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height)
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m selectModel) View() string {
	if m.title != "" {
		return NewStyles().Accent.Render("? "+m.title) + "\n" + m.list.View() + "\n"
	}
	return m.list.View() + "\n"
}

// Select presents a menu of options and returns the selected label.
// An empty item list is an error.
func Select(title string, items []string) (string, error) {
	if !IsInteractive() {
		return "", RequireInteractive("La sélection")
	}
	if len(items) == 0 {
		return "", fmt.Errorf("aucune option disponible")
	}

	raw := make([]list.Item, 0, len(items))
	for _, label := range items {
		raw = append(raw, selectItem{label: label})
	}

	maxWidth := 0
	for _, label := range items {
		if w := lipgloss.Width(label); w > maxWidth {
			maxWidth = w
		}
	}
	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(1)
	delegate.SetSpacing(0)
	delegate.ShowDescription = false
	delegate.Styles = list.NewDefaultItemStyles()
	s := NewStyles()
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().Foreground(lipgloss.Color(s.palette.accent)).Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedTitle
	delegate.Styles.NormalTitle = lipgloss.NewStyle()
	delegate.Styles.NormalDesc = delegate.Styles.NormalTitle

	l := list.New(raw, delegate, min(maxWidth+12, 80), len(items)+2)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	m := selectModel{title: title, list: l}
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return "", err
	}
	fm, ok := final.(selectModel)
	if !ok {
		return "", fmt.Errorf("sélection terminée de manière inattendue")
	}
	if fm.cancel {
		return "", fmt.Errorf("opération annulée")
	}
	return fm.result, nil
}

// confirmModel is a minimal yes/no prompt.
type confirmModel struct {
	title  string
	defYes bool
	result bool
	cancel bool
}

func (m confirmModel) Init() tea.Cmd {
	return nil
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y":
			m.result = true
			return m, tea.Quit
		case "n":
			m.result = false
			return m, tea.Quit
		case "enter":
			m.result = m.defYes
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.cancel = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m confirmModel) View() string {
	dflt := "Oui"
	if !m.defYes {
		dflt = "Non"
	}
	s := NewStyles()
	return s.Accent.Render("? "+m.title) + " (oui/non) [" + s.Info.Render(dflt) + "] : \n"
}

// Confirm asks a yes/no question. defYes is the answer given by pressing
// strictly <enter>.
func Confirm(title string, defYes bool) (bool, error) {
	if !IsInteractive() {
		return false, RequireInteractive("La confirmation")
	}
	m := confirmModel{title: title, defYes: defYes}
	p := tea.NewProgram(m)
	final, err := p.Run()
	if err != nil {
		return false, err
	}
	fm, ok := final.(confirmModel)
	if !ok {
		return false, fmt.Errorf("confirmation terminée de manière inattendue")
	}
	if fm.cancel {
		return false, fmt.Errorf("opération annulée")
	}
	return fm.result, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
