package logger

import (
	"bytes"
	"context"
	stdlog "log"
	"strings"
	"testing"
)

type logCase struct {
	name    string
	write   func(ctx context.Context, msg string)
	tag     string
	visible map[Level]bool
}

func TestLevels_MessageVisibilityAndTag(t *testing.T) {
	cases := []logCase{
		{
			name:    "Error",
			write:   Error,
			tag:     "[ERROR]",
			visible: map[Level]bool{LevelDebug: true, LevelInfo: true, LevelError: true, LevelFatal: false},
		},
		{
			name:    "Debug",
			write:   Debug,
			tag:     "[DEBUG]",
			visible: map[Level]bool{LevelDebug: true, LevelInfo: false, LevelError: false, LevelFatal: false},
		},
		{
			name:    "Info",
			write:   Info,
			tag:     "[INFO]",
			visible: map[Level]bool{LevelDebug: true, LevelInfo: true, LevelError: false, LevelFatal: false},
		},
		{
			name:    "Warning",
			write:   Warning,
			tag:     "[WARN]",
			visible: map[Level]bool{LevelDebug: true, LevelInfo: true, LevelError: false, LevelFatal: false},
		},
	}

	for _, c := range cases {
		for level, visible := range c.visible {
			t.Run(c.name+"/"+level.String(), func(t *testing.T) {
				out := captureLog(t, level)

				c.write(context.Background(), "test message")

				assertLine(t, out.String(), c.tag, visible)
			})
		}
	}
}

func TestError_NotTaggedAsDebug(t *testing.T) {
	out := captureLog(t, LevelDebug)

	Error(context.Background(), "test message")

	if strings.Contains(out.String(), "[DEBUG]") {
		t.Fatalf("Error message tagged as DEBUG: %q", out.String())
	}
}

// captureLog Подменяет вывод и уровень логгера на время теста.
func captureLog(t *testing.T, level Level) *bytes.Buffer {
	t.Helper()

	prevLogger := get()
	prevLevel := GetLevel()
	t.Cleanup(func() {
		SetLogger(prevLogger)
		SetLevel(prevLevel)
	})

	var out bytes.Buffer
	SetLogger(stdlog.New(&out, "", 0))
	SetLevel(level)

	return &out
}

func assertLine(t *testing.T, out string, tag string, visible bool) {
	t.Helper()

	if !visible {
		if out != "" {
			t.Fatalf("message must be hidden, got %q", out)
		}

		return
	}

	if !strings.Contains(out, " "+tag+" test message") {
		t.Fatalf("want %s test message, got %q", tag, out)
	}
}
