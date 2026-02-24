package ui

import (
	"database/sql"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/markovic-nikola/sqlitui/db"
)

// QueryResultMsg is sent when the user successfully executes a query.
// The parent model handles this to populate the right pane.
type QueryResultMsg struct {
	Columns []string
	Rows    [][]string
}

// QueryInputModel is the SQL query popup component.
// It presents a textarea for writing SQL and executes it on ctrl+r.
type QueryInputModel struct {
	textarea textarea.Model
	queryErr string
	database *sql.DB
	width    int
	height   int
}

// NewQueryInputModel creates the popup, sized ~70% wide x ~50% tall.
// Returns a tea.Cmd for the textarea cursor blink.
func NewQueryInputModel(database *sql.DB, termWidth, termHeight int) (QueryInputModel, tea.Cmd) {
	popupWidth := termWidth * 70 / 100
	popupHeight := termHeight * 50 / 100
	if popupWidth < 50 {
		popupWidth = 50
	}
	if popupHeight < 12 {
		popupHeight = 12
	}

	// PopupStyle has border (2) + padding (2 horiz each side = 4, 1 vert each side = 2).
	// Vertical overhead: border(2) + padding(2) + title(1) + gap(1) + error(1) + help(1) = 8.
	contentWidth := popupWidth - 6
	textareaHeight := popupHeight - 8
	if textareaHeight < 4 {
		textareaHeight = 4
	}

	ta := textarea.New()
	ta.Placeholder = "SELECT * FROM ..."
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	// Remove the textarea's own border so it doesn't double-border inside PopupStyle.
	ta.FocusedStyle.Base = lipgloss.NewStyle()
	ta.BlurredStyle.Base = lipgloss.NewStyle()
	ta.SetWidth(contentWidth)
	ta.SetHeight(textareaHeight)
	cmd := ta.Focus()

	return QueryInputModel{
		textarea: ta,
		database: database,
		width:    popupWidth,
		height:   popupHeight,
	}, cmd
}

func (m QueryInputModel) Update(msg tea.Msg) (QueryInputModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return CloseDetailMsg{} }

		case "ctrl+r", "ctrl+enter":
			query := m.textarea.Value()
			if query == "" {
				return m, nil
			}
			cols, rows, err := db.ExecQuery(m.database, query)
			if err != nil {
				m.queryErr = err.Error()
				return m, nil
			}
			return m, func() tea.Msg {
				return QueryResultMsg{Columns: cols, Rows: rows}
			}
		}
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m QueryInputModel) View() string {
	title := TitleStyle.Render(" SQL Query ")
	help := StatusBarStyle.Render("ctrl+r/ctrl+enter: run | esc: close")

	// Always reserve the error line to prevent layout jumps.
	errLine := " "
	if m.queryErr != "" {
		errLine = ErrorStyle.Render("Error: " + m.queryErr)
	}

	return PopupStyle.
		Width(m.width - 2).
		Height(m.height - 2).
		Render(title + "\n\n" + m.textarea.View() + "\n" + errLine + "\n" + help)
}
