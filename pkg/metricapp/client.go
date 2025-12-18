package metricapp

import (
	"github.com/exgamer/gosdk-core/pkg/exception"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
	"sync"
	"time"
)

var MetricsApp *Metrics
var once sync.Once

func InitMetrics(serviceName string) *Metrics {
	m := &Metrics{serviceName: serviceName}
	m.initCollector()
	m.registrCollector()

	MetricsApp = m

	return MetricsApp
}

type Metrics struct {
	serviceName        string
	httpRequestMetrics *prometheus.HistogramVec
}

// initCollector - инициализируем наши метрики
func (m *Metrics) initCollector() {
	var (
		// Метрика для измерения  обработки запросов
		httpRequestDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:        MetricNameHttpRequest,
				Help:        "Histogram of response time for handler in seconds.",
				Buckets:     prometheus.DefBuckets, // Дефолтные интервалы для гистограммы
				ConstLabels: prometheus.Labels{"service": m.serviceName},
			},
			[]string{MetricLabelHttpStatus, MetricLabelHttpMethod, MetricLabelHttpUrl},
		)
	)

	m.httpRequestMetrics = httpRequestDuration
}

// registrCollector -  регистрируем наши метрики
func (m *Metrics) registrCollector() {
	once.Do(func() {
		prometheus.MustRegister(m.httpRequestMetrics)
	})
}

func MetricsMiddlewaree(c *gin.Context, now time.Time) {
	duration := time.Since(now).Seconds()

	// парсинг ошибок
	excep := c.Keys["exception"]

	var statusCode int
	switch v := excep.(type) {
	case *exception.AppException:
		statusCode = v.Code
	default:
		statusCode = c.Writer.Status()
	}

	if MetricsApp == nil || MetricsApp.httpRequestMetrics == nil {
		return
	}

	// Будьте осторожны, порядок передачи аргументов должны быть как и при инициализации
	MetricsApp.httpRequestMetrics.WithLabelValues(strconv.Itoa(statusCode), c.Request.Method, c.FullPath()).Observe(duration)
}
