package main

type Comment struct {
	ID       int    `json:"id"`
	NewsID   int    `json:"news_id"`
	ParentID *int   `json:"parent_id,omitempty"` // NULL, если это не ответ на другой комментарий
	Content  string `json:"content"`
}
