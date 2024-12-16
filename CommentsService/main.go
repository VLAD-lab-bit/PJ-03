package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	// Инициализация базы данных PostgreSQL
	db := InitDB()
	defer db.Close()

	// Создаём роутер
	router := mux.NewRouter()

	// Регистрируем маршруты и middleware
	api := NewAPI(db)
	api.RegisterRoutes(router)

	// Запуск HTTP-сервера
	log.Println("Starting comments service on :8081...")
	log.Fatal(http.ListenAndServe(":8081", router))
}
