package main

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/sci-ecommerce/issuesherpa/internal/core"
)

var (
	steelStyle  = lipgloss.NewStyle()
	inkStyle    = lipgloss.NewStyle()
	ashStyle    = lipgloss.NewStyle()
	amberStyle  = lipgloss.NewStyle()
	emberStyle  = lipgloss.NewStyle()
	mossStyle   = lipgloss.NewStyle()
	activeStyle = lipgloss.NewStyle()
)

func init() {
	applyTUIStyles()
}

func applyTUIStyles() {
	if usesNoColor() {
		steelStyle = lipgloss.NewStyle().Bold(true)
		inkStyle = lipgloss.NewStyle()
		ashStyle = lipgloss.NewStyle()
		amberStyle = lipgloss.NewStyle().Bold(true)
		emberStyle = lipgloss.NewStyle().Bold(true)
		mossStyle = lipgloss.NewStyle().Bold(true)
		activeStyle = lipgloss.NewStyle().Bold(true)
		return
	}

	steelStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))
	inkStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	ashStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	amberStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("215"))
	emberStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("203"))
	mossStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("107"))
	activeStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))
}

func usesNoColor() bool {
	if strings.TrimSpace(os.Getenv("NO_COLOR")) != "" {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("CLICOLOR")), "0") {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(os.Getenv("TERM")), "dumb")
}

var sortOptions = []string{"updated", "created", "project", "reporter", "status", "title", "source", "id"}

type triageScope string

const (
	scopeAll      triageScope = "all"
	scopeOpen     triageScope = "open"
	scopeResolved triageScope = "resolved"
)

type layoutInfo struct {
	width        int
	height       int
	bodyHeight   int
	queueWidth   int
	queueHeight  int
	detailWidth  int
	detailHeight int
	stacked      bool
}

type model struct {
	issues           []Issue
	active           []Issue
	visible          []Issue
	leaderboard      []core.LeaderboardEntry
	projects         []string
	sources          []string
	projectFilter    string
	sourceFilter     string
	cacheInfo        core.CacheInfo
	offline          bool
	scope            triageScope
	sortBy           string
	sortDesc         bool
	openCount        int
	resolvedCount    int
	cursor           int
	scroll           int
	restoreCursorPos int
	restoreScrollPos int
	width            int
	height           int
	searchMode       bool
	searchQuery      string
	searchDraft      string
	searchBefore     string
	zoomPreview      bool
}

func newModel(issues []Issue, cacheInfo core.CacheInfo, offline bool) *model {
	m := &model{
		issues:           issues,
		projects:         core.CollectProjects(issues),
		sources:          core.CollectSources(issues),
		cacheInfo:        cacheInfo,
		offline:          offline,
		scope:            scopeOpen,
		sortBy:           "updated",
		sortDesc:         true,
		restoreCursorPos: 0,
		restoreScrollPos: 0,
		width:            120,
		height:           32,
	}
	m.syncData("")
	return m
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureSelectionVisible()
		return m, nil
	case tea.KeyMsg:
		if m.searchMode {
			return m.updateSearch(msg)
		}

		if m.zoomPreview {
			switch msg.String() {
			case "esc", "enter":
				m.zoomPreview = false
				if m.restoreCursorPos >= 0 && m.restoreCursorPos < len(m.visible) {
					m.cursor = m.restoreCursorPos
				}
				if m.restoreScrollPos >= 0 && m.restoreScrollPos <= len(m.visible) {
					m.scroll = m.restoreScrollPos
				}
				m.ensureSelectionVisible()
				return m, nil
			case "o":
				issue := m.selectedIssue()
				if issue != nil && strings.TrimSpace(issue.URL) != "" {
					return m, openIssueURL(issue.URL)
				}
				return m, nil
			default:
				return m, nil
			}
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
		case "/":
			m.searchMode = true
			m.searchBefore = m.searchQuery
			m.searchDraft = m.searchQuery
			return m, nil
		case "1":
			m.scope = scopeAll
			m.syncData(m.selectedID())
			return m, nil
		case "2":
			m.scope = scopeOpen
			m.syncData(m.selectedID())
			return m, nil
		case "3":
			m.scope = scopeResolved
			m.syncData(m.selectedID())
			return m, nil
		case "p":
			m.projectFilter = cycleFilterValue(m.projects, m.projectFilter)
			m.syncData("")
			return m, nil
		case "v":
			m.sourceFilter = cycleFilterValue(m.sources, m.sourceFilter)
			m.syncData("")
			return m, nil
		case "s":
			m.cycleSort()
			m.syncData(m.selectedID())
			return m, nil
		case "r":
			m.sortDesc = !m.sortDesc
			m.syncData(m.selectedID())
			return m, nil
		case "enter":
			if len(m.visible) > 0 {
				m.restoreCursorPos = m.cursor
				m.restoreScrollPos = m.scroll
				m.zoomPreview = !m.zoomPreview
			}
			return m, nil
		case "o":
			issue := m.selectedIssue()
			if issue != nil && strings.TrimSpace(issue.URL) != "" {
				return m, openIssueURL(issue.URL)
			}
			return m, nil
		case "home", "g":
			m.cursor = 0
			m.ensureSelectionVisible()
			return m, nil
		case "end", "G":
			m.cursor = max(len(m.visible)-1, 0)
			m.ensureSelectionVisible()
			return m, nil
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.ensureSelectionVisible()
			}
			return m, nil
		case "down", "j":
			if m.cursor < len(m.visible)-1 {
				m.cursor++
				m.ensureSelectionVisible()
			}
			return m, nil
		case "pgup":
			m.cursor -= m.queueRows()
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.ensureSelectionVisible()
			return m, nil
		case "pgdown":
			m.cursor += m.queueRows()
			if m.cursor >= len(m.visible) {
				m.cursor = max(len(m.visible)-1, 0)
			}
			m.ensureSelectionVisible()
			return m, nil
		}
	}
	return m, nil
}

