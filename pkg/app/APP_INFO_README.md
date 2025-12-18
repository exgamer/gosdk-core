### Использование APP INFO в приложении

<hr style="border: 1px solid orange;"/>

### Используем middleware RequestMiddleware

```go
	app.Router.Use(httpMiddleware.RequestMiddleware(app.BaseConfig))
```

### Получение AppInfo в Http Handler

```go
"gitlab.almanit.kz/jmart/gosdk/pkg/gin"

	appInfo := gin.GetAppInfoCollectorFromGinContext(c)
```

### Получение AppInfo в сервисах

```go
    "gitlab.almanit.kz/jmart/gosdk/pkg/helpers"

	appInfo := helpers.GetAppInfoFromContext(ctx)
```