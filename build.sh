#!/bin/bash
# Production build script

set -e

echo "Building for production..."

# Load environment variables
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

# Install dependencies
echo "Installing dependencies..."
cd frontend && pnpm install
cd ../backend && go mod download
cd ../scraper && go mod download
cd ..

# Build frontend with production API URL
echo "Building frontend..."
cd frontend
VITE_API_URL="${PROD_API_URL:-/api}" pnpm run build
cd ..

# Build backend
echo "Building backend..."
cd backend
go build -o ../robotregistry main.go
cd ..

# Build scraper
echo "Building scraper..."
cd scraper
go build -o ../scraper-bin main.go
cd ..

echo "Build complete!"
echo ""
echo "Production binary: ./robotregistry"
echo "Scraper binary: ./scraper-bin"
echo ""
echo "To run in production:"
echo "  ./robotregistry"
