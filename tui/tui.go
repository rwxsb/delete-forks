package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	gh "github.com/suheybbecerek/delete-forks/github"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF6B6B")).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B")).
			Bold(true)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EEEEEE"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

	checkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			MarginTop(1)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFCC00")).
			Bold(true)

	deletedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")).
			Bold(true)

	failedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000"))
)

// --- Messages ---

type forksLoadedMsg struct{ forks []gh.Repo }
type forksErrorMsg struct{ err error }
type forkDeletedMsg struct {
	index int
	err   error
}
type allDeletedMsg struct{}

// --- Phase ---

type phase int

const (
	phaseLoading phase = iota
	phaseList
	phaseConfirm
	phaseDeleting
	phaseDone
)

// --- Fork item state ---

type forkState int

const (
	forkPending forkState = iota
	forkDeleting
	forkDeleted
	forkFailed
)

type forkItem struct {
	repo  gh.Repo
	state forkState
	err   error
}

// --- Model ---

type Model struct {
	client   *gh.Client
	username string

	phase   phase
	spinner spinner.Model

	forks    []forkItem
	cursor   int
	selected map[int]bool

	deleteIndex int
	deleteTotal int
	deleteErrs  int

	width  int
	height int
	err    error
}

func NewModel(client *gh.Client, username string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B"))
	return Model{
		client:   client,
		username: username,
		phase:    phaseLoading,
		spinner:  s,
		selected: make(map[int]bool),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.loadForks())
}

func (m Model) loadForks() tea.Cmd {
	return func() tea.Msg {
		forks, err := m.client.ListForks(m.username)
		if err != nil {
			return forksErrorMsg{err}
		}
		return forksLoadedMsg{forks}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case forksLoadedMsg:
		m.phase = phaseList
		m.forks = make([]forkItem, len(msg.forks))
		for i, f := range msg.forks {
			m.forks[i] = forkItem{repo: f, state: forkPending}
		}
		return m, nil

	case forksErrorMsg:
		m.err = msg.err
		m.phase = phaseList
		return m, nil

	case forkDeletedMsg:
		if msg.err != nil {
			m.forks[msg.index].state = forkFailed
			m.forks[msg.index].err = msg.err
			m.deleteErrs++
		} else {
			m.forks[msg.index].state = forkDeleted
		}
		return m, m.deleteNext()

	case allDeletedMsg:
		m.phase = phaseDone
		return m, nil
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.phase {
	case phaseList:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.forks)-1 {
				m.cursor++
			}
		case " ":
			m.selected[m.cursor] = !m.selected[m.cursor]
		case "a":
			allSelected := len(m.selected) == len(m.forks)
			if allSelected {
				m.selected = make(map[int]bool)
			} else {
				for i := range m.forks {
					m.selected[i] = true
				}
			}
		case "enter", "d":
			if len(m.selectedIndices()) == 0 {
				return m, nil
			}
			m.phase = phaseConfirm
		}

	case phaseConfirm:
		switch msg.String() {
		case "y", "Y":
			m.phase = phaseDeleting
			m.deleteIndex = 0
			m.deleteTotal = len(m.selectedIndices())
			return m, tea.Batch(m.spinner.Tick, m.deleteNext())
		case "n", "N", "escape":
			m.phase = phaseList
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case phaseDone:
		return m, tea.Quit

	default:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *Model) selectedIndices() []int {
	var indices []int
	for i, sel := range m.selected {
		if sel {
			indices = append(indices, i)
		}
	}
	sort.Ints(indices)
	return indices
}

func (m *Model) deleteNext() tea.Cmd {
	indices := m.selectedIndices()
	// find next pending
	for _, idx := range indices {
		if m.forks[idx].state == forkPending {
			m.forks[idx].state = forkDeleting
			i := idx
			repo := m.forks[i].repo
			return func() tea.Msg {
				err := m.client.DeleteRepo(repo.Owner.Login, repo.Name)
				return forkDeletedMsg{index: i, err: err}
			}
		}
	}
	return func() tea.Msg { return allDeletedMsg{} }
}

func (m Model) View() string {
	switch m.phase {
	case phaseLoading:
		return m.viewLoading()
	case phaseList:
		return m.viewList()
	case phaseConfirm:
		return m.viewConfirm()
	case phaseDeleting:
		return m.viewDeleting()
	case phaseDone:
		return m.viewDone()
	}
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}
	return ""
}

func (m Model) viewLoading() string {
	return fmt.Sprintf("\n  %s Loading your forks...\n", m.spinner.View())
}

