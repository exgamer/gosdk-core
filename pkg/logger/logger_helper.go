package logger

import (
	stdlog "log"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	once sync.Once
	mu   sync.RWMutex
	l    *stdlog.Logger
)

func Init() {
	once.Do(func() {
		l = stdlog.New(os.Stdout, "", 0)
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

func printLog(level, msg string) {
	var b strings.Builder
	b.Grow(64 + len(msg))

	b.WriteString(time.Now().Format("2006-01-02 15:04:05.000"))
	b.WriteString(" ")
	b.WriteString(level)
	b.WriteString(" ")
	b.WriteString(msg)

	get().Println(b.String())
}

func Info(msg string)  { printLog("INFO", msg) }
func Error(msg string) { printLog("ERROR", msg) }
func Debug(msg string) { printLog("DEBUG", msg) }
