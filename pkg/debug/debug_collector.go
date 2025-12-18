package debug

import (
	"context"
	"github.com/gin-gonic/gin"
	"gitlab.almanit.kz/jmart/gosdk/pkg/helpers"
	"time"
)

const DebugKey string = "debug"

func GetDebugFromContext(ctx context.Context) *DebugCollector {
	if dbg, ok := ctx.Value(DebugKey).(*DebugCollector); ok {
		return dbg
	}

	return nil
}

func GetDebugCollectorFromGinContext(c *gin.Context) *DebugCollector {
	if dbg, ok := c.Request.Context().Value(DebugKey).(*DebugCollector); ok {
		return dbg
	}

	return nil
}

func AddDebugStep(ctx context.Context, step string) {
	if dbg := GetDebugFromContext(ctx); dbg != nil {
		dbg.Steps = append(dbg.Steps, step)
	}
}

func NewDebugCollector() *DebugCollector {
	return &DebugCollector{
		Start:       time.Now(),
		Time:        make(map[string]interface{}),
		Meta:        make(map[string]interface{}),
		SqlQueries:  NewSqlQueries(),
		HttpQueries: NewHttpQueries(),
		Steps:       make([]string, 0),
	}
}

type DebugCollector struct {
	Meta         map[string]interface{} `json:"meta,omitempty"`
	TotalTime    string                 `json:"total_time,omitempty"`
	Steps        []string               `json:"steps,omitempty"`
	HttpQueries  HttpQueries            `json:"http_queries,omitempty"`
	SqlQueries   SqlQueries             `json:"sql_queries,omitempty"`
	RedisQueries RedisQueries           `json:"redis_queries,omitempty"`
	Time         map[string]interface{} `json:"time,omitempty"`
	Start        time.Time              `json:"-"`
}

func (d *DebugCollector) addDebugMeta(c *gin.Context, key string, value interface{}) {
	if dbg := GetDebugCollectorFromGinContext(c); dbg != nil {
		dbg.Meta[key] = value
	}
}

func (d *DebugCollector) CalculateSqlTotalTime() {
	total := time.Duration(0)

	for _, statement := range d.SqlQueries.Statements {
		total += statement.Duration
	}

	d.SqlQueries.TotalTime = helpers.GetDurationAsString(total)
	d.SqlQueries.Duration = total
}

func (d *DebugCollector) CalculateHttpTotalTime() {
	total := time.Duration(0)

	for _, statement := range d.HttpQueries.Statements {
		total += statement.Duration
	}

	d.HttpQueries.TotalTime = helpers.GetDurationAsString(total)
	d.HttpQueries.Duration = total
}

func (d *DebugCollector) CalculateRedisTotalTime() {
	total := time.Duration(0)

	for _, statement := range d.RedisQueries.Statements {
		total += statement.Duration
	}

	d.RedisQueries.TotalTime = helpers.GetDurationAsString(total)
	d.RedisQueries.Duration = total
}

func (d *DebugCollector) CalculateTotalTime() {
	execTime := time.Since(d.Start)
	d.TotalTime = helpers.GetDurationAsString(execTime)
}
