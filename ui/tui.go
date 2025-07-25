package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gandol/okx-tui-monitor/core"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles
var (
	baseStyle = lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	titleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true).
		Padding(0, 1)

	currentTimeStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true)

	timeStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	cardStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("238")).
		Padding(1).
		Margin(0, 1, 1, 0).
		Width(24).
		Height(12)

	cardHeaderStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true).
		Align(lipgloss.Center)

	labelStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	valueStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("255"))

	positiveStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("46"))

	negativeStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))

	neutralStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("255"))

	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)

	debugStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Margin(1, 0, 0, 0)

	debugHeaderStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)
)

// Model represents the application state
type Model struct {
	positions       map[string]core.PositionData
	balances        map[string]core.BalanceData
	previousBalance float64 // Store previous total balance for color comparison
	positionCh      <-chan core.PositionData
	balanceCh       <-chan core.BalanceData
	errorCh         <-chan string
	lastUpdate      time.Time
	width           int
	height          int
	scrollOffset    int
	errorMsg        string
	hasError        bool
	debugMessages   []string
	maxDebugLines   int
	showDebug       bool // Toggle for debug output visibility
}

// NewProgram creates a new Bubble Tea program
func NewProgram(positionCh <-chan core.PositionData, balanceCh <-chan core.BalanceData, errorCh <-chan string) *tea.Program {
	model := NewModel(positionCh, balanceCh, errorCh)
	return tea.NewProgram(model, tea.WithAltScreen())
}

// NewModel creates a new model
func NewModel(positionCh <-chan core.PositionData, balanceCh <-chan core.BalanceData, errorCh <-chan string) Model {
	return Model{
		positions:     make(map[string]core.PositionData),
		balances:      make(map[string]core.BalanceData),
		positionCh:    positionCh,
		balanceCh:     balanceCh,
		errorCh:       errorCh,
		width:         80,
		height:        24,
		debugMessages: make([]string, 0),
		maxDebugLines: 10, // Keep last 10 debug messages
		showDebug:     false, // Debug output hidden by default
	}
}

