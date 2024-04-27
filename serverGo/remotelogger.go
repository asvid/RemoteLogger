package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/common-nighthawk/go-figure"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

func convertStringToEventLevel(levelStr string) EventLevel {
	switch strings.ToUpper(levelStr) {
	case "INFO":
		return INFO
	case "ERROR":
		return ERROR
	case "DEBUG":
		return DEBUG
	case "VERBOSE":
		return VERBOSE
	case "WARNING":
		return WARNING
	default:
		return EventLevel(0)
	}
}

func (e *EventLevel) UnmarshalJSON(data []byte) error {
	var levelStr string
	if err := json.Unmarshal(data, &levelStr); err != nil {
		return err
	}

	*e = convertStringToEventLevel(levelStr)
	return nil
}

func main() {
	myFigure := figure.NewFigure("Remote Logger", "doom", true)
	myFigure.Print()

	//	currentTime := time.Now()
	// formattedDate := currentTime.Format("2006-01-02-15-04-05")
	// fileName := fmt.Sprintf("logs-%s.txt", formattedDate)

	port := flag.Int("port", 3000, "Provide port for running Remote Logger")
	logLevelStr := flag.String("logLevel", "", "Show only provided level of logs")
	tag := flag.String("tag", "", "Show only logs with provided tag")
	toFile := flag.String("toFile", "", "Save logs to provided file")

	flag.Parse()

	logLevel := convertStringToEventLevel(*logLevelStr)
	log.Info("Starting server on", "port", *port, "logLevelStr", *logLevelStr, "tag", *tag, "toFile", *toFile)

	remoteLogger := log.NewWithOptions(os.Stderr, log.Options{
		Prefix:          "🤖",
		ReportTimestamp: true,
		TimeFormat:      time.StampMilli,
	})

	var file *os.File
	if *toFile != "" {
		f, err := os.OpenFile(*toFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Error(err)
		}
		file = f
		defer file.Close()
		log.Info("created a file at", "file", file)

	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Error(err)
			return
		}
		defer conn.Close()

		log.Info("New client connected", "IP", r.RemoteAddr)

		for {
			_, p, err := conn.ReadMessage()
			if err != nil {
				log.Error(err)
				return
			}
			var event Event
			if err := json.Unmarshal(p, &event); err != nil {
				log.Error("Error parsing JSON: ", err)
				continue
			}

			if *toFile != "" {
				timeStr := time.Now().Format("2006-01-02 15:04:05.000")
				if event.Time != nil {
					timeStr = *event.Time
				}

				newLine := fmt.Sprintf("%s %s %s %s\n", timeStr, event.Level.String(), event.Tag, event.Message)
				_, err := file.WriteString(newLine)
				if err != nil {
					log.Error("Error when writing to file: ", err)
				}
				log.Info("Writing line to file", "newLine", newLine)
			}

			format := "%s %s"

			if (logLevel == 0 || logLevel == event.Level) || (*tag == "" || event.Tag == *tag) {
				switch event.Level {
				case INFO:
					remoteLogger.Infof(format, event.Tag, event.Message)
				case ERROR:
					remoteLogger.Errorf(format, event.Tag, event.Message)
				case DEBUG:
					remoteLogger.Debugf(format, event.Tag, event.Message)
				case VERBOSE:
					remoteLogger.Infof(format, event.Tag, event.Message)
				case WARNING:
					remoteLogger.Warnf(format, event.Tag, event.Message)
				}
			}
		}
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}

type Event struct {
	Time    *string    `json:"time,omitempty"`
	Tag     string     `json:"tag"`
	Message string     `json:"message"`
	Level   EventLevel `json:"level"`
}

type EventLevel int

const (
	INFO EventLevel = iota
	ERROR
	DEBUG
	VERBOSE
	WARNING
)

func (e EventLevel) String() string {
	return [...]string{"INFO", "ERROR", "DEBUG", "VERBOSE", "WARNING"}[e]
}
