package ui

import (
	"database/sql"
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/markovic-nikola/sqlitui/db"
)

// RowSelectedMsg is sent when the user presses enter on a row.
// Carries column names + that row's values so the popup can display them.
type RowSelectedMsg struct {
	Columns []string
	Values  []string
}

// filterState tracks the two-step filter flow.
type filterState int

const (
	filterOff     filterState = iota // normal table mode
	filterPickCol                    // picking a column
	filterInput                      // typing a value
)

// TableDataModel wraps bubbles/table.Model to display rows from a DB table.
// It also stores the raw data so we can pass it to the popup on selection.
type TableDataModel struct {
	table     table.Model
	tableName string
	columns   []string
	allRows   [][]string // initial loaded rows (displayed when no filter)
	database  *sql.DB    // for DB-level filter queries
	width     int
	height    int

	// Filter state.
	fState     filterState
	fColIndex  int             // highlighted column in the picker
	fColScroll int             // scroll offset for column picker
	fCol       string          // selected column name
	fInput     textinput.Model // value input
}

func NewTableDataModel(name string, columns []string, rows [][]string, width, height int, database *sql.DB) TableDataModel {
	innerWidth := width - 2
	// height is the pane border-box. Content area = height - 2.
	// bubbles/table with WithHeight(N) outputs N+1 lines.
	// We need N+1 <= height-2, so N = height-3.
	tableHeight := height - 3

	colWidths := calcColumnWidths(len(columns), innerWidth)
	cols := make([]table.Column, len(columns))
	for i, c := range columns {
		cols[i] = table.Column{Title: c, Width: colWidths[i]}
	}

	tableRows := make([]table.Row, len(rows))
	for i, r := range rows {
		tableRows[i] = r
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithRows(tableRows),
		table.WithFocused(true),
		table.WithHeight(tableHeight),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	ti := textinput.New()
	ti.Placeholder = "filter..."
	ti.Width = innerWidth - 3
	// Disable suggestion keybinds to avoid up/down conflicts with the table.
	ti.KeyMap.NextSuggestion = key.NewBinding()
	ti.KeyMap.PrevSuggestion = key.NewBinding()

	return TableDataModel{
		table:     t,
		tableName: name,
		columns:   columns,
		allRows:   rows,
		database:  database,
		width:     width,
		height:    height,
		fInput:    ti,
	}
}

// pickerVisibleCount returns how many column names are visible in the picker.
func (m TableDataModel) pickerVisibleCount() int {
	maxVisible := (m.height - 3) / 2
	if maxVisible < 3 {
		maxVisible = 3
	}
	if len(m.columns) < maxVisible {
		return len(m.columns)
	}
	return maxVisible
}

func (m *TableDataModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	innerWidth := width - 2
	tableHeight := m.tableHeight()

	colWidths := calcColumnWidths(len(m.table.Columns()), innerWidth)
	cols := m.table.Columns()
	for i := range cols {
		cols[i].Width = colWidths[i]
	}
	m.table.SetColumns(cols)
	m.table.SetHeight(tableHeight)
	m.fInput.Width = innerWidth - 3
}

// tableHeight returns the bubbles/table height accounting for the filter UI.
func (m TableDataModel) tableHeight() int {
	h := m.height - 3
	switch m.fState {
	case filterPickCol:
		h -= m.pickerVisibleCount()
	case filterInput:
		h--
	}
	if h < 3 {
		h = 3
	}
	return h
}

func (m TableDataModel) Update(msg tea.Msg) (TableDataModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.fState {
		case filterPickCol:
			return m.updatePickCol(msg)
		case filterInput:
			return m.updateFilterInput(msg)
		default:
			return m.updateNormal(msg)
		}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m TableDataModel) updateNormal(msg tea.KeyMsg) (TableDataModel, tea.Cmd) {
	if msg.String() == "f" {
		m.fState = filterPickCol
		m.fColIndex = 0
		m.fColScroll = 0
		m.table.SetHeight(m.tableHeight())
		return m, nil
	}

	if key.Matches(msg, Keys.Select) {
		selected := m.table.SelectedRow()
		if selected != nil {
			return m, func() tea.Msg {
				return RowSelectedMsg{
					Columns: m.columns,
					Values:  selected,
				}
			}
		}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m TableDataModel) updatePickCol(msg tea.KeyMsg) (TableDataModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.fState = filterOff
		m.table.SetRows(toTableRows(m.allRows))
		m.table.SetCursor(0)
		m.table.SetHeight(m.tableHeight())
		return m, nil

	case "up", "k":
		if m.fColIndex > 0 {
			m.fColIndex--
			if m.fColIndex < m.fColScroll {
				m.fColScroll = m.fColIndex
			}
		}
		return m, nil

	case "down", "j":
		if m.fColIndex < len(m.columns)-1 {
			m.fColIndex++
			visible := m.pickerVisibleCount()
			if m.fColIndex >= m.fColScroll+visible {
				m.fColScroll = m.fColIndex - visible + 1
			}
		}
		return m, nil

	case "enter":
		m.fCol = m.columns[m.fColIndex]
		m.fState = filterInput
		m.fInput.Prompt = m.fCol + ": "
		m.fInput.Reset()
		m.table.SetHeight(m.tableHeight())
		cmd := m.fInput.Focus()
		return m, cmd
	}

	return m, nil
}

func (m TableDataModel) updateFilterInput(msg tea.KeyMsg) (TableDataModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.fInput.Blur()
		m.fInput.Reset()
		m.fState = filterOff
		m.table.SetRows(toTableRows(m.allRows))
		m.table.SetCursor(0)
		m.table.SetHeight(m.tableHeight())
		return m, nil

	case "enter":
		m.fInput.Blur()
		m.fState = filterOff
		m.table.SetHeight(m.tableHeight())
		return m, nil
	}

	var cmd tea.Cmd
	m.fInput, cmd = m.fInput.Update(msg)
	m.applyFilter()
	return m, cmd
}

