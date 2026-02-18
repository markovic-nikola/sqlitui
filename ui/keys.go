package ui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines shared key bindings used across all views.
// Centralizing them here (DRY) means one place to change shortcuts.
type KeyMap struct {
	Quit       key.Binding
	SwitchTab  key.Binding
	FocusRight key.Binding
	FocusLeft  key.Binding
	Select     key.Binding
	OpenQuery  key.Binding
}

var Keys = KeyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	SwitchTab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch pane"),
	),
	FocusRight: key.NewBinding(
		key.WithKeys("right"),
		key.WithHelp("→", "open & focus"),
	),
	FocusLeft: key.NewBinding(
		key.WithKeys("left"),
		key.WithHelp("←", "back"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	OpenQuery: key.NewBinding(
		key.WithKeys("ctrl+e"),
		key.WithHelp("ctrl+e", "SQL query"),
	),
}
