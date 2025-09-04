package middleware

import (
	"net/http"

	"main/utils"

	"go.uber.org/zap"
)

// HTTPHeadersLoggingMiddleware логирует все входящие HTTP заголовки (для отладки)
func HTTPHeadersLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Собираем все заголовки в map для логирования
		headers := make(map[string][]string)
		for key, values := range r.Header {
			headers[key] = values
		}

		// Логируем все заголовки
		utils.Logger.Debug("Incoming HTTP request headers",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("remote_addr", r.RemoteAddr),
			zap.Any("headers", headers),
		)

		next.ServeHTTP(w, r)
	})
}
