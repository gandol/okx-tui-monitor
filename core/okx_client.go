package core

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"sync"
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
	PnL           float64 `json:"upl,string"`        // Use 'upl' for unrealized PnL
	PnLRatio      float64 `json:"uplRatio,string"`   // Use 'uplRatio' for unrealized PnL ratio
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

// TickerData represents ticker information
type TickerData struct {
	InstrumentID string  `json:"instId"`
	LastPrice    float64 `json:"last,string"`
	BidPrice     float64 `json:"bidPx,string"`
	AskPrice     float64 `json:"askPx,string"`
	Volume24h    float64 `json:"vol24h,string"`
	Timestamp    int64   `json:"ts,string"`
}

// OKXClient handles WebSocket connection to OKX API
type OKXClient struct {
	conn         *websocket.Conn
	tickerConn   *websocket.Conn  // Separate connection for ticker data
	positionCh   chan<- PositionData
	balanceCh    chan<- BalanceData
	errorCh      chan<- string
	apiKey       string
	secretKey    string
	passphrase   string
	currentPositions map[string]bool // Track current positions for ticker subscription
	isDemo       bool               // Track if running in demo mode
	demoPositions map[string]PositionData // Store demo positions
	connMutex    sync.Mutex         // Protect main WebSocket writes
	tickerMutex  sync.Mutex         // Protect ticker WebSocket writes
}

// NewOKXClient creates a new OKX WebSocket client
func NewOKXClient(positionCh chan<- PositionData, balanceCh chan<- BalanceData, errorCh chan<- string) *OKXClient {
	return &OKXClient{
		positionCh:       positionCh,
		balanceCh:        balanceCh,
		errorCh:          errorCh,
		currentPositions: make(map[string]bool),
		demoPositions:    make(map[string]PositionData),
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
	// Always establish public WebSocket connection for ticker data
	if err := c.connectTickerWebSocket(); err != nil {
		c.errorCh <- fmt.Sprintf("Failed to connect to ticker WebSocket: %v", err)
		// Continue without ticker connection - not critical
	}

	// Use public endpoint for demo/testing without credentials
	var wsURL string
	if c.apiKey == "" || c.secretKey == "" || c.passphrase == "" {
		// Set demo mode flag
		c.isDemo = true
		
		// Public WebSocket for demo data
		wsURL = "wss://ws.okx.com:8443/ws/v5/public"
		c.errorCh <- "Connecting to OKX public WebSocket"
		
		// Create demo positions for display
		c.createDemoPositions()
	} else {
		// Private WebSocket for real trading data
		wsURL = "wss://ws.okx.com:8443/ws/v5/private"
		c.errorCh <- "Connecting to OKX private WebSocket"
	}

	u, err := url.Parse(wsURL)
	if err != nil {
		return fmt.Errorf("invalid WebSocket URL: %v", err)
	}
	
	c.conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect to OKX WebSocket: %v", err)
	}

	c.errorCh <- "WebSocket connection established"

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

// connectTickerWebSocket establishes a separate WebSocket connection for ticker data
func (c *OKXClient) connectTickerWebSocket() error {
	// Use the public WebSocket endpoint for ticker data as specified in requirements
	wsURL := "wss://wspri.okx.com:8443/ws/v5/ipublic"
	
	u, err := url.Parse(wsURL)
	if err != nil {
		return fmt.Errorf("invalid ticker WebSocket URL: %v", err)
	}
	
	c.tickerConn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect to ticker WebSocket: %v", err)
	}

	c.errorCh <- "DEBUG: Ticker WebSocket connection established"
	
	// Start ticker listener in a separate goroutine
	go c.startTickerListener()
	
	return nil
}

// startTickerListener listens for ticker data on the separate connection
func (c *OKXClient) startTickerListener() {
	defer func() {
		if c.tickerConn != nil {
			c.tickerConn.Close()
		}
	}()

	// Start heartbeat for ticker connection
	go c.tickerHeartbeat()

	for {
		if c.tickerConn == nil {
			break
		}
		
		_, message, err := c.tickerConn.ReadMessage()
		if err != nil {
			c.errorCh <- fmt.Sprintf("DEBUG: Ticker WebSocket read error: %v", err)
			return
		}

		// Handle simple pong response
		if string(message) == "pong" {
			continue
		}

		// Parse ticker message
		var response map[string]interface{}
		if err := json.Unmarshal(message, &response); err != nil {
			c.errorCh <- fmt.Sprintf("DEBUG: Failed to parse ticker message: %v", err)
			continue
		}

		// Handle ticker data
		if data, ok := response["data"].([]interface{}); ok {
			if arg, ok := response["arg"].(map[string]interface{}); ok {
				if channel, ok := arg["channel"].(string); ok && channel == "tickers" {
					c.errorCh <- fmt.Sprintf("DEBUG: Received %d ticker items", len(data))
					for _, item := range data {
						if tickerData, ok := item.(map[string]interface{}); ok {
							c.handleTickerData(tickerData)
						}
					}
				}
			}
		}
	}
}

