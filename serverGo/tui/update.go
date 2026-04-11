package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"remotelogger/ws"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case serverEvent:
		switch msg.Kind {
		case ws.EventConnected:
			m.connected = true
			m.clientAddr = msg.Addr
		case ws.EventDisconnected:
			m.connected = false
			m.clientAddr = ""
		case ws.EventLog:
			m.appendLog(msg.Entry)
		}
		cmds = append(cmds, waitForEvent(m.eventCh))

	case clearStatusMsg:
		m.saveMsg = ""

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateTableSize()

	case tea.KeyMsg:
		// Inputs consume keys when focused.
		if m.focus == FocusSearch {
			switch msg.String() {
			case "esc", "enter":
				m.searchInput.Blur()
				m.focus = FocusTable
				m.table.Focus()
			default:
				m.searchInput, cmd = m.searchInput.Update(msg)
				cmds = append(cmds, cmd)
				m.filter.Search = m.searchInput.Value()
				m.applyFilter()
			}
			return m, tea.Batch(cmds...)
		}

		if m.focus == FocusTag {
			switch msg.String() {
			case "esc", "enter":
				m.tagInput.Blur()
				m.focus = FocusTable
				m.table.Focus()
			default:
				m.tagInput, cmd = m.tagInput.Update(msg)
				cmds = append(cmds, cmd)
				m.filter.Tag = m.tagInput.Value()
				m.applyFilter()
			}
			return m, tea.Batch(cmds...)
		}

		// Table focus key handling.
		switch msg.String() {
		case "q", "ctrl+c":
			if m.fileHandle != nil {
				m.fileHandle.Close()
			}
			return m, tea.Quit

		case "/":
			m.focus = FocusSearch
			m.table.Blur()
			m.searchInput.Focus()
			cmds = append(cmds, textinput.Blink)

		case "t":
			m.focus = FocusTag
			m.table.Blur()
			m.tagInput.Focus()
			cmds = append(cmds, textinput.Blink)

		case "l":
			m.filter = m.filter.nextLevel()
			m.applyFilter()

		case "c":
			m.filter = Filter{}
			m.searchInput.SetValue("")
			m.tagInput.SetValue("")
			m.applyFilter()

		case "s":
			filename, err := m.saveLogs()
			if err != nil {
				m.saveMsg = fmt.Sprintf("save failed: %v", err)
			} else {
				m.saveMsg = fmt.Sprintf("saved → %s (%d entries)", filename, len(m.logs))
			}
			cmds = append(cmds, clearStatusAfter(3_000_000_000)) // 3s

		case "G":
			m.following = true
			m.table.GotoBottom()

		case "up", "k", "pgup":
			m.following = false
			m.table, cmd = m.table.Update(msg)
			cmds = append(cmds, cmd)

		default:
			m.table, cmd = m.table.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}