func (m *model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searchMode = false
		m.searchDraft = m.searchBefore
		m.searchQuery = m.searchBefore
		m.syncData("")
		return m, nil
	case "enter":
		m.searchMode = false
		m.searchQuery = strings.TrimSpace(m.searchDraft)
		m.syncData("")
		return m, nil
	case "backspace", "ctrl+h":
		runes := []rune(m.searchDraft)
		if len(runes) > 0 {
			m.searchDraft = string(runes[:len(runes)-1])
		}
	case "ctrl+u":
		m.searchDraft = ""
	default:
		if len(msg.Runes) > 0 {
			m.searchDraft += string(msg.Runes)
		}
	}

	m.searchQuery = strings.TrimSpace(m.searchDraft)
	m.syncData("")
	return m, nil
}

func (m *model) View() string {
	layout := m.layout()
	parts := []string{
		m.viewHeader(),
		m.viewBody(layout),
		m.viewFooter(layout.width),
	}
	return strings.Join(parts, "\n")
}

func (m *model) viewHeader() string {
	title := steelStyle.Render("IssueSherpa triage desk")
	mode := fmt.Sprintf("mode[%s]", m.modeLabel())
	cache := m.cacheLabel()

	stats := fmt.Sprintf(
		"showing %d issues  open %d  resolved %d  reporters %d",
		len(m.visible),
		m.openCount,
		m.resolvedCount,
		len(m.leaderboard),
	)

	scopeLine := "scope " + strings.Join([]string{
		m.scopeChip("1 all", scopeAll),
		m.scopeChip("2 open", scopeOpen),
		m.scopeChip("3 resolved", scopeResolved),
	}, "  ")

	filterLine := fmt.Sprintf(
		"filters project[p]=%s  provider[v]=%s  sort[s]=%s  order[r]=%s",
		emptyLabel(m.projectFilter),
		emptyLabel(m.sourceFilter),
		m.sortBy,
		sortDirectionLabel(m.sortDesc),
	)

	searchLine := fmt.Sprintf("search[/] %s", m.searchLabel())

	lines := []string{
		title + "  " + ashStyle.Render(mode) + "  " + ashStyle.Render(cache),
		inkStyle.Render(stats),
		scopeLine,
		ashStyle.Render(filterLine),
		ashStyle.Render(searchLine),
	}
	return strings.Join(lines, "\n")
}

