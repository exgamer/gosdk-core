✅ Использование debug-информации в эндпойнтах

**Функционал показывает sql запросы, http запросы и обращения в redis**

Если переменная оружения APP_ENV = 'prod' функционал работать не будет

Чтобы включить сбор и вывод debug-информации при обработке HTTP-запросов, выполните следующие шаги:

1. Указать заголовок в запросе
   Добавьте следующий HTTP-заголовок:

Apelsin: sanya

Это активирует сбор debug-информации по запросу.

2. Подключить middleware RequestMiddleware
   В маршрутизаторе обязательно подключите middleware:

```go
routes.Use(httpMiddleware.RequestMiddleware(baseConfig))
```

3. Подключить middleware FormattedResponseMiddleware
   Также подключите middleware для форматированного ответа:
```go
routes.Use(httpMiddleware.FormattedResponseMiddleware())
```

4. Получить контекст gin
   В каждом handler'е получайте контекст с помощью:
```go
"github.com/gin-gonic/gin"

c *gin.Context

ctx := c.Request.Context()
```

5. Использовать корректный ctx во всех функциях
   Передавайте ctx, полученный через c.Request.Context(), во все вызовы, где требуется логирование, запросы в БД и т. п.
   Это обязательное условие для корректной работы debug-сборщика.

6. Добавить свои debug-шаги
   Для логирования пользовательских этапов используйте:
```go
    debug.AddDebugStep(ctx, "Моя инфа")
```
Это добавит сообщение в общий debug-лог запроса.

7) В результате ответ от эндпойнтов будет примерно такой:

```json
{
    "success": true,
    "data": [
        {
            "id": 1,
            "name": "listing_page",
            "caption": "Листинг"
        }
    ],
    "debug": {
         "total_time": "4.758 ms",
         "http_queries": {
            "total_time": "260.873 ms",
            "statements": [
                {
                    "time": "260.873 ms",
                    "status": 200,
                    "timeout": "30.000 s",
                    "method": "POST",
                    "url": "http://company-service.loc/company/v1/company/by-bins",
                    "curl": "curl -X 'POST' -d '' -H 'Accept-Language: ru' -H 'City-Id: 443' -H 'Content-Type: application/json' -H 'Request-Id: d535ce60-47cc-4805-83cc-ce2f48c890c2' -H 'X-Client-Name: company-registration-service' 'http://company-service.loc/company/v1/company/by-bins'",
                    "raw_payload": "{\"bins\":[\"880102300353\"]}",
                    "headers": {
                        "Accept-Language": "ru",
                        "City-Id": "443",
                        "Request-Id": "d535ce60-47cc-4805-83cc-ce2f48c890c2"
                    },
                    "duration": 260872709
                }
            ]
        },
        "sql_queries": {
            "total_time": "4.708 µs",
            "statements": [
                {
                    "time": "1.708 µs",
                    "operation": "SELECT",
                    "sql": "SELECT count(*) FROM `rates` LEFT JOIN rates_store_locations rsl ON rsl.rate_id = rates.id GROUP BY `rates`.`id` ORDER BY rates.id DESC",
                    "duration": 1708
                },
                {
                    "time": "3.000 µs",
                    "operation": "SELECT",
                    "sql": "SELECT rates.*, GROUP_CONCAT(rsl.store_location_id) AS store_location_ids_string FROM `rates` LEFT JOIN rates_store_locations rsl ON rsl.rate_id = rates.id GROUP BY `rates`.`id` ORDER BY rates.id DESC LIMIT ? OFFSET ?",
                    "params": [
                        30,
                        30
                    ],
                    "duration": 3000
                }
            ]
        },
        "redis_queries": {
            "total_time": "43.518 ms",
            "statements": [
                {
                    "time": "43.518 ms",
                    "operation": "GET",
                    "keys": "banner-service:locations",
                    "duration": 43517666
                }
            ]
        },
        "meta": {
            "id": "27e77282-4117-4c29-bbc9-9b03b5b854a7",
            "method": "GET",
            "url": "/banner/v1/location"
        }
    }
}
```