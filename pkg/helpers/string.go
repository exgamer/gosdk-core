package helpers

import (
	"fmt"
	"time"
)

// GetDurationAsString - Возвращает Duration в виде строки с указанием секунды, миллисекунды и тп
func GetDurationAsString(duration time.Duration) string {
	result := ""

	if duration >= time.Second {
		result = fmt.Sprintf("%.3f s", duration.Seconds())
	} else if duration >= time.Millisecond {
		result = fmt.Sprintf("%.3f ms", float64(duration)/float64(time.Millisecond))
	} else if duration >= time.Microsecond {
		result = fmt.Sprintf("%.3f µs", float64(duration)/float64(time.Microsecond))
	} else {
		result = fmt.Sprintf("%d ns", duration.Nanoseconds())
	}

	return result
}
