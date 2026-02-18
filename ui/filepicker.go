package ui

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/markovic-nikola/sqlitui/db"
)

type pickerFocus int

const (
	focusInput pickerFocus = iota
	focusList
)

// dbOpenedMsg is sent when a database is successfully opened.
type dbOpenedMsg struct {
	db     *sql.DB
	tables []string
}

// FilePickerModel shows a text input for typing a path and a list of
// SQLite files found in the current directory.
type FilePickerModel struct {
	input   textinput.Model
	files   []string
	cursor  int
	focused pickerFocus
	pathErr string
	width   int
	height  int
}

// validExtensions are the file extensions we recognize as SQLite databases.
var validExtensions = map[string]bool{
	".db":      true,
	".sqlite":  true,
	".sqlite3": true,
}

func NewFilePickerModel() FilePickerModel {
	ti := textinput.New()
	ti.Placeholder = "/path/to/database.db"
	ti.Width = 50

	files := findSQLiteFiles()

	focused := focusInput
	if len(files) > 0 {
		focused = focusList
	} else {
		ti.Focus()
	}

	return FilePickerModel{
		input:   ti,
		files:   files,
		focused: focused,
	}
}

func (m FilePickerModel) Init() tea.Cmd {
	if m.focused == focusInput {
		return textinput.Blink
	}
	return nil
}

func (m FilePickerModel) Update(msg tea.Msg) (FilePickerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			return m.submit()

		case tea.KeyEsc, tea.KeyCtrlC:
			return m, tea.Quit

		case tea.KeyUp:
			if len(m.files) == 0 {
				return m, nil
			}
			if m.focused == focusInput {
				// Move from input to last file in list.
				return m.switchToList(len(m.files) - 1)
			}
			if m.cursor > 0 {
				m.cursor--
			} else {
				// At top of list, move to input.
				return m.switchToInput()
			}
			return m, nil

		case tea.KeyDown:
			if len(m.files) == 0 {
				return m, nil
			}
			if m.focused == focusInput {
				// Move from input to first file in list.
				return m.switchToList(0)
			}
			if m.cursor < len(m.files)-1 {
				m.cursor++
			}
			return m, nil
		}

		if m.focused == focusList {
			switch msg.String() {
			case "k":
				if m.cursor > 0 {
					m.cursor--
				} else {
					return m.switchToInput()
				}
				return m, nil
			case "j":
				if m.cursor < len(m.files)-1 {
					m.cursor++
				}
				return m, nil
			}
		}
	}

	if m.focused == focusInput {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m FilePickerModel) View() string {
	boxWidth := 50

	// Input box.
	inputStyle := UnfocusedPaneStyle
	if m.focused == focusInput {
		inputStyle = FocusedPaneStyle
	}
	inputBox := inputStyle.
		Width(boxWidth).
		Padding(0, 1).
		Render(m.input.View())

	// File list box.
	var fileListBox string
	if len(m.files) > 0 {
		listStyle := UnfocusedPaneStyle
		if m.focused == focusList {
			listStyle = FocusedPaneStyle
		}

		var lines []string
		for i, f := range m.files {
			if m.focused == focusList && i == m.cursor {
				lines = append(lines, TitleStyle.Render(" > "+f))
			} else {
				lines = append(lines, "   "+f)
			}
		}

		fileListBox = listStyle.
			Width(boxWidth).
			Padding(0, 1).
			Render(strings.Join(lines, "\n"))
	}

	errLine := ""
	if m.pathErr != "" {
		errLine = ErrorStyle.Render("Error: " + m.pathErr)
	}

	help := StatusBarStyle.Render("enter: open | esc: quit")

	sections := []string{
		Logo,
		"",
		StatusBarStyle.Render("  Database path"),
		inputBox,
	}

	if fileListBox != "" {
		sections = append(sections, "", StatusBarStyle.Render("  Files in current directory"), fileListBox)
	}

	if errLine != "" {
		sections = append(sections, "", errLine)
	}

	sections = append(sections, "", help)

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}
	return content
}

func (m FilePickerModel) switchToList(cursor int) (FilePickerModel, tea.Cmd) {
	m.focused = focusList
	m.cursor = cursor
	m.input.Blur()
	return m, nil
}

func (m FilePickerModel) switchToInput() (FilePickerModel, tea.Cmd) {
	m.focused = focusInput
	cmd := m.input.Focus()
	return m, cmd
}

func (m FilePickerModel) submit() (FilePickerModel, tea.Cmd) {
	var path string
	if m.focused == focusList && len(m.files) > 0 {
		path = m.files[m.cursor]
	} else {
		path = m.input.Value()
	}

	if path == "" {
		return m, nil
	}

	if err := validatePath(path); err != nil {
		m.pathErr = err.Error()
		return m, nil
	}

	database, err := db.Open(path)
	if err != nil {
		m.pathErr = err.Error()
		return m, nil
	}

	tables, err := db.ListTables(database)
	if err != nil {
		database.Close()
		m.pathErr = err.Error()
		return m, nil
	}

	return m, func() tea.Msg {
		return dbOpenedMsg{db: database, tables: tables}
	}
}

// validatePath checks that the path points to an existing regular file
// with a recognized SQLite extension.
func validatePath(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", path)
	}
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", path)
	}
	ext := strings.ToLower(filepath.Ext(path))
	if !validExtensions[ext] {
		return fmt.Errorf("unsupported file extension %q (expected .db, .sqlite, or .sqlite3)", ext)
	}
	return nil
}

// findSQLiteFiles returns SQLite files in the current working directory.
func findSQLiteFiles() []string {
	entries, err := os.ReadDir(".")
	if err != nil {
		return nil
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if validExtensions[ext] {
			files = append(files, e.Name())
		}
	}
	return files
}
