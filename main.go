package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2).Foreground(lipgloss.Color("blue"))
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("green"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
	inputStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
)

const (
	modeList = iota
	modeInput
	modeTagInput
)

// Custom keymaps for our list
type listKeyMap struct {
	createNote key.Binding
}

// Define our custom keybindings
var customListKeys = listKeyMap{
	createNote: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new note"),
	),
}

type noteItem struct {
	filename string
	tags     string
}

func (i noteItem) FilterValue() string { 
	// Use both filename and tags for filtering
	return i.filename + " " + i.tags
}

// Implement list.Item interface
func (i noteItem) Title() string {
	// Return filename without .md extension
	return strings.TrimSuffix(i.filename, ".md")
}

func (i noteItem) Description() string { return i.tags }

type model struct {
	list         list.Model
	items        []noteItem
	choice       string
	quitting     bool
	mode         int
	textInput    textinput.Model
	tagInput     textinput.Model
	keys         listKeyMap
	newNoteTags  string
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Enter filename (without .md extension)"
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 40
	
	tagInput := textinput.New()
	tagInput.Placeholder = "Enter tags (e.g. work important todo)"
	tagInput.CharLimit = 100
	tagInput.Width = 40
	
	return model{
		textInput: ti,
		tagInput:  tagInput,
		mode:      modeList,
		keys:      customListKeys,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.list.StartSpinner(),
		tea.EnterAltScreen,
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	
	switch m.mode {
	case modeList:
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
				
			case "n":
				// Only trigger new note creation if not filtering
				if !m.list.SettingFilter() {
					m.mode = modeInput
					return m, textinput.Blink
				}
			}

		case tea.WindowSizeMsg:
			// Use the full height of the terminal, minus 1 for status bar
			h := msg.Height - 1
			m.list.SetHeight(h)
			m.list.SetWidth(msg.Width)
			
			// Return a command to redraw the UI after resize
			return m, tea.ClearScreen
		}

		m.list, cmd = m.list.Update(msg)
		return m, cmd
		
	case modeInput:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				// Return to list mode
				m.mode = modeList
				return m, nil
				
			case "enter":
				// Create and open the new file
				filename := m.textInput.Value()
				if filename != "" {
					// Remove any .md extension the user might have added
					filename = strings.TrimSuffix(filename, ".md")
					// Always add .md extension
					filename += ".md"
					
					m.choice = filename
					// Switch to tag input
					m.mode = modeTagInput
					m.tagInput.Focus()
					return m, textinput.Blink
				}
			}
		}
		
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
		
	case modeTagInput:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				// Return to filename input
				m.mode = modeInput
				m.textInput.Focus()
				return m, nil
				
			case "enter":
				// Save the tags and exit
				m.newNoteTags = m.tagInput.Value()
				return m, tea.Quit
			}
		}
		
		m.tagInput, cmd = m.tagInput.Update(msg)
		return m, cmd
	}
	
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return quitTextStyle.Render("Bye!")
	}
	
	if m.choice != "" && m.mode != modeTagInput && m.mode != modeInput {
		return quitTextStyle.Render("Bye!")
	}
	
	switch m.mode {
	case modeList:
		return m.list.View()
	case modeInput:
		return fmt.Sprintf(
			"\n\n  %s\n\n  %s\n\n",
			"Enter the filename for your new note (without .md extension):",
			m.textInput.View(),
		) + "  (press ESC to cancel)"
	case modeTagInput:
		return fmt.Sprintf(
			"\n\n  %s\n\n  %s\n\n",
			"Enter tags for your note (e.g. work important todo):",
			m.tagInput.View(),
		) + "  (press ESC to go back to filename)"
	}
	
	return ""
}

func formatTagsWithPlus(tags string) string {
	words := strings.Fields(tags)
	tagWords := make([]string, 0)
	
	for _, word := range words {
		// Only add + if it doesn't already have one
		if !strings.HasPrefix(word, "+") {
			tagWords = append(tagWords, "+" + word)
		} else {
			tagWords = append(tagWords, word)
		}
	}
	
	return strings.Join(tagWords, " ")
}

// Capitalize first letter of a string
func capitalizeFirstLetter(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func openInEditor(filename, tags string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		return fmt.Errorf("EDITOR environment variable not set")
	}

	// Create file if it doesn't exist, or open for writing if it does
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	
	// Extract the title from filename (without extension)
	title := strings.TrimSuffix(filename, ".md")
	// Capitalize the first letter of the title
	title = capitalizeFirstLetter(title)
	
	// If tags were provided, write them as the first line
	if tags != "" {
		// Format tags with + for each word
		formattedTags := formatTagsWithPlus(tags)
		file.WriteString("// " + formattedTags + "\n")
	}
	
	// Add the title as a markdown heading
	file.WriteString("# " + title + "\n\n")
	
	file.Close()

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
	
	// Add additional key bindings to the help menu
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			customListKeys.createNote,
		}
	}
	
	// Add additional active key bindings
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			customListKeys.createNote,
		}
	}
	
	// Enable filter mode on startup
	l.ShowFilter()
	l.FilterInput.Focus() // Give focus to the filter input

	m := initialModel()
	m.list = l
	m.items = files
	
	// Use WithAltScreen to use the full terminal space
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
	
	// Get the final model state
	if m, ok := finalModel.(model); ok && m.choice != "" {
		// We need to wait until the program has completely exited before running the editor
		if err := openInEditor(m.choice, m.newNoteTags); err != nil {
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