// tickerHeartbeat sends ping messages to keep ticker connection alive
func (c *OKXClient) tickerHeartbeat() {
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if c.tickerConn != nil {
				// Protect ticker WebSocket writes with mutex
				c.tickerMutex.Lock()
				err := c.tickerConn.WriteMessage(websocket.TextMessage, []byte("ping"))
				c.tickerMutex.Unlock()
				
				if err != nil {
					c.errorCh <- fmt.Sprintf("DEBUG: Failed to send ticker ping: %v", err)
					return
				}
			}
		}
	}
}

// handleTickerData processes ticker data and updates position current prices
func (c *OKXClient) handleTickerData(data map[string]interface{}) {
	instId := getString(data, "instId")
	if instId == "" {
		return
	}

	// Parse last price
	var lastPrice float64
	if last, ok := data["last"].(string); ok {
		fmt.Sscanf(last, "%f", &lastPrice)
	}

	c.errorCh <- fmt.Sprintf("DEBUG: Ticker update for %s: %.6f", instId, lastPrice)

	// In demo mode, update demo positions with ticker data
	if c.isDemo {
		if demoPos, exists := c.demoPositions[instId]; exists {
			// Update the current price and recalculate PnL
			demoPos.CurrentPrice = lastPrice
			
			// Calculate PnL based on position side
			if demoPos.PositionSide == "short" {
				// For short positions, profit when price goes down
				demoPos.PnL = (demoPos.AvgPrice - demoPos.CurrentPrice) * demoPos.Size
			} else {
				// For long positions, profit when price goes up
				demoPos.PnL = (demoPos.CurrentPrice - demoPos.AvgPrice) * demoPos.Size
			}
			
			if demoPos.AvgPrice > 0 {
				demoPos.PnLRatio = (demoPos.PnL / (demoPos.AvgPrice * demoPos.Size)) * 100
			}
			demoPos.Timestamp = time.Now().UnixNano() / int64(time.Millisecond)
			
			// Update stored demo position
			c.demoPositions[instId] = demoPos
			
			// Send updated demo position to UI
			c.positionCh <- demoPos
			return
		}
	}

	// For real positions, only send price updates (not full position data)
	// The UI will merge this with existing position data
	position := PositionData{
		InstrumentID: instId,
		CurrentPrice: lastPrice,
		Timestamp:    time.Now().UnixNano() / int64(time.Millisecond),
	}

	// Send to position channel to update current price
	c.positionCh <- position
}

// createDemoPositions creates demo trading positions for display in demo mode
func (c *OKXClient) createDemoPositions() {
	// Create demo positions for 10 different trading pairs
	demoInstruments := []struct {
		instId   string
		avgPrice float64
		size     float64
		side     string
	}{
		{"BTC-USDT-SWAP", 45000.0, 0.1, "long"},
		{"ETH-USDT-SWAP", 2800.0, 1.0, "long"},
		{"SOL-USDT-SWAP", 178.0, 2.7, "short"},
		{"ADA-USDT-SWAP", 0.45, 1000.0, "long"},
		{"DOT-USDT-SWAP", 6.8, 50.0, "short"},
		{"LINK-USDT-SWAP", 14.2, 25.0, "long"},
		{"AVAX-USDT-SWAP", 28.5, 15.0, "short"},
		{"MATIC-USDT-SWAP", 0.85, 500.0, "long"},
		{"UNI-USDT-SWAP", 7.3, 40.0, "short"},
		{"LTC-USDT-SWAP", 95.0, 3.0, "long"},
	}

	for _, demo := range demoInstruments {
		position := PositionData{
			InstrumentID: demo.instId,
			PositionSide: demo.side,
			Size:         demo.size,
			AvgPrice:     demo.avgPrice,
			CurrentPrice: demo.avgPrice, // Will be updated by ticker data
			PnL:          0.0,           // Will be calculated when ticker updates
			PnLRatio:     0.0,           // Will be calculated when ticker updates
			Leverage:     10.0,          // Demo leverage
			Timestamp:    time.Now().UnixNano() / int64(time.Millisecond),
		}
		
		// Store demo position
		c.demoPositions[demo.instId] = position
		
		// Send initial demo position to UI
		c.positionCh <- position
		
		c.errorCh <- fmt.Sprintf("DEBUG: Created demo position for %s", demo.instId)
	}
	
	// Also create a demo balance
	demoBalance := BalanceData{
		Currency:     "USDT",
		TotalEquity:  10000.0, // Demo balance of $10,000
		AvailBalance: 5000.0,  // $5,000 available
		Timestamp:    time.Now().UnixNano() / int64(time.Millisecond),
	}
	
	c.balanceCh <- demoBalance
	c.errorCh <- "DEBUG: Created demo balance"
}

