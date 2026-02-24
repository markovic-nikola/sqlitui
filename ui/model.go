package ui

import (
	"database/sql"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/markovic-nikola/sqlitui/db"
)

// pane tracks which panel currently receives keyboard input.
type pane int

const (
	paneList pane = iota
	paneData
)

// --- Custom message types ---

type tablesLoadedMsg struct {
	tables []string
}

type tableDataLoadedMsg struct {
	tableName string
	columns   []string
	rows      [][]string
	page      int
	pageSize  int
	totalRows int
}

type errMsg struct {
	err error
}

// --- Root Model ---

type Model struct {
	db      *sql.DB
	focused pane
	loaded  bool // true once the table list is ready

	width  int
	height int
	err    error

	// File picker screen — shown when no CLI arg is provided.
	showPathInput bool
	filePicker    FilePickerModel

	tableList  TableListModel
	tableData  TableDataModel
	dataLoaded bool // true once any table's data has been fetched

	// Modal popup for row detail.
	rowDetail  RowDetailModel
	showDetail bool

	// Modal popup for SQL query input.
	queryInput QueryInputModel
	showQuery  bool

	// Pane dimensions — recalculated on every WindowSizeMsg.
	leftWidth  int
	rightWidth int
}

func NewModel(path string) Model {
	if path != "" {
		if err := validatePath(path); err != nil {
			return Model{err: err}
		}
		database, err := db.Open(path)
		if err != nil {
			return Model{err: err}
		}
		return Model{
			db:      database,
			focused: paneList,
		}
	}

	return Model{
		showPathInput: true,
		filePicker:    NewFilePickerModel(),
		focused:       paneList,
	}
}

func (m Model) Init() tea.Cmd {
	if m.showPathInput {
		return m.filePicker.Init()
	}
	if m.db == nil {
		return nil
	}
	return func() tea.Msg {
		tables, err := db.ListTables(m.db)
		if err != nil {
			return errMsg{err: err}
		}
		return tablesLoadedMsg{tables: tables}
	}
}

// calcPaneSizes splits the terminal width into left (~30%) and right (~70%).
func (m *Model) calcPaneSizes() {
	available := m.width - 4
	m.leftWidth = available * 30 / 100
	if m.leftWidth < 25 {
		m.leftWidth = 25
	}
	m.rightWidth = available - m.leftWidth
}

// paneHeight returns the total height for a pane's border box.
func (m Model) paneHeight() int {
	return max(m.height-4, 5)
}

// pageSize returns the number of visible data rows in the table, used as page size.
// paneHeight-3 is the bubbles table Height, and the header (with border-bottom)
// takes 2 of those lines, leaving Height-2 for actual data rows.
func (m Model) pageSize() int {
	return max(m.paneHeight()-5, 1)
}

// helpItem is a key binding + description pair for the status bar.
type helpItem struct {
	key  string
	desc string
}

