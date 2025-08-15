package utils

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func InitLogger() {
	level := "debug"
	minLevel := parseLevel(level)
	logger := slog.New(NewPrettyHandler(minLevel))
	slog.SetDefault(logger)
}

type PrettyHandler struct {
	minLevel slog.Level
}

func NewPrettyHandler(level slog.Level) *PrettyHandler {
	return &PrettyHandler{minLevel: level}
}

func (h *PrettyHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.minLevel
}

func (h *PrettyHandler) Handle(_ context.Context, record slog.Record) error {
	ts := time.Now().Format("2006/01/02 15:04:05")
	levelStr, _ := styledLevel(record.Level)
	msg := record.Message
	attrs := ""
	record.Attrs(func(a slog.Attr) bool {
		attrs += fmt.Sprintf(" %s=%v", a.Key, a.Value)
		return true
	})

	msg += attrs

	fmt.Fprintf(os.Stderr, "%s %s %s\n", ts, levelStr, msg)
	return nil
}

func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Optional: support structured fields
	return h
}

func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	return h
}

func styledLevel(level slog.Level) (string, lipgloss.Style) {
	var code string
	levelStr := level.String()

	switch level {
	case slog.LevelDebug:
		code = "8" // gray
	case slog.LevelInfo:
		code = "12" // blue
	case slog.LevelWarn:
		code = "11" // yellow
	case slog.LevelError:
		code = "9" // red
	default:
		code = "7" // white
	}

	style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(code))
	return style.Render("[" + levelStr + "]"), style
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
