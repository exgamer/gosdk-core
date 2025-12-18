package middleware

import (
	"github.com/gin-gonic/gin"
	"gitlab.almanit.kz/jmart/gosdk/pkg/http/helpers"
)

// FormattedResponseMiddleware Middleware для обработки ответа
func FormattedResponseMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		helpers.FormattedResponse(c)
	}
}
