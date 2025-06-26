package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const listHeight = 14

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

type item string

func (i item) FilterValue() string { return string(i) }

// Implement list.Item interface
func (i item) Title() string       { return string(i) }
func (i item) Description() string { return "" }

type model struct {
	list     list.Model
	items    []item
	choice   string
	quitting bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.choice = string(i)
			}
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		h, v := listHeight, msg.Height
		if v <= h {
			h = v - 1
		}
		m.list.SetHeight(h)
		m.list.SetWidth(msg.Width)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.quitting {
		return quitTextStyle.Render("Bye!")
	}
	if m.choice != "" {
		return quitTextStyle.Render(fmt.Sprintf("You chose: %s", m.choice))
	}
	return m.list.View()
}

func main() {
	files, err := findMarkdownFiles(".")
	if err != nil {
		fmt.Printf("Error finding markdown files: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Println("No markdown files found in the current directory.")
		os.Exit(0)
	}

	items := make([]list.Item, len(files))
	for i, file := range files {
		items[i] = item(file)
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Simple notes"
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	m := model{list: l}

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

// findMarkdownFiles returns a list of all .md files in the specified directory
func findMarkdownFiles(dir string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}