func (m Model) viewList() string {
	var b strings.Builder

	title := fmt.Sprintf("🍴 Forks for %s (%d total)", m.username, len(m.forks))
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("  Press q to quit"))
		return b.String()
	}

	if len(m.forks) == 0 {
		b.WriteString(dimStyle.Render("  No forks found. You're clean!"))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("  Press q to quit"))
		return b.String()
	}

	sel := len(m.selectedIndices())
	b.WriteString(statusBarStyle.Render(fmt.Sprintf("  %d selected", sel)))
	b.WriteString("\n\n")

	// Scrolling viewport
	visibleLines := m.height - 10
	if visibleLines < 5 {
		visibleLines = 15
	}
	start := 0
	if m.cursor >= visibleLines {
		start = m.cursor - visibleLines + 1
	}
	end := start + visibleLines
	if end > len(m.forks) {
		end = len(m.forks)
	}

	for i := start; i < end; i++ {
		f := m.forks[i]
		cursor := "  "
		if m.cursor == i {
			cursor = "▸ "
		}

		check := "○"
		if m.selected[i] {
			check = checkStyle.Render("●")
		}

		name := f.repo.FullName
		parent := ""
		if f.repo.Parent != nil {
			parent = dimStyle.Render(fmt.Sprintf(" ← %s", f.repo.Parent.FullName))
		}

		lang := ""
		if f.repo.Language != nil {
			lang = dimStyle.Render(fmt.Sprintf(" [%s]", f.repo.Language.Name))
		}

		line := fmt.Sprintf("%s%s %s%s%s", cursor, check, name, parent, lang)
		if m.cursor == i {
			line = selectedStyle.Render(line)
		} else {
			line = normalStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	if len(m.forks) > visibleLines {
		b.WriteString(dimStyle.Render(fmt.Sprintf("\n  showing %d-%d of %d", start+1, end, len(m.forks))))
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("  ↑/↓ navigate • space select • a select all • d/enter delete • q quit"))

	return b.String()
}

func (m Model) viewConfirm() string {
	indices := m.selectedIndices()
	count := len(indices)
	var b strings.Builder

	b.WriteString(titleStyle.Render("⚠️  Confirm Deletion"))
	b.WriteString("\n\n")
	b.WriteString(errorStyle.Render(fmt.Sprintf("  You are about to permanently delete %d fork(s):\n", count)))
	b.WriteString("\n")

	// Scrollable list — reserve lines for header (5) + footer (4)
	maxVisible := m.height - 9
	if maxVisible < 3 {
		maxVisible = 3
	}

	start := 0
	end := count
	if count > maxVisible {
		end = maxVisible
	}

	for _, idx := range indices[start:end] {
		b.WriteString(fmt.Sprintf("    • %s\n", m.forks[idx].repo.FullName))
	}

	if count > maxVisible {
		b.WriteString(dimStyle.Render(fmt.Sprintf("    ... and %d more\n", count-maxVisible)))
	}

	b.WriteString(fmt.Sprintf("\n  %s\n", errorStyle.Render("This action CANNOT be undone.")))
	b.WriteString(helpStyle.Render("\n  Press y to confirm, n to cancel"))

	return b.String()
}

func (m Model) viewDeleting() string {
	indices := m.selectedIndices()
	var b strings.Builder

	// Count progress
	done := 0
	for _, idx := range indices {
		if m.forks[idx].state == forkDeleted || m.forks[idx].state == forkFailed {
			done++
		}
	}

	b.WriteString(titleStyle.Render(fmt.Sprintf("🗑  Deleting forks... (%d/%d)", done, len(indices))))
	b.WriteString("\n\n")

	// Scrollable — reserve lines for header (4) + bottom padding (2)
	maxVisible := m.height - 6
	if maxVisible < 3 {
		maxVisible = 3
	}

	// Auto-scroll: keep the currently-active item visible
	activePos := 0
	for i, idx := range indices {
		if m.forks[idx].state == forkDeleting {
			activePos = i
			break
		}
		if m.forks[idx].state == forkPending {
			activePos = i
			break
		}
	}

	start := 0
	if activePos >= maxVisible {
		start = activePos - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(indices) {
		end = len(indices)
	}

	for _, idx := range indices[start:end] {
		f := m.forks[idx]
		var status string
		switch f.state {
		case forkPending:
			status = dimStyle.Render("  ○ " + f.repo.FullName)
		case forkDeleting:
			status = fmt.Sprintf("  %s %s", m.spinner.View(), f.repo.FullName)
		case forkDeleted:
			status = deletedStyle.Render("  ✓ " + f.repo.FullName)
		case forkFailed:
			status = failedStyle.Render(fmt.Sprintf("  ✗ %s (%v)", f.repo.FullName, f.err))
		}
		b.WriteString(status)
		b.WriteString("\n")
	}

	if len(indices) > maxVisible {
		b.WriteString(dimStyle.Render(fmt.Sprintf("\n  showing %d-%d of %d", start+1, end, len(indices))))
	}

	return b.String()
}

func (m Model) viewDone() string {
	var b strings.Builder

	total := m.deleteTotal
	failed := m.deleteErrs
	deleted := total - failed

	b.WriteString(titleStyle.Render("✅ Done!"))
	b.WriteString("\n\n")
	b.WriteString(deletedStyle.Render(fmt.Sprintf("  %d fork(s) deleted successfully", deleted)))
	if failed > 0 {
		b.WriteString("\n")
		b.WriteString(failedStyle.Render(fmt.Sprintf("  %d fork(s) failed to delete", failed)))
	}
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("  Press any key to exit"))

	return b.String()
}
