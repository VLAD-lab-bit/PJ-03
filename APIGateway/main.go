package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	ID      int    `json:"id"`
	NewsID  int    `json:"news_id"`
	Content string `json:"content"`
}

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		ctx := context.WithValue(r.Context(), "request_id", requestID)
		r = r.WithContext(ctx)
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r)
	})
}
func logRequestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Обертка для захвата статуса ответа
		rw := &responseWriterWrapper{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		// Если обработчик обновил Request ID, он будет учтен
		log.Printf("Completed request: %s %s with status %d in %v [Client IP: %s]",
			r.Method, r.URL.Path, rw.statusCode, duration, getClientIP(r))
	})
}

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func forwardRequest(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	// Передаем текущий Request ID из контекста
	requestID := ctx.Value("request_id")
	if requestID != nil {
		req.Header.Set("X-Request-ID", requestID.(string))
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	// Проверяем и логируем новый Request ID из ответа
	externalRequestID := resp.Header.Get("X-Request-ID")
	if externalRequestID != "" {
		log.Printf("External service responded with Request ID: %s", externalRequestID)
	}

	return resp, nil
}

func processResponse(w http.ResponseWriter, resp *http.Response) {
	ctx := resp.Request.Context()
	requestID := ctx.Value("request_id").(string)

	// Проверяем и обновляем Request ID, если он пришел из внешнего сервиса
	externalRequestID := resp.Header.Get("X-Request-ID")
	if externalRequestID != "" {
		requestID = externalRequestID
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[Request ID: %s] Error reading response body: %v", requestID, err)
		http.Error(w, "Error reading response from service", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)

	log.Printf("[Request ID: %s] Completed request with status: %d", requestID, resp.StatusCode)
}

func getLastNPosts(w http.ResponseWriter, r *http.Request) {
	n := r.URL.Query().Get("n")
	if n == "" {
		http.Error(w, "Missing 'n' parameter", http.StatusBadRequest)
		return
	}
	apiURL := fmt.Sprintf("http://localhost:8082/news/%s", n)
	requestID := r.Context().Value("request_id").(string)
	log.Printf("%s [Request ID: %s] Fetching last %s posts",
		time.Now().Format(time.RFC3339), requestID, n)

	resp, err := forwardRequest(r.Context(), http.MethodGet, apiURL, nil)
	if err != nil {
		http.Error(w, "Error contacting news service", http.StatusInternalServerError)
		log.Printf("%s [Request ID: %s] Error contacting news service: %v",
			time.Now().Format(time.RFC3339), requestID, err)
		return
	}
	processResponse(w, resp)
}

// Получить новости с фильтрацией и пагинацией
func getNews(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("s")
	page := r.URL.Query().Get("page")
	if page == "" {
		page = "1"
	}

	baseURL := "http://localhost:8082/news"
	params := url.Values{}
	params.Add("s", query)
	params.Add("page", page)

	apiURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	resp, err := forwardRequest(r.Context(), http.MethodGet, apiURL, nil)
	if err != nil {
		http.Error(w, "Error contacting news service", http.StatusInternalServerError)
		log.Printf("Error contacting news service: %v", err)
		return
	}

	processResponse(w, resp)
}

// Получить детали новости
func getNewsDetails(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing 'id' parameter", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	requestID := ctx.Value("request_id").(string)

	// 1. Получаем информацию о новости
	newsAPIURL := fmt.Sprintf("http://localhost:8082/news/details?id=%s", url.QueryEscape(id))
	newsResp, err := forwardRequest(ctx, http.MethodGet, newsAPIURL, nil)
	if err != nil {
		http.Error(w, "Error contacting news service", http.StatusInternalServerError)
		log.Printf("[Request ID: %s] Error contacting news service: %v", requestID, err)
		return
	}
	defer newsResp.Body.Close()

	// Обновляем Request ID
	externalRequestID := newsResp.Header.Get("X-Request-ID")
	if externalRequestID != "" {
		requestID = externalRequestID
	}

	if newsResp.StatusCode != http.StatusOK {
		http.Error(w, "Failed to get news details", newsResp.StatusCode)
		processResponse(w, newsResp)
		return
	}

	// 2. Получаем комментарии для новости
	commentsAPIURL := fmt.Sprintf("http://localhost:8081/comments?news_id=%s", url.QueryEscape(id))
	commentsResp, err := forwardRequest(r.Context(), http.MethodGet, commentsAPIURL, nil)
	if err != nil {
		http.Error(w, "Error contacting comments service", http.StatusInternalServerError)
		log.Printf("%s [Request ID: %s] Error contacting comments service: %v", time.Now().Format(time.RFC3339), requestID, err)
		return
	}
	defer commentsResp.Body.Close()

	if commentsResp.StatusCode != http.StatusOK {
		http.Error(w, "Failed to get comments", commentsResp.StatusCode)
		processResponse(w, commentsResp)
		return
	}

	// 3. Объединяем информацию о новости и комментарии
	newsBody, err := io.ReadAll(newsResp.Body)
	if err != nil {
		http.Error(w, "Error reading news response", http.StatusInternalServerError)
		log.Printf("%s [Request ID: %s] Error reading news response: %v", time.Now().Format(time.RFC3339), requestID, err)
		return
	}

	commentsBody, err := io.ReadAll(commentsResp.Body)
	if err != nil {
		http.Error(w, "Error reading comments response", http.StatusInternalServerError)
		log.Printf("%s [Request ID: %s] Error reading comments response: %v", time.Now().Format(time.RFC3339), requestID, err)
		return
	}

	// Объединяем информацию о новости и комментарии в одну структуру
	type NewsWithComments struct {
		News     json.RawMessage `json:"news"`
		Comments json.RawMessage `json:"comments"`
	}

	response := NewsWithComments{
		News:     newsBody,
		Comments: commentsBody,
	}

	// Отправляем объединенный ответ клиенту
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Преобразуем данные в JSON и отправляем клиенту
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		log.Printf("%s [Request ID: %s] Error encoding response: %v", time.Now().Format(time.RFC3339), requestID, err)
	}
	// Логируем успешное завершение обработки
	log.Printf("%s [Request ID: %s] Successfully retrieved details for news ID: %s", time.Now().Format(time.RFC3339), requestID, id)
}

// Добавить комментарий
func addComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	apiURL := "http://localhost:8081/comments"

	resp, err := forwardRequest(r.Context(), http.MethodPost, apiURL, r.Body)
	if err != nil {
		http.Error(w, "Error contacting comments service", http.StatusInternalServerError)
		log.Printf("Error contacting comments service: %v", err)
		return
	}
	defer resp.Body.Close()

	// Извлечение Request ID из заголовков ответа
	externalRequestID := resp.Header.Get("X-Request-ID")
	if externalRequestID != "" {
		log.Printf("%s [Request ID: %s] Received external Request ID from comments service: %s", time.Now().Format(time.RFC3339), r.Context().Value("request_id").(string), externalRequestID)
	}

	processResponse(w, resp)
}