// renderPositionCards renders all position cards in a dynamic grid based on terminal width
func (m Model) renderPositionCards() string {
	if len(m.positions) == 0 {
		return ""
	}

	// Convert map to slice for sorting
	var positions []core.PositionData
	for _, pos := range m.positions {
		positions = append(positions, pos)
	}

	// Sort positions by InstrumentID (coin name) in ascending order
	sort.Slice(positions, func(i, j int) bool {
		return positions[i].InstrumentID < positions[j].InstrumentID
	})

	// Create cards from sorted positions
	var cards []string
	for _, pos := range positions {
		cards = append(cards, m.renderPositionCard(pos))
	}

	// Calculate dynamic cards per row based on terminal width
	cardWidth := 26 // Each card is 24 chars wide + 2 chars margin
	availableWidth := m.width - 8 // Account for base style padding and borders
	if availableWidth < 40 {
		availableWidth = 40 // Minimum width
	}
	
	cardsPerRow := availableWidth / cardWidth
	if cardsPerRow < 1 {
		cardsPerRow = 1 // At least 1 card per row
	}
	if cardsPerRow > 8 {
		cardsPerRow = 8 // Maximum 8 cards per row for readability
	}
	
	// Create rows with calculated cards per row
	var rows []string
	for i := 0; i < len(cards); i += cardsPerRow {
		end := i + cardsPerRow
		if end > len(cards) {
			end = len(cards)
		}
		
		rowCards := cards[i:end]
		row := lipgloss.JoinHorizontal(lipgloss.Top, rowCards...)
		rows = append(rows, row)
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// renderBalance renders the total account balance with color coding based on change
func (m Model) renderBalance() string {
	if len(m.balances) == 0 {
		return ""
	}

	// Calculate total equity across all currencies
	var totalEquity float64
	var mainCurrency string
	
	// Look for USDT first as the main currency, then USD, then any other
	for currency, balance := range m.balances {
		totalEquity += balance.TotalEquity
		if currency == "USDT" || (mainCurrency == "" && currency == "USD") || mainCurrency == "" {
			mainCurrency = currency
		}
	}

	if totalEquity == 0 {
		return ""
	}

	// Format balance with appropriate styling
	balanceText := fmt.Sprintf("%.2f %s", totalEquity, mainCurrency)
	
	// Style based on balance change compared to previous balance
	var styledBalance string
	if m.previousBalance > 0 { // Only apply color coding if we have a previous balance to compare
		if totalEquity > m.previousBalance {
			// Current balance is greater than previous - green
			styledBalance = positiveStyle.Render(balanceText)
		} else if totalEquity < m.previousBalance {
			// Current balance is less than previous - red
			styledBalance = negativeStyle.Render(balanceText)
		} else {
			// Balance unchanged - neutral
			styledBalance = neutralStyle.Render(balanceText)
		}
	} else {
		// First time showing balance - use neutral style
		styledBalance = valueStyle.Render(balanceText)
	}

	return fmt.Sprintf("%s %s", 
		labelStyle.Render("Balance:"), 
		styledBalance)
}

// renderPositionCard renders a single position card
func (m Model) renderPositionCard(pos core.PositionData) string {
	var content strings.Builder
	
	// Use full instrument ID (e.g., "SOL-USDT-SWAP") instead of just coin name
	instrumentName := pos.InstrumentID
	
	// Card header with prominent instrument name - full trading pair
	content.WriteString(cardHeaderStyle.Render(fmt.Sprintf("▶ %s ◀", instrumentName)))
	content.WriteString("\n")
	
	// Position details
	content.WriteString(fmt.Sprintf("%s %s\n", 
		labelStyle.Render("Side:"), 
		valueStyle.Render(pos.PositionSide)))
	
	content.WriteString(fmt.Sprintf("%s %s\n", 
		labelStyle.Render("Size:"), 
		valueStyle.Render(fmt.Sprintf("%.4f", pos.Size))))
	
	// Format entry price with appropriate precision
	var entryPriceStr string
	if pos.AvgPrice < 0.001 {
		entryPriceStr = fmt.Sprintf("%.5f", pos.AvgPrice)
	} else {
		entryPriceStr = fmt.Sprintf("%.2f", pos.AvgPrice)
	}
	content.WriteString(fmt.Sprintf("%s %s\n", 
		labelStyle.Render("Entry:"), 
		valueStyle.Render(entryPriceStr)))
	
	// Format current price with appropriate precision
	var currentPriceStr string
	if pos.CurrentPrice < 0.001 {
		currentPriceStr = fmt.Sprintf("%.5f", pos.CurrentPrice)
	} else {
		currentPriceStr = fmt.Sprintf("%.2f", pos.CurrentPrice)
	}
	content.WriteString(fmt.Sprintf("%s %s\n", 
		labelStyle.Render("Current:"), 
		valueStyle.Render(currentPriceStr)))
	
	// PnL with color
	pnlStr := fmt.Sprintf("%.2f", pos.PnL)
	if pos.PnL > 0 {
		pnlStr = "+" + pnlStr
		pnlStr = positiveStyle.Render(pnlStr)
	} else if pos.PnL < 0 {
		pnlStr = negativeStyle.Render(pnlStr)
	} else {
		pnlStr = neutralStyle.Render(pnlStr)
	}
	
	content.WriteString(fmt.Sprintf("%s %s\n", 
		labelStyle.Render("PnL:"), pnlStr))
	
	// PnL percentage with color
	pnlPercentStr := fmt.Sprintf("%.2f%%", pos.PnLRatio)
	if pos.PnLRatio > 0 {
		pnlPercentStr = "+" + pnlPercentStr
		pnlPercentStr = positiveStyle.Render(pnlPercentStr)
	} else if pos.PnLRatio < 0 {
		pnlPercentStr = negativeStyle.Render(pnlPercentStr)
	} else {
		pnlPercentStr = neutralStyle.Render(pnlPercentStr)
	}
	
	content.WriteString(fmt.Sprintf("%s %s\n", 
		labelStyle.Render("PnL %:"), pnlPercentStr))
	
	content.WriteString(fmt.Sprintf("%s %s", 
		labelStyle.Render("Leverage:"), 
		valueStyle.Render(fmt.Sprintf("%.0fx", pos.Leverage))))
	
	// Render the entire card with border and styling
	return cardStyle.Render(content.String())
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		waitForPositionUpdate(m.positionCh),
		waitForBalanceUpdate(m.balanceCh),
		waitForError(m.errorCh),
		tick(),
	)
}

// Update handles model updates
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Calculate content lines and max scroll for boundary checking
		var mainContent string
		if len(m.positions) == 0 {
			// Show different messages based on whether we have received any updates
			var statusMsg, detailMsg string
			if m.lastUpdate.IsZero() {
				statusMsg = "Initializing..."
				detailMsg = "Starting OKX connection\nPlease wait while we connect to the API"
			} else {
				statusMsg = "Connected - No Positions"
				detailMsg = "Successfully connected to OKX\nNo open positions found"
			}
			
			waitingMsg := cardStyle.Render(
				labelStyle.Render("Status: ") + valueStyle.Render(statusMsg) + "\n" +
				labelStyle.Render("Time: ") + valueStyle.Render(time.Now().Format("15:04:05")) + "\n\n" +
				neutralStyle.Render(detailMsg))
			mainContent = waitingMsg
		} else {
			mainContent = m.renderPositionCards()
		}
		
		lines := strings.Split(mainContent, "\n")
		availableHeight := m.height - 10 // Reserve space for header, footer, padding, etc.
		if availableHeight < 5 {
			availableHeight = 5
		}
		
		// Calculate maximum scroll offset
		maxScroll := len(lines) - availableHeight
		if maxScroll < 0 {
			maxScroll = 0
		}
		
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "d":
			// Toggle debug output visibility
			m.showDebug = !m.showDebug
			// Clear existing debug messages when turning debug off
			if !m.showDebug {
				m.debugMessages = make([]string, 0)
			}
		case "up", "k":
			// Scroll up
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}
		case "down", "j":
			// Scroll down
			if m.scrollOffset < maxScroll {
				m.scrollOffset++
			}
		case "pgup":
			// Page up (scroll up by 5 lines)
			m.scrollOffset -= 5
			if m.scrollOffset < 0 {
				m.scrollOffset = 0
			}
		case "pgdown":
			// Page down (scroll down by 5 lines)
			m.scrollOffset += 5
			if m.scrollOffset > maxScroll {
				m.scrollOffset = maxScroll
			}
		case "home":
			// Go to top
			m.scrollOffset = 0
		case "end":
			// Go to bottom
			m.scrollOffset = maxScroll
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case positionUpdateMsg:
		// Handle position updates - could be full position data or just ticker updates
		if msg.PositionSide != "" {
			// Full position data with position side
			key := fmt.Sprintf("%s-%s", msg.InstrumentID, msg.PositionSide)
			m.positions[key] = core.PositionData(msg)
			m.lastUpdate = time.Now()

			// Add debug message for position update
			m.AddDebugMessage(fmt.Sprintf("Position updated: %s %s %.4f @ %.2f", 
				msg.InstrumentID, msg.PositionSide, msg.Size, msg.CurrentPrice))
		} else {
			// Ticker update - find existing positions for this instrument and update price
			updated := false
			for key, position := range m.positions {
				if position.InstrumentID == msg.InstrumentID {
					// Update current price and recalculate PnL
					position.CurrentPrice = msg.CurrentPrice
					position.Timestamp = msg.Timestamp
					
					// Recalculate PnL if we have position data
					if position.Size > 0 && position.AvgPrice > 0 {
						if position.PositionSide == "short" {
							// For short positions, profit when price goes down
							position.PnL = (position.AvgPrice - position.CurrentPrice) * position.Size
						} else {
							// For long positions, profit when price goes up
							position.PnL = (position.CurrentPrice - position.AvgPrice) * position.Size
						}
						position.PnLRatio = (position.PnL / (position.AvgPrice * position.Size)) * 100
					}
					
					m.positions[key] = position
					updated = true
				}
			}
			
			if updated {
				m.lastUpdate = time.Now()
				// Add debug message for ticker update
				m.AddDebugMessage(fmt.Sprintf("Ticker updated: %s @ %.2f", 
					msg.InstrumentID, msg.CurrentPrice))
			}
		}

		// Clear any previous errors when we get successful updates
		m.ClearError()

		return m, waitForPositionUpdate(m.positionCh)

	case balanceUpdateMsg:
		// Calculate current total balance before updating
		var currentTotalBalance float64
		for _, balance := range m.balances {
			currentTotalBalance += balance.TotalEquity
		}
		
		// Store the current total as previous balance
		if currentTotalBalance > 0 {
			m.previousBalance = currentTotalBalance
		}
		
		// Update balance data
		m.balances[msg.Currency] = core.BalanceData(msg)
		m.lastUpdate = time.Now()

		// Add debug message for balance update
		m.AddDebugMessage(fmt.Sprintf("Balance updated: %s Total: %.4f Available: %.4f", 
			msg.Currency, msg.TotalEquity, msg.AvailBalance))

		return m, waitForBalanceUpdate(m.balanceCh)

	case errorMsg:
		// Check if this is a debug message (starts with "DEBUG:")
		msgStr := string(msg)
		if strings.HasPrefix(msgStr, "DEBUG:") {
			// Remove "DEBUG:" prefix and add to debug messages
			debugMsg := strings.TrimPrefix(msgStr, "DEBUG:")
			m.AddDebugMessage(strings.TrimSpace(debugMsg))
		} else {
			// Set as error message
			m.SetError(msgStr)
		}
		return m, waitForError(m.errorCh)

	case tickMsg:
		return m, tick()
	}

	return m, cmd
}

