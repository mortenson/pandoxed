package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"
)

// 1MB
const MaxFileSize = 1000000

// 10 seconds
const MaxPandocTime = 10

func httpError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	w.Write([]byte(message))
}

func mdToPdf(w http.ResponseWriter, req *http.Request) {
	startTime := time.Now()
	defer func() {
		log.Printf("[%s] %s %s %s %s", startTime.Format(time.RFC3339), req.RemoteAddr, req.Method, req.URL.Path, time.Since(startTime))
	}()

	// Parse request.
	if req.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, "wrong request method")
		return
	}
	bodyReader := http.MaxBytesReader(w, req.Body, MaxFileSize)
	bodyBytes, err := io.ReadAll(bodyReader)
	if err != nil {
		httpError(w, http.StatusBadRequest, "can't read the body. maybe too big")
		return
	}

	// Gen some tmp files.
	inFile, err := os.CreateTemp("", "in*.md")
	if err != nil {
		httpError(w, http.StatusInternalServerError, "tmp in problems")
		return
	}
	defer os.Remove(inFile.Name())
	outFile, err := os.CreateTemp("", "out*.pdf")
	if err != nil {
		httpError(w, http.StatusInternalServerError, "tmp out problems")
		return
	}
	defer os.Remove(outFile.Name())

	// Write in file.
	_, err = inFile.Write(bodyBytes)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "can't write bytes to tmp file")
		return
	}

	// Exec pandoc
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(MaxPandocTime)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "pandoc", "--verbose", "--pdf-engine=lualatex", "-f", "markdown+raw_tex", inFile.Name(), "-o", outFile.Name())
	if err := cmd.Run(); err != nil {
		httpError(w, http.StatusInternalServerError, fmt.Sprintf("pandoc err: %s", err))
		return
	}

	// Read PDF.
	outBytes, err := os.ReadFile(outFile.Name())
	if err != nil {
		httpError(w, http.StatusInternalServerError, "can't read PDF bytes")
		return
	}

	// Return PDF.
	w.Header().Add("Content-Type", "application/pdf")
	w.Write(outBytes)
}

func main() {
	http.HandleFunc("/md-to-pdf", mdToPdf)
	http.ListenAndServe(":1337", nil)
}
