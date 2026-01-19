package logger

import (
	"context"
	"encoding/json"
	"fmt"
	context2 "github.com/exgamer/gosdk-core/pkg/context"
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

func printLog(ctx context.Context, level Level, msg string) {
	if level < GetLevel() {
		return
	}

	appInfo := context2.GetAppInfoFromContext(ctx)
	var b strings.Builder
	b.Grow(64 + len(msg))

	b.WriteString(time.Now().Format("2006-01-02 15:04:05.000"))
	b.WriteString(" ")

	if appInfo == nil {
		b.WriteString("[" + level.String() + "]")
	} else {
		b.WriteString("[" + level.String() + "," + appInfo.ServiceName + "]")
	}

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

func IsTraceLevel() bool {
	if GetLevel() > LevelTrace {
		return false
	}

	return true
}

func Info(ctx context.Context, msg string)    { printLog(ctx, LevelInfo, msg) }
func Error(ctx context.Context, msg string)   { printLog(ctx, LevelDebug, msg) }
func Warning(ctx context.Context, msg string) { printLog(ctx, LevelWarn, msg) }
func Fatal(ctx context.Context, msg string)   { printLog(ctx, LevelFatal, msg) }
func Trace(ctx context.Context, msg string)   { printLog(ctx, LevelTrace, msg) }
func Debug(ctx context.Context, msg string)   { printLog(ctx, LevelDebug, msg) }

func Dump(ctx context.Context, v ...any) {
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

	printLog(ctx, LevelDebug, "DUMP "+msg)
}

func dumpValue(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%#v", v)
	}
	return string(b)
}
