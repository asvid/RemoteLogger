package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	figure "github.com/common-nighthawk/go-figure"
	"remotelogger/logentry"
	"remotelogger/tui"
	"remotelogger/ws"
)

func main() {
	myFigure := figure.NewFigure("Remote Logger", "doom", true)
	myFigure.Print()

	port := flag.Int("p", 3000, "Port for Remote Logger")
	logLevelStr := flag.String("ll", "", "Initial log level filter (VERBOSE/DEBUG/INFO/WARNING/ERROR)")
	tag := flag.String("t", "", "Initial tag filter")
	toFile := flag.String("f", "", "Save logs to file continuously (filters do not apply)")
	noTUI := flag.Bool("no-tui", false, "Plain stdout output, suitable for piping or CI")
	flag.Parse()

	wsServer := ws.NewServer()

	mux := http.NewServeMux()
	mux.HandleFunc("/", wsServer.Handler)

	go func() {
		addr := fmt.Sprintf(":%d", *port)
		log.Info("WebSocket server listening", "port", *port)
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Fatal("Server error", "err", err)
		}
	}()

	if *noTUI {
		runPlain(wsServer, *toFile, *logLevelStr, *tag)
		return
	}

	var initialLevel *logentry.Level
	if *logLevelStr != "" {
		if l, ok := logentry.ParseLevel(*logLevelStr); ok {
			initialLevel = &l
		} else {
			log.Warn("Unknown log level, ignoring", "level", *logLevelStr)
		}
	}

	model := tui.New(wsServer.Ch, *port, *toFile, initialLevel, *tag)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}

func runPlain(s *ws.Server, toFile, levelStr, tagFilter string) {
	var file *os.File
	if toFile != "" {
		f, err := os.OpenFile(toFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Fatal("Cannot open file", "err", err, "path", toFile)
		}
		file = f
		defer file.Close()
	}

	var levelFilter logentry.Level
	hasLevelFilter := false
	if levelStr != "" {
		if l, ok := logentry.ParseLevel(levelStr); ok {
			levelFilter = l
			hasLevelFilter = true
		}
	}

	log.Info("Running in plain mode. Ctrl+C to quit.")

	for event := range s.Ch {
		switch event.Kind {
		case ws.EventConnected:
			log.Info("Client connected", "addr", event.Addr)
		case ws.EventDisconnected:
			log.Info("Client disconnected")
		case ws.EventLog:
			e := event.Entry
			if file != nil {
				t := "unknown"
				if e.Time != nil {
					t = *e.Time
				}
				fmt.Fprintf(file, "%s %s %s %s\n", t, e.Level, e.Tag, e.Message)
			}
			if hasLevelFilter && e.Level != levelFilter {
				continue
			}
			if tagFilter != "" && e.Tag != tagFilter {
				continue
			}
			fmt.Printf("[%s] %s: %s\n", tui.StyledLevel(e.Level), e.Tag, e.Message)
		}
	}
}
