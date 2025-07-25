package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/gandol/okx-tui-monitor/core"
	"github.com/gandol/okx-tui-monitor/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
)

func validateAPICredentials(apiKey, secretKey, passphrase string) bool {
	// Check for placeholder values
	if strings.Contains(apiKey, "your-actual-api-key") ||
		strings.Contains(secretKey, "your-actual-api-secret") ||
		strings.Contains(passphrase, "your-actual-passphrase") {
		return false
	}

	// Basic format validation
	// API Key should be a UUID-like format (32 chars hex)
	if matched, _ := regexp.MatchString(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$`, apiKey); !matched {
		return false
	}

	// Secret should be base64-like (at least 32 chars)
	if len(secretKey) < 32 {
		return false
	}

	// Passphrase should not be empty
	if len(passphrase) == 0 {
		return false
	}

	return true
}

func main() {
	// Parse command line flags
	var debugMode bool
	flag.BoolVar(&debugMode, "debug", false, "Enable debug mode")
	flag.BoolVar(&debugMode, "d", false, "Enable debug mode (shorthand)")
	flag.Parse()

	// Create channels for communication first
	positionCh := make(chan core.PositionData, 100)
	balanceCh := make(chan core.BalanceData, 100)
	errorCh := make(chan string, 10)

	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		errorCh <- fmt.Sprintf("DEBUG: Warning: Could not load .env file: %v", err)
	}

	// Get API credentials from environment variables
	apiKey := os.Getenv("OKX_API_KEY")
	secretKey := os.Getenv("OKX_API_SECRET")
	passphrase := os.Getenv("OKX_API_PASSPHRASE")

	// Validate credentials
	validCredentials := validateAPICredentials(apiKey, secretKey, passphrase)
	
	if !validCredentials {
		errorCh <- "DEBUG: Running in demo mode - Invalid or missing API credentials"
		errorCh <- "DEBUG: To use live data, please set valid OKX API credentials in .env file"
		// Clear invalid credentials to ensure demo mode
		apiKey, secretKey, passphrase = "", "", ""
	} else {
		errorCh <- "DEBUG: Running in authenticated mode with valid API credentials"
	}

	// Create and start the TUI immediately with debug mode setting
	var program *tea.Program
	if debugMode {
		program = ui.NewProgramWithDebug(positionCh, balanceCh, errorCh)
	} else {
		program = ui.NewProgram(positionCh, balanceCh, errorCh)
	}
	
	// Start API connection in a separate goroutine
	go func() {
		// Create OKX client with channels
		client := core.NewOKXClient(positionCh, balanceCh, errorCh)

		// Set API credentials if available and valid
		if validCredentials {
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
		// Send error to error channel and exit gracefully
		errorCh <- fmt.Sprintf("FATAL: Error running program: %v", err)
		os.Exit(1)
	}
}