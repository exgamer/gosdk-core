package metricapp

const (
	// metricNameHttpRequest -  название метрик для HTTP запросов (Method, Url, StatusCode)
	MetricNameHttpRequest = "http_request_metrics_info"
)

var (
	MetricLabelHttpStatus = "status"
	MetricLabelHttpMethod = "method"
	MetricLabelHttpUrl    = "url"
)
