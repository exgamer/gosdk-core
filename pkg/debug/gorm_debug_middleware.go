package debug

import (
	"gitlab.almanit.kz/jmart/gosdk/pkg/helpers"
	"gorm.io/gorm"
	"strings"
	"time"
)

func NewGormDebugMiddleware() gorm.Plugin {
	return &GormDebugMiddleware{}
}

// GormDebugMiddleware мидлвар для вывода запросов в дебаг инфу
type GormDebugMiddleware struct {
}

func (p *GormDebugMiddleware) Name() string {
	return "plugin_debug"
}

func (p *GormDebugMiddleware) Initialize(db *gorm.DB) error {
	// Before any operation callback
	db.Callback().Create().Before("gorm:before_create").Register("plugin_debug:before_create", p.before())
	db.Callback().Query().Before("gorm:before_query").Register("plugin_debug:before_query", p.before())
	db.Callback().Delete().Before("gorm:before_delete").Register("plugin_debug:before_delete", p.before())
	db.Callback().Update().Before("gorm:before_update").Register("plugin_debug:before_update", p.before())
	db.Callback().Row().Before("gorm:before_row").Register("plugin_debug:before_row", p.before())
	db.Callback().Raw().Before("gorm:before_raw").Register("plugin_debug:before_raw", p.before())

	// After any operation callback
	db.Callback().Create().After("gorm:after_create").Register("plugin_debug:after_create", p.after)
	db.Callback().Query().After("gorm:after_query").Register("plugin_debug:after_query", p.after)
	db.Callback().Delete().After("gorm:after_delete").Register("plugin_debug:after_delete", p.after)
	db.Callback().Update().After("gorm:after_update").Register("plugin_debug:after_update", p.after)
	db.Callback().Row().After("gorm:after_row").Register("plugin_debug:after_row", p.after)
	db.Callback().Raw().After("gorm:after_raw").Register("plugin_debug:after_raw", p.after)

	return nil
}

func (p *GormDebugMiddleware) before() func(*gorm.DB) {
	return func(db *gorm.DB) {
		startTime := time.Now()
		db.InstanceSet("start_time", startTime)
	}
}

func (p *GormDebugMiddleware) after(db *gorm.DB) {
	debugCollector := GetDebugFromContext(db.Statement.Context)

	if debugCollector == nil {

		return
	}

	statement := SqlStatement{}
	statement.Sql = db.Statement.SQL.String()
	statement.Params = db.Statement.Vars
	startTime, ok := db.InstanceGet("start_time")

	if !ok {
		return
	}

	// Рассчитываем продолжительность выполнения SQL-запроса
	duration := time.Since(startTime.(time.Time))
	statement.Duration = duration
	statement.Time = helpers.GetDurationAsString(duration)
	statement.Operation = p.detectOperationType(db.Statement.SQL.String())

	if db.Error != nil {
		statement.Error = db.Error.Error()
	}

	debugCollector.SqlQueries.Statements = append(debugCollector.SqlQueries.Statements, statement)

	debugCollector.CalculateSqlTotalTime()
}

func (p *GormDebugMiddleware) detectOperationType(sql string) string {
	if len(sql) == 0 {
		return "unknown"
	}

	sql = strings.ToUpper(strings.TrimSpace(sql))
	switch {
	case strings.HasPrefix(sql, "SELECT"):
		return "SELECT"
	case strings.HasPrefix(sql, "INSERT"):
		return "INSERT"
	case strings.HasPrefix(sql, "UPDATE"):
		return "UPDATE"
	case strings.HasPrefix(sql, "DELETE"):
		return "DELETE"
	default:
		return "raw"
	}
}