func (m *model) viewBody(layout layoutInfo) string {
	if layout.bodyHeight <= 0 {
		return ""
	}

	if m.zoomPreview {
		return m.viewDetailPanel(layout.width, layout.bodyHeight)
	}

	if layout.stacked {
		queue := m.viewQueuePanel(layout.queueWidth, layout.queueHeight)
		detail := m.viewDetailPanel(layout.detailWidth, layout.detailHeight)
		return strings.Join([]string{queue, detail}, "\n")
	}

	queue := m.viewQueuePanel(layout.queueWidth, layout.queueHeight)
	detail := m.viewDetailPanel(layout.detailWidth, layout.detailHeight)
	return lipgloss.JoinHorizontal(lipgloss.Top, queue, " ", detail)
}

func (m *model) viewQueuePanel(width, height int) string {
	lines := []string{panelHeader("Queue", fmt.Sprintf("%d visible", len(m.visible)), width)}
	rows := max(height-1, 1)

	if len(m.visible) == 0 {
		lines = append(lines, ashStyle.Render("No issues match current filters."))
		lines = append(lines, ashStyle.Render("Try scope, project, provider, or search."))
		return fitPanel(lines, height)
	}

	m.ensureSelectionVisible()

	end := m.scroll + rows
	if end > len(m.visible) {
		end = len(m.visible)
	}

	for idx := m.scroll; idx < end; idx++ {
		lines = append(lines, m.renderIssueRow(m.visible[idx], idx == m.cursor, width))
	}

	return fitPanel(lines, height)
}

func (m *model) viewDetailPanel(width, height int) string {
	lines := []string{panelHeader("Dossier", m.previewMeta(), width)}
	issue := m.selectedIssue()
	if issue == nil {
		lines = append(lines, ashStyle.Render("Queue is empty."))
		lines = append(lines, ashStyle.Render("Try adjusting scope, project, provider, or search to reveal issues."))
		return fitPanel(lines, height)
	}

	for _, line := range wrapText(sanitizeTerminalText(issue.Title), max(width-2, 12)) {
		lines = append(lines, steelStyle.Render(line))
	}

	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("status    %s", statusLabel(issue.Status)))
	lines = append(lines, fmt.Sprintf("source    %s", fallback(issue.Source, "unknown")))
	lines = append(lines, fmt.Sprintf("id        %s", fallback(issue.ShortID, issue.ID)))
	lines = append(lines, fmt.Sprintf("project   %s", fallback(issue.Project.Slug, issue.Project.Name)))
	lines = append(lines, fmt.Sprintf("reporter  %s", fallback(issue.Reporter, "unknown")))
	lines = append(lines, fmt.Sprintf("assigned  %s", assignedLabel(issue.AssignedTo)))
	lines = append(lines, fmt.Sprintf("events    %s", fallback(issue.Count, "0")))
	lines = append(lines, fmt.Sprintf("users     %d", issue.UserCount))
	lines = append(lines, fmt.Sprintf("created   %s", detailDate(issue.FirstSeen)))
	lines = append(lines, fmt.Sprintf("updated   %s", detailDate(issue.LastSeen)))

	if issue.URL != "" {
		lines = append(lines, "")
		lines = append(lines, ashStyle.Render("url"))
		lines = append(lines, wrapText(sanitizeTerminalText(issue.URL), max(width-2, 12))...)
	}

	if len(m.leaderboard) > 0 {
		lines = append(lines, "")
		lines = append(lines, ashStyle.Render("top reporters in current slice"))
		for i, entry := range m.leaderboard {
			if i == 3 {
				break
			}
			lines = append(lines, fmt.Sprintf("%d. %s  %d", i+1, sanitizeTerminalText(entry.Name), entry.Count))
		}
	}

	return fitPanel(lines, height)
}

func (m *model) viewFooter(width int) string {
	var lines []string
	if m.zoomPreview {
		lines = append(lines, ashStyle.Render("keys enter/esc dossier  1/2/3 scope  p project  v provider  s sort  r reverse  / search  q quit"))
	} else {
		lines = append(lines, ashStyle.Render("keys j/k move  pgup/pgdown jump  enter dossier  1/2/3 scope  p project  v provider  s sort  r reverse  / search  q quit"))
	}

	if width >= 96 {
		lines = append(lines, ashStyle.Render("principles: queue first, actions visible, keyboard primary, color optional, CLI still available"))
	}
	return strings.Join(lines, "\n")
}

