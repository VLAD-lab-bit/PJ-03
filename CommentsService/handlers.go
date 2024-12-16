package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// AddCommentHandler — обработчик для добавления комментария
func AddCommentHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var comment Comment
	err := json.NewDecoder(r.Body).Decode(&comment)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Проверяем комментарий через сервис цензурирования
	if !checkCensorship(comment.Content, r) {
		http.Error(w, "Comment rejected by censorship service", http.StatusBadRequest)
		return
	}

	// Сохраняем комментарий в БД
	id, err := SaveComment(db, &comment)
	if err != nil {
		http.Error(w, "Failed to save comment", http.StatusInternalServerError)
		return
	}
	comment.ID = id

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comment)
}

// GetCommentsHandler — обработчик для получения комментариев по ID новости
func GetCommentsHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	newsIDStr := r.URL.Query().Get("news_id")
	newsID, err := strconv.Atoi(newsIDStr)
	if err != nil {
		http.Error(w, "Invalid news_id parameter", http.StatusBadRequest)
		return
	}

	comments, err := GetCommentsByNewsID(db, newsID)
	if err != nil {
		http.Error(w, "Failed to get comments", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comments)
}

func checkCensorship(content string, r *http.Request) bool {
	client := &http.Client{}
	req, err := http.NewRequest("POST", "http://localhost:8082/censor", strings.NewReader(content))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")

	// Передаём trace_id в запрос
	traceID := r.Header.Get("X-Trace-ID")
	if traceID != "" {
		req.Header.Set("X-Trace-ID", traceID)
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
