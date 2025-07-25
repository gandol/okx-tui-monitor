#!/bin/bash

# OKX TUI - Multi-platform Build Script
# This script builds the application for all supported platforms and architectures

set -e

echo "🚀 Starting multi-platform build for OKX TUI..."

# Create releases directory
mkdir -p releases

# Clean previous builds
rm -f releases/*

# Build information
VERSION=${1:-"latest"}
echo "📦 Building version: $VERSION"

# Build for all platforms
echo ""
echo "🐧 Building for Linux x86_64..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o releases/okx-tui-linux-amd64 main.go

echo "🐧 Building for Linux ARM64..."
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o releases/okx-tui-linux-arm64 main.go

echo "🪟 Building for Windows x86_64..."
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o releases/okx-tui-windows-amd64.exe main.go

echo "🪟 Building for Windows ARM64..."
GOOS=windows GOARCH=arm64 go build -ldflags="-s -w" -o releases/okx-tui-windows-arm64.exe main.go

echo "🍎 Building for macOS Intel (x86_64)..."
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o releases/okx-tui-darwin-amd64 main.go

echo "🍎 Building for macOS Apple Silicon (ARM64)..."
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o releases/okx-tui-darwin-arm64 main.go

echo ""
echo "✅ All builds completed successfully!"
echo ""
echo "📁 Built files:"
ls -la releases/

echo ""
echo "📊 File sizes:"
du -h releases/*

echo ""
echo "🎉 Ready for release! Upload the files in the releases/ directory to GitHub Releases."
echo ""
echo "💡 To create a GitHub release:"
echo "   1. Go to your repository on GitHub"
echo "   2. Click 'Releases' → 'Create a new release'"
echo "   3. Upload all files from the releases/ directory"
echo "   4. Add release notes and publish"