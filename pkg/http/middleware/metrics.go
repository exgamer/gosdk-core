package middleware

import (
	"github.com/gin-gonic/gin"
	"gitlab.almanit.kz/jmart/gosdk/pkg/metricapp"
	"time"
)

// MetricsCollect - мидлвар для обработки HTTP запросов метрик
func MetricsCollect() gin.HandlerFunc {
	return func(c *gin.Context) {
		now := time.Now()
		c.Next()

		metricapp.MetricsMiddlewaree(c, now)
	}
}
