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

// pageDataLoadedMsg carries the result of loading a specific page.
type pageDataLoadedMsg struct {
	rows      [][]string
	page      int
	pageSize  int
	totalRows int
	cursorEnd bool // when true, place cursor at the last row
}

const (
	minColWidth     = 10 // minimum width for any data column
	maxColWidth     = 40 // maximum width for any data column
	colPadding      = 3  // padding added to measured content width
	indicatorColLen = 12 // reserved width for the "+ N cols" indicator column
)

// TableDataModel wraps bubbles/table.Model to display rows from a DB table.
// It also stores the raw data so we can pass it to the popup on selection.
type TableDataModel struct {
	table       table.Model
	tableName   string
	columns     []string   // all columns from the DB
	displayCols int        // number of columns shown in the table (dynamically computed)
	allRows     [][]string // rows for the current page (all columns)
	database    *sql.DB    // for DB-level filter queries
	width       int
	height      int

	// Pagination state.
	page      int // current page (0-indexed)
	pageSize  int // rows per page
	totalRows int // total rows in table (from COUNT(*))

	// Filter state.
	fState     filterState
	fColIndex  int             // highlighted column in the picker
	fColScroll int             // scroll offset for column picker
	fCol       string          // selected column name
	fInput     textinput.Model // value input
	fActive    bool            // true when a confirmed filter is applied
	fQuery     string          // the confirmed filter text
	fTotalRows int             // total count of filtered rows
	fPrevPage  int             // page before filter was opened
}

