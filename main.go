package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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
	titleStyle        = lipgloss.NewStyle().MarginLeft(2).Foreground(lipgloss.Color("white"))
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("green"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
	inputStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	
	// Pill styling
	tagPillStyle      = lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("255"))
	selectedTagPillStyle = lipgloss.NewStyle().Background(lipgloss.Color("99")).Foreground(lipgloss.Color("255")).Bold(true)
	
	// Circle styling - foreground matches the background of the pill
	circleStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("62"))
	selectedCircleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("99"))
)

const (
	modeList = iota
	modeInput
	modeTagInput
	
	// Unicode half circles for pill styling
	leftHalfCircle  = ""
	rightHalfCircle = ""
)

// Custom item delegate for styling the list items
type customItemDelegate struct {
	list.DefaultDelegate
}

func NewCustomDelegate() list.ItemDelegate {
	delegate := customItemDelegate{
		DefaultDelegate: list.NewDefaultDelegate(),
	}
	
	// Style base delegate
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.
		Foreground(lipgloss.Color("255")) // White for unselected items
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("10")). // Bright green for selected items
		Bold(true)
	
	// Clear description styles (we'll handle them in Render)
	delegate.Styles.NormalDesc = lipgloss.NewStyle()
	delegate.Styles.SelectedDesc = lipgloss.NewStyle()
	
	return delegate
}

// Override Render to customize the appearance of list items
func (d customItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(noteItem)
	if !ok {
		d.DefaultDelegate.Render(w, m, index, listItem)
		return
	}

	isSelected := index == m.Index()
	var title string
	var tags string

	if isSelected {
		title = d.Styles.SelectedTitle.Render(item.Title())
	} else {
		title = d.Styles.NormalTitle.Render(item.Title())
	}

	// Format tags as pills
	if item.tags != "" {
		tagWords := strings.Fields(item.tags)
		var formattedTags []string
		
		for _, tag := range tagWords {
			// Remove + prefix if present
			tagText := tag
			if strings.HasPrefix(tagText, "+") {
				tagText = tagText[1:]
			}
			
			// Style each tag as a pill with matching circle foreground
			if isSelected {
				formattedTags = append(formattedTags, 
					selectedCircleStyle.Render(leftHalfCircle) + 
					selectedTagPillStyle.Render(tagText) + 
					selectedCircleStyle.Render(rightHalfCircle))
			} else {
				formattedTags = append(formattedTags, 
					circleStyle.Render(leftHalfCircle) + 
					tagPillStyle.Render(tagText) + 
					circleStyle.Render(rightHalfCircle))
			}
		}
		
		tags = strings.Join(formattedTags, " ")
	}

	// Write title and tags with spacing
	fmt.Fprintf(w, "%s\n", title)
	if tags != "" {
		fmt.Fprintf(w, "  %s", tags)
	}
}

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
	notesDir     string
}

func initialModel(notesDir string) model {
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
		notesDir:  notesDir,
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

// Expand ~ to home directory
func expandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path // Return original if we can't expand
		}
		return filepath.Join(homeDir, path[2:])
	}
	return path
}

func openInEditor(fullPath string, tags string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		return fmt.Errorf("EDITOR environment variable not set")
	}

	// Create file if it doesn't exist, or open for writing if it does
	file, err := os.OpenFile(fullPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	
	// Extract the title from filename (without extension)
	filename := filepath.Base(fullPath)
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

	cmd := exec.Command(editor, fullPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	return cmd.Run()
}

func main() {
	// Expand the path to the notes directory
	notesDir := expandTilde("~/notes/")
	
	// Ensure the notes directory exists
	if err := os.MkdirAll(notesDir, 0755); err != nil {
		fmt.Printf("Error creating notes directory: %v\n", err)
		os.Exit(1)
	}
	
	files, err := findMarkdownFiles(notesDir)
	if err != nil {
		fmt.Printf("Error finding markdown files: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Printf("No markdown files found in %s\n", notesDir)
		os.Exit(0)
	}

	items := make([]list.Item, len(files))
	for i, fileInfo := range files {
		items[i] = fileInfo
	}

	delegate := NewCustomDelegate()
	l := list.New(items, delegate, 0, 0)
	l.Title = fmt.Sprintf("Notes at %s", notesDir)
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

	m := initialModel(notesDir)
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
		// Create full file path in the notes directory
		fullPath := filepath.Join(m.notesDir, m.choice)
		
		// We need to wait until the program has completely exited before running the editor
		if err := openInEditor(fullPath, m.newNoteTags); err != nil {
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
			filePath := filepath.Join(dir, filename)
			file, err := os.Open(filePath)
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
