// Go Text-to-Speech Starter - Backend Server
//
// This is a simple Go HTTP server that provides a text-to-speech API endpoint
// powered by Deepgram's Text-to-Speech service. It's designed to be easily
// modified and extended for your own projects.
//
// Key Features:
// - Contract-compliant API endpoint: POST /api/text-to-speech
// - Accepts text in body and model as query parameter
// - Returns binary audio data (audio/mpeg)
// - JWT session auth for API protection
// - CORS enabled for frontend communication
// - Pure API server (frontend served separately)

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"

	speakrest "github.com/deepgram/deepgram-go-sdk/v3/pkg/api/speak/v1/rest"
	dginterfaces "github.com/deepgram/deepgram-go-sdk/v3/pkg/client/interfaces"
	speak "github.com/deepgram/deepgram-go-sdk/v3/pkg/client/speak"
)

// ============================================================================
// CONFIGURATION - Customize these values for your needs
// ============================================================================

// DefaultModel is the default text-to-speech model to use when none is specified.
// Options: "aura-2-thalia-en", "aura-2-theia-en", "aura-2-andromeda-en", etc.
// See: https://developers.deepgram.com/docs/text-to-speech-models
const DefaultModel = "aura-2-thalia-en"

// JWTExpiry is the JWT token expiry duration (1 hour).
const JWTExpiry = time.Hour

// ============================================================================
// TYPES - Structures for request/response handling
// ============================================================================

// TTSRequest represents the JSON body for the text-to-speech endpoint.
type TTSRequest struct {
	Text string `json:"text"`
}

// ErrorDetail holds structured error information matching the contract format.
type ErrorDetail struct {
	Type    string      `json:"type"`
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details"`
}

// ErrorResponse wraps an ErrorDetail in the contract-compliant structure.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// SessionResponse holds the JWT token issued by /api/session.
type SessionResponse struct {
	Token string `json:"token"`
}

// HealthResponse holds the health check response.
type HealthResponse struct {
	Status string `json:"status"`
}

// DeepgramToml represents the parsed deepgram.toml file.
type DeepgramToml struct {
	Meta map[string]interface{} `toml:"meta"`
}

// ============================================================================
// SESSION AUTH - JWT tokens for API protection
// ============================================================================

// sessionSecret holds the secret used for signing JWTs.
// Auto-generated in development, should be set via env var in production.
var sessionSecret string

// generateRandomHex produces a random hex string of the given byte length.
func generateRandomHex(n int) string {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatalf("Failed to generate random bytes: %v", err)
	}
	return hex.EncodeToString(b)
}

// initSessionSecret loads SESSION_SECRET from env or generates a random one.
func initSessionSecret() {
	sessionSecret = os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		sessionSecret = generateRandomHex(32)
	}
}

// createJWT signs a new JWT with the configured session secret.
func createJWT() (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(JWTExpiry)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(sessionSecret))
}

// verifyJWT validates a JWT token string and returns an error if invalid.
func verifyJWT(tokenString string) error {
	_, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(sessionSecret), nil
	})
	return err
}

// ============================================================================
// API KEY LOADING - Load Deepgram API key from environment
// ============================================================================

// loadAPIKey reads the Deepgram API key from environment variables.
// Exits with a helpful error message if not found.
func loadAPIKey() string {
	apiKey := os.Getenv("DEEPGRAM_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "\nERROR: Deepgram API key not found!\n")
		fmt.Fprintln(os.Stderr, "Please set your API key using one of these methods:\n")
		fmt.Fprintln(os.Stderr, "1. Create a .env file (recommended):")
		fmt.Fprintln(os.Stderr, "   DEEPGRAM_API_KEY=your_api_key_here\n")
		fmt.Fprintln(os.Stderr, "2. Environment variable:")
		fmt.Fprintln(os.Stderr, "   export DEEPGRAM_API_KEY=your_api_key_here\n")
		fmt.Fprintln(os.Stderr, "Get your API key at: https://console.deepgram.com\n")
		os.Exit(1)
	}
	return apiKey
}

// ============================================================================
// CORS MIDDLEWARE - Enable cross-origin requests
// ============================================================================

