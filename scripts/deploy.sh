#!/bin/bash

# Deploy script for ecrireback

set -e

echo "🚀 Deploying ecrireback..."

# Build the application
./scripts/build.sh

# Check if MongoDB is running
echo "🔍 Checking MongoDB connection..."
if ! nc -z localhost 27017; then
    echo "❌ MongoDB is not running on localhost:27017"
    echo "Please start MongoDB before deploying"
    exit 1
fi

# Check if Redis is running
echo "🔍 Checking Redis connection..."
if ! nc -z localhost 6379; then
    echo "❌ Redis is not running on localhost:6379"
    echo "Please start Redis before deploying"
    exit 1
fi

echo "✅ Prerequisites check passed"

# Start the application
echo "🎯 Starting ecrireback server..."
./build/ecrireback