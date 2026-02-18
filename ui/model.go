package ui

import (
	"database/sql"

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
					return m, loadTableDataCmd(m.db, item.Name)
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
			return m, loadTableDataCmd(m.db, msg.tables[0])
		}
		return m, nil

	case tableDataLoadedMsg:
		m.tableData = NewTableDataModel(
			msg.tableName, msg.columns, msg.rows,
			m.rightWidth, m.paneHeight(), m.db,
		)
		m.dataLoaded = true
		return m, nil

	case TableSelectedMsg:
		return m, loadTableDataCmd(m.db, msg.Name)

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

	h := m.paneHeight()
	contentH := h - 2

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

	statusText := " ←→/tab: navigate | enter: detail | f: filter | ctrl+e: query | esc: back | q: quit"
	if m.dataLoaded {
		statusText = " " + m.tableData.StatusText() + " | " + statusText[1:]
	}
	status := StatusBarStyle.Render(statusText)

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

func loadTableDataCmd(database *sql.DB, tableName string) tea.Cmd {
	return func() tea.Msg {
		cols, rows, err := db.GetRows(database, tableName, 1000)
		if err != nil {
			return errMsg{err: err}
		}
		return tableDataLoadedMsg{
			tableName: tableName,
			columns:   cols,
			rows:      rows,
		}
	}
}
