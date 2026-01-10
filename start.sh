#!/bin/bash
# Production start script

set -e

# Load environment variables
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

# Check if binary exists
if [ ! -f "./robotregistry" ]; then
    echo "Backend binary not found. Running build script..."
    ./build.sh
fi

echo "Starting in production mode..."
echo "Server: http://localhost:${PORT:-8080}"
echo ""

# Run the binary
./robotregistry