func (m *model) syncData(preferredID string) {
	m.active = core.ApplyFilters(m.issues, core.IssueFilter{
		Project:  m.projectFilter,
		Source:   m.sourceFilter,
		Search:   m.searchQuery,
		SortBy:   m.sortBy,
		SortDesc: m.sortDesc,
	})

	m.openCount = len(core.FilterByStatus(m.active, "open"))
	m.resolvedCount = len(core.FilterByStatus(m.active, "resolved"))
	m.leaderboard = core.BuildLeaderboard(m.active)

	switch m.scope {
	case scopeResolved:
		m.visible = core.FilterByStatus(m.active, "resolved")
	case scopeAll:
		m.visible = append([]Issue(nil), m.active...)
	default:
		m.visible = core.FilterByStatus(m.active, "open")
	}

	m.restoreCursorToSelection(preferredID)
}

func (m *model) restoreCursorToSelection(preferredID string) {
	if len(m.visible) == 0 {
		m.cursor = 0
		m.scroll = 0
		return
	}

	if preferredID != "" {
		for idx, issue := range m.visible {
			if strings.EqualFold(issue.ID, preferredID) || strings.EqualFold(issue.ShortID, preferredID) {
				m.cursor = idx
				m.ensureSelectionVisible()
				return
			}
		}
	}

	if m.cursor >= len(m.visible) {
		m.cursor = len(m.visible) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.ensureSelectionVisible()
}

func (m *model) ensureSelectionVisible() {
	rows := m.queueRows()
	if rows <= 0 {
		rows = 1
	}
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}
	if m.cursor >= m.scroll+rows {
		m.scroll = m.cursor - rows + 1
	}
	if m.scroll < 0 {
		m.scroll = 0
	}
}

func (m *model) queueRows() int {
	layout := m.layout()
	if m.zoomPreview {
		return 1
	}
	return max(layout.queueHeight-1, 1)
}

func (m *model) selectedIssue() *Issue {
	if len(m.visible) == 0 || m.cursor < 0 || m.cursor >= len(m.visible) {
		return nil
	}
	issue := m.visible[m.cursor]
	return &issue
}

func (m *model) selectedID() string {
	issue := m.selectedIssue()
	if issue == nil {
		return ""
	}
	if issue.ID != "" {
		return issue.ID
	}
	return issue.ShortID
}

func (m *model) cycleSort() {
	for idx, value := range sortOptions {
		if value == m.sortBy {
			m.sortBy = sortOptions[(idx+1)%len(sortOptions)]
			return
		}
	}
	m.sortBy = sortOptions[0]
}

func (m *model) modeLabel() string {
	switch {
	case m.searchMode:
		return "search"
	case m.zoomPreview:
		return "dossier"
	default:
		return "browse"
	}
}

func (m *model) cacheLabel() string {
	if !m.cacheInfo.HasSync {
		if m.offline {
			return "cache[offline no sync]"
		}
		return "cache[not synced]"
	}

	age := time.Since(m.cacheInfo.LastSyncAt).Round(time.Minute)
	bits := []string{
		fmt.Sprintf("cache[%s", m.cacheInfo.LastSyncAt.Local().Format("Jan 02 15:04")),
		fmt.Sprintf("%s ago", age),
	}
	if m.cacheInfo.Stale {
		bits = append(bits, "stale")
	}
	if m.offline {
		bits = append(bits, "offline")
	}
	return strings.Join(bits, " ") + "]"
}

func (m *model) searchLabel() string {
	if m.searchMode {
		value := m.searchDraft
		if value == "" {
			value = "_"
		} else {
			value += "_"
		}
		return fmt.Sprintf("editing %q  enter apply  esc cancel", truncateText(value, 42))
	}
	if m.searchQuery == "" {
		return "none"
	}
	return fmt.Sprintf("%q", truncateText(m.searchQuery, 48))
}

func (m *model) scopeChip(label string, scope triageScope) string {
	if m.scope == scope {
		return activeStyle.Render("[" + label + "]")
	}
	return ashStyle.Render("[" + label + "]")
}

func (m *model) previewMeta() string {
	switch {
	case len(m.visible) == 0:
		return "no selection"
	case m.zoomPreview:
		return "zoomed"
	default:
		return fmt.Sprintf("%d/%d", m.cursor+1, len(m.visible))
	}
}