// updateTickerSubscriptions subscribes to tickers for current positions
func (c *OKXClient) updateTickerSubscriptions() error {
	if c.tickerConn == nil {
		return fmt.Errorf("ticker connection not established")
	}

	// Protect ticker WebSocket writes with mutex
	c.tickerMutex.Lock()
	defer c.tickerMutex.Unlock()

	// Build subscription args for current positions
	var args []map[string]string
	for instId := range c.currentPositions {
		args = append(args, map[string]string{
			"channel": "tickers",
			"instId":  instId,
		})
	}

	// If no positions, subscribe to all 10 demo tickers
	if len(args) == 0 {
		args = []map[string]string{
			{"channel": "tickers", "instId": "BTC-USDT-SWAP"},
			{"channel": "tickers", "instId": "ETH-USDT-SWAP"},
			{"channel": "tickers", "instId": "SOL-USDT-SWAP"},
			{"channel": "tickers", "instId": "ADA-USDT-SWAP"},
			{"channel": "tickers", "instId": "DOT-USDT-SWAP"},
			{"channel": "tickers", "instId": "LINK-USDT-SWAP"},
			{"channel": "tickers", "instId": "AVAX-USDT-SWAP"},
			{"channel": "tickers", "instId": "MATIC-USDT-SWAP"},
			{"channel": "tickers", "instId": "UNI-USDT-SWAP"},
			{"channel": "tickers", "instId": "LTC-USDT-SWAP"},
		}
	}

	subMsg := map[string]interface{}{
		"op":   "subscribe",
		"args": args,
	}

	c.errorCh <- fmt.Sprintf("DEBUG: Subscribing to %d ticker channels", len(args))
	return c.tickerConn.WriteJSON(subMsg)
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

	// Protect main WebSocket writes with mutex
	c.connMutex.Lock()
	defer c.connMutex.Unlock()
	
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
		// Subscribe to public ticker data for all 10 demo pairs
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
				{
					"channel": "tickers",
					"instId": "ADA-USDT-SWAP",
				},
				{
					"channel": "tickers",
					"instId": "DOT-USDT-SWAP",
				},
				{
					"channel": "tickers",
					"instId": "LINK-USDT-SWAP",
				},
				{
					"channel": "tickers",
					"instId": "AVAX-USDT-SWAP",
				},
				{
					"channel": "tickers",
					"instId": "MATIC-USDT-SWAP",
				},
				{
					"channel": "tickers",
					"instId": "UNI-USDT-SWAP",
				},
				{
					"channel": "tickers",
					"instId": "LTC-USDT-SWAP",
				},
			},
		}
	}

	// Protect main WebSocket writes with mutex
	c.connMutex.Lock()
	err := c.conn.WriteJSON(subMsg)
	c.connMutex.Unlock()
	
	if err != nil {
		return err
	}

	// Also trigger initial ticker subscriptions if ticker connection is available
	if c.tickerConn != nil {
		go func() {
			if err := c.updateTickerSubscriptions(); err != nil {
				c.errorCh <- fmt.Sprintf("Failed to initialize ticker subscriptions: %v", err)
			}
		}()
	}

	return nil
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
					c.errorCh <- msg
				}
			case "subscribe":
				c.errorCh <- "DEBUG: Successfully subscribed to OKX channels"
			case "error":
				errMsg := fmt.Sprintf("OKX error: %v", response["msg"])
				c.errorCh <- fmt.Sprintf("OKX error received: %v", errMsg)
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
			if c.conn != nil {
				// Protect main WebSocket writes with mutex
				c.connMutex.Lock()
				err := c.conn.WriteMessage(websocket.TextMessage, []byte("ping"))
				c.connMutex.Unlock()
				
				if err != nil {
					c.errorCh <- fmt.Sprintf("Failed to send ping: %v", err)
					return
				}
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

	// Parse PnL fields - prioritize actual OKX data over calculations
	pnlFound := false
	if upl, ok := data["upl"].(string); ok && upl != "" && upl != "0" {
		fmt.Sscanf(upl, "%f", &position.PnL)
		pnlFound = true
		c.errorCh <- fmt.Sprintf("DEBUG: Using UPL (unrealized PnL): %s = %.4f", upl, position.PnL)
	} else if pnl, ok := data["pnl"].(string); ok && pnl != "" && pnl != "0" {
		// Fallback to 'pnl' for realized PnL or other data
		fmt.Sscanf(pnl, "%f", &position.PnL)
		pnlFound = true
		c.errorCh <- fmt.Sprintf("DEBUG: Using PNL (realized PnL): %s = %.4f", pnl, position.PnL)
	}
	
	// Only calculate mock PnL if no real PnL data is available and we have valid prices
	if !pnlFound && position.CurrentPrice > 0 && position.AvgPrice > 0 && position.Size > 0 {
		if position.PositionSide == "short" {
			// For short positions, profit when price goes down
			position.PnL = (position.AvgPrice - position.CurrentPrice) * position.Size
		} else {
			// For long positions, profit when price goes up
			position.PnL = (position.CurrentPrice - position.AvgPrice) * position.Size
		}
		c.errorCh <- fmt.Sprintf("DEBUG: Calculated PnL for %s: %.4f (avg: %.4f, current: %.4f, size: %.4f)", 
			position.PositionSide, position.PnL, position.AvgPrice, position.CurrentPrice, position.Size)
	}

	// Parse PnL ratio - prioritize actual OKX data
	ratioFound := false
	if uplRatio, ok := data["uplRatio"].(string); ok && uplRatio != "" && uplRatio != "0" {
		fmt.Sscanf(uplRatio, "%f", &position.PnLRatio)
		position.PnLRatio *= 100 // Convert from decimal to percentage
		ratioFound = true
		c.errorCh <- fmt.Sprintf("DEBUG: Using UPL Ratio: %s = %.2f%%", uplRatio, position.PnLRatio)
	} else if pnlRatio, ok := data["pnlRatio"].(string); ok && pnlRatio != "" && pnlRatio != "0" {
		// Fallback to 'pnlRatio' for other data
		fmt.Sscanf(pnlRatio, "%f", &position.PnLRatio)
		position.PnLRatio *= 100 // Convert from decimal to percentage
		ratioFound = true
		c.errorCh <- fmt.Sprintf("DEBUG: Using PNL Ratio: %s = %.2f%%", pnlRatio, position.PnLRatio)
	}
	
	// Only calculate ratio if no real ratio data is available and we have valid data
	if !ratioFound && position.AvgPrice > 0 && position.Size > 0 {
		position.PnLRatio = (position.PnL / (position.AvgPrice * position.Size)) * 100
		c.errorCh <- fmt.Sprintf("DEBUG: Calculated PnL ratio: %.2f%%", position.PnLRatio)
	}

	if lever, ok := data["lever"].(string); ok {
		fmt.Sscanf(lever, "%f", &position.Leverage)
	} else {
		position.Leverage = 1.0 // Default leverage
	}

	// Track current positions for ticker subscriptions
	if position.InstrumentID != "" && position.Size > 0 {
		// Update current positions map
		c.currentPositions[position.InstrumentID] = true
		
		// Update ticker subscriptions if we have a ticker connection
		// Note: Removed goroutine to prevent concurrent WebSocket writes
		if c.tickerConn != nil {
			if err := c.updateTickerSubscriptions(); err != nil {
				c.errorCh <- fmt.Sprintf("Failed to update ticker subscriptions: %v", err)
			}
		}
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

// Close closes the WebSocket connections
func (c *OKXClient) Close() error {
	var err error
	
	// Close main connection
	if c.conn != nil {
		if closeErr := c.conn.Close(); closeErr != nil {
			err = closeErr
		}
	}
	
	// Close ticker connection
	if c.tickerConn != nil {
		if closeErr := c.tickerConn.Close(); closeErr != nil {
			if err == nil {
				err = closeErr
			}
		}
	}
	
	return err
}