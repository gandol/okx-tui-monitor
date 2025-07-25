# OKX Trading Terminal (TUI)

A professional terminal-based user interface for monitoring OKX trading positions and account balances in real-time with advanced grid layout.

## 🎬 Preview

https://github.com/gandol/okx-tui-monitor/raw/main/assets/preview.mp4

*See the OKX TUI Monitor in action - real-time position monitoring, live PnL calculations, and responsive grid layout.*

## ✨ Enhanced Features

### 🎯 **Core Functionality**
- **Real-time Position Monitoring** - Live WebSocket feeds from OKX API
- **Dynamic Grid Layout** - Responsive grid displaying up to 10+ positions simultaneously
- **Account Balance Tracking** - Real-time balance updates with color-coded changes
- **Advanced Demo Mode** - 10 diverse cryptocurrency positions with live market data
- **Professional UI** - Clean, organized terminal interface with modern styling

### 📊 **Trading Intelligence**
- **Live PnL Calculations** - Real-time profit/loss tracking with percentage changes
- **Position Analytics** - Entry price, current price, leverage, and position size
- **Market Data Integration** - Live ticker feeds for all major trading pairs
- **Balance Monitoring** - Track available balance and total equity changes
- **Multi-Asset Support** - BTC, ETH, SOL, ADA, DOT, LINK, AVAX, MATIC, UNI, LTC

### 🔧 **Technical Excellence**
- **Dual Mode Operation** - Authenticated mode (real data) + Demo mode (test data)
- **Error-Free Logging** - All debug information routed through error channels
- **Secure Credential Handling** - Environment variable isolation with validation
- **WebSocket Reliability** - Automatic reconnection and heartbeat monitoring
- **Cross-Platform Support** - Linux, Windows, macOS (Intel & Apple Silicon)

### 🎮 **User Experience**
- **Interactive Controls** - Keyboard navigation (q/Ctrl+C to quit, d for debug toggle)
- **Responsive Design** - Adapts to terminal width (1-8 cards per row)
- **Real-time Updates** - Sub-second data refresh rates
- **Debug Mode** - Toggle debug information visibility
- **Status Indicators** - Connection status and last update timestamps

## 🚀 **Quick Start**

### Demo Mode (No Setup Required)
```bash
# Clone and run immediately
git clone https://github.com/gandol/okx-tui-monitor.git
cd okx-tui-monitor
go run main.go
```

### Live Trading Mode
```bash
# Set up API credentials
cp .env.example .env
# Edit .env with your OKX API credentials
go run main.go
```

## Security & Setup

### 🔒 **IMPORTANT SECURITY NOTICE**

This application handles sensitive trading API credentials. Please follow these security guidelines:

1. **Never commit your `.env` file** - It's already in `.gitignore`
2. **Keep your API credentials private**
3. **Use API keys with minimal required permissions**
4. **Regularly rotate your API keys**

### Installation

1. Clone this repository:
```bash
git clone https://github.com/gandol/okx-tui-monitor.git
cd okx-tui-monitor
```

2. Install dependencies:
```bash
go mod download
```

3. Set up your API credentials:
```bash
cp .env.example .env
```

4. Edit `.env` file with your actual OKX API credentials:
```bash
# Get these from your OKX account -> API Management
OKX_API_KEY=your-actual-api-key
OKX_API_SECRET=your-actual-api-secret  
OKX_API_PASSPHRASE=your-actual-passphrase
```

### Getting OKX API Credentials

1. Log into your OKX account
2. Go to **Profile** → **API Management**
3. Create a new API key with these permissions:
   - **Read**: For viewing positions and balances
   - **Trade**: Only if you plan to add trading features (optional)
4. Copy the API Key, Secret, and Passphrase to your `.env` file

### Running the Application

```bash
# With your API credentials (private data)
go run main.go
```


## Build & Releases

### 📦 **Pre-built Releases**

Download the latest release for your platform:

