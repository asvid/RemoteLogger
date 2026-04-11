package tui

import (
	"strings"

	"remotelogger/logentry"
)

// levelCycle defines the order for cycling with the `l` key.
// ALL is represented by HasLevel=false.
var levelCycle = []logentry.Level{
	logentry.VERBOSE,
	logentry.DEBUG,
	logentry.INFO,
	logentry.WARNING,
	logentry.ERROR,
}

type Filter struct {
	HasLevel bool
	Level    logentry.Level
	Tag      string
	Search   string
}

func (f Filter) matches(e logentry.Entry) bool {
	if f.HasLevel && e.Level != f.Level {
		return false
	}
	if f.Tag != "" && !strings.EqualFold(e.Tag, f.Tag) {
		return false
	}
	if f.Search != "" {
		s := strings.ToLower(f.Search)
		if !strings.Contains(strings.ToLower(e.Message), s) &&
			!strings.Contains(strings.ToLower(e.Tag), s) {
			return false
		}
	}
	return true
}

func (f Filter) nextLevel() Filter {
	next := f
	if !f.HasLevel {
		next.HasLevel = true
		next.Level = levelCycle[0]
		return next
	}
	for i, l := range levelCycle {
		if l == f.Level {
			if i+1 >= len(levelCycle) {
				next.HasLevel = false
			} else {
				next.Level = levelCycle[i+1]
			}
			return next
		}
	}
	next.HasLevel = false
	return next
}

func (f Filter) levelStr() string {
	if !f.HasLevel {
		return "ALL"
	}
	return f.Level.String()
}
