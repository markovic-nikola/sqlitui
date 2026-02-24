package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
		prefix := label + " : "
		indentWidth := lipgloss.Width(prefix)
		valueWidth := contentWidth - indentWidth
		if valueWidth < 10 {
			valueWidth = 10
		}

		wrapped := wrapText(val, valueWidth)
		b.WriteString(prefix + wrapped[0] + "\n")
		indent := strings.Repeat(" ", indentWidth)
		for _, line := range wrapped[1:] {
			b.WriteString(indent + line + "\n")
		}
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

// wrapText breaks text into lines that fit within maxWidth visible characters.
// It splits on spaces when possible, hard-breaking mid-word only when a single
// word exceeds maxWidth.
func wrapText(text string, maxWidth int) []string {
	if maxWidth <= 0 || lipgloss.Width(text) <= maxWidth {
		return []string{text}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	current := words[0]
	for _, word := range words[1:] {
		if lipgloss.Width(current+" "+word) <= maxWidth {
			current += " " + word
		} else {
			lines = append(lines, current)
			current = word
		}
	}
	lines = append(lines, current)

	// Hard-break any lines where a single word exceeds maxWidth.
	var result []string
	for _, line := range lines {
		if lipgloss.Width(line) <= maxWidth {
			result = append(result, line)
			continue
		}
		runes := []rune(line)
		for len(runes) > 0 {
			end := len(runes)
			for end > 0 && lipgloss.Width(string(runes[:end])) > maxWidth {
				end--
			}
			if end == 0 {
				end = 1
			}
			result = append(result, string(runes[:end]))
			runes = runes[end:]
		}
	}
	return result
}