// setCORSHeaders sets standard CORS headers on the response.
func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

// corsMiddleware wraps a handler with CORS support, including preflight handling.
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setCORSHeaders(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

// ============================================================================
// AUTH MIDDLEWARE - JWT Bearer token validation
// ============================================================================

// requireAuth wraps a handler with JWT Bearer token validation.
// Returns 401 with structured error if token is missing or invalid.
func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			writeJSON(w, http.StatusUnauthorized, ErrorResponse{
				Error: ErrorDetail{
					Type:    "AuthenticationError",
					Code:    "MISSING_TOKEN",
					Message: "Authorization header with Bearer token is required",
				},
			})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		err := verifyJWT(tokenString)
		if err != nil {
			message := "Invalid session token"
			if strings.Contains(err.Error(), "expired") {
				message = "Session expired, please refresh the page"
			}
			writeJSON(w, http.StatusUnauthorized, ErrorResponse{
				Error: ErrorDetail{
					Type:    "AuthenticationError",
					Code:    "INVALID_TOKEN",
					Message: message,
				},
			})
			return
		}

		next(w, r)
	}
}

// ============================================================================
// HELPER FUNCTIONS - Modular logic for easier understanding and testing
// ============================================================================

// writeJSON encodes a value as JSON and writes it to the response.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// formatErrorResponse builds a contract-compliant error response.
// It auto-detects the error code from the message if not explicitly provided.
func formatErrorResponse(message string, statusCode int, errorCode string) ErrorResponse {
	// Auto-detect error code if not provided
	if errorCode == "" {
		msgLower := strings.ToLower(message)
		if statusCode == 400 {
			switch {
			case strings.Contains(msgLower, "empty"):
				errorCode = "EMPTY_TEXT"
			case strings.Contains(msgLower, "model"):
				errorCode = "MODEL_NOT_FOUND"
			case strings.Contains(msgLower, "long") || strings.Contains(msgLower, "limit") || strings.Contains(msgLower, "exceed"):
				errorCode = "TEXT_TOO_LONG"
			default:
				errorCode = "INVALID_TEXT"
			}
		} else {
			errorCode = "INVALID_TEXT"
		}
	}

	errorType := "GenerationError"
	if statusCode == 400 {
		errorType = "ValidationError"
	}

	return ErrorResponse{
		Error: ErrorDetail{
			Type:    errorType,
			Code:    errorCode,
			Message: message,
			Details: map[string]string{
				"originalError": message,
			},
		},
	}
}

// ============================================================================
// DEEPGRAM API - Official Deepgram Go SDK (Text-to-Speech REST)
// ============================================================================

// generateAudio calls the Deepgram TTS REST API via the official Go SDK and
// returns the raw audio bytes. No encoding is set, so Deepgram returns MP3 by
// default — preserving the audio/mpeg contract expected by the frontend.
func generateAudio(apiKey, text, model string) ([]byte, error) {
	c := speak.NewREST(apiKey, &dginterfaces.ClientOptions{})
	dg := speakrest.New(c)

	var buf dginterfaces.RawResponse // RawResponse embeds bytes.Buffer
	_, err := dg.ToStream(context.Background(), text, &dginterfaces.SpeakOptions{Model: model}, &buf)
	if err != nil {
		return nil, fmt.Errorf("Deepgram TTS failed: %w", err)
	}

	return buf.Bytes(), nil
}

// ============================================================================
// ROUTE HANDLERS - API endpoint implementations
// ============================================================================

// handleSession issues a signed JWT for session authentication.
// GET /api/session
func handleSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	token, err := createJWT()
	if err != nil {
		log.Printf("Failed to create JWT: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create session"})
		return
	}

	writeJSON(w, http.StatusOK, SessionResponse{Token: token})
}

