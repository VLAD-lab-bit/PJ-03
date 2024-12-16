package config

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
)

// Config - структура для хранения конфигурации приложения.
type Config struct {
	DatabaseURL   string   `json:"database_url"`   // URL для подключения к базе данных
	RSS           []string `json:"rss"`            // Ссылки на RSS-ленты
	RequestPeriod int      `json:"request_period"` // Интервал опроса (в минутах)
	ServerPort    int      `json:"server_port"`    // Порт для запуска сервера
}

// LoadConfig загружает конфигурационный файл.
func LoadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(byteValue, &config); err != nil {
		return nil, err
	}

	log.Println("Config loaded successfully:", config)
	return &config, nil
}