// View renders the UI
func (m Model) View() string {
	// Build the UI components
	var content strings.Builder
	
	// Create header with title on left and time info on right
	title := titleStyle.Render("OKX Position Monitor")
	
	// Get balance display
	balance := m.renderBalance()
	
	// Current time and last update time
	currentTime := time.Now()
	var timeInfo string
	if m.lastUpdate.IsZero() {
		timeInfo = fmt.Sprintf("%s\n%s", 
			currentTimeStyle.Render(currentTime.Format("2006-01-02 15:04:05")),
			timeStyle.Render("Last update: --:--:--"))
	} else {
		timeInfo = fmt.Sprintf("%s\n%s", 
			currentTimeStyle.Render(currentTime.Format("2006-01-02 15:04:05")),
			timeStyle.Render(fmt.Sprintf("Last update: %s", m.lastUpdate.Format("15:04:05"))))
	}
	
	// Calculate available width for header layout
	headerWidth := m.width - 8 // Account for base style padding and borders
	if headerWidth < 40 {
		headerWidth = 40 // Minimum width
	}
	
	// Create header layout
	var header string
	if balance != "" {
		// Three-column layout: title | balance | time
		titleWidth := lipgloss.Width(title)
		balanceWidth := lipgloss.Width(balance)
		timeWidth := lipgloss.Width(timeInfo)
		
		// Calculate spacing
		totalContentWidth := titleWidth + balanceWidth + timeWidth
		if totalContentWidth + 4 <= headerWidth { // +4 for minimum spacing
			// Distribute remaining space
			remainingSpace := headerWidth - totalContentWidth
			leftSpacing := remainingSpace / 2
			rightSpacing := remainingSpace - leftSpacing
			
			leftSpacer := strings.Repeat(" ", leftSpacing)
			rightSpacer := strings.Repeat(" ", rightSpacing)
			
			header = lipgloss.JoinHorizontal(lipgloss.Top, title, leftSpacer, balance, rightSpacer, timeInfo)
		} else {
			// Not enough space for three columns, stack vertically
			header = lipgloss.JoinVertical(lipgloss.Left, 
				lipgloss.JoinHorizontal(lipgloss.Top, title, strings.Repeat(" ", headerWidth-titleWidth-timeWidth), timeInfo),
				lipgloss.NewStyle().Align(lipgloss.Center).Width(headerWidth).Render(balance))
		}
	} else {
		// Two-column layout: title | time (original layout)
		titleWidth := lipgloss.Width(title)
		timeWidth := lipgloss.Width(timeInfo)
		spacingWidth := headerWidth - titleWidth - timeWidth
		
		if spacingWidth > 0 {
			spacing := strings.Repeat(" ", spacingWidth)
			header = lipgloss.JoinHorizontal(lipgloss.Top, title, spacing, timeInfo)
		} else {
			// If not enough space, stack vertically
			header = lipgloss.JoinVertical(lipgloss.Left, title, timeInfo)
		}
	}
	
	content.WriteString(header)
	content.WriteString("\n\n")
	
	// Add position cards or waiting message
	var mainContent string
	if len(m.positions) == 0 {
		// Show different messages based on whether we have received any updates
		var statusMsg, detailMsg string
		if m.lastUpdate.IsZero() {
			statusMsg = "Initializing..."
			detailMsg = "Starting OKX connection\nPlease wait while we connect to the API"
		} else {
			statusMsg = "Connected - No Positions"
			detailMsg = "Successfully connected to OKX\nNo open positions found"
		}
		
		waitingMsg := cardStyle.Render(
			labelStyle.Render("Status: ") + valueStyle.Render(statusMsg) + "\n" +
			labelStyle.Render("Time: ") + valueStyle.Render(currentTime.Format("15:04:05")) + "\n\n" +
			neutralStyle.Render(detailMsg))
		mainContent = waitingMsg
	} else {
		mainContent = m.renderPositionCards()
	}
	
	// Split main content into lines for scrolling
	lines := strings.Split(mainContent, "\n")
	
	// Calculate available height for content (excluding header, footer, and padding)
	availableHeight := m.height - 10 // Reserve space for header, footer, padding, etc.
	if availableHeight < 5 {
		availableHeight = 5 // Minimum height
	}
	
	// Apply scroll offset and limit to available lines
	startLine := m.scrollOffset
	endLine := startLine + availableHeight
	
	// Ensure we don't go beyond the content
	if endLine > len(lines) {
		endLine = len(lines)
	}
	
	// Get visible lines
	var visibleLines []string
	if startLine < len(lines) {
		visibleLines = lines[startLine:endLine]
	}
	
	// Add visible content
	content.WriteString(strings.Join(visibleLines, "\n"))
	content.WriteString("\n")
	
	// Add debug section if enabled and there are debug messages
	if m.showDebug {
		debugSection := m.renderDebugSection()
		if debugSection != "" {
			content.WriteString("\n")
			content.WriteString(debugSection)
			content.WriteString("\n")
		}
	}
	
	// Add error message if there is one
	if m.hasError {
		content.WriteString("\n")
		content.WriteString(errorStyle.Render(fmt.Sprintf("Error: %s", m.errorMsg)))
		content.WriteString("\n")
	}
	
	// Add footer with navigation instructions and scroll indicator
	content.WriteString("\n")
	
	// Show scroll indicator if there's more content
	scrollInfo := ""
	if len(lines) > availableHeight {
		maxScroll := len(lines) - availableHeight
		scrollInfo = fmt.Sprintf(" | Scroll: %d/%d", startLine, maxScroll)
	}
	
	// Show debug status
	debugStatus := ""
	if m.showDebug {
		debugStatus = " | Debug: ON"
	} else {
		debugStatus = " | Debug: OFF"
	}
	
	footerText := "Press q or Ctrl+C to quit | d to toggle debug | ↑↓ or j/k to scroll | PgUp/PgDn | Home/End" + scrollInfo + debugStatus
	content.WriteString(footerText)

	return baseStyle.Render(content.String())
}