// renderStatusBar builds the full-width status bar with an info section on the
// left and wrapped help hints on the right.
func (m Model) renderStatusBar(info string, items []helpItem) string {
	barW := m.width - 4 // account for AppStyle horizontal margin
	if barW < 1 {
		barW = 1
	}

	// Render the info section.
	var infoRendered string
	infoW := 0
	if info != "" {
		infoRendered = StatusBarInfoStyle.Render(" " + info + " ")
		infoW = lipgloss.Width(infoRendered)
	}

	// Render help items with wrapping.
	helpW := barW - infoW
	if helpW < 10 {
		helpW = barW
		infoRendered = ""
		infoW = 0
	}

	var helpLines []string
	var lineItems []string
	lineW := 0

	for _, item := range items {
		rendered := StatusBarKeyStyle.Render(" "+item.key+" ") + StatusBarDescStyle.Render(item.desc+" ")
		itemW := lipgloss.Width(rendered)

		// First line has less space (info section is there); subsequent lines use full width.
		maxLineW := helpW
		if len(helpLines) > 0 {
			maxLineW = barW
		}

		if lineW > 0 && lineW+itemW > maxLineW {
			helpLines = append(helpLines, strings.Join(lineItems, ""))
			lineItems = nil
			lineW = 0
		}
		lineItems = append(lineItems, rendered)
		lineW += itemW
	}
	if len(lineItems) > 0 {
		helpLines = append(helpLines, strings.Join(lineItems, ""))
	}

	// Build each line: info on first line only, padding on the right to fill barW.
	var barLines []string
	for i, hl := range helpLines {
		hlW := lipgloss.Width(hl)

		if i == 0 && infoRendered != "" {
			lineContent := infoRendered + StatusBarBgStyle.Render(" ") + hl
			pad := barW - infoW - 1 - hlW
			if pad < 0 {
				pad = 0
			}
			barLines = append(barLines, lineContent+StatusBarBgStyle.Render(strings.Repeat(" ", pad)))
		} else {
			pad := barW - hlW
			if pad < 0 {
				pad = 0
			}
			barLines = append(barLines, hl+StatusBarBgStyle.Render(strings.Repeat(" ", pad)))
		}
	}

	if len(barLines) == 0 {
		if infoRendered != "" {
			pad := barW - infoW
			if pad < 0 {
				pad = 0
			}
			return infoRendered + StatusBarBgStyle.Render(strings.Repeat(" ", pad))
		}
		return StatusBarBgStyle.Render(strings.Repeat(" ", barW))
	}

	return strings.Join(barLines, "\n")
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Always track terminal size.
	if wsm, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = wsm.Width
		m.height = wsm.Height
	}

	// File picker captures all input when shown.
	if m.showPathInput {
		switch msg := msg.(type) {
		case dbOpenedMsg:
			m.db = msg.db
			m.showPathInput = false
			m.calcPaneSizes()
			return m, func() tea.Msg {
				return tablesLoadedMsg{tables: msg.tables}
			}
		default:
			var cmd tea.Cmd
			m.filePicker, cmd = m.filePicker.Update(msg)
			return m, cmd
		}
	}

	// Query popup captures all input when open.
	if m.showQuery {
		switch msg := msg.(type) {
		case CloseDetailMsg:
			m.showQuery = false
			return m, nil
		case QueryResultMsg:
			m.showQuery = false
			m.tableData = NewTableDataModel(
				"query result", msg.Columns, msg.Rows,
				m.rightWidth, m.paneHeight(), m.db,
				0, len(msg.Rows), len(msg.Rows),
			)
			m.dataLoaded = true
			m.focused = paneData
			return m, nil
		default:
			var cmd tea.Cmd
			m.queryInput, cmd = m.queryInput.Update(msg)
			return m, cmd
		}
	}

	// Row detail popup captures all input when open.
	if m.showDetail {
		switch msg.(type) {
		case CloseDetailMsg:
			m.showDetail = false
			return m, nil
		default:
			var cmd tea.Cmd
			m.rowDetail, cmd = m.rowDetail.Update(msg)
			return m, cmd
		}
	}

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.calcPaneSizes()
		if m.loaded {
			m.tableList.SetSize(m.leftWidth, m.paneHeight())
		}
		if m.dataLoaded {
			m.tableData.SetSize(m.rightWidth, m.paneHeight())
		}
		return m, nil

	case tea.KeyMsg:
		if key.Matches(msg, Keys.SwitchTab) {
			if m.focused == paneList {
				m.focused = paneData
			} else {
				m.focused = paneList
			}
			return m, nil
		}

		if key.Matches(msg, Keys.FocusRight) && m.focused == paneList && m.loaded {
			if m.tableList.list.FilterState() != list.Filtering {
				m.focused = paneData
				item, ok := m.tableList.list.SelectedItem().(TableItem)
				if ok && (!m.dataLoaded || m.tableData.tableName != item.Name) {
					return m, loadTableDataCmd(m.db, item.Name, m.pageSize())
				}
			}
			return m, nil
		}

		if key.Matches(msg, Keys.FocusLeft) && m.focused == paneData {
			m.focused = paneList
			return m, nil
		}

		if msg.Type == tea.KeyEsc {
			if m.focused == paneList && m.tableList.list.FilterState() == list.Filtering {
				break // let the list handle esc to cancel filter
			}
			if m.db != nil {
				m.db.Close()
				m.db = nil
			}
			m.loaded = false
			m.dataLoaded = false
			m.showPathInput = true
			m.filePicker = NewFilePickerModel()
			m.filePicker.width = m.width
			m.filePicker.height = m.height
			return m, m.filePicker.Init()
		}

		if key.Matches(msg, Keys.Quit) {
			if m.focused == paneList && m.tableList.list.FilterState() == list.Filtering {
				break
			}
			return m, tea.Quit
		}

		if key.Matches(msg, Keys.Refresh) && m.dataLoaded && m.tableData.tableName != "query result" {
			return m, loadTableDataCmd(m.db, m.tableData.tableName, m.pageSize())
		}

		if key.Matches(msg, Keys.OpenQuery) {
			qi, cmd := NewQueryInputModel(m.db, m.width, m.height)
			m.queryInput = qi
			m.showQuery = true
			return m, cmd
		}

	case tablesLoadedMsg:
		m.tableList = NewTableListModel(msg.tables, m.leftWidth, m.paneHeight())
		m.loaded = true
		if len(msg.tables) > 0 {
			return m, loadTableDataCmd(m.db, msg.tables[0], m.pageSize())
		}
		return m, nil

	case tableDataLoadedMsg:
		m.tableData = NewTableDataModel(
			msg.tableName, msg.columns, msg.rows,
			m.rightWidth, m.paneHeight(), m.db,
			msg.page, msg.pageSize, msg.totalRows,
		)
		m.dataLoaded = true
		return m, nil

	case pageDataLoadedMsg:
		m.tableData.allRows = msg.rows
		m.tableData.page = msg.page
		if m.tableData.fActive {
			m.tableData.fTotalRows = msg.totalRows
		} else {
			m.tableData.totalRows = msg.totalRows
		}
		m.tableData.table.SetRows(truncateRows(msg.rows, m.tableData.displayCols, m.tableData.hasHiddenCols()))
		if msg.cursorEnd && len(msg.rows) > 0 {
			m.tableData.table.SetCursor(len(msg.rows) - 1)
			m.tableData.table.GotoBottom()
		} else {
			m.tableData.table.SetCursor(0)
		}
		return m, nil

	case TableSelectedMsg:
		return m, loadTableDataCmd(m.db, msg.Name, m.pageSize())

	case RowSelectedMsg:
		m.rowDetail = NewRowDetailModel(msg.Columns, msg.Values, m.width, m.height)
		m.showDetail = true
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil
	}

	switch m.focused {
	case paneList:
		if m.loaded {
			var cmd tea.Cmd
			m.tableList, cmd = m.tableList.Update(msg)
			return m, cmd
		}
	case paneData:
		if m.dataLoaded {
			var cmd tea.Cmd
			m.tableData, cmd = m.tableData.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m Model) View() string {
	if m.showPathInput {
		return m.filePicker.View()
	}

	if m.err != nil {
		return AppStyle.Render(
			ErrorStyle.Render("Error: "+m.err.Error()) +
				"\n\n" + StatusBarStyle.Render("Press q to quit."),
		)
	}

	if !m.loaded {
		return AppStyle.Render(
			Logo + "\n\nLoading tables...",
		)
	}

	leftStyle, rightStyle := UnfocusedPaneStyle, UnfocusedPaneStyle
	if m.focused == paneList {
		leftStyle = FocusedPaneStyle
	} else {
		rightStyle = FocusedPaneStyle
	}

	// Build the status bar first so we know how many lines it needs.
	hints := []helpItem{
		{"←→/tab", "navigate"},
		{"enter", "detail"},
		{"f", "filter"},
		{"[/]", "page"},
		{"ctrl+e", "query"},
		{"ctrl+r", "refresh"},
		{"esc", "back"},
		{"q", "quit"},
	}
	var info string
	if m.dataLoaded {
		info = m.tableData.StatusText()
	}
	status := m.renderStatusBar(info, hints)
	statusLines := strings.Count(status, "\n") + 1

	// 3 = top margin (1) + bottom margin (1) + status bar base (1 line already counted in statusLines adjustment)
	contentH := max(m.height-3-statusLines, 3) - 2

	leftClip := lipgloss.NewStyle().MaxHeight(contentH).MaxWidth(m.leftWidth - 2)
	rightClip := lipgloss.NewStyle().MaxHeight(contentH).MaxWidth(m.rightWidth - 2)

	leftPanel := leftStyle.
		Width(m.leftWidth - 2).
		Height(contentH).
		Render(leftClip.Render(m.tableList.View()))

	var rightContent string
	if m.dataLoaded {
		rightContent = m.tableData.View()
	} else {
		rightContent = lipgloss.Place(
			m.rightWidth-2, contentH,
			lipgloss.Center, lipgloss.Center,
			StatusBarStyle.Render("← Select a table"),
		)
	}
	rightPanel := rightStyle.
		Width(m.rightWidth - 2).
		Height(contentH).
		Render(rightClip.Render(rightContent))

	split := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	base := AppStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, split, status),
	)

	if m.showDetail {
		popup := m.rowDetail.View()
		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			popup,
		)
	}
	if m.showQuery {
		popup := m.queryInput.View()
		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			popup,
		)
	}

	return base
}

func loadTableDataCmd(database *sql.DB, tableName string, pageSize int) tea.Cmd {
	return func() tea.Msg {
		total, err := db.CountRows(database, tableName)
		if err != nil {
			return errMsg{err: err}
		}
		cols, rows, err := db.GetRows(database, tableName, pageSize, 0)
		if err != nil {
			return errMsg{err: err}
		}
		return tableDataLoadedMsg{
			tableName: tableName,
			columns:   cols,
			rows:      rows,
			page:      0,
			pageSize:  pageSize,
			totalRows: total,
		}
	}
}
