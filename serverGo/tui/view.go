package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"remotelogger/logentry"
)

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62")).
			Padding(0, 1)

	connectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Bold(true)

	disconnectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Bold(true)

	filterLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))

	filterValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")).
				Bold(true)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)

	saveMsgStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82"))

	levelStyles = map[logentry.Level]lipgloss.Style{
		logentry.VERBOSE: lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		logentry.DEBUG:   lipgloss.NewStyle().Foreground(lipgloss.Color("39")),
		logentry.INFO:    lipgloss.NewStyle().Foreground(lipgloss.Color("82")),
		logentry.WARNING: lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
		logentry.ERROR:   lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
	}
)

func (m Model) View() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		m.headerView(),
		m.table.View(),
		m.filterBarView(),
		m.statusView(),
	)
}

func (m Model) headerView() string {
	title := "RemoteLogger"
	portStr := fmt.Sprintf(":%d", m.port)

	var connIndicator string
	if m.connected {
		addr := m.clientAddr
		if addr == "" {
			addr = "client"
		}
		connIndicator = connectedStyle.Render("● CONNECTED  " + addr)
	} else {
		connIndicator = disconnectedStyle.Render("○ WAITING...")
	}

	left := headerStyle.Render(title + "  " + portStr)
	right := connIndicator

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}
	padding := lipgloss.NewStyle().Width(gap).Render("")

	return left + padding + right
}

func (m Model) filterBarView() string {
	// Search input
	searchLabel := filterLabelStyle.Render("[/] search: ")
	var searchVal string
	if m.focus == FocusSearch {
		searchVal = m.searchInput.View()
	} else {
		v := m.searchInput.Value()
		if v == "" {
			searchVal = filterLabelStyle.Render("—")
		} else {
			searchVal = filterValueStyle.Render(v)
		}
	}

	// Level cycling
	levelLabel := filterLabelStyle.Render("  [l] level: ")
	levelVal := filterValueStyle.Render(m.filter.levelStr())

	// Tag input
	tagLabel := filterLabelStyle.Render("  [t] tag: ")
	var tagVal string
	if m.focus == FocusTag {
		tagVal = m.tagInput.View()
	} else {
		v := m.tagInput.Value()
		if v == "" {
			tagVal = filterLabelStyle.Render("—")
		} else {
			tagVal = filterValueStyle.Render(v)
		}
	}

	clearHint := filterLabelStyle.Render("  [c] clear  [s] save  [q] quit")

	return searchLabel + searchVal + levelLabel + levelVal + tagLabel + tagVal + clearHint
}

func (m Model) statusView() string {
	if m.saveMsg != "" {
		return saveMsgStyle.Render(m.saveMsg)
	}
	followStr := ""
	if m.following {
		followStr = "  [G] following"
	} else {
		followStr = "  [G] jump to latest"
	}
	return statusStyle.Render(fmt.Sprintf("showing %d/%d logs%s", len(m.display), len(m.logs), followStr))
}

// StyledLevel renders a level string with its color — exported for use in plain mode.
func StyledLevel(l logentry.Level) string {
	if style, ok := levelStyles[l]; ok {
		return style.Render(l.String())
	}
	return l.String()
}
