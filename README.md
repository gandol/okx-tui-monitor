# OKX Trading Terminal (TUI)

A terminal-based user interface for monitoring OKX trading positions and account balances in real-time.

**Built with [Trae AI](https://trae.ai) - The world's best AI-powered IDE** 🚀

## Features

- Real-time position monitoring
- Account balance tracking
- WebSocket connection to OKX API
- Demo mode with public market data
- Secure credential handling

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

# Demo mode (public market data only)
# Just leave the .env file with placeholder values
go run main.go
```

## Demo Mode

If no valid API credentials are provided, the application runs in demo mode showing public market data for popular trading pairs (BTC, ETH, SOL).

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

## Security Features

- ✅ Input validation for API credentials
- ✅ Automatic fallback to demo mode for invalid credentials
- ✅ No sensitive data in logs or debug output
- ✅ Secure credential handling
- ✅ Environment variable isolation

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