// applyFilter queries the DB for rows matching the filter value in the selected column.
func (m *TableDataModel) applyFilter() {
	query := m.fInput.Value()
	if query == "" {
		m.table.SetRows(toTableRows(m.allRows))
		m.table.SetCursor(0)
		return
	}
	_, rows, err := db.FilterColumn(m.database, m.tableName, m.fCol, query, 1000)
	if err != nil {
		m.table.SetRows(toTableRows(m.allRows))
		m.table.SetCursor(0)
		return
	}
	m.table.SetRows(toTableRows(rows))
	m.table.SetCursor(0)
}

func (m TableDataModel) View() string {
	if len(m.allRows) == 0 {
		contentW := m.width - 2
		contentH := m.height - 2
		msg := TitleStyle.Render(m.tableName) + "\n\n" + StatusBarStyle.Render("No rows in this table")
		return lipgloss.Place(contentW, contentH, lipgloss.Center, lipgloss.Center, msg)
	}

	tableView := m.table.View()

	switch m.fState {
	case filterPickCol:
		return tableView + "\n" + m.renderColumnPicker()
	case filterInput:
		return tableView + "\n" + m.fInput.View()
	}
	return tableView
}

// renderColumnPicker draws a simple selectable list of column names.
func (m TableDataModel) renderColumnPicker() string {
	visible := m.pickerVisibleCount()
	var s string
	for i := m.fColScroll; i < m.fColScroll+visible && i < len(m.columns); i++ {
		name := m.columns[i]
		if i == m.fColIndex {
			s += TitleStyle.Render("▸ " + name)
		} else {
			s += StatusBarStyle.Render("  " + name)
		}
		if i < m.fColScroll+visible-1 && i < len(m.columns)-1 {
			s += "\n"
		}
	}
	return s
}

// StatusText returns info about the table for the parent's status bar.
func (m TableDataModel) StatusText() string {
	displayed := len(m.table.Rows())
	total := len(m.allRows)
	if displayed != total {
		return fmt.Sprintf("%s (%d results for %s)", m.tableName, displayed, m.fCol)
	}
	return fmt.Sprintf("%s (%d rows)", m.tableName, total)
}

// toTableRows converts [][]string to []table.Row.
func toTableRows(rows [][]string) []table.Row {
	result := make([]table.Row, len(rows))
	for i, r := range rows {
		result[i] = r
	}
	return result
}

// calcColumnWidths divides available width evenly across columns.
func calcColumnWidths(numCols, totalWidth int) []int {
	if numCols == 0 {
		return nil
	}
	available := totalWidth - 2
	base := available / numCols
	if base < 10 {
		base = 10
	}
	widths := make([]int, numCols)
	for i := range widths {
		widths[i] = base
	}
	return widths
}