// Извлечение IP-адреса клиента
func getClientIP(r *http.Request) string {
	// Проверка заголовка X-Forwarded-For
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return forwarded
	}
	// Если X-Forwarded-For отсутствует, используем RemoteAddr
	return r.RemoteAddr
}

// Получить комментарии для новости
func getComments(w http.ResponseWriter, r *http.Request) {
	newsID := r.URL.Query().Get("news_id")
	if newsID == "" {
		http.Error(w, "Missing 'news_id' parameter", http.StatusBadRequest)
		return
	}

	apiURL := fmt.Sprintf("http://localhost:8081/comments?news_id=%s", url.QueryEscape(newsID))

	resp, err := forwardRequest(r.Context(), http.MethodGet, apiURL, nil)
	if err != nil {
		http.Error(w, "Error contacting comments service", http.StatusInternalServerError)
		log.Printf("Error contacting comments service: %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Error reading response from comments service", http.StatusInternalServerError)
		log.Printf("Error reading response: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/news", getNews)
	mux.HandleFunc("/news/last", getLastNPosts)
	mux.HandleFunc("/news/details", getNewsDetails)
	mux.HandleFunc("/news/comments", getComments)
	mux.HandleFunc("/news/comments/add", addComment)

	handler := requestIDMiddleware(logRequestMiddleware(mux))

	log.Println("API Gateway is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
