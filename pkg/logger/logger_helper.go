package logger

import (
	"encoding/json"
	"fmt"
	stdlog "log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	logLevel atomic.Int32
	once     sync.Once
	mu       sync.RWMutex
	l        *stdlog.Logger
)

func Init() {
	once.Do(func() {
		l = stdlog.New(os.Stdout, "", 0)
		logLevel.Store(int32(LevelInfo))
	})
}

func SetLogger(custom *stdlog.Logger) {
	if custom == nil {
		return
	}

	Init() // гарантируем, что l не nil (не обязательно, но полезно)

	mu.Lock()
	l = custom
	mu.Unlock()
}

func SetLevel(level Level) {
	Init()
	logLevel.Store(int32(level))
}

func GetLevel() Level {
	return Level(logLevel.Load())
}

// internal getter (safe)
func get() *stdlog.Logger {
	Init()

	mu.RLock()
	cur := l
	mu.RUnlock()

	// на всякий случай (теоретически)
	if cur == nil {
		return stdlog.New(os.Stdout, "", 0)
	}
	return cur
}

func printLog(level Level, msg string) {
	if level < GetLevel() {
		return
	}

	var b strings.Builder
	b.Grow(64 + len(msg))

	b.WriteString(time.Now().Format("2006-01-02 15:04:05.000"))
	b.WriteString(" ")
	b.WriteString(level.String())
	b.WriteString(" ")
	b.WriteString(msg)

	get().Println(b.String())
}

func ParseLevel(name string) Level {
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "TRACE":
		return LevelTrace
	case "DEBUG":
		return LevelDebug
	case "INFO":
		return LevelInfo
	case "WARN":
		return LevelWarn
	case "ERROR":
		return LevelError
	case "FATAL":
		return LevelFatal
	default:
		return LevelInfo
	}
}

func IsDebugLevel() bool {
	if GetLevel() > LevelDebug {
		return false
	}

	return true
}

func Info(msg string)    { printLog(LevelInfo, msg) }
func Error(msg string)   { printLog(LevelDebug, msg) }
func Warning(msg string) { printLog(LevelWarn, msg) }
func Fatal(msg string)   { printLog(LevelFatal, msg) }
func Trace(msg string)   { printLog(LevelTrace, msg) }
func Debug(msg string)   { printLog(LevelDebug, msg) }

func Dump(v ...any) {
	if !IsDebugLevel() {
		return
	}

	var msg string

	switch len(v) {
	case 1:
		msg = dumpValue(v[0])
	case 2:
		msg = fmt.Sprintf("%v=%s", v[0], dumpValue(v[1]))
	default:
		msg = dumpValue(v)
	}

	printLog(LevelDebug, "DUMP "+msg)
}

func dumpValue(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%#v", v)
	}
	return string(b)
}
