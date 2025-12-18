package debug

import "time"

func NewRedisQueries() RedisQueries {
	return RedisQueries{}
}

type RedisQueries struct {
	TotalTime  string           `json:"total_time,omitempty"`
	Duration   time.Duration    `json:"duration,omitempty"`
	Statements []RedisStatement `json:"statements,omitempty"`
}

type RedisStatement struct {
	Time      string        `json:"time,omitempty"`
	Operation string        `json:"operation,omitempty"`
	Keys      interface{}   `json:"keys,omitempty"`
	Error     string        `json:"error,omitempty"`
	Duration  time.Duration `json:"duration,omitempty"`
}
