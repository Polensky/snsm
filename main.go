package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const listHeight = 14

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2).Foreground(lipgloss.Color("blue"))
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("green"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

type noteItem struct {
	filename string
	tags     string
}

func (i noteItem) FilterValue() string { 
	// Use both filename and tags for filtering
	return i.filename + " " + i.tags
}

// Implement list.Item interface
func (i noteItem) Title() string       { return i.filename }
func (i noteItem) Description() string { return i.tags }

type model struct {
	list     list.Model
	items    []noteItem
	choice   string
	quitting bool
}

func (m model) Init() tea.Cmd {
	return m.list.StartSpinner()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(noteItem)
			if ok {
				m.choice = i.filename
				return m, tea.Quit
			}
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
	if m.quitting || m.choice != "" {
		return quitTextStyle.Render("Bye!")
	}
	return m.list.View()
}

func openInEditor(filename string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		return fmt.Errorf("EDITOR environment variable not set")
	}

	cmd := exec.Command(editor, filename)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	return cmd.Run()
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
	for i, fileInfo := range files {
		items[i] = fileInfo
	}

	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 0, 0)
	l.Title = "Simple notes"
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle
	
	// Change "item/items" to "note/notes" in status messages
	l.SetStatusBarItemName("note", "notes")
	
	// Enable filter mode on startup
	l.ShowFilter()
	l.FilterInput.Focus() // Give focus to the filter input

	m := model{list: l}
	
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
	
	// Get the final model state
	if m, ok := finalModel.(model); ok && m.choice != "" {
		// We need to wait until the program has completely exited before running the editor
		if err := openInEditor(m.choice); err != nil {
			fmt.Printf("Error opening file in editor: %v\n", err)
			os.Exit(1)
		}
	}
}

// Extract tags that start with "+" from a string
func extractTags(line string) string {
	var tags []string
	tagRegex := regexp.MustCompile(`\+\w+`)
	
	matches := tagRegex.FindAllString(line, -1)
	if matches != nil {
		tags = matches
	}
	
	return strings.Join(tags, " ")
}

// findMarkdownFiles returns a list of all .md files in the specified directory
// along with tags extracted from their first line
func findMarkdownFiles(dir string) ([]noteItem, error) {
	var files []noteItem

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		// Skip hidden files (dot files) and directories
		if !entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") && strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			filename := entry.Name()
			tags := ""
			
			// Open file and read first line to extract tags
			file, err := os.Open(filename)
			if err == nil {
				scanner := bufio.NewScanner(file)
				if scanner.Scan() {
					firstLine := scanner.Text()
					// If the first line starts with //, extract tags
					if strings.HasPrefix(firstLine, "//") {
						tags = extractTags(firstLine)
					}
				}
				file.Close()
			}
			
			files = append(files, noteItem{
				filename: filename,
				tags:     tags,
			})
		}
	}

	return files, nil
}
