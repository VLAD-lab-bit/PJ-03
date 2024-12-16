package rss

import (
	"testing"
	"time"
)

// Тест для функции FetchRSS, проверяющий успешное получение и парсинг RSS-фида из URL.
func TestFetchRSS(t *testing.T) {
	// Указываем URL с реальным RSS-фидом
	url := "https://habr.com/ru/rss/best/daily/?fl=ru"

	// Вызываем функцию FetchRSS для получения данных фида
	posts, err := FetchRSS(url)
	if err != nil {
		t.Fatalf("ошибка при получении RSS фида: %v", err)
	}

	// Проверяем, что фид содержит хотя бы одну публикацию
	if len(posts) == 0 {
		t.Fatal("данные не декодированы или публикации отсутствуют")
	}

	// Логируем количество и первые новости (для проверки успешного разбора)
	t.Logf("получено %d новостей", len(posts))
	for _, post := range posts[:3] { // Логируем первые три публикации для наглядности
		t.Logf("Заголовок: %s, Ссылка: %s, Дата: %s", post.Title, post.Link, post.PubDate)
	}
}

// Тест для функции FetchAllRSS, проверяющий асинхронное получение и парсинг из нескольких источников.
func TestFetchAllRSS(t *testing.T) {
	// URL нескольких RSS-фидов
	urls := []string{
		"https://habr.com/ru/rss/best/daily/?fl=ru",
		"https://habr.com/ru/rss/news/?fl=ru",
	}

	// Устанавливаем ограничение времени выполнения теста
	timeout := time.After(10 * time.Second)
	done := make(chan bool)

	go func() {
		// Вызываем функцию FetchAllRSS для получения данных из всех фидов
		allPosts := FetchAllRSS(urls)
		// Проверяем, что из всех фидов пришло хотя бы несколько публикаций
		if len(allPosts) == 0 {
			t.Error("данные не декодированы или публикации отсутствуют")
		}
		// Логируем общее количество новостей и информацию о первых публикациях
		t.Logf("получено %d новостей", len(allPosts))
		for _, post := range allPosts[:3] { // Логируем первые три публикации для наглядности
			t.Logf("Заголовок: %s, Ссылка: %s, Дата: %s", post.Title, post.Link, post.PubDate)
		}
		done <- true
	}()

	// Завершаем тест, если функция не завершилась за заданное время
	select {
	case <-timeout:
		t.Fatal("TestFetchAllRSS timed out")
	case <-done:
		// Успешное выполнение теста
	}
}
