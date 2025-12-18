### Нужны переменные окружения в файле <code>.env</code>
```env
TRACE_IS_ENABLED=true
TRACE_URL=http://localhost:14268/api/traces
TRACE_SERVICE_NAME=banner-service
```

<hr style="border: 1px solid orange;"/>

###  Если есть необходимость создать http запрос с трассировкой
```go
func main() {
	ctx := context.Background()

	builder := builder.NewGetHttpRequestBuilderWithCtx[interface{}](ctx, "https://google.com")

	_, err := builder.GetResult()
}
```

**Примечаение** надо  инициализировать клиента трассировки

```go
    traceClient, err := tracer.InitTraceClient()
    
    if err != nil {
        return err
    }
```
Данный код инициализирует подключение к Jaeger.И если есть необходимость в этом клиенте где-то еще, то уже на свое усмотрение прокидывать куда надо. 

<hr style="border: 1px solid orange;"/>

###  Если есть необходимость трассировать входящие запросы то есть Middleware

```go
	v1 := router.Group("/banner/v1")
	v1.Use(tracer.MiddleWareExtractTraceId())
```

При использовании этого мидлвара рекомендуется использовать внутри хендлера <code>context.Request.Context()</code>  
<hr style="border: 1px solid orange;"/>

###  Если есть необходимость создать спан
```go
    ctx, span = tracer.CreateSpan(context, "handler")
    defer span.End()
```
**Примечаение: **
1) закрыть текущий спан надо обязательно, если не закроете, то данный спан не попадет в систему трассировки
2)   <code>CreateSpan</code> создает родительский спан если в контексте, который передали в качестве аргумента, нет ключа "ctx_parent". Если есть то создает дочерний спан
