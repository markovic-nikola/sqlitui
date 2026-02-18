package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// TableItem implements the list.Item interface from bubbles.
// The list component needs items that can provide a title, description,
// and a filter value (used for the built-in fuzzy search).
type TableItem struct {
	Name string
}

func (t TableItem) Title() string       { return t.Name }
func (t TableItem) Description() string { return "" }
func (t TableItem) FilterValue() string { return t.Name }

// TableSelectedMsg is sent when the user presses enter on a table.
// This is how the table list communicates upward to the parent model —
// through messages, not direct function calls.
type TableSelectedMsg struct {
	Name string
}

// TableListModel wraps bubbles/list.Model. This is the component
// composition pattern: our model contains a child model and delegates
// messages to it.
type TableListModel struct {
	list list.Model
}

// NewTableListModel creates the table list from a slice of table names.
func NewTableListModel(tables []string, width, height int) TableListModel {
	// Convert []string to []list.Item — the list component works with
	// its own Item interface, so we wrap our data.
	items := make([]list.Item, len(tables))
	for i, t := range tables {
		items[i] = TableItem{Name: t}
	}

	// Parent passes the pane border-box dimensions.
	// Content area inside the border = width-2 x height-2.
	// The list must match this exactly or its lines will wrap inside the border.
	contentW := width - 2
	contentH := height - 2
	listDelegate := list.NewDefaultDelegate()
	listDelegate.SetHeight(1)  // 1 line per item (no description line)
	listDelegate.SetSpacing(0) // no blank line between items
	listDelegate.ShowDescription = false
	l := list.New(items, listDelegate, contentW, contentH)
	l.Title = fmt.Sprintf("Tables (%d)", len(tables))
	l.SetShowStatusBar(false) // count is in the title now
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false) // parent's status bar shows keybindings

	// Disable left/right pagination — parent uses these for pane switching.
	l.KeyMap.NextPage.SetEnabled(false)
	l.KeyMap.PrevPage.SetEnabled(false)

	return TableListModel{list: l}
}

// SetSize updates the list dimensions. Called when the terminal resizes.
func (m *TableListModel) SetSize(width, height int) {
	m.list.SetSize(width-2, height-2)
}

// Update delegates messages to the inner list and checks for selection.
// Notice the return type is (TableListModel, tea.Cmd) — not (tea.Model, tea.Cmd).
// Sub-models don't need to satisfy the tea.Model interface; only the root does.
func (m TableListModel) Update(msg tea.Msg) (TableListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't intercept keys when the list is filtering (user is typing)
		if m.list.FilterState() == list.Filtering {
			break
		}
		if msg.String() == "enter" {
			item, ok := m.list.SelectedItem().(TableItem)
			if ok {
				return m, func() tea.Msg {
					return TableSelectedMsg{Name: item.Name}
				}
			}
		}
	}

	// Forward all messages to the inner list so it can handle
	// navigation, filtering, pagination, etc.
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View returns the raw list output — no border wrapping.
// The parent model handles borders/layout so this component
// can be placed in any layout without double-bordering.
func (m TableListModel) View() string {
	return m.list.View()
}