// Message types
type positionUpdateMsg core.PositionData
type balanceUpdateMsg core.BalanceData
type tickMsg time.Time
type errorMsg string

// SetError sets an error message
func (m *Model) SetError(msg string) {
	m.errorMsg = msg
	m.hasError = true
}

// ClearError clears the error message
func (m *Model) ClearError() {
	m.errorMsg = ""
	m.hasError = false
}

// AddDebugMessage adds a debug message to the debug output
func (m *Model) AddDebugMessage(msg string) {
	// Only add debug messages when debug mode is enabled
	if !m.showDebug {
		return
	}
	
	timestamp := time.Now().Format("15:04:05")
	debugMsg := fmt.Sprintf("[%s] %s", timestamp, msg)
	
	m.debugMessages = append(m.debugMessages, debugMsg)
	
	// Keep only the last maxDebugLines messages
	if len(m.debugMessages) > m.maxDebugLines {
		m.debugMessages = m.debugMessages[len(m.debugMessages)-m.maxDebugLines:]
	}
}

// renderDebugSection renders the debug output section
func (m Model) renderDebugSection() string {
	if len(m.debugMessages) == 0 {
		return ""
	}
	
	var content strings.Builder
	content.WriteString(debugHeaderStyle.Render("Debug Output"))
	content.WriteString("\n")
	
	for _, msg := range m.debugMessages {
		content.WriteString(msg)
		content.WriteString("\n")
	}
	
	return debugStyle.Render(content.String())
}

// waitForPositionUpdate waits for position updates from the channel
func waitForPositionUpdate(ch <-chan core.PositionData) tea.Cmd {
	return func() tea.Msg {
		pos, ok := <-ch
		if !ok {
			return errorMsg("Connection to OKX API lost. Please restart the application.")
		}
		return positionUpdateMsg(pos)
	}
}

// waitForBalanceUpdate waits for balance updates from the channel
func waitForBalanceUpdate(ch <-chan core.BalanceData) tea.Cmd {
	return func() tea.Msg {
		balance, ok := <-ch
		if !ok {
			return errorMsg("Balance channel closed.")
		}
		return balanceUpdateMsg(balance)
	}
}

// waitForError waits for error messages from the channel
func waitForError(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		err, ok := <-ch
		if !ok {
			return nil
		}
		return errorMsg(err)
	}
}

// tick sends a tick message every second for UI updates
func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}