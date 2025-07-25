package main

import (
	"fmt"
	"log"
	"os"

	"github.com/bibib/okx-tui/core"
	"github.com/bibib/okx-tui/ui"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	// Get API credentials from environment variables
	apiKey := os.Getenv("OKX_API_KEY")
	secretKey := os.Getenv("OKX_API_SECRET")
	passphrase := os.Getenv("OKX_API_PASSPHRASE")

	if apiKey == "" || secretKey == "" || passphrase == "" {
		log.Printf("Warning: OKX API credentials not found. Using demo mode.")
	}

	// Create channels for communication
	positionCh := make(chan core.PositionData, 100)
	balanceCh := make(chan core.BalanceData, 100)
	errorCh := make(chan string, 10)

	// Create and start the TUI immediately
	program := ui.NewProgram(positionCh, balanceCh, errorCh)
	
	// Start API connection in a separate goroutine
	go func() {
		// Create OKX client with channels
		client := core.NewOKXClient(positionCh, balanceCh, errorCh)

		// Set API credentials if available
		if apiKey != "" && secretKey != "" && passphrase != "" {
			client.SetCredentials(apiKey, secretKey, passphrase)
		}

		// Connect to OKX WebSocket
		if err := client.Connect(); err != nil {
			errorCh <- fmt.Sprintf("Failed to connect to OKX: %v", err)
			return
		}
		defer client.Close()

		// Start listening for position updates
		client.StartListening()
	}()

	// Run the TUI (this blocks until the user quits)
	if _, err := program.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
	}
}