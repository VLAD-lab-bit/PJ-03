package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// RequestIDMiddleware генерирует или извлекает request_id для каждого запроса.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, передан ли request_id в заголовке
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			// Если request_id нет, генерируем новый
			requestID = uuid.New().String()
		}

		// Добавляем request_id в контекст запроса
		ctx := context.WithValue(r.Context(), "request_id", requestID)

		// Устанавливаем request_id в заголовок ответа
		w.Header().Set("X-Request-ID", requestID)

		// Передаем обработку запроса дальше
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// LoggingMiddleware логирует все запросы, включая информацию о запросе.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Извлекаем request_id из контекста
		requestID := r.Context().Value("request_id")
		if requestID == nil {
			requestID = "unknown"
		}

		// Логируем запрос
		log.Printf("IP: %s, Method: %s, URL: %s, Request ID: %s, Started at: %s", r.RemoteAddr, r.Method, r.URL.Path, requestID, start.Format(time.RFC3339))

		// Продолжаем обработку запроса
		next.ServeHTTP(w, r)

		// Логируем время завершения
		duration := time.Since(start)
		log.Printf("Request ID: %s, Method: %s, URL: %s, Duration: %v", requestID, r.Method, r.URL.Path, duration)
	})
}
