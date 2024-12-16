package rss

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
)

// Post представляет собой структуру для одной публикации (статьи) в RSS.
type Post struct {
	ID      int    `json:"id"`         // Уникальный идентификатор
	Title   string `xml:"title"`       // Заголовок статьи
	Link    string `xml:"link"`        // Ссылка на оригинальную статью
	PubDate string `xml:"pubDate"`     // Дата публикации
	Content string `xml:"description"` // Описание или краткое содержание статьи
}

// RSSFeed описывает структуру RSS-ленты.
type RSSFeed struct {
	Channel struct {
		Title string `xml:"title"` // Название канала RSS
		Items []Post `xml:"item"`  // Массив публикаций
	} `xml:"channel"`
}

// FetchRSS делает HTTP-запрос к RSS-ленте и возвращает массив публикаций.
func FetchRSS(url string) ([]Post, error) {
	client := &http.Client{Timeout: 10 * time.Second} // Устанавливаем таймаут для запроса

	resp, err := client.Get(url) // Выполняем запрос по ссылке RSS
	if err != nil {
		return nil, fmt.Errorf("failed to fetch RSS: %v", err)
	}
	defer resp.Body.Close()

	// Читаем содержимое ответа
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read RSS response body: %v", err)
	}

	// Парсим XML-ответ в структуру RSSFeed
	var feed RSSFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("failed to unmarshal RSS XML: %v", err)
	}

	// Парсим дату публикации в каждом посте в формате Unix
	for i, item := range feed.Channel.Items {
		pubDate, err := time.Parse("Mon, 02 Jan 2006 15:04:05 MST", item.PubDate)
		if err != nil {
			log.Printf("Ошибка парсинга даты у статьи %s: %v. Дата: %s", item.Title, err, item.PubDate)
			continue
		}
		// Перезаписываем поле PubDate в формате Unix (для совместимости с базой данных)
		feed.Channel.Items[i].PubDate = time.Unix(pubDate.Unix(), 0).Format(time.RFC1123Z)
	}

	// Возвращаем массив публикаций
	return feed.Channel.Items, nil
}

// FetchAllRSS принимает список URL и собирает публикации из всех источников асинхронно.
func FetchAllRSS(urls []string) []Post {
	var allPosts []Post
	var wg sync.WaitGroup
	postsChan := make(chan []Post, len(urls)) // Канал для публикаций

	// Обрабатываем каждый URL в списке в горутине
	for _, url := range urls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			posts, err := FetchRSS(url)
			if err != nil {
				log.Printf("Ошибка при получении RSS с %s: %v", url, err)
				return
			}
			postsChan <- posts
		}(url)
	}

	// Закрываем канал после завершения всех горутин
	go func() {
		wg.Wait()
		close(postsChan)
	}()

	// Обрабатываем результаты после завершения всех горутин
	for posts := range postsChan {
		allPosts = append(allPosts, posts...) // Добавляем публикации в общий массив
	}

	return allPosts
}
