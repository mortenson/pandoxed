package main

import (
	"context"
	"crypto/subtle"
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

func BasicAuth(handler http.HandlerFunc, username, password, realm string) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		user, pass, ok := r.BasicAuth()

		if !ok || subtle.ConstantTimeCompare([]byte(user), []byte(username)) != 1 || subtle.ConstantTimeCompare([]byte(pass), []byte(password)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
			w.WriteHeader(401)
			w.Write([]byte("Unauthorised.\n"))
			return
		}

		handler(w, r)
	}
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
	username := os.Getenv("BASIC_AUTH_USERNAME")
	password := os.Getenv("BASIC_AUTH_PASWORD")
	http.HandleFunc("/md-to-pdf", BasicAuth(mdToPdf, username, password, "Pandoxed"))
	http.ListenAndServe(":1337", nil)
}
