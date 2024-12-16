package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

func main() {
	http.HandleFunc("/censor", CensorHandler)
	log.Println("Starting censorship service on :8083...")
	log.Fatal(http.ListenAndServe(":8083", nil))
}

func CensorHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Println("Invalid JSON format")
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	log.Printf("Text for censorship: %s", request.Text)
	if containsForbiddenWords(request.Text) {
		log.Println("Text contains forbidden words")
		http.Error(w, "Text contains forbidden words", http.StatusBadRequest)
		return
	}

	log.Println("Text passed censorship")
	w.WriteHeader(http.StatusOK)
}

func containsForbiddenWords(text string) bool {
	text = strings.ToLower(text)
	for _, word := range forbiddenWords {
		if strings.Contains(text, word) {
			return true
		}
	}
	return false
}