func NewTableDataModel(name string, columns []string, rows [][]string, width, height int, database *sql.DB, page, pageSize, totalRows int) TableDataModel {
	innerWidth := width - 2
	// height is the pane border-box. Content area = height - 2.
	// bubbles/table with WithHeight(N) outputs N+1 lines.
	// We need N+1 <= height-2, so N = height-3.
	tableHeight := height - 3
	displayCols, colWidths := fitColumns(columns, rows, innerWidth)

	tableCols := buildTableColumns(columns, displayCols, colWidths, len(columns))

	t := table.New(
		table.WithColumns(tableCols),
		table.WithRows(truncateRows(rows, displayCols, displayCols < len(columns))),
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
		table:       t,
		tableName:   name,
		columns:     columns,
		displayCols: displayCols,
		allRows:     rows,
		database:    database,
		width:       width,
		height:      height,
		page:        page,
		pageSize:    pageSize,
		totalRows:   totalRows,
		fInput:      ti,
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

func (m TableDataModel) totalPages() int {
	total := m.totalRows
	if m.fActive {
		total = m.fTotalRows
	}
	if total <= 0 {
		return 1
	}
	return (total + m.pageSize - 1) / m.pageSize
}

func (m TableDataModel) hasNextPage() bool {
	return m.page < m.totalPages()-1
}

func (m TableDataModel) hasPrevPage() bool {
	return m.page > 0
}

func loadPageCmd(database *sql.DB, tableName string, page, pageSize int, cursorEnd bool) tea.Cmd {
	return func() tea.Msg {
		offset := page * pageSize
		_, rows, err := db.GetRows(database, tableName, pageSize, offset)
		if err != nil {
			return errMsg{err: err}
		}
		total, err := db.CountRows(database, tableName)
		if err != nil {
			return errMsg{err: err}
		}
		return pageDataLoadedMsg{
			rows:      rows,
			page:      page,
			pageSize:  pageSize,
			totalRows: total,
			cursorEnd: cursorEnd,
		}
	}
}

func loadFilteredPageCmd(database *sql.DB, tableName, fCol, fQuery string, page, pageSize int, cursorEnd bool) tea.Cmd {
	return func() tea.Msg {
		offset := page * pageSize
		_, rows, err := db.FilterColumn(database, tableName, fCol, fQuery, pageSize, offset)
		if err != nil {
			return errMsg{err: err}
		}
		total, err := db.CountFilteredRows(database, tableName, fCol, fQuery)
		if err != nil {
			return errMsg{err: err}
		}
		return pageDataLoadedMsg{
			rows:      rows,
			page:      page,
			pageSize:  pageSize,
			totalRows: total,
			cursorEnd: cursorEnd,
		}
	}
}

func (m TableDataModel) nextPageCmd() tea.Cmd {
	if m.fActive {
		return loadFilteredPageCmd(m.database, m.tableName, m.fCol, m.fQuery, m.page+1, m.pageSize, false)
	}
	return loadPageCmd(m.database, m.tableName, m.page+1, m.pageSize, false)
}

func (m TableDataModel) prevPageCmd() tea.Cmd {
	if m.fActive {
		return loadFilteredPageCmd(m.database, m.tableName, m.fCol, m.fQuery, m.page-1, m.pageSize, true)
	}
	return loadPageCmd(m.database, m.tableName, m.page-1, m.pageSize, true)
}

func (m *TableDataModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	innerWidth := width - 2

	displayCols, colWidths := fitColumns(m.columns, m.allRows, innerWidth)
	m.displayCols = displayCols
	m.table.SetColumns(buildTableColumns(m.columns, displayCols, colWidths, len(m.columns)))
	m.table.SetRows(truncateRows(m.allRows, m.displayCols, m.hasHiddenCols()))
	m.table.SetHeight(m.tableHeight())
	m.fInput.Width = innerWidth - 3
}

func (m TableDataModel) hasHiddenCols() bool {
	return len(m.columns) > m.displayCols
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
		m.fPrevPage = m.page
		m.table.SetHeight(m.tableHeight())
		return m, nil
	}

	if key.Matches(msg, Keys.NextPage) && m.hasNextPage() {
		return m, m.nextPageCmd()
	}

	if key.Matches(msg, Keys.PrevPage) && m.hasPrevPage() {
		return m, m.prevPageCmd()
	}

	// Auto-advance to next page when pressing down on the last row.
	switch msg.String() {
	case "down", "j":
		if m.table.Cursor() >= len(m.table.Rows())-1 && m.hasNextPage() {
			return m, m.nextPageCmd()
		}
	case "up", "k":
		if m.table.Cursor() <= 0 && m.hasPrevPage() {
			return m, m.prevPageCmd()
		}
	}

	if key.Matches(msg, Keys.Select) {
		cursor := m.table.Cursor()
		if cursor >= 0 && cursor < len(m.allRows) {
			return m, func() tea.Msg {
				return RowSelectedMsg{
					Columns: m.columns,
					Values:  m.allRows[cursor],
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
		m.fActive = false
		m.fQuery = ""
		m.fTotalRows = 0
		m.page = m.fPrevPage
		m.table.SetRows(truncateRows(m.allRows, m.displayCols, m.hasHiddenCols()))
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
		m.fActive = false
		m.fQuery = ""
		m.fTotalRows = 0
		m.page = m.fPrevPage
		m.table.SetRows(truncateRows(m.allRows, m.displayCols, m.hasHiddenCols()))
		m.table.SetCursor(0)
		m.table.SetHeight(m.tableHeight())
		return m, nil

	case "enter":
		m.fInput.Blur()
		m.fActive = m.fInput.Value() != ""
		m.fQuery = m.fInput.Value()
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
		m.table.SetRows(truncateRows(m.allRows, m.displayCols, m.hasHiddenCols()))
		m.table.SetCursor(0)
		m.fTotalRows = 0
		return
	}
	_, rows, err := db.FilterColumn(m.database, m.tableName, m.fCol, query, m.pageSize, 0)
	if err != nil {
		m.table.SetRows(truncateRows(m.allRows, m.displayCols, m.hasHiddenCols()))
		m.table.SetCursor(0)
		return
	}
	total, err := db.CountFilteredRows(m.database, m.tableName, m.fCol, query)
	if err != nil {
		total = len(rows)
	}
	m.fTotalRows = total
	m.page = 0
	m.table.SetRows(truncateRows(rows, m.displayCols, m.hasHiddenCols()))
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
	currentPage := m.page + 1
	pages := m.totalPages()

	if m.fActive {
		return fmt.Sprintf("%s (page %d/%d, %d results for %s)", m.tableName, currentPage, pages, m.fTotalRows, m.fCol)
	}

	// During live filter typing, show result count without page info.
	if m.fState != filterOff {
		displayed := len(m.table.Rows())
		return fmt.Sprintf("%s (%d results for %s)", m.tableName, displayed, m.fCol)
	}

	return fmt.Sprintf("%s (page %d/%d, %d rows)", m.tableName, currentPage, pages, m.totalRows)
}

// measureColWidth returns the ideal width for a column based on its header and data.
func measureColWidth(colIndex int, header string, rows [][]string) int {
	w := len(header)
	for _, r := range rows {
		if colIndex < len(r) && len(r[colIndex]) > w {
			w = len(r[colIndex])
		}
	}
	w += colPadding
	if w < minColWidth {
		w = minColWidth
	}
	if w > maxColWidth {
		w = maxColWidth
	}
	return w
}

// fitColumns determines how many columns fit within the available width and
// returns the number of display columns along with their widths.
func fitColumns(columns []string, rows [][]string, innerWidth int) (int, []int) {
	available := innerWidth - 2 // account for table border
	if available < minColWidth {
		available = minColWidth
	}

	widths := make([]int, 0, len(columns))
	used := 0

	for i, col := range columns {
		w := measureColWidth(i, col, rows)
		remaining := len(columns) - i - 1

		// If this isn't the last column, check if we need to reserve space for the indicator.
		needed := w
		if remaining > 0 {
			needed += indicatorColLen // must still fit the "+ N cols" column
		}

		if used+needed > available && i > 0 {
			break
		}
		widths = append(widths, w)
		used += w
	}

	// Distribute leftover space evenly across displayed columns.
	displayCols := len(widths)
	hiddenCols := len(columns) - displayCols
	leftover := available - used
	if hiddenCols > 0 {
		leftover -= indicatorColLen
	}
	if leftover > 0 && displayCols > 0 {
		extra := leftover / displayCols
		for i := range widths {
			widths[i] += extra
		}
	}

	return displayCols, widths
}

// buildTableColumns creates bubbles table column definitions from pre-computed widths.
func buildTableColumns(columns []string, displayCols int, widths []int, totalCols int) []table.Column {
	hiddenCols := totalCols - displayCols
	numCols := displayCols
	if hiddenCols > 0 {
		numCols++
	}
	cols := make([]table.Column, numCols)
	for i := range displayCols {
		cols[i] = table.Column{Title: columns[i], Width: widths[i]}
	}
	if hiddenCols > 0 {
		cols[displayCols] = table.Column{
			Title: fmt.Sprintf("+ %d cols", hiddenCols),
			Width: indicatorColLen,
		}
	}
	return cols
}

// truncateRows converts [][]string to []table.Row, keeping only the first maxCols values per row.
// When hasExtra is true, an empty trailing cell is added to match the extra header column.
func truncateRows(rows [][]string, maxCols int, hasExtra bool) []table.Row {
	result := make([]table.Row, len(rows))
	for i, r := range rows {
		row := r
		if len(r) > maxCols {
			row = r[:maxCols]
		}
		if hasExtra {
			row = append(row, "")
		}
		result[i] = row
	}
	return result
}
