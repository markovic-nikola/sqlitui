package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// CloseDetailMsg is sent when the user dismisses the row detail popup.
type CloseDetailMsg struct{}

// RowDetailModel displays a single row's data as a vertical key-value list
// inside a scrollable viewport. This is the "popup" component.
type RowDetailModel struct {
	viewport viewport.Model
	width    int
	height   int
}

// NewRowDetailModel creates the popup. It renders column:value pairs
// with aligned colons so the values line up neatly.
func NewRowDetailModel(columns, values []string, termWidth, termHeight int) RowDetailModel {
	// Size the popup to ~60% of terminal width, ~70% of terminal height.
	popupWidth := termWidth * 60 / 100
	popupHeight := termHeight * 70 / 100
	if popupWidth < 40 {
		popupWidth = 40
	}
	if popupHeight < 10 {
		popupHeight = 10
	}

	// Account for PopupStyle border (2) + padding (2 each side = 4).
	// The viewport content area is smaller than the popup box.
	// Extra -3 vertical: title line + blank line + help line.
	contentWidth := popupWidth - 6
	contentHeight := popupHeight - 4 - 3

	// Find the longest column name for alignment.
	maxLabel := 0
	for _, col := range columns {
		if len(col) > maxLabel {
			maxLabel = len(col)
		}
	}

	// Build the key-value content.
	var b strings.Builder
	for i, col := range columns {
		val := ""
		if i < len(values) {
			val = values[i]
		}
		// Left-pad column names so the colons align.
		label := PopupLabelStyle.Render(fmt.Sprintf("%*s", maxLabel, col))
		b.WriteString(label + " : " + val + "\n")
	}

	vp := viewport.New(contentWidth, contentHeight)
	vp.SetContent(b.String())

	return RowDetailModel{
		viewport: vp,
		width:    popupWidth,
		height:   popupHeight,
	}
}

func (m RowDetailModel) Update(msg tea.Msg) (RowDetailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "enter":
			// Close the popup by sending a message to the parent.
			return m, func() tea.Msg { return CloseDetailMsg{} }
		}
	}

	// Delegate to viewport for up/down scrolling.
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the viewport content inside the popup border.
func (m RowDetailModel) View() string {
	title := TitleStyle.Render(" Row Detail ")
	content := m.viewport.View()
	help := StatusBarStyle.Render("↑↓: scroll | esc/enter: close")

	return PopupStyle.
		Width(m.width - 2).   // -2 for border chars
		Height(m.height - 2). // -2 for border chars
		Render(title + "\n\n" + content + "\n" + help)
}
