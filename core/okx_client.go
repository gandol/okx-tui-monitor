package core

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

// PositionData represents a trading position
type PositionData struct {
	InstrumentID  string  `json:"instId"`
	PositionSide  string  `json:"posSide"`
	Size          float64 `json:"pos,string"`
	AvgPrice      float64 `json:"avgPx,string"`
	CurrentPrice  float64 `json:"markPx,string"`
	PnL           float64 `json:"pnl,string"`
	PnLRatio      float64 `json:"pnlRatio,string"`
	Leverage      float64 `json:"lever,string"`
	Timestamp     int64   `json:"ts,string"`
}

// BalanceData represents account balance information
type BalanceData struct {
	Currency      string  `json:"ccy"`
	TotalEquity   float64 `json:"totalEq,string"`
	AvailBalance  float64 `json:"availBal,string"`
	Timestamp     int64   `json:"ts,string"`
}

// OKXClient handles WebSocket connection to OKX API
type OKXClient struct {
	conn       *websocket.Conn
	positionCh chan<- PositionData
	balanceCh  chan<- BalanceData
	errorCh    chan<- string
	apiKey     string
	secretKey  string
	passphrase string
}

// NewOKXClient creates a new OKX WebSocket client
func NewOKXClient(positionCh chan<- PositionData, balanceCh chan<- BalanceData, errorCh chan<- string) *OKXClient {
	return &OKXClient{
		positionCh: positionCh,
		balanceCh:  balanceCh,
		errorCh:    errorCh,
	}
}

// SetCredentials sets the API credentials
func (c *OKXClient) SetCredentials(apiKey, secretKey, passphrase string) {
	c.apiKey = apiKey
	c.secretKey = secretKey
	c.passphrase = passphrase
}

// Connect establishes WebSocket connection to OKX
func (c *OKXClient) Connect() error {
	// Use public endpoint for demo/testing without credentials
	var wsURL string
	if c.apiKey == "" || c.secretKey == "" || c.passphrase == "" {
		// Public WebSocket for demo data
		wsURL = "wss://ws.okx.com:8443/ws/v5/public"
		c.errorCh <- "DEBUG: Connecting to OKX public WebSocket"
	} else {
		// Private WebSocket for real trading data
		wsURL = "wss://ws.okx.com:8443/ws/v5/private"
		c.errorCh <- "DEBUG: Connecting to OKX private WebSocket"
	}

	u, err := url.Parse(wsURL)
	if err != nil {
		return fmt.Errorf("invalid WebSocket URL: %v", err)
	}
	
	c.conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect to OKX WebSocket: %v", err)
	}

	c.errorCh <- "DEBUG: WebSocket connection established"

	// If we have credentials, authenticate first, then subscribe
	if c.apiKey != "" && c.secretKey != "" && c.passphrase != "" {
		if err := c.authenticate(); err != nil {
			c.conn.Close()
			return fmt.Errorf("authentication failed: %v", err)
		}
		// Note: We'll subscribe after receiving successful login response
	} else {
		// For public WebSocket, subscribe immediately
		if err := c.subscribe(); err != nil {
			c.conn.Close()
			return fmt.Errorf("subscription failed: %v", err)
		}
	}

	return nil
}

// authenticate sends authentication message for private WebSocket
func (c *OKXClient) authenticate() error {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	message := timestamp + "GET" + "/users/self/verify"
	
	h := hmac.New(sha256.New, []byte(c.secretKey))
	h.Write([]byte(message))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	authMsg := map[string]interface{}{
		"op": "login",
		"args": []map[string]string{
			{
				"apiKey":     c.apiKey,
				"passphrase": c.passphrase,
				"timestamp":  timestamp,
				"sign":       signature,
			},
		},
	}

	return c.conn.WriteJSON(authMsg)
}

