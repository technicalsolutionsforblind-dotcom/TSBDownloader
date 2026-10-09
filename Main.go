package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

func homeHandler(w http.ResponseWriter, r *http.Request) {
	// CORS इनेबल करें ताकि Netlify से रिक्वेस्ट आ सके
	w.Header().Set("Access-Control-Allow-Origin", "*")
	http.ServeFile(w, r, "templates/yt-downloader.html")
}

type ConvertResponse struct {
	Success     bool   `json:"success"`
	Title       string `json:"title,omitempty"`
	DownloadURL string `json:"download_url,omitempty"`
	Error       string `json:"error,omitempty"`
}

func convertHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	videoURL := r.URL.Query().Get("url")
	format := r.URL.Query().Get("format")

	if videoURL == "" || format == "" {
		json.NewEncoder(w).Encode(ConvertResponse{Success: false, Error: "URL and Format are required"})
		return
	}

	// 1. yt-dlp से वीडियो का टाइटल और डायरेक्ट स्ट्रीमिंग/डेटा URL एक साथ निकालें (-g और --get-title)
	// इससे सर्वर पर फाइल डाउनलोड करने की जरूरत ही नहीं पड़ेगी!
	cmd := exec.Command("yt-dlp", "--get-title", "-g", videoURL)
	outputBytes, err := cmd.Output()
	if err != nil {
		json.NewEncoder(w).Encode(ConvertResponse{Success: false, Error: "Failed to fetch stream URL. Invalid or restricted link."})
		return
	}

	outputStr := strings.TrimSpace(string(outputBytes))
	lines := strings.Split(outputStr, "\n")

	if len(lines) < 2 {
		json.NewEncoder(w).Encode(ConvertResponse{Success: false, Error: "Could not extract direct media links."})
		return
	}

	mediaTitle := sanitizeFilename(strings.TrimSpace(lines[0]))
	if mediaTitle == "" {
		mediaTitle = "tsb_media"
	}

	// अगर ऑडियो फॉर्मेट मांगा गया है, तो yt-dlp का ऑडियो डायरेक्ट लिंक उठाएं, वरना वीडियो लिंक
	directURL := strings.TrimSpace(lines[len(lines)-1])

	json.NewEncoder(w).Encode(ConvertResponse{
		Success:     true,
		Title:       mediaTitle,
		DownloadURL: directURL, // सीधा यूट्यूब का हाई-स्पीड CDN लिंक यूजर को मिल जाएगा!
	})
}

func sanitizeFilename(name string) string {
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", "\n", "\r"}
	result := name
	for _, char := range invalidChars {
		result = strings.ReplaceAll(result, char, "")
	}
	if len(result) > 100 {
		return result[:100]
	}
	return result
}

func startSelfPing() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	ticker := time.NewTicker(12 * time.Minute)
	go func() {
		for range ticker.C {
			resp, err := http.Get("http://localhost:" + port + "/")
			if err == nil {
				resp.Body.Close()
				log.Println("Self-ping sent to stay awake.")
			}
		}
	}()
}

func main() {
	startSelfPing()

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/convert", convertHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("🚀 TSB High-Speed Direct Stream Server started on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}