| Platform | Architecture | Download |
|----------|-------------|----------|
| **Linux** | x86_64 | [okx-tui-linux-amd64](https://github.com/gandol/okx-tui-monitor/releases/latest/download/okx-tui-linux-amd64) |
| **Linux** | ARM64 | [okx-tui-linux-arm64](https://github.com/gandol/okx-tui-monitor/releases/latest/download/okx-tui-linux-arm64) |
| **Windows** | x86_64 | [okx-tui-windows-amd64.exe](https://github.com/gandol/okx-tui-monitor/releases/latest/download/okx-tui-windows-amd64.exe) |
| **Windows** | ARM64 | [okx-tui-windows-arm64.exe](https://github.com/gandol/okx-tui-monitor/releases/latest/download/okx-tui-windows-arm64.exe) |
| **macOS** | Intel (x86_64) | [okx-tui-darwin-amd64](https://github.com/gandol/okx-tui-monitor/releases/latest/download/okx-tui-darwin-amd64) |
| **macOS** | Apple Silicon (ARM64) | [okx-tui-darwin-arm64](https://github.com/gandol/okx-tui-monitor/releases/latest/download/okx-tui-darwin-arm64) |

### 🔨 **Build from Source**

#### Quick Build (Current Platform)
```bash
go build -o okx-tui main.go
./okx-tui
```

#### Cross-Platform Builds

**Linux:**
```bash
# Linux x86_64
GOOS=linux GOARCH=amd64 go build -o okx-tui-linux-amd64 main.go

# Linux ARM64
GOOS=linux GOARCH=arm64 go build -o okx-tui-linux-arm64 main.go
```

**Windows:**
```bash
# Windows x86_64
GOOS=windows GOARCH=amd64 go build -o okx-tui-windows-amd64.exe main.go

# Windows ARM64
GOOS=windows GOARCH=arm64 go build -o okx-tui-windows-arm64.exe main.go
```

**macOS:**
```bash
# macOS Intel (x86_64)
GOOS=darwin GOARCH=amd64 go build -o okx-tui-darwin-amd64 main.go

# macOS Apple Silicon (ARM64)
GOOS=darwin GOARCH=arm64 go build -o okx-tui-darwin-arm64 main.go
```

#### Build All Platforms at Once
```bash
# Create a build script
cat > build-all.sh << 'EOF'
#!/bin/bash

# Create releases directory
mkdir -p releases

# Build for all platforms
echo "Building for Linux x86_64..."
GOOS=linux GOARCH=amd64 go build -o releases/okx-tui-linux-amd64 main.go

echo "Building for Linux ARM64..."
GOOS=linux GOARCH=arm64 go build -o releases/okx-tui-linux-arm64 main.go

echo "Building for Windows x86_64..."
GOOS=windows GOARCH=amd64 go build -o releases/okx-tui-windows-amd64.exe main.go

echo "Building for Windows ARM64..."
GOOS=windows GOARCH=arm64 go build -o releases/okx-tui-windows-arm64.exe main.go

echo "Building for macOS Intel..."
GOOS=darwin GOARCH=amd64 go build -o releases/okx-tui-darwin-amd64 main.go

echo "Building for macOS Apple Silicon..."
GOOS=darwin GOARCH=arm64 go build -o releases/okx-tui-darwin-arm64 main.go

echo "All builds completed! Check the releases/ directory."
EOF

chmod +x build-all.sh
./build-all.sh
```

### 🚀 **Installation Instructions**

**Linux/macOS:**
```bash
# Download the appropriate binary for your platform
# Make it executable
chmod +x okx-tui-*

# Run the application
./okx-tui-*
```

**Windows:**
```cmd
# Download the .exe file for your architecture
# Run directly
okx-tui-windows-*.exe
```



## 🔒 **Security & Technical Features**

### **Enhanced Security**
- ✅ **Input validation** for API credentials
- ✅ **Zero sensitive data exposure** - No credentials in logs or debug output
- ✅ **Secure credential handling** with environment variable isolation
- ✅ **Error channel routing** - All debug information through secure channels
- ✅ **No log.Printf usage** - Eliminated all standard logging for security

### **Technical Excellence**
- ✅ **WebSocket reliability** - Automatic reconnection and heartbeat monitoring
- ✅ **Real-time data processing** - Sub-second ticker updates
- ✅ **Memory efficient** - Optimized data structures and channel management
- ✅ **Cross-platform compatibility** - Tested on Linux, Windows, macOS
- ✅ **Responsive UI** - Dynamic grid layout adapting to terminal size
- ✅ **Error resilience** - Graceful handling of network issues and API errors

### **Performance Features**
- ✅ **Concurrent processing** - Separate goroutines for WebSocket connections
- ✅ **Efficient data updates** - Smart position merging and ticker integration
- ✅ **Minimal resource usage** - Optimized for terminal environments
- ✅ **Real-time calculations** - Live PnL updates without blocking UI

## Contributing

1. Fork the repository
2. Create a feature branch
3. **Never commit your `.env` file**
4. Submit a pull request

## Support the Project

If you find this project helpful, consider supporting its development:

### 💰 **Donations**

**Bitcoin (BTC):**
```
bc1qepznrxnps9v6x7fh2leh6kd7hzpxmgdrq7u5svj8yrj9ryu3mycq4877xs
```

**Ethereum (ETH) - ERC20:**
```
0xb55ec1c4cbb800cdc9d46bed3e55aa35ad3d90a7
```

Your support helps maintain and improve this project. Thank you! 🙏

## Disclaimer

This software is for educational and monitoring purposes. Always verify trades and positions through the official OKX interface. The developers are not responsible for any trading losses.

## License

MIT License - see LICENSE file for details.