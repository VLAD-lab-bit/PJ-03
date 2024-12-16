package storage

import (
	"Task36a41/pkg/rss"
	"context"
	"testing"
	"time"
)

func setupDatabase(db *Storage) error {
	// Drop table если создано, иначе создать
	_, err := db.db.Exec(`DROP TABLE IF EXISTS posts;
        CREATE TABLE posts (
            id SERIAL PRIMARY KEY,
            title TEXT NOT NULL,
            content TEXT NOT NULL,
            pub_time BIGINT NOT NULL,
            link TEXT NOT NULL UNIQUE
        );`)
	return err
}

func TestSaveAndGetPosts(t *testing.T) {
	// Используем строку подключения к тестовой базе данных
	db, err := New("user=postgres password=vlad5043 dbname=News sslmode=disable")
	if err != nil {
		t.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	// Настраиваем базу данных
	if err := setupDatabase(db); err != nil {
		t.Fatalf("Error setting up database: %v", err)
	}

	// Очищаем таблицу перед началом тестов
	_, err = db.db.Exec("DELETE FROM posts")
	if err != nil {
		t.Fatalf("Error cleaning table: %v", err)
	}

	// Пример поста для тестирования
	post := rss.Post{
		Title:   "Test Post",
		Content: "This is a test post",
		PubDate: time.Now().Format(time.RFC1123Z), // Используем текущее время
		Link:    "http://example.com/test",
	}

	// Сохраняем пост
	err = db.SavePost(post)
	if err != nil {
		t.Fatalf("Error saving post: %v", err)
	}

	// Получаем посты
	posts, err := db.GetLastNPosts(1)
	if err != nil {
		t.Fatalf("Error getting posts: %v", err)
	}

	// Проверяем результат
	if len(posts) != 1 || posts[0].Title != post.Title {
		t.Errorf("Expected 1 post, got %d", len(posts))
	}
}

func TestSearchPostsByTitle(t *testing.T) {
	// Используем строку подключения к тестовой базе данных
	db, err := New("user=postgres password=vlad5043 dbname=News sslmode=disable")
	if err != nil {
		t.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	// Настраиваем базу данных
	if err := setupDatabase(db); err != nil {
		t.Fatalf("Error setting up database: %v", err)
	}

	// Пример данных для поиска
	posts := []rss.Post{
		{Title: "Go Programming Language", Content: "Learn Go", PubDate: time.Now().Format(time.RFC1123Z), Link: "http://example.com/go"},
		{Title: "Learning Python", Content: "Python basics", PubDate: time.Now().Format(time.RFC1123Z), Link: "http://example.com/python"},
		{Title: "Advanced Go Concepts", Content: "Deep dive into Go", PubDate: time.Now().Format(time.RFC1123Z), Link: "http://example.com/advanced-go"},
	}

	// Сохраняем тестовые данные
	for _, post := range posts {
		if err := db.SavePost(post); err != nil {
			t.Fatalf("Error saving post: %v", err)
		}
	}

	// Выполняем поиск
	ctx := context.Background()
	query := "Go"
	limit := 2
	offset := 0

	results, total, err := db.SearchPostsByTitle(ctx, query, limit, offset)
	if err != nil {
		t.Fatalf("Error searching posts: %v", err)
	}

	// Проверяем общее количество результатов
	if total != 2 {
		t.Errorf("Expected total 2, got %d", total)
	}

	// Проверяем количество возвращенных записей
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Проверяем правильность заголовков
	expectedTitles := []string{"Go Programming Language", "Advanced Go Concepts"}
	for i, result := range results {
		if result.Title != expectedTitles[i] {
			t.Errorf("Expected title %s, got %s", expectedTitles[i], result.Title)
		}
	}
}
