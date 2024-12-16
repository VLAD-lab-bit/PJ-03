package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"Task36a41/pkg/storage"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// API представляет структуру для API с доступом к хранилищу данных.
type API struct {
	storage *storage.Storage
}

// New создает новый экземпляр API.
func New(storage *storage.Storage) *API {
	return &API{storage: storage}
}

// RegisterRoutes регистрирует маршруты API.
func (api *API) RegisterRoutes(router *mux.Router) {
	// Middleware для request_id и логирования
	router.Use(RequestIDMiddleware)
	router.Use(LoggingMiddleware)

	// Регистрация маршрутов
	router.HandleFunc("/news/details", api.getNewsDetails).Methods(http.MethodGet)   // Обработчик деталей новости
	router.HandleFunc("/news/{n:[0-9]+}", api.getLastNPosts).Methods(http.MethodGet) // Ограничение для {n} только числами
	router.HandleFunc("/news", api.getNews).Methods(http.MethodGet)                  // Уже существующий маршрут
}

func (api *API) getLastNPosts(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("request_id")
	log.Printf("[Request ID: %s] Processing getLastNPosts", requestID)

	vars := mux.Vars(r)
	n, err := strconv.Atoi(vars["n"])
	if err != nil {
		log.Printf("[Request ID: %s] Invalid 'n' parameter: %v", requestID, vars["n"])
		http.Error(w, "Invalid number format", http.StatusBadRequest)
		return
	}

	// Получаем последние N постов из хранилища
	posts, err := api.storage.GetLastNPosts(n)
	if err != nil {
		log.Printf("[Request ID: %s] Error retrieving posts: %v", requestID, err)
		http.Error(w, "Error retrieving posts", http.StatusInternalServerError)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(posts); err != nil {
		log.Printf("[Request ID: %s] Error encoding response: %v", requestID, err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

// getNewsDetails обрабатывает запрос для получения деталей новости по ID.
func (api *API) getNewsDetails(w http.ResponseWriter, r *http.Request) {
	// Извлекаем request_id из контекста для логирования
	requestID := r.Context().Value("request_id")
	log.Printf("[Request ID: %s] Processing getNewsDetails", requestID)

	// Получаем параметр id из строки запроса
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		log.Printf("[Request ID: %s] Missing 'id' parameter", requestID)
		http.Error(w, "Missing 'id' parameter", http.StatusBadRequest)
		return
	}

	// Проверяем, что id — это валидное число
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("[Request ID: %s] Invalid 'id' parameter: %s", requestID, idStr)
		http.Error(w, "Invalid 'id' parameter format: must be a number", http.StatusBadRequest)
		return
	}

	// Получаем новость из хранилища по ID
	post, err := api.storage.GetPostByID(id)
	if err != nil {
		log.Printf("[Request ID: %s] Error retrieving post with ID %d: %v", requestID, id, err)
		http.Error(w, "Error retrieving post", http.StatusInternalServerError)
		return
	}

	// Если пост не найден
	if post == nil {
		log.Printf("[Request ID: %s] Post with ID %d not found", requestID, id)
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}

	// Возвращаем найденный пост
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(post); err != nil {
		log.Printf("[Request ID: %s] Error encoding response: %v", requestID, err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

// getNews обрабатывает запрос для поиска новостей.
func (api *API) getNews(w http.ResponseWriter, r *http.Request) {
	requestID := r.Context().Value("request_id")
	log.Printf("[Request ID: %s] Processing getNews", requestID)

	query := r.URL.Query().Get("s")
	log.Printf("[Request ID: %s] Search query: %s", requestID, query)

	pageStr := r.URL.Query().Get("page")
	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	log.Printf("[Request ID: %s] Pagination page: %d", requestID, page)

	itemsPerPage := 15
	offset := (page - 1) * itemsPerPage

	posts, total, err := api.storage.SearchPostsByTitle(r.Context(), query, itemsPerPage, offset)
	if err != nil {
		log.Printf("[Request ID: %s] Error retrieving posts: %v", requestID, err)
		http.Error(w, "Error retrieving posts", http.StatusInternalServerError)
		return
	}

	totalPages := (total + itemsPerPage - 1) / itemsPerPage
	log.Printf("[Request ID: %s] Retrieved %d posts (page %d of %d)", requestID, len(posts), page, totalPages)

	pagination := map[string]int{
		"current_page":   page,
		"total_pages":    totalPages,
		"items_per_page": itemsPerPage,
	}

	response := map[string]interface{}{
		"posts":      posts,
		"pagination": pagination,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("[Request ID: %s] Error encoding response: %v", requestID, err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

// RequestIDMiddleware генерирует или извлекает request_id для каждого запроса.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, передан ли request_id в заголовке
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			// Если request_id нет, генерируем новый
			requestID = uuid.New().String()
			log.Printf("Generated new Request ID: %s", requestID)
		} else {
			log.Printf("Using provided Request ID: %s", requestID)
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

		log.Printf("[Request ID: %s] Incoming request: %s %s from %s", requestID, r.Method, r.URL.Path, r.RemoteAddr)

		// Обертка для записи статуса ответа
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		// Логируем время завершения и статус ответа
		duration := time.Since(start)
		log.Printf("[Request ID: %s] Completed request: %s %s with status %d in %v",
			requestID, r.Method, r.URL.Path, rw.statusCode, duration)
	})
}

// Обертка для http.ResponseWriter для записи статуса ответа
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
