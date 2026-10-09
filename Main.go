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

	// बुलेटप्रूफ yt-dlp कमांड: --no-playlist, असली ब्राउज़र का User-Agent, और Geo-bypass फ्लैग ताकि ब्लॉक न हो
	cmd := exec.Command(
		"yt-dlp",
		"--no-playlist",
		"--user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"--geo-bypass",
		"--get-title",
		"-g",
		videoURL,
	)

	outputBytes, err := cmd.Output()
	if err != nil {
		// अगर पहला तरीका फेल हो, तो एक बार yt-dlp को खुद को अपडेट करने की कोशिश करने का फॉલबैक
		log.Printf("Primary extraction failed: %v. Attempting update fallback...", err)
		
		updateCmd := exec.Command("yt-dlp", "-U")
		_ = updateCmd.Run()

		// दोबारा कमांड चलाएं
		cmdRetry := exec.Command(
			"yt-dlp",
			"--no-playlist",
			"--user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			"--geo-bypass",
			"--get-title",
			"-g",
			videoURL,
		)
		outputBytes, err = cmdRetry.Output()
		if err != nil {
			json.NewEncoder(w).Encode(ConvertResponse{Success: false, Error: "Failed to fetch stream URL. YouTube might be blocking the server IP or the link is invalid."})
			return
		}
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

	directURL := strings.TrimSpace(lines[len(lines)-1])

	json.NewEncoder(w).Encode(ConvertResponse{
		Success:     true,
		Title:       mediaTitle,
		DownloadURL: directURL,
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

	fmt.Printf("🚀 TSB Bulletproof Direct Stream Server started on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}