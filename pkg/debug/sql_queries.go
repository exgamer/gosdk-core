package debug

import "time"

func NewSqlQueries() SqlQueries {
	return SqlQueries{}
}

type SqlQueries struct {
	TotalTime  string         `json:"total_time,omitempty"`
	Duration   time.Duration  `json:"duration,omitempty"`
	Statements []SqlStatement `json:"statements,omitempty"`
}

type SqlStatement struct {
	Time      string        `json:"time,omitempty"`
	Operation string        `json:"operation,omitempty"`
	Sql       string        `json:"sql,omitempty"`
	Error     string        `json:"error,omitempty"`
	Params    []interface{} `json:"params,omitempty"`
	Duration  time.Duration `json:"duration,omitempty"`
}
