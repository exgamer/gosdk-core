package logger

import (
	"log"
	"os"
)

// Info Обычный лог
func Info(format string, v ...any) {
	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	infoLog.Printf(format, v)
}

// Error лог с ошибкой
func Error(format string, v ...any) {
	infoLog := log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime)
	infoLog.Printf(format, v)
}

// LogError лог с ошибкой
func LogError(err error) {
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)
	errorLog.Println(err)
}