// handleTextToSpeech converts text to speech audio via the Deepgram API.
// POST /api/text-to-speech?model=aura-2-thalia-en
//
// Accepts JSON body: {"text": "Hello world"}
// Returns binary audio data (audio/mpeg) on success.
func handleTextToSpeech(apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
			return
		}

		// Parse the model from query parameter (default to DefaultModel)
		model := r.URL.Query().Get("model")
		if model == "" {
			model = DefaultModel
		}

		// Parse the JSON body
		var req TTSRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			errResp := formatErrorResponse("Invalid request body", 400, "INVALID_TEXT")
			writeJSON(w, http.StatusBadRequest, errResp)
			return
		}

		// Validate input - text is required
		if req.Text == "" {
			errResp := formatErrorResponse("Text parameter is required", 400, "EMPTY_TEXT")
			writeJSON(w, http.StatusBadRequest, errResp)
			return
		}

		if strings.TrimSpace(req.Text) == "" {
			errResp := formatErrorResponse("Text must be a non-empty string", 400, "EMPTY_TEXT")
			writeJSON(w, http.StatusBadRequest, errResp)
			return
		}

		// Generate audio from text via Deepgram API
		audioData, err := generateAudio(apiKey, req.Text, model)
		if err != nil {
			log.Printf("Text-to-speech error: %v", err)
			errMsg := err.Error()
			errMsgLower := strings.ToLower(errMsg)

			// Determine error type and status code based on error message
			statusCode := http.StatusInternalServerError
			errorCode := ""

			switch {
			case strings.Contains(errMsgLower, "model") || strings.Contains(errMsgLower, "not found"):
				statusCode = http.StatusBadRequest
				errorCode = "MODEL_NOT_FOUND"
			case strings.Contains(errMsgLower, "too long") || strings.Contains(errMsgLower, "length") || strings.Contains(errMsgLower, "limit") || strings.Contains(errMsgLower, "exceed"):
				statusCode = http.StatusBadRequest
				errorCode = "TEXT_TOO_LONG"
			case strings.Contains(errMsgLower, "invalid") || strings.Contains(errMsgLower, "malformed"):
				statusCode = http.StatusBadRequest
				errorCode = "INVALID_TEXT"
			}

			errResp := formatErrorResponse(errMsg, statusCode, errorCode)
			writeJSON(w, statusCode, errResp)
			return
		}

		// Return binary audio data with proper content type
		w.Header().Set("Content-Type", "audio/mpeg")
		w.WriteHeader(http.StatusOK)
		w.Write(audioData)
	}
}

// handleMetadata returns project metadata from deepgram.toml.
// GET /api/metadata
func handleMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var config DeepgramToml
	if _, err := toml.DecodeFile("deepgram.toml", &config); err != nil {
		log.Printf("Error reading deepgram.toml: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "INTERNAL_SERVER_ERROR",
			"message": "Failed to read metadata from deepgram.toml",
		})
		return
	}

	if config.Meta == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "INTERNAL_SERVER_ERROR",
			"message": "Missing [meta] section in deepgram.toml",
		})
		return
	}

	writeJSON(w, http.StatusOK, config.Meta)
}

// handleHealth returns a simple health check response.
// GET /health
func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	writeJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
}

// ============================================================================
// SERVER START
// ============================================================================

func main() {
	// Load .env file (ignore error if not present)
	_ = godotenv.Load()

	// Load API key and initialize session
	apiKey := loadAPIKey()
	initSessionSecret()

	// Read port and host from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	host := os.Getenv("HOST")
	if host == "" {
		host = "0.0.0.0"
	}

	// Register routes with middleware
	mux := http.NewServeMux()

	// Unprotected routes
	mux.HandleFunc("/api/session", corsMiddleware(handleSession))
	mux.HandleFunc("/api/metadata", corsMiddleware(handleMetadata))
	mux.HandleFunc("/health", corsMiddleware(handleHealth))

	// Protected routes (auth required)
	mux.HandleFunc("/api/text-to-speech", corsMiddleware(requireAuth(handleTextToSpeech(apiKey))))

	addr := host + ":" + port

	fmt.Println()
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("Backend API running at http://localhost:%s\n", port)
	fmt.Println("GET  /api/session")
	fmt.Println("POST /api/text-to-speech (auth required)")
	fmt.Println("GET  /api/metadata")
	fmt.Println("GET  /health")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println()

	log.Fatal(http.ListenAndServe(addr, mux))
}
