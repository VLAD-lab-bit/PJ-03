package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"Task36a41/pkg/rss"

	_ "github.com/lib/pq" // PostgreSQL драйвер
)

// Storage представляет структуру для работы с БД.
type Storage struct {
	db *sql.DB
}

// MockStorage - имитация хранилища для тестирования
type MockStorage struct {
	posts []rss.Post
}

// NewMockStorage создает новый экземпляр MockStorage с заданными постами
func NewMockStorage(posts []rss.Post) *MockStorage {
	return &MockStorage{posts: posts}
}

// GetLastNPosts возвращает последние N публикаций из имитационного хранилища
func (m *MockStorage) GetLastNPosts(n int) ([]rss.Post, error) {
	if n > len(m.posts) {
		n = len(m.posts)
	}
	return m.posts[:n], nil
}

// New создает новое подключение к базе данных и сбрасывает таблицу
func New(connectionString string) (*Storage, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("could not connect to database: %v", err)
	}

	// Проверим соединение
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("could not ping database: %v", err)
	}

	// Создаем и сбрасываем таблицу
	if err := resetTable(db); err != nil {
		return nil, fmt.Errorf("could not reset table: %v", err)
	}

	return &Storage{db: db}, nil
}

// resetTable сбрасывает таблицу и создаёт её заново
func resetTable(db *sql.DB) error {
	// Удаляем таблицу, если она существует
	_, err := db.Exec(`DROP TABLE IF EXISTS posts;`)
	if err != nil {
		return fmt.Errorf("could not drop table: %v", err)
	}

	// Создаем таблицу заново
	_, err = db.Exec(`
		CREATE TABLE posts (
			id SERIAL PRIMARY KEY,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			pub_time BIGINT DEFAULT 0,
			link TEXT NOT NULL UNIQUE
		);
	`)
	if err != nil {
		return fmt.Errorf("could not create table: %v", err)
	}

	return nil
}

// Close закрывает соединение с базой данных.
func (s *Storage) Close() error {
	return s.db.Close()
}

// SavePost сохраняет одну публикацию в БД.
func (s *Storage) SavePost(post rss.Post) error {
	query := `
        INSERT INTO posts (title, content, pub_time, link)
        VALUES ($1, $2, $3, $4)
        RETURNING id
    `
	// Парсим дату публикации
	var pubTime time.Time
	var err error
	timeFormats := []string{
		time.RFC1123Z,
		time.RFC1123,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 MST",
	}

	for _, format := range timeFormats {
		pubTime, err = time.Parse(format, post.PubDate)
		if err == nil {
			break
		}
	}

	if err != nil {
		return fmt.Errorf("couldn't parse publication time: %v", err)
	}

	unixTime := pubTime.Unix()

	// Вставляем пост в таблицу и извлекаем его ID
	var id int
	err = s.db.QueryRow(query, post.Title, post.Content, unixTime, post.Link).Scan(&id)
	if err != nil {
		return fmt.Errorf("couldn't insert post: %v", err)
	}

	// Устанавливаем ID в структуру post
	post.ID = id
	return nil
}

// SavePosts сохраняет несколько публикаций в БД с использованием транзакции.
func (s *Storage) SavePosts(posts []rss.Post) error {
	// Начинаем транзакцию
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("could not begin transaction: %v", err)
	}
	defer tx.Rollback() // откатим транзакцию в случае ошибки

	for _, post := range posts {
		if err := s.SavePostTx(tx, post); err != nil {
			return fmt.Errorf("could not save post: %v", err)
		}
	}

	// Фиксируем транзакцию
	return tx.Commit()
}

