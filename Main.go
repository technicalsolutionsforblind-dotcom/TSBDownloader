package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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

	// सिस्टम की टेम्परेरी डायरेक्टरी इस्तेमाल करें (कोई लोकल फोल्डर नहीं बनेगा)
	tempDir := os.TempDir()

	// असली टाइटल प्राप्त करें
	titleCmd := exec.Command("yt-dlp", "--get-title", videoURL)
	titleBytes, err := titleCmd.Output()
	mediaTitle := "tsb_media"
	if err == nil {
		cleanTitle := strings.TrimSpace(string(titleBytes))
		mediaTitle = sanitizeFilename(cleanTitle)
	}
	if mediaTitle == "" {
		mediaTitle = "media_file"
	}

	isAudio := isAudioFormat(format)
	var cmd *exec.Cmd

	if isAudio {
		cmd = exec.Command("yt-dlp", "-x", "--audio-format", format, "--audio-quality", "5", "--no-playlist", "-o", filepath.Join(tempDir, mediaTitle+".%(ext)s"), videoURL)
	} else {
		cmd = exec.Command("yt-dlp", "-S", "res,ext:mp4:m4a", "--no-playlist", "-o", filepath.Join(tempDir, mediaTitle+".%(ext)s"), videoURL)
	}

	err = cmd.Run()
	if err != nil {
		fallbackCmd := exec.Command("yt-dlp", "--no-playlist", "-o", filepath.Join(tempDir, mediaTitle+".%(ext)s"), videoURL)
	if errFB := fallbackCmd.Run(); errFB != nil {
			json.NewEncoder(w).Encode(ConvertResponse{Success: false, Error: "Conversion failed: " + err.Error()})
			return
		}
	}

	finalFile := findMatchingFile(tempDir, mediaTitle)
	if finalFile == "" {
		json.NewEncoder(w).Encode(ConvertResponse{Success: false, Error: "Generated file not found."})
		return
	}

	finalFileName := filepath.Base(finalFile)

	json.NewEncoder(w).Encode(ConvertResponse{
		Success:     true,
		Title:       mediaTitle,
		DownloadURL: "/download-file?file=" + finalFileName,
	})
}

func serveFileHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	fileName := r.URL.Query().Get("file")
	
	// सुरक्षा जांच: पाथ ट्रावेर्सा से बचने के लिए सिर्फ बेसनेम लें
	fileName = filepath.Base(fileName)
	filePath := filepath.Join(os.TempDir(), fileName)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "File not found or expired", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	http.ServeFile(w, r, filePath)

	// डाउनलोड होते ही टेम्परेरी फाइल को तुरंत डिलीट कर दें
	go func() {
		os.Remove(filePath)
	}()
}

func isAudioFormat(format string) bool {
	audioFormats := map[string]bool{"mp3": true, "m4a": true, "wav": true, "flac": true, "aac": true, "opus": true}
	return audioFormats[format]
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

func findMatchingFile(dir, prefix string) string {
	files, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, f := range files {
		if strings.HasPrefix(f.Name(), prefix) && !f.IsDir() {
			return filepath.Join(dir, f.Name())
		}
	}
	return ""
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
	http.HandleFunc("/download-file", serveFileHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("🚀 TSB Backend Server started on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}