// subscribe subscribes to position updates
func (c *OKXClient) subscribe() error {
	var subMsg map[string]interface{}

	if c.apiKey != "" {
		// Subscribe to private position and account channels
		subMsg = map[string]interface{}{
			"op": "subscribe",
			"args": []map[string]string{
				{
					"channel": "positions",
					"instType": "SWAP",
				},
				{
					"channel": "account",
				},
			},
		}
	} else {
		// Subscribe to public ticker data for demo
		subMsg = map[string]interface{}{
			"op": "subscribe",
			"args": []map[string]string{
				{
					"channel": "tickers",
					"instId": "BTC-USDT-SWAP",
				},
				{
					"channel": "tickers",
					"instId": "ETH-USDT-SWAP",
				},
				{
					"channel": "tickers",
					"instId": "SOL-USDT-SWAP",
				},
			},
		}
	}

	return c.conn.WriteJSON(subMsg)
}

// StartListening starts listening for position updates
func (c *OKXClient) StartListening() {
	defer c.conn.Close()

	// Start heartbeat goroutine
	go c.heartbeat()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			c.errorCh <- fmt.Sprintf("WebSocket read error: %v", err)
			return
		}

		// Handle simple pong response
		if string(message) == "pong" {
			continue
		}

		// Parse the message and extract position data
		var response map[string]interface{}
		if err := json.Unmarshal(message, &response); err != nil {
			c.errorCh <- fmt.Sprintf("Failed to parse message: %v", err)
			continue
		}

		// Handle different message types
		if event, ok := response["event"].(string); ok {
			switch event {
			case "login":
				if code, ok := response["code"].(string); ok && code == "0" {
					c.errorCh <- "DEBUG: Successfully authenticated with OKX"
					// Now subscribe to position updates after successful authentication
					if err := c.subscribe(); err != nil {
						c.errorCh <- fmt.Sprintf("Subscription failed after authentication: %v", err)
					}
				} else {
					msg := "Authentication failed"
					if errMsg, ok := response["msg"]; ok {
						msg = fmt.Sprintf("Authentication failed: %v", errMsg)
					}
					log.Printf("Authentication error: %v", msg)
					c.errorCh <- msg
				}
			case "subscribe":
				c.errorCh <- "DEBUG: Successfully subscribed to OKX channels"
			case "error":
				errMsg := fmt.Sprintf("OKX error: %v", response["msg"])
				log.Printf("OKX error received: %v", errMsg)
				c.errorCh <- errMsg
			}
			continue
		}

		// Handle position/ticker data
		if data, ok := response["data"].([]interface{}); ok {
			// Check if this is account balance data or position data
			if arg, ok := response["arg"].(map[string]interface{}); ok {
				if channel, ok := arg["channel"].(string); ok {
					switch channel {
					case "account":
						// Handle balance data
						c.errorCh <- fmt.Sprintf("DEBUG: Received %d balance items", len(data))
						for _, item := range data {
							if balData, ok := item.(map[string]interface{}); ok {
								balance := c.parseBalanceData(balData)
								c.errorCh <- fmt.Sprintf("DEBUG: Parsed balance data for %s", balance.Currency)
								c.balanceCh <- balance
							}
						}
					case "positions", "tickers":
						// Handle position/ticker data
						c.errorCh <- fmt.Sprintf("DEBUG: Received %d position/ticker items", len(data))
						for _, item := range data {
							if posData, ok := item.(map[string]interface{}); ok {
								position := c.parsePositionData(posData)
								c.errorCh <- fmt.Sprintf("DEBUG: Parsed position data for %s", position.InstrumentID)
								c.positionCh <- position
							}
						}
					}
				}
			} else {
				// Fallback for data without arg (older format)
				c.errorCh <- fmt.Sprintf("DEBUG: Received %d data items (fallback)", len(data))
				for _, item := range data {
					if posData, ok := item.(map[string]interface{}); ok {
						position := c.parsePositionData(posData)
						c.errorCh <- fmt.Sprintf("DEBUG: Parsed position data for %s", position.InstrumentID)
						c.positionCh <- position
					}
				}
			}
		}
	}
}

// heartbeat sends ping messages to keep connection alive
func (c *OKXClient) heartbeat() {
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Use simple string "ping" for OKX WebSocket
			if err := c.conn.WriteMessage(websocket.TextMessage, []byte("ping")); err != nil {
				log.Printf("Failed to send ping: %v", err)
				return
			}
		}
	}
}

