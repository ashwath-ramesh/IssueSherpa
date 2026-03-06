package main

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).MarginBottom(1)
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	normalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	statusOpen    = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	statusClosed  = lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
	barStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	headerStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81")).MarginBottom(1)
	filterStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
)

var sortOptions = []string{"created", "updated", "project", "reporter", "status", "title", "source", "id"}

type screen int

const (
	screenMenu screen = iota
	screenAllIssues
	screenOpenIssues
	screenResolvedIssues
	screenLeaderboard
	screenIssueDetail
)

type model struct {
	issues        []Issue
	projects      []string
	projectFilter string // "" = all
	sources       []string
	sourceFilter  string // "" = all
	screen        screen
	prevScreen    screen
	cursor        int
	scroll        int
	filtered      []Issue
	selected      *Issue
	termHeight    int
	sortBy        string
	sortDesc      bool
	searchMode    bool
	searchQuery   string
}

func (m model) activeIssues() []Issue {
	return applyFilters(m.issues, IssueFilter{
		Project:  m.projectFilter,
		Source:   m.sourceFilter,
		Search:   m.searchQuery,
		SortBy:   m.sortBy,
		SortDesc: m.sortDesc,
	})
}

// activeIssuesForScreen returns issues with status filtering applied.
func (m model) activeIssuesForScreen() []Issue {
	active := m.activeIssues()
	switch m.screen {
	case screenOpenIssues:
		return filterByStatus(active, "open")
	case screenResolvedIssues:
		return filterByStatus(active, "resolved")
	default:
		return active
	}
}

func newModel(issues []Issue) model {
	return model{
		issues:      issues,
		projects:    collectProjects(issues),
		sources:     collectSources(issues),
		screen:      screenMenu,
		sortBy:      "created",
		sortDesc:    true,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termHeight = msg.Height
		return m, nil
	case tea.KeyMsg:
		if m.searchMode {
			switch msg.String() {
			case "esc", "enter":
				m.searchMode = false
				return m, nil
			case "backspace", "ctrl+h":
				if m.searchQuery != "" {
					runes := []rune(m.searchQuery)
					m.searchQuery = string(runes[:len(runes)-1])
				}
			case "ctrl+u":
				m.searchQuery = ""
			default:
				if len(msg.Runes) > 0 {
					m.searchQuery += string(msg.Runes)
				}
			}
			m.cursor = 0
			m.scroll = 0
			m.refilter()
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			if m.screen == screenMenu {
				return m, tea.Quit
			}
			m.screen = screenMenu
			m.cursor = 0
			m.scroll = 0
			m.selected = nil
			return m, nil
		case "esc":
			if m.screen == screenIssueDetail {
				m.screen = m.prevScreen
				m.selected = nil
				return m, nil
			}
			m.screen = screenMenu
			m.cursor = 0
			m.scroll = 0
			return m, nil
		case "/":
			m.searchMode = true
			m.searchQuery = ""
			m.cursor = 0
			m.scroll = 0
			m.refilter()
			return m, nil
		case "f":
			if m.screen != screenIssueDetail {
				m.cycleProject()
				m.cursor = 0
				m.scroll = 0
				m.refilter()
				return m, nil
			}
		case "x":
			if m.screen != screenIssueDetail {
				m.cycleSource()
				m.cursor = 0
				m.scroll = 0
				m.refilter()
				return m, nil
			}
		case "s":
			if m.screen != screenIssueDetail {
				m.cycleSort()
				m.cursor = 0
				m.scroll = 0
				m.refilter()
				return m, nil
			}
		case "r":
			if m.screen != screenIssueDetail {
				m.sortDesc = !m.sortDesc
				m.cursor = 0
				m.scroll = 0
				m.refilter()
				return m, nil
			}
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.scroll {
					m.scroll = m.cursor
				}
			}
		case "down", "j":
			maxIdx := m.maxCursorIndex()
			if m.cursor < maxIdx {
				m.cursor++
				visibleLines := m.visibleLines()
				if m.cursor >= m.scroll+visibleLines {
					m.scroll = m.cursor - visibleLines + 1
				}
			}
		case "enter":
			return m.handleEnter()
		}
	}
	return m, nil
}

func (m *model) refilter() {
	m.filtered = m.activeIssuesForScreen()
}

func (m *model) cycleProject() {
	if m.projectFilter == "" {
		if len(m.projects) > 0 {
			m.projectFilter = m.projects[0]
		}
		return
	}
	for i, p := range m.projects {
		if p == m.projectFilter {
			if i+1 < len(m.projects) {
				m.projectFilter = m.projects[i+1]
			} else {
				m.projectFilter = ""
			}
			return
		}
	}
	m.projectFilter = ""
}

func (m *model) cycleSource() {
	if m.sourceFilter == "" {
		if len(m.sources) > 0 {
			m.sourceFilter = m.sources[0]
		}
		return
	}
	for i, s := range m.sources {
		if s == m.sourceFilter {
			if i+1 < len(m.sources) {
				m.sourceFilter = m.sources[i+1]
			} else {
				m.sourceFilter = ""
			}
			return
		}
	}
	m.sourceFilter = ""
}

