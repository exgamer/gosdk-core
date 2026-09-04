package errorreporter

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// Level - уровень серьёзности события для error-репортера
type Level string

const (
	LevelWarning Level = "warning"
	LevelError   Level = "error"
	LevelFatal   Level = "fatal"
)

// Options - дополнительные данные события
type Options struct {
	Level Level
	Tags  map[string]string
	Extra map[string]any

	// Force заставляет отправить событие, даже если ошибка уже была
	// помечена как отправленная (WasReported). Нужен для намеренной
	// эскалации: например, ошибку сначала отрепортили через CaptureSoft
	// (warning) при первой неудачной попытке, а после исчерпания ретраев
	// её же нужно отрепортить повторно как CaptureError.
	Force bool
}

// Reporter - абстракция над системой трекинга ошибок (Sentry, Rollbar и т.п.).
// gosdk-core ничего не знает про конкретного вендора - реальную реализацию
// регистрирует соответствующий адаптер (например SentryKernel из gosdk-sentry-core).
type Reporter interface {
	Capture(ctx context.Context, err error, opts Options)
}

type noopReporter struct{}

func (noopReporter) Capture(context.Context, error, Options) {}

var (
	mu           sync.RWMutex
	reporter     Reporter = noopReporter{}
	isConfigured atomic.Bool
)

// SetReporter регистрирует активную реализацию Reporter.
// Вызывается один раз адаптером при инициализации (например SentryKernel.Init).
func SetReporter(r Reporter) {
	if r == nil {
		return
	}

	mu.Lock()
	reporter = r
	mu.Unlock()

	isConfigured.Store(true)
}

// IsConfigured сообщает, зарегистрирован ли реальный (не noop) Reporter.
// Полезно, если код хочет заранее понять, что репортинг реально куда-то улетит.
func IsConfigured() bool {
	return isConfigured.Load()
}

func get() Reporter {
	mu.RLock()
	defer mu.RUnlock()

	return reporter
}

// Flusher - опциональный интерфейс для Reporter. Некоторые реализации
// (например sentry-go) отправляют события асинхронно в фоновой горутине -
// без явного ожидания перед завершением процесса последнее событие можно
// потерять. Актуально для one-shot процессов без graceful shutdown
// (консольные команды: обычный сервис успевает отправить и так, пока живёт
// или в момент штатной остановки по SIGTERM).
type Flusher interface {
	Flush(timeout time.Duration) bool
}

// Flush синхронно ждёт отправки уже поставленных в очередь событий, если
// текущий Reporter это поддерживает (реализует Flusher). Для noop-репортера
// или реализации без поддержки Flush - no-op, возвращает true сразу.
func Flush(timeout time.Duration) bool {
	if f, ok := get().(Flusher); ok {
		return f.Flush(timeout)
	}

	return true
}

// reportedError - маркер "эта ошибка уже отправлена через errorreporter".
// Оборачивает исходную ошибку, сохраняя цепочку errors.Is/errors.As.
type reportedError struct {
	err error
}

func (e *reportedError) Error() string { return e.err.Error() }
func (e *reportedError) Unwrap() error { return e.err }

// WasReported сообщает, что ошибка уже была отправлена в error-трекер
// через этот пакет. Используется адаптерами верхних слоёв (например
// HTTP-транспортом), чтобы не отправить один и тот же инцидент дважды,
// когда ошибка репортится в одном слое (Domain/Infrastructure) и затем
// пробрасывается дальше до другого (Transport).
func WasReported(err error) bool {
	var re *reportedError

	return errors.As(err, &re)
}

// Capture отправляет ошибку в error-трекер с явно указанными опциями и
// возвращает ту же ошибку, помеченную как отправленная - это позволяет
// репортить и пробрасывать ошибку одной строкой:
//
//	if err != nil {
//	    return errorreporter.Capture(ctx, err, opts)
//	}
//
// Если err уже был помечен (WasReported), повторной отправки не будет -
// это и есть защита от дублей при прохождении ошибки через несколько слоёв.
// Обойти это можно через Options.Force (осознанная эскалация). Не прерывает
// выполнение - что делать дальше (пробросить, вернуть nil после fallback,
// залогировать), решает вызывающий код.
func Capture(ctx context.Context, err error, opts Options) error {
	if err == nil {
		return nil
	}

	if opts.Force || !WasReported(err) {
		get().Capture(ctx, err, opts)
	}

	if WasReported(err) {
		return err
	}

	return &reportedError{err: err}
}

// CaptureSoft фиксирует некритичную ошибку и НЕ прерывает выполнение.
// Используется, когда у вызывающего кода есть fallback и саму ошибку
// не нужно пробрасывать наверх - например Redis недоступен, но можно
// сходить в БД напрямую. Возвращаемое значение можно игнорировать -
// это чисто fire-and-forget сценарий.
func CaptureSoft(ctx context.Context, err error, tags map[string]string) error {
	return Capture(ctx, err, Options{Level: LevelWarning, Tags: tags})
}

// CaptureError фиксирует ошибку с уровнем error и возвращает её обратно,
// готовую к пробросу наверх - удобно для сценария "выбить ошибку и
// отправить в Sentry" одной строкой:
//
//	if err != nil {
//	    return errorreporter.CaptureError(ctx, err, tags)
//	}
func CaptureError(ctx context.Context, err error, tags map[string]string) error {
	return Capture(ctx, err, Options{Level: LevelError, Tags: tags})
}