func (m *model) renderIssueRow(issue Issue, selected bool, width int) string {
	status := strings.ToUpper(statusWord(issue.Status))
	source := strings.ToLower(fallback(issue.Source, "?"))
	shortID := fallback(issue.ShortID, issue.ID)
	project := fallback(issue.Project.Slug, issue.Project.Name)
	title := sanitizeTerminalText(issue.Title)

	line := renderIssueLine(width, status, source, shortID, project, title)
	prefix := "  "
	style := inkStyle
	if selected {
		prefix = "> "
		style = activeStyle
	}
	return style.Render(prefix + line)
}

func (m *model) layout() layoutInfo {
	width := max(m.width, 20)
	height := max(m.height, 8)
	bodyHeight := max(height-7, 8)

	if m.zoomPreview {
		return layoutInfo{
			width:        width,
			height:       height,
			bodyHeight:   bodyHeight,
			queueWidth:   width,
			queueHeight:  0,
			detailWidth:  width,
			detailHeight: bodyHeight,
		}
	}

	if width < 110 || height < 26 {
		queueHeight := max(bodyHeight*3/5, 5)
		detailHeight := max(bodyHeight-queueHeight-1, 3)
		return layoutInfo{
			width:        width,
			height:       height,
			bodyHeight:   bodyHeight,
			queueWidth:   width,
			queueHeight:  queueHeight,
			detailWidth:  width,
			detailHeight: detailHeight,
			stacked:      true,
		}
	}

	queueWidth := width * 7 / 12
	if queueWidth < 54 {
		queueWidth = 54
	}
	detailWidth := max(width-queueWidth-1, 20)

	return layoutInfo{
		width:        width,
		height:       height,
		bodyHeight:   bodyHeight,
		queueWidth:   queueWidth,
		queueHeight:  bodyHeight,
		detailWidth:  detailWidth,
		detailHeight: bodyHeight,
	}
}

func panelHeader(title, meta string, width int) string {
	left := steelStyle.Render(title)
	if meta == "" {
		return left
	}

	raw := title + " - " + meta
	rawWidth := runewidth.StringWidth(raw)
	if width <= rawWidth {
		return steelStyle.Render(truncateText(raw, width))
	}

	padding := width - rawWidth - 1
	if padding < 1 {
		padding = 1
	}
	return left + ashStyle.Render(" - "+meta+" "+strings.Repeat("-", padding))
}

func renderIssueLine(width int, status, source, shortID, project, title string) string {
	switch {
	case width >= 72:
		return fitColumns(width, []fieldSpec{
			{value: status, minWidth: 4, preferredWidth: 4},
			{value: source, minWidth: 4, preferredWidth: 7},
			{value: shortID, minWidth: 4, preferredWidth: 12},
			{value: project, minWidth: 8, preferredWidth: 14},
			{value: title, minWidth: 8, preferredWidth: width},
		})
	case width >= 56:
		return fitColumns(width, []fieldSpec{
			{value: status, minWidth: 4, preferredWidth: 4},
			{value: shortID, minWidth: 4, preferredWidth: 12},
			{value: project, minWidth: 8, preferredWidth: 14},
			{value: title, minWidth: 8, preferredWidth: width},
		})
	default:
		return fitColumns(width, []fieldSpec{
			{value: status, minWidth: 4, preferredWidth: 4},
			{value: shortID, minWidth: 4, preferredWidth: 12},
			{value: title, minWidth: 6, preferredWidth: width},
		})
	}
}

type fieldSpec struct {
	value          string
	minWidth       int
	preferredWidth int
}

func fitColumns(width int, specs []fieldSpec) string {
	if width <= 0 || len(specs) == 0 {
		return ""
	}

	const separatorWidth = 2
	if width <= separatorWidth*(len(specs)-1) {
		return truncateText(specs[0].value, width)
	}

	widths := make([]int, len(specs))
	sum := separatorWidth * (len(specs) - 1)
	for i, spec := range specs {
		if spec.minWidth < 1 {
			spec.minWidth = 1
		}
		if spec.preferredWidth < spec.minWidth {
			spec.preferredWidth = spec.minWidth
		}
		widths[i] = spec.preferredWidth
		sum += widths[i]
	}

	if sum > width {
		excess := sum - width
		for i := len(specs) - 1; i >= 0 && excess > 0; i-- {
			room := widths[i] - specs[i].minWidth
			if room <= 0 {
				continue
			}
			reduce := min(room, excess)
			widths[i] -= reduce
			excess -= reduce
		}

		for i := 0; i < len(specs) && excess > 0; i++ {
			room := widths[i] - 1
			if room <= 0 {
				continue
			}
			reduce := min(room, excess)
			widths[i] -= reduce
			excess -= reduce
		}
	}

	parts := make([]string, len(specs))
	for i, spec := range specs {
		parts[i] = displayField(spec.value, widths[i])
	}
	return strings.Join(parts, "  ")
}