// SavePostTx сохраняет одну публикацию в БД внутри транзакции.
func (s *Storage) SavePostTx(tx *sql.Tx, post rss.Post) error {
	query := `
        INSERT INTO posts (title, content, pub_time, link)
        VALUES ($1, $2, $3, $4)
        RETURNING id
    `
	// Парсим дату публикации
	var pubTime time.Time
	var err error
	timeFormats := []string{
		time.RFC1123Z,
		time.RFC1123,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 MST",
	}

	for _, format := range timeFormats {
		pubTime, err = time.Parse(format, post.PubDate)
		if err == nil {
			break
		}
	}

	if err != nil {
		return fmt.Errorf("couldn't parse publication time: %v", err)
	}

	unixTime := pubTime.Unix()

	// Вставляем пост в таблицу и извлекаем его ID
	var id int
	err = tx.QueryRow(query, post.Title, post.Content, unixTime, post.Link).Scan(&id)
	if err != nil {
		return fmt.Errorf("couldn't insert post: %v", err)
	}

	// Устанавливаем ID в структуру post
	post.ID = id
	return nil
}

// GetLastNPosts возвращает последние N публикаций.
func (s *Storage) GetLastNPosts(n int) ([]rss.Post, error) {
	query :=
		`SELECT id, title, content, pub_time, link
		FROM posts
		ORDER BY pub_time DESC
		LIMIT $1`

	rows, err := s.db.Query(query, n)
	if err != nil {
		return nil, fmt.Errorf("could not get posts: %v", err)
	}
	defer rows.Close()

	var posts []rss.Post
	for rows.Next() {
		var post rss.Post
		var pubTime int64

		// Извлекаем pub_time как Unix timestamp
		if err := rows.Scan(&post.ID, &post.Title, &post.Content, &pubTime, &post.Link); err != nil {
			return nil, fmt.Errorf("could not scan post: %v", err)
		}

		// Преобразуем Unix timestamp обратно в строку в формате RFC1123Z
		post.PubDate = time.Unix(pubTime, 0).Format(time.RFC1123Z)
		posts = append(posts, post)
	}

	return posts, nil
}

func (s *Storage) GetPostByID(id int) (*rss.Post, error) {
	query := `SELECT id, title, content, pub_time, link FROM posts WHERE id = $1`

	var post rss.Post
	var pubTime int64

	// Выполняем запрос
	err := s.db.QueryRow(query, id).Scan(&post.ID, &post.Title, &post.Content, &pubTime, &post.Link)
	if err != nil {
		if err == sql.ErrNoRows {
			// Если запись с таким ID не найдена, возвращаем nil
			return nil, nil
		}
		return nil, fmt.Errorf("could not get post: %v", err)
	}

	// Преобразуем pub_time из Unix timestamp в строку формата RFC1123Z
	post.PubDate = time.Unix(pubTime, 0).Format(time.RFC1123Z)
	return &post, nil
}

// SearchPostsByTitle ищет новости по заголовку с учетом пагинации.
func (s *Storage) SearchPostsByTitle(ctx context.Context, query string, limit, offset int) ([]rss.Post, int, error) {
	// Сначала получаем общее количество результатов для пагинации
	var totalCount int
	countQuery := `SELECT COUNT(*) FROM posts WHERE title ILIKE $1`
	err := s.db.QueryRowContext(ctx, countQuery, "%"+query+"%").Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("could not count posts: %v", err)
	}

	// Затем выполняем запрос на получение нужной страницы
	selectQuery := `SELECT id, title, content, pub_time, link FROM posts WHERE title ILIKE $1 ORDER BY pub_time DESC LIMIT $2 OFFSET $3`
	rows, err := s.db.QueryContext(ctx, selectQuery, "%"+query+"%", limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("could not get posts: %v", err)
	}
	defer rows.Close()

	var posts []rss.Post
	for rows.Next() {
		var post rss.Post
		var pubTime int64
		if err := rows.Scan(&post.ID, &post.Title, &post.Content, &pubTime, &post.Link); err != nil {
			return nil, 0, fmt.Errorf("could not scan post: %v", err)
		}
		post.PubDate = time.Unix(pubTime, 0).Format(time.RFC1123Z)
		posts = append(posts, post)
	}

	return posts, totalCount, nil
}
