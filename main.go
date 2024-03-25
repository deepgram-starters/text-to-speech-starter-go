package main

import (
	"log"
	"net/http"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Serve static files from the public directory
	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("./public"))))

	// Start the server
	err = http.ListenAndServe(":3000", nil)
	if err != nil {
		log.Fatal("Error starting server: ", err)
	}
}
