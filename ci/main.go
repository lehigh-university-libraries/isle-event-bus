package main

import (
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	http.HandleFunc("/", uploadFile)
	http.Handle("/files/", http.StripPrefix("/files/", customFileHandler(http.Dir("/tmp"))))

	slog.Info("Starting test file GET/POST server")

	http.ListenAndServe(":80", nil)
}

func customFileHandler(root http.FileSystem) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			ext := filepath.Ext(r.URL.Path)
			w.Header().Set("Content-Type", typeByExtension(ext))
			w.WriteHeader(http.StatusOK)
			return
		}
		f, err := root.Open(r.URL.Path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", typeByExtension(ext))
		w.WriteHeader(http.StatusOK)
		io.Copy(w, f)
	})
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	filePath := r.Header.Get("Content-Location")
	if filePath == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	slog.Info("Uploading", "filePath", filePath)

	f, err := os.Create(filePath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer f.Close()
	_, err = io.Copy(f, r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func typeByExtension(ext string) string {
	extToMime := map[string]string{
		".md":   "text/markdown",
		".m3u8": "application/vnd.apple.mpegurl",
		".xls":  "application/vnd.ms-excel",
		".mp4":  "video/mp4",
	}
	if mime, ok := extToMime[ext]; ok {
		return mime
	}

	return mime.TypeByExtension(ext)
}
