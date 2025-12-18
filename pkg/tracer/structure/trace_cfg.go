package structure

type TraceConfig struct {
	// IsTraceEnabled - этот парамет нужен для включения трассировки или выключения
	IsTraceEnabled bool `mapstructure:"TRACE_IS_ENABLED"`
	// Url - хост урл jaeger-а
	Url string `mapstructure:"TRACE_URL"`
	// ServiceName -  название сервиса, в трейсах будет
	ServiceName string `mapstructure:"TRACE_SERVICE_NAME"`
	// IsHttpBodyEnabled -  этот параметр нужен для того чтобы мидлвар записывал в трэйс все входящие тела запроса -  HTTP BODY
	IsHttpBodyEnabled bool `mapstructure:"TRACE_IS_HTTP_BODY_ENABLED"`
}
