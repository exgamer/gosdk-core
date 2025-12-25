package debug

import (
	"context"
	"fmt"
	"time"
)

// безопасный ключ (без коллизий со строками)
type ctxKey struct{}

var DebugCollectorKey = ctxKey{}

// AddDebugStep — удобный хелпер из бизнес-кода (где есть ctx)
func AddDebugStep(ctx context.Context, step string) {
	if dbg := GetDebugFromContext(ctx); dbg != nil {
		dbg.AddStep(step)
	}
}

// GetDebugFromContext достаёт коллектор из context.Context
func GetDebugFromContext(ctx context.Context) *DebugCollector {
	if ctx == nil {
		return nil
	}

	if dbg, ok := ctx.Value(DebugCollectorKey).(*DebugCollector); ok {
		return dbg
	}

	return nil
}

// WithDebugCollector кладёт коллектор в context
func WithDebugCollector(ctx context.Context, dbg *DebugCollector) context.Context {
	return context.WithValue(ctx, DebugCollectorKey, dbg)
}

func NewDebugCollector() *DebugCollector {
	return &DebugCollector{
		Start: time.Now(),
		Time:  make(map[string]any),
		Meta:  make(map[string]any),
		Steps: make([]string, 0),
		Cats:  make(map[string]*Category),
	}
}

type DebugCollector struct {
	Meta      map[string]any       `json:"meta,omitempty"`
	TotalTime string               `json:"total_time,omitempty"`
	Steps     []string             `json:"steps,omitempty"`
	Time      map[string]any       `json:"time,omitempty"`
	Cats      map[string]*Category `json:"cats,omitempty"` // <-- модульно, без SQL/HTTP

	Start time.Time `json:"-"`
}

type Category struct {
	TotalTime  string `json:"total_time,omitempty"`
	Count      int    `json:"count,omitempty"`
	Statements []any  `json:"statements,omitempty"`

	Duration time.Duration `json:"-"`
}

// Cat возвращает категорию (создаёт при отсутствии)
func (d *DebugCollector) Cat(name string) *Category {
	if d.Cats == nil {
		d.Cats = map[string]*Category{}
	}
	if c, ok := d.Cats[name]; ok {
		return c
	}
	c := &Category{Statements: make([]any, 0)}
	d.Cats[name] = c
	return c
}

// AddStep добавляет шаг (удобно для бизнес-логики)
func (d *DebugCollector) AddStep(step string) {
	d.Steps = append(d.Steps, step)
}

// AddStatement добавляет statement в категорию и обновляет total по категории
func (d *DebugCollector) AddStatement(cat string, duration time.Duration, stmt any) {
	c := d.Cat(cat)
	c.Statements = append(c.Statements, stmt)
	c.Duration += duration
	c.Count++
	c.TotalTime = d.getDurationAsString(c.Duration)
}

func (d *DebugCollector) CalculateTotalTime() {
	execTime := time.Since(d.Start)
	d.TotalTime = d.getDurationAsString(execTime)
}

// getDurationAsString - Duration в виде строки (s/ms/µs/ns)
func (d *DebugCollector) getDurationAsString(duration time.Duration) string {
	if duration >= time.Second {
		return fmt.Sprintf("%.3f s", duration.Seconds())
	}
	if duration >= time.Millisecond {
		return fmt.Sprintf("%.3f ms", float64(duration)/float64(time.Millisecond))
	}
	if duration >= time.Microsecond {
		return fmt.Sprintf("%.3f µs", float64(duration)/float64(time.Microsecond))
	}
	return fmt.Sprintf("%d ns", duration.Nanoseconds())
}
