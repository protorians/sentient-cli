package tui

import (
	"errors"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func sendKeyMsg(t *testing.T, m tea.Model, msg tea.Msg) tea.Model {
	t.Helper()
	next, _ := m.Update(msg)
	return next
}

func enterKey() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyEnter}
}

func escKey() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyEsc}
}

func runeKey(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func TestInputModelEnterReturnsValue(t *testing.T) {
	m := inputModel{title: "Nom", input: textinput.New()}
	m.input.SetValue("blog-manager")

	next := sendKeyMsg(t, m, enterKey())
	fm, ok := next.(inputModel)
	if !ok {
		t.Fatalf("type = %T", next)
	}
	if fm.result != "blog-manager" {
		t.Errorf("result = %q, want blog-manager", fm.result)
	}
}

func TestInputModelCancelsOnEsc(t *testing.T) {
	m := inputModel{title: "Nom", input: textinput.New()}
	m.input.SetValue("blog")

	next := sendKeyMsg(t, m, escKey())
	fm, ok := next.(inputModel)
	if !ok {
		t.Fatalf("type = %T", next)
	}
	if !fm.cancel {
		t.Error("esc doit annuler le prompt")
	}
}

func TestConfirmModelYesAndNo(t *testing.T) {
	yes := sendKeyMsg(t, confirmModel{title: "Confirmer", defYes: false}, runeKey('y'))
	if y := yes.(confirmModel); !y.result {
		t.Error("la touche y doit valider")
	}

	no := sendKeyMsg(t, confirmModel{title: "Confirmer", defYes: true}, runeKey('n'))
	if n := no.(confirmModel); n.result {
		t.Error("la touche n doit refuser")
	}
}

func TestConfirmModelEnterUsesDefault(t *testing.T) {
	m := confirmModel{title: "Confirmer", defYes: true}
	next := sendKeyMsg(t, m, enterKey())
	if !next.(confirmModel).result {
		t.Error("enter doit retenir la valeur par défaut (true)")
	}
}

func TestSelectModelEnterReturnsSelection(t *testing.T) {
	items := []list.Item{
		selectItem{label: "alpha"},
		selectItem{label: "beta"},
		selectItem{label: "gamma"},
	}
	l := list.New(items, list.NewDefaultDelegate(), 24, 5)
	m := selectModel{title: "Sélection", list: l}

	next := sendKeyMsg(t, m, enterKey())
	fm, ok := next.(selectModel)
	if !ok {
		t.Fatalf("type = %T", next)
	}
	if fm.result != "alpha" {
		t.Errorf("result = %q, want alpha (premier élément)", fm.result)
	}
}

func TestSelectModelDownThenEnter(t *testing.T) {
	items := []list.Item{
		selectItem{label: "alpha"},
		selectItem{label: "beta"},
	}
	l := list.New(items, list.NewDefaultDelegate(), 24, 5)
	m := selectModel{title: "Sélection", list: l}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(tea.KeyMsg{Type: tea.KeyDown})
	if cmd != nil {
		_ = cmd
	}

	next, _ := m.Update(enterKey())
	fm := next.(selectModel)
	if fm.result != "beta" {
		t.Errorf("result = %q, want beta après ↓", fm.result)
	}
}

func TestRunWithSpinnerNonInteractive(t *testing.T) {
	// In a non-interactive environment the spinner degrades to a blocking
	// synchronous call.
	value, err := RunWithSpinner("Tâche", func() (int, error) {
		return 42, nil
	})
	if err != nil {
		t.Fatalf("RunWithSpinner: %v", err)
	}
	if value != 42 {
		t.Errorf("value = %d, want 42", value)
	}

	_, err = RunWithSpinner("Tâche", func() (int, error) {
		return 0, errors.New("boom")
	})
	if err == nil || err.Error() != "boom" {
		t.Errorf("erreur inattendue: %v", err)
	}
}