// parsePositionData converts raw data to PositionData struct
func (c *OKXClient) parsePositionData(data map[string]interface{}) PositionData {
	position := PositionData{
		InstrumentID: getString(data, "instId"),
		PositionSide: getString(data, "posSide"),
		Timestamp:    time.Now().UnixNano() / int64(time.Millisecond),
	}

	// Handle both position data and ticker data
	if position.PositionSide == "" {
		position.PositionSide = "long" // Default for ticker data
	}

	// Parse numeric fields with proper error handling
	if size, ok := data["pos"].(string); ok {
		fmt.Sscanf(size, "%f", &position.Size)
	} else {
		position.Size = 1.0 // Default size for ticker data
	}

	if avgPx, ok := data["avgPx"].(string); ok {
		fmt.Sscanf(avgPx, "%f", &position.AvgPrice)
	} else if last, ok := data["last"].(string); ok {
		fmt.Sscanf(last, "%f", &position.AvgPrice)
	}

	if markPx, ok := data["markPx"].(string); ok {
		fmt.Sscanf(markPx, "%f", &position.CurrentPrice)
	} else if last, ok := data["last"].(string); ok {
		fmt.Sscanf(last, "%f", &position.CurrentPrice)
	}

	// Parse PnL fields - use 'upl' for unrealized PnL (live positions)
	if upl, ok := data["upl"].(string); ok {
		fmt.Sscanf(upl, "%f", &position.PnL)
		c.errorCh <- fmt.Sprintf("DEBUG: Using UPL (unrealized PnL): %s", upl)
	} else if pnl, ok := data["pnl"].(string); ok {
		// Fallback to 'pnl' for realized PnL or other data
		fmt.Sscanf(pnl, "%f", &position.PnL)
		c.errorCh <- fmt.Sprintf("DEBUG: Using PNL (realized PnL): %s", pnl)
	} else {
		// Calculate mock PnL for ticker data
		if position.CurrentPrice > 0 && position.AvgPrice > 0 {
			position.PnL = (position.CurrentPrice - position.AvgPrice) * position.Size
			c.errorCh <- fmt.Sprintf("DEBUG: Calculated mock PnL: %.4f", position.PnL)
		}
	}

	// Parse PnL ratio - use 'uplRatio' for unrealized PnL ratio
	if uplRatio, ok := data["uplRatio"].(string); ok {
		fmt.Sscanf(uplRatio, "%f", &position.PnLRatio)
		position.PnLRatio *= 100 // Convert from decimal to percentage
		c.errorCh <- fmt.Sprintf("DEBUG: Using UPL Ratio: %s (converted to %.2f%%)", uplRatio, position.PnLRatio)
	} else if pnlRatio, ok := data["pnlRatio"].(string); ok {
		// Fallback to 'pnlRatio' for other data
		fmt.Sscanf(pnlRatio, "%f", &position.PnLRatio)
		position.PnLRatio *= 100 // Convert from decimal to percentage
		c.errorCh <- fmt.Sprintf("DEBUG: Using PNL Ratio: %s (converted to %.2f%%)", pnlRatio, position.PnLRatio)
	} else if position.AvgPrice > 0 {
		position.PnLRatio = (position.PnL / (position.AvgPrice * position.Size)) * 100
		c.errorCh <- fmt.Sprintf("DEBUG: Calculated PnL ratio: %.2f%%", position.PnLRatio)
	}

	if lever, ok := data["lever"].(string); ok {
		fmt.Sscanf(lever, "%f", &position.Leverage)
	} else {
		position.Leverage = 1.0 // Default leverage
	}

	return position
}

// parseBalanceData converts raw balance data to BalanceData struct
func (c *OKXClient) parseBalanceData(data map[string]interface{}) BalanceData {
	balance := BalanceData{
		Currency:  getString(data, "ccy"),
		Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
	}

	// Parse numeric fields with proper error handling
	if totalEq, ok := data["totalEq"].(string); ok {
		fmt.Sscanf(totalEq, "%f", &balance.TotalEquity)
	}

	if availBal, ok := data["availBal"].(string); ok {
		fmt.Sscanf(availBal, "%f", &balance.AvailBalance)
	}

	return balance
}

// getString safely extracts string value from map
func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key].(string); ok {
		return val
	}
	return ""
}

// Close closes the WebSocket connection
func (c *OKXClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}