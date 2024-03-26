package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	speak "github.com/deepgram/deepgram-go-sdk/pkg/api/speak/v1"
	interfaces "github.com/deepgram/deepgram-go-sdk/pkg/client/interfaces"
	client "github.com/deepgram/deepgram-go-sdk/pkg/client/speak"
)

const (
	audioFolder = "./public/audio/"
)

func synthesizeAudio(text string, model string) (string, error) {

	ctx := context.Background()

	options := interfaces.SpeakOptions{
		Model: model,
	}

	// Create a new Deepgram client
	// Note: The Deepgram API KEY is read from the environment variable DEEPGRAM_API_KEY
	c := client.NewWithDefaults()
	dg := speak.New(c)

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(audioFolder, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	_, err := dg.ToSave(ctx, audioFolder+"output.mp3", text, options)
	if err != nil {
		return "", err
	}

	return audioFolder + "output.mp3", nil
}

func handleSynthesizeSpeech(w http.ResponseWriter, r *http.Request) {

	var requestData struct {
		Text  string `json:"text"`
		Model string `json:"model"`
	}

	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if requestData.Text == "" {
		http.Error(w, "Text is required in the request", http.StatusBadRequest)
		return
	}

	audioFilePath, err := synthesizeAudio(requestData.Text, requestData.Model)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to synthesize speech: %v", err), http.StatusInternalServerError)
		return
	}

	// Open the audio file
	audioFile, err := os.Open(audioFilePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to open audio file: %v", err), http.StatusInternalServerError)
		return
	}
	defer audioFile.Close()

	// Set the appropriate content type
	w.Header().Set("Content-Type", "audio/mpeg")

	// Copy the audio file content to the response writer
	if _, err := io.Copy(w, audioFile); err != nil {
		http.Error(w, fmt.Sprintf("Failed to write audio file to response: %v", err), http.StatusInternalServerError)
		return
	}
}

func main() {
	client.InitWithDefault()
	http.HandleFunc("/api", handleSynthesizeSpeech)
	http.Handle("/", http.FileServer(http.Dir("./public")))
	log.Fatal(http.ListenAndServe(":3000", nil))
}
