package utils

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"log/slog"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"unknown", slog.LevelInfo}, // default
	}

	for _, tt := range tests {
		got := parseLevel(tt.input)
		if got != tt.want {
			t.Errorf("parseLevel(%q) = %v; want %v", tt.input, got, tt.want)
		}
	}
}

func TestPrettyHandler_Enabled(t *testing.T) {
	handler := NewPrettyHandler(slog.LevelInfo)

	if !handler.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("Enabled returned false for level equal to minLevel")
	}

	if !handler.Enabled(context.Background(), slog.LevelError) {
		t.Error("Enabled returned false for level higher than minLevel")
	}

	if handler.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("Enabled returned true for level lower than minLevel")
	}
}

func TestPrettyHandler_Handle(t *testing.T) {
	handler := NewPrettyHandler(slog.LevelDebug)

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	attrs := []slog.Attr{
		slog.String("key1", "value1"),
		slog.Int("key2", 42),
	}

	record := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
	record.AddAttrs(attrs...)

	err := handler.Handle(context.Background(), record)
	errClose := w.Close()
	if errClose != nil {
		t.Fatalf("failed to close pipe writer: %v", errClose)
	}

	os.Stderr = oldStderr

	if err != nil {
		t.Errorf("Handle returned error: %v", err)
	}

	// Read from pipe
	var outputBuf bytes.Buffer
	_, err = outputBuf.ReadFrom(r)
	if err != nil {
		t.Fatalf("failed to read stderr pipe: %v", err)
	}

	output := outputBuf.String()

	// Check timestamp format (simple check)
	if len(output) < 20 || output[4] != '/' || output[7] != '/' {
		t.Errorf("Output timestamp format incorrect: %q", output)
	}

	// Check level string is present
	if !strings.Contains(output, "[INFO]") {
		t.Errorf("Output missing level string: %q", output)
	}

	// Check message and attrs presence
	if !strings.Contains(output, "test message") || !strings.Contains(output, "key1=value1") || !strings.Contains(output, "key2=42") {
		t.Errorf("Output missing message or attributes: %q", output)
	}
}

func TestStyledLevel(t *testing.T) {
	tests := []struct {
		level    slog.Level
		expected string
	}{
		{slog.LevelDebug, "[DEBUG]"},
		{slog.LevelInfo, "[INFO]"},
		{slog.LevelWarn, "[WARN]"},
		{slog.LevelError, "[ERROR]"},
		{slog.Level(999), "[" + slog.Level(999).String() + "]"}, // unknown/custom levels
	}

	for _, tt := range tests {
		got, _ := styledLevel(tt.level)
		if !strings.Contains(got, tt.expected) {
			t.Errorf("styledLevel(%v) = %q; want to contain %q", tt.level, got, tt.expected)
		}
	}
}
