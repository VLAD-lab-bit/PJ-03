package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"Task36a41/pkg/api"
	"Task36a41/pkg/config"
	"Task36a41/pkg/rss"
	"Task36a41/pkg/storage"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Middleware для генерации и логирования request_id
func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.URL.Query().Get("request_id")
		if requestID == "" {
			requestID = uuid.NewString()
		}
		r.Header.Set("X-Request-ID", requestID)

		next.ServeHTTP(w, r)

		log.Printf("Request ID: %s, IP: %s, Method: %s, URL: %s, Status: %d",
			requestID, r.RemoteAddr, r.Method, r.URL.Path, http.StatusOK)
	})
}

func main() {
	// Загружаем конфигурацию
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Инициализируем подключение к базе данных
	db, err := storage.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	// Создаем API
	apiService := api.New(db)

	// Загружаем RSS-фиды каждые RequestPeriod минут
	go func() {
		for {
			posts := rss.FetchAllRSS(cfg.RSS)
			if err := db.SavePosts(posts); err != nil {
				log.Println("Error saving posts:", err)
			}
			time.Sleep(time.Duration(cfg.RequestPeriod) * time.Minute)
		}
	}()

	// Настраиваем маршрутизатор и регистрируем маршруты
	router := mux.NewRouter()
	router.Use(requestIDMiddleware) // Добавляем middleware
	apiService.RegisterRoutes(router)

	// Добавляем обработчик для статических файлов фронтенда
	router.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("./webapp"))))

	// Запускаем сервер
	port := cfg.ServerPort
	fmt.Printf("Server running on port %d\n", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), router); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
