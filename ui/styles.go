package ui

import "github.com/charmbracelet/lipgloss"

// All styles live here — one place to change the look of the entire app.
// lipgloss works like CSS: you build styles by chaining methods, and
// they're immutable (each method returns a new style).

var (
	AppStyle = lipgloss.NewStyle().Margin(1, 2)

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	// StatusBarInfoStyle is for the left section showing table name and page info.
	StatusBarInfoStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Background(lipgloss.Color("236")).
				Bold(true).
				Padding(0, 1)

	// StatusBarKeyStyle highlights the key binding name (e.g. "f", "enter").
	StatusBarKeyStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Background(lipgloss.Color("235"))

	// StatusBarDescStyle is for the key description (e.g. "filter", "detail").
	StatusBarDescStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("242")).
				Background(lipgloss.Color("235"))

	// StatusBarBgStyle is the base background for the full status bar.
	StatusBarBgStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("235"))

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	// FocusedPaneStyle has a bright border — applied to the active panel.
	// Width/Height are set dynamically at render time via .Width()/.Height().
	FocusedPaneStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62"))

	// UnfocusedPaneStyle has a dim border — applied to the inactive panel.
	UnfocusedPaneStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240"))

	// PopupStyle wraps the row detail modal. Bright border + background
	// so it visually "floats" above the split pane behind it.
	PopupStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")).
			Padding(1, 2)

	// PopupLabelStyle is for the column names in the key-value list.
	PopupLabelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("63"))

	Logo = TitleStyle.Render(
		" ▄▄▄▄  ▄▄▄  ▄▄    ▄▄ ▄▄▄▄▄▄ ▄▄ ▄▄ ▄▄ \n" +
			"███▄▄ ██▀██ ██    ██   ██   ██ ██ ██ \n" +
			"▄▄██▀ ▀███▀ ██▄▄▄ ██   ██   ▀███▀ ██ \n" +
			"         ▀▀                          ")
)
