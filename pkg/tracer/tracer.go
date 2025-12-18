package tracer

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gookit/goutil/netutil/httpctype"
	"github.com/gookit/goutil/netutil/httpheader"
	"gitlab.almanit.kz/jmart/gosdk/pkg/config"
	"gitlab.almanit.kz/jmart/gosdk/pkg/exception"
	span2 "gitlab.almanit.kz/jmart/gosdk/pkg/tracer/span"
	"gitlab.almanit.kz/jmart/gosdk/pkg/tracer/structure"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	trace2 "go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
)

// TraceClient - данная глобальная переменная нужна для пакета http-билдера в основном.
var TraceClient *Tracer

type Tracer struct {
	tp                 *tracesdk.TracerProvider
	cfg                *structure.TraceConfig
	IsEnabled          bool
	ServiceName        string
	tracingFlagEnabled int32
}

// InitTraceClient - создание клиента трассировки
func InitTraceClient() (*Tracer, error) {
	TraceClient = &Tracer{}
	// config init
	if err := TraceClient.initTraceConfig(); err != nil {
		return nil, err
	}

	if !TraceClient.cfg.IsTraceEnabled {
		return TraceClient, nil
	}

	// отключил провеку сертификата, так как на тесте были ошибки "x509: certificate signed by unknown authority error"
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	// Create the Jaeger exporter
	exp, err := jaeger.New(
		jaeger.WithCollectorEndpoint(
			jaeger.WithEndpoint(TraceClient.cfg.Url),
			jaeger.WithHTTPClient(&http.Client{Transport: transport}),
		),
	)

	if err != nil {
		return nil, err
	}

	tp := tracesdk.NewTracerProvider(
		// Always be sure to batch in production.
		tracesdk.WithBatcher(exp),
		// Record information about this application in a Resource.
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(TraceClient.cfg.ServiceName),
			attribute.String("environment", "development"),
			attribute.Int64("ID", 1),
		)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			b3.New(
				b3.WithInjectEncoding(b3.B3MultipleHeader),
			),
		),
	)
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		fmt.Printf("OpenTelemetry error: %v\n", err)
	}))

	TraceClient.tp = tp

	return TraceClient, nil
}

func (t *Tracer) EnableTracing() {
	atomic.StoreInt32(&TraceClient.tracingFlagEnabled, 1)
}

func (t *Tracer) DisableTracing() {
	atomic.StoreInt32(&TraceClient.tracingFlagEnabled, 0)
}

func (t *Tracer) IsTraceFlagEnabled() bool {
	return TraceClient.tracingFlagEnabled == 1
}

// Shutdown -
func (t *Tracer) Shutdown(ctx context.Context) error {
	return TraceClient.tp.Shutdown(ctx)
}

// InjectHttpTraceId -  записывает  trace id  в запрос, требует  *http.Request
func (t *Tracer) InjectHttpTraceId(ctx context.Context, req *http.Request) {
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
}

// isNeedEnableTrace -  функция для  мидлвара, который выключает флаг трассировки если не нашел заголовок, если нашел то включает
func (t *Tracer) IsNeedEnableTrace(traceSampled string) bool {
	if traceSampled == TraceDisable && TraceClient.tracingFlagEnabled == 1 {
		TraceClient.DisableTracing()

		return false
	}

	if traceSampled == TraceEnable && TraceClient.tracingFlagEnabled == 0 {
		TraceClient.EnableTracing()

		return true
	}

	if traceSampled == TraceEnable && TraceClient.tracingFlagEnabled == 1 {
		return true
	}

	return false
}

// MiddleWareExtractTraceId -  мидлвар который записывает трассировку
func (t *Tracer) MiddleWareExtractTraceId() gin.HandlerFunc {
	return func(c *gin.Context) {
		if TraceClient == nil || !TraceClient.cfg.IsTraceEnabled {
			c.Next()

			return
		}

		traceSampled := c.Request.Header.Get(ApiGwHeaderTraceSampled)
		if !TraceClient.IsNeedEnableTrace(traceSampled) {
			c.Next()

			return
		}

		// Извлечение контекста из заголовков запроса
		propagator := otel.GetTextMapPropagator()
		parentCtx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		parentCtx, span := TraceClient.CreateSpan(parentCtx, "["+c.Request.Method+"] "+c.FullPath())
		defer span.End()

		// парсинг body
		if TraceClient.cfg.IsHttpBodyEnabled {
			// нет смысла копировать тело запроса при наличии файла
			if !strings.HasPrefix(c.GetHeader(httpheader.ContentType), httpctype.MIMEDataForm) {
				bodyBytes, _ := io.ReadAll(c.Request.Body)
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				span.SetAttributes(attribute.String(span2.AttributeReqBody, string(bodyBytes)))
			}
		}

		c.Request = c.Request.WithContext(parentCtx)
		c.Next()

		// парсинг ошибок
		{
			excep := c.Keys["exception"]

			switch v := excep.(type) {
			case *exception.AppException:
				span.SetAttributes(attribute.Int(span2.AttributeRespHttpCode, v.Code))
				if v.Error != nil {
					span.SetAttributes(attribute.String(span2.AttributeRespErrMsg, v.Error.Error()))
				}
			default:
				span.SetAttributes(attribute.Int(span2.AttributeRespHttpCode, c.Writer.Status()))
			}
		}
	}
}

// CreateSpan - Создает родительский спан,и возвращает контекст, этот контекст нужен для дочернего спана.
// В случае если в ctx нет контекста родителя то создается контекст родителя
// Не забыть вызывать span.End()
func (t *Tracer) CreateSpan(ctx context.Context, name string) (context.Context, trace2.Span) {
	if TraceClient == nil || TraceClient.tp == nil {
		return context.Background(), noop.Span{}
	}

	return TraceClient.tp.Tracer(TraceClient.ServiceName).Start(ctx, name)
}

// CreateSpanWithCustomTraceId -  экспериментальный метод, создаем спан на основе кастомного трайс айди
func (t *Tracer) CreateSpanWithCustomTraceId(ctx context.Context, traceId, name string) (context.Context, trace2.Span, error) {
	tId, err := trace2.TraceIDFromHex(traceId)

	if err != nil {
		return nil, noop.Span{}, err
	}

	spanContext := trace2.NewSpanContext(trace2.SpanContextConfig{
		TraceID: tId,
	})

	ctx1 := trace2.ContextWithSpanContext(ctx, spanContext)
	ctx1, span := TraceClient.tp.Tracer(TraceClient.ServiceName).Start(ctx1, name)

	return ctx1, span, nil
}

// initTraceConfig -  инициализирует конфиг трассировки, читает  из файла  .env переменки
func (t *Tracer) initTraceConfig() error {
	if err := config.ReadEnv(); err != nil {
		return err
	}

	traceCfg := &structure.TraceConfig{}
	err := config.InitConfig(traceCfg)

	if err != nil {
		return err
	}

	TraceClient.cfg = traceCfg
	TraceClient.ServiceName = traceCfg.ServiceName
	TraceClient.IsEnabled = traceCfg.IsTraceEnabled

	return nil
}