func (m *model) cycleSort() {
	idx := 0
	for i, option := range sortOptions {
		if option == m.sortBy {
			idx = i
			break
		}
	}
	m.sortBy = sortOptions[(idx+1)%len(sortOptions)]
}

func (m model) visibleLines() int {
	h := m.termHeight
	if h < 10 {
		h = 30
	}
	return h - 8
}

func (m model) maxCursorIndex() int {
	switch m.screen {
	case screenMenu:
		return 4
	case screenAllIssues, screenOpenIssues, screenResolvedIssues:
		return len(m.filtered) - 1
	case screenLeaderboard:
		return len(buildLeaderboard(m.activeIssues())) - 1
	}
	return 0
}

func (m model) handleEnter() (tea.Model, tea.Cmd) {
	active := m.activeIssuesForScreen()
	switch m.screen {
	case screenMenu:
		switch m.cursor {
		case 0:
			m.filtered = active
			m.screen = screenAllIssues
		case 1:
			m.filtered = filterByStatus(active, "open")
			m.screen = screenOpenIssues
		case 2:
			m.filtered = filterByStatus(active, "resolved")
			m.screen = screenResolvedIssues
		case 3:
			m.screen = screenLeaderboard
		case 4:
			return m, tea.Quit
		}
		m.cursor = 0
		m.scroll = 0
	case screenAllIssues, screenOpenIssues, screenResolvedIssues:
		if m.cursor < len(m.filtered) {
			issue := m.filtered[m.cursor]
			m.selected = &issue
			m.prevScreen = m.screen
			m.screen = screenIssueDetail
		}
	}
	return m, nil
}

func (m model) View() string {
	switch m.screen {
	case screenMenu:
		return m.viewMenu()
	case screenAllIssues:
		return m.viewIssueList("All Issues")
	case screenOpenIssues:
		return m.viewIssueList("Open Issues")
	case screenResolvedIssues:
		return m.viewIssueList("Resolved Issues")
	case screenLeaderboard:
		return m.viewLeaderboard()
	case screenIssueDetail:
		return m.viewIssueDetail()
	}
	return ""
}

