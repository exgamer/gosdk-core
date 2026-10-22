package logger

import (
	"bytes"
	"context"
	stdlog "log"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/exgamer/gosdk-core/pkg/config"
	context2 "github.com/exgamer/gosdk-core/pkg/context"
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

func TestPrintLog_EscapesControlInMessage(t *testing.T) {
	out := captureLog(t, LevelDebug)

	Error(context.Background(), "a\nb\r\x1b[31mc d\te")

	line := singleLine(t, out.String())
	if !strings.HasSuffix(line, " [ERROR] "+`a\nb\r\x1b[31mc d`+"\te") {
		t.Fatalf("want escaped message with real tab, got %q", line)
	}

	if strings.ContainsAny(line, "\r\x1b") {
		t.Fatalf("raw control bytes in %q", line)
	}
}

func TestPrintLog_EscapesServiceName(t *testing.T) {
	out := captureLog(t, LevelDebug)
	ctx := context2.WithAppInfo(context.Background(), &config.AppInfo{ServiceName: "svc\n[INFO]"})

	Info(ctx, "m")

	line := singleLine(t, out.String())
	if !strings.HasSuffix(line, " [INFO,"+`svc\n[INFO]`+"] m") {
		t.Fatalf("want escaped service name, got %q", line)
	}
}

func TestPrintLog_EscapesOnEveryLevel(t *testing.T) {
	out := captureLog(t, LevelTrace)
	writers := []func(ctx context.Context, msg string){Trace, Debug, Info, Warning, Error, Fatal}

	for _, write := range writers {
		write(context.Background(), "l1\nl2")
	}

	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	if len(lines) != len(writers) {
		t.Fatalf("want %d lines, got %d: %q", len(writers), len(lines), out.String())
	}

	for _, line := range lines {
		if !strings.HasSuffix(line, `l1\nl2`) {
			t.Fatalf("want escaped message, got %q", line)
		}
	}
}

func TestPrintLog_EscapesBeforeCustomLogger(t *testing.T) {
	out := captureLog(t, LevelInfo)
	SetLogger(stdlog.New(out, "pfx ", stdlog.LstdFlags))

	Info(context.Background(), "a\nb")

	line := singleLine(t, out.String())
	if !strings.HasPrefix(line, "pfx ") || !strings.HasSuffix(line, `a\nb`) {
		t.Fatalf("want custom prefix and escaped message, got %q", line)
	}
}

func TestPrintLog_CleanMessageUnchanged(t *testing.T) {
	for _, msg := range []string{"GET /api 200", "Привет, мир", `C:\path\n`, "tab\there"} {
		t.Run(msg, func(t *testing.T) {
			out := captureLog(t, LevelInfo)

			Info(context.Background(), msg)

			line := singleLine(t, out.String())
			if !strings.HasSuffix(line, " [INFO] "+msg) {
				t.Fatalf("want message %q unchanged, got %q", msg, line)
			}
		})
	}
}

// TestPrintLog_BackslashNotEscaped Фиксирует принятую неоднозначность: \ не экранируется,
// поэтому перевод строки и литерал \n выглядят в логе одинаково.
func TestPrintLog_BackslashNotEscaped(t *testing.T) {
	out := captureLog(t, LevelInfo)

	Info(context.Background(), "a\nb")
	Info(context.Background(), "a\\nb")

	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	if len(lines) != 2 || lines[0] != lines[1] {
		t.Fatalf("want two equal lines, got %q", out.String())
	}
}

func TestPrintLog_ConcurrentWrites(t *testing.T) {
	out := captureLog(t, LevelInfo)

	const writers = 20
	var wg sync.WaitGroup
	for i := range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			Info(context.Background(), "g\n"+strconv.Itoa(i))
		}()
	}
	wg.Wait()

	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	if len(lines) != writers {
		t.Fatalf("want %d lines, got %d", writers, len(lines))
	}
}

func TestPrintLog_BelowLevelNoAllocs(t *testing.T) {
	out := captureLog(t, LevelError)

	allocs := testing.AllocsPerRun(100, func() {
		Info(context.Background(), "a\nb")
	})

	if allocs != 0 {
		t.Fatalf("want 0 allocs below level, got %v", allocs)
	}

	if out.Len() != 0 {
		t.Fatalf("want no output, got %q", out.String())
	}
}

func TestDump_SingleLine(t *testing.T) {
	cases := []struct {
		name string
		args []any
		want string
	}{
		{name: "map", args: []any{map[string]any{"a": 1, "b": "x"}}, want: `[DEBUG] DUMP {"a":1,"b":"x"}`},
		{name: "named", args: []any{"cfg", map[string]any{"a": 1}}, want: `[DEBUG] DUMP cfg={"a":1}`},
		{name: "many", args: []any{1, 2, 3}, want: `[DEBUG] DUMP [1,2,3]`},
		{name: "json escaped once", args: []any{map[string]string{"k": "a\nb"}}, want: `[DEBUG] DUMP {"k":"a\nb"}`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := captureLog(t, LevelDebug)

			Dump(context.Background(), c.args...)

			line := singleLine(t, out.String())
			if !strings.HasSuffix(line, c.want) {
				t.Fatalf("want suffix %q, got %q", c.want, line)
			}
		})
	}
}

func TestDump_MarshalFailureFallsBackToSingleLine(t *testing.T) {
	for _, v := range []any{make(chan int), struct{ F func() }{}} {
		out := captureLog(t, LevelDebug)

		Dump(context.Background(), v)

		line := singleLine(t, out.String())
		if !strings.Contains(line, "[DEBUG] DUMP ") {
			t.Fatalf("want dump line, got %q", line)
		}
	}
}

func TestDump_HiddenAboveDebug(t *testing.T) {
	out := captureLog(t, LevelInfo)

	Dump(context.Background(), map[string]any{"a": 1})

	if out.Len() != 0 {
		t.Fatalf("want no output, got %q", out.String())
	}
}

// singleLine Проверяет, что вывод — ровно одна физическая строка, и возвращает её без перевода строки.
func singleLine(t *testing.T, out string) string {
	t.Helper()

	line, ok := strings.CutSuffix(out, "\n")
	if !ok || strings.Contains(line, "\n") {
		t.Fatalf("want exactly one line, got %q", out)
	}

	return line
}
