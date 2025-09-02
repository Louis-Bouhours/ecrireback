#!/bin/bash

# Build script for ecrireback

set -e

echo "🔨 Building ecrireback..."

# Clean previous builds
rm -rf build/*

# Build for current platform
go build -o build/ecrireback ./cmd/server

echo "✅ Build complete! Binary available at: build/ecrireback"

# Make sure the binary is executable
chmod +x build/ecrireback

echo "📦 Build size:"
ls -lh build/ecrireback