func (m model) viewMenu() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("IssueSherpa"))
	b.WriteString("  ")
	filter := fmt.Sprintf("[project: %s] [source: %s] [sort: %s%s] [search: %q]",
		m.projectLabel(),
		m.sourceLabel(),
		m.sortBy,
		func() string {
			if m.sortDesc {
				return " (desc)"
			}
			return " (asc)"
		}(),
		m.searchQuery,
	)
	b.WriteString(filterStyle.Render(filter))
	b.WriteString("\n\n")

	active := m.activeIssuesForScreen()
	total := len(active)
	open := len(filterByStatus(active, "open"))
	resolved := len(filterByStatus(active, "resolved"))

	b.WriteString(dimStyle.Render(fmt.Sprintf("  %d issues across %d projects / %d sources", len(m.issues), len(m.projects), len(m.sources))))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("  Showing: %d total / %d open / %d resolved", total, open, resolved)))
	b.WriteString("\n\n")

	items := []string{
		fmt.Sprintf("All Issues (%d)", total),
		fmt.Sprintf("Open Issues (%d)", open),
		fmt.Sprintf("Resolved Issues (%d)", resolved),
		"Leaderboard",
		"Quit",
	}

	for i, item := range items {
		if i == m.cursor {
			b.WriteString(selectedStyle.Render("  > " + item))
		} else {
			b.WriteString(normalStyle.Render("    " + item))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  j/k: navigate | enter: select | f: project | x: source | s: sort | r: reverse | /: search | q: quit"))

	return b.String()
}

func (m model) viewIssueList(title string) string {
	var b strings.Builder
	showResolved := m.screen == screenResolvedIssues

	header := fmt.Sprintf("  %s (%d)", title, len(m.filtered))
	if m.projectFilter != "" {
		header += "  " + filterStyle.Render("["+m.projectFilter+"]")
	}
	if m.sourceFilter != "" {
		header += "  " + filterStyle.Render("["+m.sourceFilter+"]")
	}
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")

	visible := m.visibleLines()
	end := m.scroll + visible
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	if len(m.filtered) == 0 {
		b.WriteString(dimStyle.Render("  No issues match current filters"))
		b.WriteString("\n")
	} else {
		for i := m.scroll; i < end; i++ {
			issue := m.filtered[i]
			status := statusOpen.Render("OPEN")
			if issue.Status == "resolved" {
				status = statusClosed.Render("DONE")
			}

			created := FormatDate(issue.FirstSeen)
			line := fmt.Sprintf("  %-8s %-14s %s  %s", issue.Source, issue.ShortID, status, created)

			shortTitle := issue.Title
			if showResolved {
				line += "  " + issue.Project.Slug + "  " + shortTitle
			} else {
				line += "  " + shortTitle
			}
			if len(shortTitle) > 58 {
				line = line[:len(line)-len(shortTitle)] + shortTitle[:55] + "..."
			}

			if i == m.cursor {
				b.WriteString(selectedStyle.Render(line))
			} else {
				b.WriteString(normalStyle.Render(line))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  enter: details | f: project | x: source | s: sort | r: reverse | /: search | esc/q: back"))

	if m.searchMode {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  / to add search, ESC/Enter to apply"))
	}

	return b.String()
}

func (m model) viewIssueDetail() string {
	if m.selected == nil {
		return "No issue selected"
	}
	i := m.selected
	var b strings.Builder
	b.WriteString(headerStyle.Render("  Issue Detail"))
	b.WriteString("\n\n")

	status := statusOpen.Render("OPEN")
	if i.Status == "resolved" {
		status = statusClosed.Render("RESOLVED")
	}

	rows := []struct{ label, value string }{
		{"Source", i.Source},
		{"ID", i.ShortID},
		{"Project", i.Project.Slug},
		{"Title", i.Title},
		{"Status", ""},
		{"Reporter", i.Reporter},
		{"Events", i.Count},
		{"Users", fmt.Sprintf("%d", i.UserCount)},
		{"Created", FormatDate(i.FirstSeen)},
		{"Last Seen", FormatDate(i.LastSeen)},
	}
	if i.URL != "" {
		rows = append(rows, struct{ label, value string }{"URL", i.URL})
	}
	if i.AssignedTo != nil {
		rows = append(rows, struct{ label, value string }{"Assigned To", i.AssignedTo.Name})
	}

	for _, r := range rows {
		if r.label == "Status" {
			b.WriteString(fmt.Sprintf("  %-14s %s\n", r.label+":", status))
		} else {
			b.WriteString(fmt.Sprintf("  %-14s %s\n", r.label+":", r.value))
		}
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  esc: back to list | q: main menu"))

	return b.String()
}

func (m model) viewLeaderboard() string {
	var b strings.Builder

	header := "  Leaderboard - Issues Reported"
	if m.projectFilter != "" {
		header += "  " + filterStyle.Render("["+m.projectFilter+"]")
	}
	if m.sourceFilter != "" {
		header += "  " + filterStyle.Render("["+m.sourceFilter+"]")
	}
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n\n")

	active := m.activeIssues()
	entries := buildLeaderboard(active)

	maxCount := 0
	if len(entries) > 0 {
		maxCount = entries[0].count
	}

	visible := m.visibleLines()
	end := m.scroll + visible
	if end > len(entries) {
		end = len(entries)
	}

	if len(entries) == 0 {
		b.WriteString(dimStyle.Render("  No issues match current filters"))
		b.WriteString("\n")
	} else {
		for i := m.scroll; i < end; i++ {
			e := entries[i]
			pct := float64(e.count) / float64(len(active)) * 100

			barLen := 0
			if maxCount > 0 {
				barLen = e.count * 30 / maxCount
			}
			bar := barStyle.Render(strings.Repeat("█", barLen))
			line := fmt.Sprintf("  %-25s %3d (%4.1f%%)  %s", e.name, e.count, pct, bar)

			if i == m.cursor {
				b.WriteString(selectedStyle.Render("" + line[0:]))
			} else {
				b.WriteString(normalStyle.Render(line))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	sort := m.sortBy
	if m.sortDesc {
		sort += " desc"
	} else {
		sort += " asc"
	}
	b.WriteString(dimStyle.Render(fmt.Sprintf("  f: project | x: source | /: search | q/esc: back | sort: %s", sort)))

	return b.String()
}

func (m model) projectLabel() string {
	if m.projectFilter == "" {
		return "all"
	}
	return m.projectFilter
}

func (m model) sourceLabel() string {
	if m.sourceFilter == "" {
		return "all"
	}
	return m.sourceFilter
}

func truncateText(value string, max int) string {
	if max <= 3 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max-3]) + "..."
}


func issueByID(id string, issues []Issue) *Issue {
	for _, issue := range issues {
		if issue.ShortID == id {
			copy := issue
			return &copy
		}
	}
	return nil
}

func runesLen(value string) int {
	return len([]rune(value))
}

func sortByLabel(sortBy string) string {
	return sortBy
}

func activeIssueCount(issues []Issue) int {
	return len(issues)
}

func (m model) projectMenu() []string {
	return m.projects
}

func (m *model) sortAll() {
	m.filtered = sortIssues(m.issues, m.sortBy, m.sortDesc)
}

func (m model) sortByCurrent() string {
	sort.Slice(m.filtered, func(i, j int) bool {
		return m.filtered[i].ID < m.filtered[j].ID
	})
	return m.sortBy
}

func (m *model) applyFilters() {
	m.filtered = m.activeIssuesForScreen()
}
