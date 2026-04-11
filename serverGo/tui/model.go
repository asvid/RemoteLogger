package tui

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"remotelogger/logentry"
	"remotelogger/ws"
)

const (
	maxLogs = 50_000
	trimTo  = 40_000

	headerHeight    = 1
	filterBarHeight = 2 // filter inputs + status line
)

type FocusTarget int

const (
	FocusTable FocusTarget = iota
	FocusSearch
	FocusTag
)

// serverEvent wraps a ws.Event as a tea.Msg.
type serverEvent ws.Event

// clearStatusMsg clears the transient save-confirmation message.
type clearStatusMsg struct{}

func waitForEvent(ch <-chan ws.Event) tea.Cmd {
	return func() tea.Msg {
		return serverEvent(<-ch)
	}
}

func clearStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

type Model struct {
	logs        []logentry.Entry
	display     []logentry.Entry
	rows        []table.Row // mirrors display, kept in sync to avoid O(n) rebuild on each append
	filter      Filter
	table       table.Model
	searchInput textinput.Model
	tagInput    textinput.Model
	focus       FocusTarget
	following   bool // auto-scroll to latest
	connected   bool
	clientAddr  string
	width       int
	height      int
	eventCh     <-chan ws.Event
	port        int
	fileHandle  *os.File // -f continuous output
	saveMsg     string   // transient status from `s` key
}

func New(ch <-chan ws.Event, port int, fileOutput string, initialLevel *logentry.Level, initialTag string) *Model {
	cols := []table.Column{
		{Title: "Time", Width: 15},
		{Title: "Level", Width: 7},
		{Title: "Tag", Width: 20},
		{Title: "Message", Width: 60},
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	searchIn := textinput.New()
	searchIn.Placeholder = "search..."
	searchIn.CharLimit = 100

	tagIn := textinput.New()
	tagIn.Placeholder = "tag..."
	tagIn.CharLimit = 50

	filter := Filter{}
	if initialLevel != nil {
		filter.HasLevel = true
		filter.Level = *initialLevel
	}
	if initialTag != "" {
		filter.Tag = initialTag
		tagIn.SetValue(initialTag)
	}

	var fileHandle *os.File
	if fileOutput != "" {
		f, err := os.OpenFile(fileOutput, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err == nil {
			fileHandle = f
		}
	}

	return &Model{
		logs:        make([]logentry.Entry, 0, 1000),
		display:     make([]logentry.Entry, 0, 1000),
		rows:        make([]table.Row, 0, 1000),
		filter:      filter,
		table:       t,
		searchInput: searchIn,
		tagInput:    tagIn,
		focus:       FocusTable,
		following:   true,
		eventCh:     ch,
		port:        port,
		fileHandle:  fileHandle,
	}
}

func (m Model) Init() tea.Cmd {
	return waitForEvent(m.eventCh)
}

func (m *Model) appendLog(e logentry.Entry) {
	if m.fileHandle != nil {
		t := time.Now().Format("2006-01-02 15:04:05.000")
		if e.Time != nil {
			t = *e.Time
		}
		fmt.Fprintf(m.fileHandle, "%s %s %s %s\n", t, e.Level, e.Tag, e.Message)
	}

	m.logs = append(m.logs, e)
	if m.filter.matches(e) {
		m.display = append(m.display, e)
		m.rows = append(m.rows, entryToRow(e))
		m.table.SetRows(m.rows)
		if m.following {
			m.table.GotoBottom()
		}
	}

	if len(m.logs) > maxLogs {
		m.logs = m.logs[len(m.logs)-trimTo:]
		m.applyFilter()
	}
}

func (m *Model) applyFilter() {
	m.display = m.display[:0]
	m.rows = m.rows[:0]
	for _, e := range m.logs {
		if m.filter.matches(e) {
			m.display = append(m.display, e)
			m.rows = append(m.rows, entryToRow(e))
		}
	}
	m.table.SetRows(m.rows)
	if m.following {
		m.table.GotoBottom()
	}
}

func (m *Model) updateTableSize() {
	// Time(15) + Level(7) + Tag(20) = 42, plus 3 column separators
	fixedWidth := 42 + 3
	msgWidth := m.width - fixedWidth
	if msgWidth < 20 {
		msgWidth = 20
	}
	tableHeight := m.height - headerHeight - filterBarHeight - 3 // -3 for table header + borders
	if tableHeight < 1 {
		tableHeight = 1
	}
	m.table.SetColumns([]table.Column{
		{Title: "Time", Width: 15},
		{Title: "Level", Width: 7},
		{Title: "Tag", Width: 20},
		{Title: "Message", Width: msgWidth},
	})
	m.table.SetHeight(tableHeight)
}

func (m *Model) saveLogs() (string, error) {
	filename := fmt.Sprintf("remote-logs-%s.txt", time.Now().Format("20060102-150405"))
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return "", err
	}
	defer f.Close()
	for _, e := range m.logs {
		t := time.Now().Format("2006-01-02 15:04:05.000")
		if e.Time != nil {
			t = *e.Time
		}
		fmt.Fprintf(f, "%s %s %s %s\n", t, e.Level, e.Tag, e.Message)
	}
	return filename, nil
}

func entryToRow(e logentry.Entry) table.Row {
	t := ""
	if e.Time != nil {
		t = *e.Time
	}
	return table.Row{t, e.Level.String(), e.Tag, e.Message}
}