func fitPanel(lines []string, height int) string {
	if height <= 0 {
		return ""
	}

	clean := append([]string(nil), lines...)
	if len(clean) > height {
		clean = clean[:height]
		clean[height-1] = ashStyle.Render("...")
	}
	for len(clean) < height {
		clean = append(clean, "")
	}
	return strings.Join(clean, "\n")
}

func wrapText(value string, width int) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if width <= 4 {
		return []string{truncateText(value, width)}
	}

	var lines []string
	for _, paragraph := range strings.Split(value, "\n") {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			lines = append(lines, "")
			continue
		}

		words := strings.Fields(paragraph)
		current := ""
		for _, word := range words {
			for runewidth.StringWidth(word) > width {
				if current != "" {
					lines = append(lines, current)
					current = ""
				}
				chunk := runewidth.Truncate(word, width, "")
				if chunk == "" {
					if _, w := utf8.DecodeRuneInString(word); w > 0 {
						chunk = word[:w]
					}
				}
				lines = append(lines, chunk)
				word = strings.TrimSpace(strings.TrimPrefix(word, chunk))
			}

			candidate := word
			if current != "" {
				candidate = current + " " + word
			}

			if runewidth.StringWidth(candidate) <= width {
				current = candidate
				continue
			}

			if current != "" {
				lines = append(lines, current)
			}
			current = word
		}
		if current != "" {
			lines = append(lines, current)
		}
	}

	return lines
}

func cycleFilterValue(values []string, current string) string {
	if len(values) == 0 {
		return ""
	}
	if current == "" {
		return values[0]
	}
	for idx, value := range values {
		if value == current {
			if idx+1 < len(values) {
				return values[idx+1]
			}
			return ""
		}
	}
	return ""
}

func statusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "resolved":
		return mossStyle.Render("RESOLVED")
	default:
		return emberStyle.Render("OPEN")
	}
}

func openIssueURL(rawURL string) tea.Cmd {
	return func() tea.Msg {
		if err := openIssueInBrowser(strings.TrimSpace(rawURL)); err != nil {
			fmt.Fprintf(os.Stderr, "Unable to open URL: %v\n", err)
		}
		return nil
	}
}

func openIssueInBrowser(rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme %q (must be http:// or https://)", parsed.Scheme)
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		cmd = exec.Command("xdg-open", rawURL)
	}
	return cmd.Start()
}

func statusWord(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "resolved":
		return "done"
	default:
		return "open"
	}
}

func emptyLabel(value string) string {
	if strings.TrimSpace(value) == "" {
		return "all"
	}
	return value
}

func sortDirectionLabel(desc bool) string {
	if desc {
		return amberStyle.Render("desc")
	}
	return mossStyle.Render("asc")
}

func assignedLabel(assignee *AssignedTo) string {
	if assignee == nil || strings.TrimSpace(assignee.Name) == "" {
		return "unassigned"
	}
	return sanitizeTerminalText(assignee.Name)
}

func fallback(primary, backup string) string {
	primary = sanitizeTerminalText(primary)
	if primary != "" {
		return primary
	}
	backup = sanitizeTerminalText(backup)
	if backup != "" {
		return backup
	}
	return "-"
}

func detailDate(value string) string {
	if value == "" {
		return "-"
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		if parsed, err = time.Parse("2006-01-02T15:04:05", value); err != nil {
			return value
		}
	}
	return parsed.Local().Format("Jan 02 15:04")
}

func truncateText(value string, max int) string {
	if max <= 0 {
		return ""
	}
	if runewidth.StringWidth(value) <= max {
		return value
	}
	if max <= 3 {
		return runewidth.Truncate(value, max, "")
	}
	return runewidth.Truncate(value, max, "...")
}

func displayField(value string, width int) string {
	value = truncateText(value, width)
	return runewidth.FillRight(value, width)
}
