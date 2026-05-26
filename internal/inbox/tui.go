package inbox

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type tuiModel struct {
	store   Store
	records []Record
	cursor  int
	detail  bool
	message string
}

func RunTUI(store Store) error {
	_, err := tea.NewProgram(newTUIModel(store)).Run()
	return err
}

func newTUIModel(store Store) tuiModel {
	model := tuiModel{store: store}
	model.reload()
	return model
}

func (m tuiModel) Init() tea.Cmd {
	return nil
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.records)-1 {
			m.cursor++
		}
	case "enter":
		m.detail = !m.detail
	case "r":
		m.reload()
	case "d":
		if len(m.records) == 0 {
			return m, nil
		}
		rec := m.records[m.cursor]
		n, err := m.store.MarkDone([]string{rec.ID})
		if err != nil {
			m.message = err.Error()
			return m, nil
		}
		m.message = fmt.Sprintf("marked %d done", n)
		m.reload()
	}
	return m, nil
}

func (m tuiModel) View() string {
	var b strings.Builder
	b.WriteString("agent-notify inbox\n")
	b.WriteString("j/k move  enter details  d done  r reload  q quit\n\n")
	if m.message != "" {
		b.WriteString(m.message)
		b.WriteString("\n\n")
	}
	if len(m.records) == 0 {
		b.WriteString("No pending notifications.\n")
		return b.String()
	}
	for i, rec := range m.records {
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		fmt.Fprintf(&b, "%s%s  %s  %s/%s  %s  %s\n", prefix, rec.ID, rec.Host, rec.Agent, rec.Event, rec.CWD, rec.Title)
		if i == m.cursor && m.detail {
			if rec.Body != "" {
				fmt.Fprintf(&b, "    body: %s\n", rec.Body)
			}
			if !rec.Time.IsZero() {
				fmt.Fprintf(&b, "    time: %s\n", rec.Time.Local().Format("2006-01-02 15:04:05"))
			}
			if rec.Tmux.Pane != "" {
				fmt.Fprintf(&b, "    tmux pane: %s\n", rec.Tmux.Pane)
			}
		}
	}
	return b.String()
}

func (m *tuiModel) reload() {
	records, err := m.store.Pending()
	if err != nil {
		m.message = err.Error()
		return
	}
	m.records = records
	if m.cursor >= len(m.records) {
		m.cursor = len(m.records) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}
