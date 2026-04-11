package logentry

import (
	"encoding/json"
	"strings"
)

type Level int

const (
	VERBOSE Level = iota
	DEBUG
	INFO
	WARNING
	ERROR
)

var levelNames = [...]string{"VERBOSE", "DEBUG", "INFO", "WARNING", "ERROR"}

func (l Level) String() string {
	if int(l) >= 0 && int(l) < len(levelNames) {
		return levelNames[l]
	}
	return "UNKNOWN"
}

func ParseLevel(s string) (Level, bool) {
	switch strings.ToUpper(s) {
	case "VERBOSE":
		return VERBOSE, true
	case "DEBUG":
		return DEBUG, true
	case "INFO":
		return INFO, true
	case "WARNING", "WARN":
		return WARNING, true
	case "ERROR":
		return ERROR, true
	default:
		return VERBOSE, false
	}
}

func (l *Level) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, _ := ParseLevel(s)
	*l = parsed
	return nil
}

type Entry struct {
	Tag     string  `json:"tag"`
	Message string  `json:"message"`
	Level   Level   `json:"level"`
	Time    *string `json:"time,omitempty"`
}
