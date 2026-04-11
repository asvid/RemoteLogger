package ws

import (
	"encoding/json"
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/gorilla/websocket"
	"remotelogger/logentry"
)

type EventKind int

const (
	EventLog EventKind = iota
	EventConnected
	EventDisconnected
)

type Event struct {
	Kind  EventKind
	Entry logentry.Entry
	Addr  string
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Server struct {
	Ch chan Event
}

func NewServer() *Server {
	return &Server{
		Ch: make(chan Event, 1000),
	}
}

func (s *Server) Handler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("WebSocket upgrade failed", "err", err)
		return
	}
	defer func() {
		conn.Close()
		s.Ch <- Event{Kind: EventDisconnected}
	}()

	s.Ch <- Event{Kind: EventConnected, Addr: r.RemoteAddr}

	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var entry logentry.Entry
		if err := json.Unmarshal(p, &entry); err != nil {
			log.Error("JSON parse error", "err", err)
			continue
		}
		s.Ch <- Event{Kind: EventLog, Entry: entry}
	}
}
