package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func InitDB() *sql.DB {
	// Подключение к PostgreSQL
	connStr := "host=localhost port=5432 user=postgres password=vlad5043 dbname=comments_service sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}

	// Проверка соединения
	err = db.Ping()
	if err != nil {
		log.Fatalf("Failed to ping the database: %v", err)
	}

	// Создание таблицы комментариев
	query := `
	CREATE TABLE IF NOT EXISTS comments (
		id SERIAL PRIMARY KEY,
		news_id INT NOT NULL,
		parent_id INT DEFAULT NULL,
		content TEXT NOT NULL
	);`
	_, err = db.Exec(query)
	if err != nil {
		log.Fatalf("Failed to create comments table: %v", err)
	}

	fmt.Println("Database connected and initialized successfully.")
	return db
}

func SaveComment(db *sql.DB, comment *Comment) (int, error) {
	var id int
	query := `
		INSERT INTO comments (news_id, parent_id, content)
		VALUES ($1, $2, $3)
		RETURNING id;`
	err := db.QueryRow(query, comment.NewsID, comment.ParentID, comment.Content).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func GetCommentsByNewsID(db *sql.DB, newsID int) ([]Comment, error) {
	query := `
		SELECT id, news_id, parent_id, content
		FROM comments
		WHERE news_id = $1;`

	rows, err := db.Query(query, newsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var comment Comment
		var parentID sql.NullInt64
		err := rows.Scan(&comment.ID, &comment.NewsID, &parentID, &comment.Content)
		if err != nil {
			return nil, err
		}
		if parentID.Valid {
			parentIDInt := int(parentID.Int64)
			comment.ParentID = &parentIDInt
		}
		comments = append(comments, comment)
	}
	return comments, nil
}
