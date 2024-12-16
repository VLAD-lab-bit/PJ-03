package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// API представляет структуру для API с доступом к базе данных.
type API struct {
	db *sql.DB
}

// NewAPI создает новый экземпляр API.
func NewAPI(db *sql.DB) *API {
	return &API{db: db}
}

// RegisterRoutes регистрирует маршруты API и middleware.
func (api *API) RegisterRoutes(router *mux.Router) {
	// Добавляем middleware для request_id и логирования
	router.Use(RequestIDMiddleware)
	router.Use(LoggingMiddleware)

	// Регистрация маршрутов
	router.HandleFunc("/comments", api.AddCommentHandler).Methods(http.MethodPost)
	router.HandleFunc("/comments", api.GetCommentsHandler).Methods(http.MethodGet)
}

// AddCommentHandler — обработчик для добавления комментария.
func (api *API) AddCommentHandler(w http.ResponseWriter, r *http.Request) {
	var comment Comment
	err := json.NewDecoder(r.Body).Decode(&comment)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Проверяем комментарий через сервис цензурирования
	if !api.checkCensorship(comment.Content) {
		http.Error(w, "Comment rejected by censorship service", http.StatusBadRequest)
		return
	}

	// Сохраняем комментарий в БД
	id, err := SaveComment(api.db, &comment)
	if err != nil {
		http.Error(w, "Failed to save comment", http.StatusInternalServerError)
		return
	}
	comment.ID = id

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comment)
}

// GetCommentsHandler — обработчик для получения комментариев по ID новости.
func (api *API) GetCommentsHandler(w http.ResponseWriter, r *http.Request) {
	newsIDStr := r.URL.Query().Get("news_id")
	newsID, err := strconv.Atoi(newsIDStr)
	if err != nil {
		http.Error(w, "Invalid news_id parameter", http.StatusBadRequest)
		return
	}

	comments, err := GetCommentsByNewsID(api.db, newsID)
	if err != nil {
		http.Error(w, "Failed to get comments", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comments)
}

func (api *API) checkCensorship(content string) bool {
	client := &http.Client{}

	// Подготавливаем тело запроса в формате JSON
	censorshipRequest := map[string]string{
		"text": content,
	}

	// Кодируем тело запроса в JSON
	jsonData, err := json.Marshal(censorshipRequest)
	if err != nil {
		log.Printf("Error marshaling censorship request: %v", err)
		return false
	}

	// Создаем новый запрос
	req, err := http.NewRequest("POST", "http://localhost:8083/censor", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error creating censorship request: %v", err)
		return false
	}

	// Устанавливаем заголовки
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос к сервису цензуры
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error calling censorship service: %v", err)
		return false
	}
	defer resp.Body.Close()

	// Если статус ответа не 200 OK, выводим ошибку и возвращаем false
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Printf("Censorship service returned an error: %s", body)
		return false
	}

	// Возвращаем true, если все в порядке
	return true
}
