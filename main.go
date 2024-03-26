package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	speak "github.com/deepgram/deepgram-go-sdk/pkg/api/speak/v1"
	interfaces "github.com/deepgram/deepgram-go-sdk/pkg/client/interfaces"
	client "github.com/deepgram/deepgram-go-sdk/pkg/client/speak"
)

func synthesizeAudioStream(text string, model string) (interfaces.RawResponse, error) {
	ctx := context.Background()

	options := interfaces.SpeakOptions{
		Model: model,
	}

	// Create a new Deepgram client
	c := client.NewWithDefaults()
	dg := speak.New(c)

	// Initialize a buffer to store the audio data
	var buffer interfaces.RawResponse

	// Stream the audio directly to the provided buffer
	_, err := dg.ToStream(ctx, text, options, &buffer)
	if err != nil {
		return buffer, err
	}

	return buffer, nil
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

	// Set the appropriate content type
	w.Header().Set("Content-Type", "audio/mpeg")

	// Synthesize the audio and get the buffer
	buffer, err := synthesizeAudioStream(requestData.Text, requestData.Model)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to synthesize speech: %v", err), http.StatusInternalServerError)
		return
	}

	// Write the contents of the buffer to the response writer
	if _, err := w.Write(buffer.Bytes()); err != nil {
		http.Error(w, fmt.Sprintf("Failed to write audio data to response: %v", err), http.StatusInternalServerError)
		return
	}
}

func main() {
	client.InitWithDefault()
	http.HandleFunc("/api", handleSynthesizeSpeech)
	http.Handle("/", http.FileServer(http.Dir("./public")))
	log.Fatal(http.ListenAndServe(":3000", nil))
}
