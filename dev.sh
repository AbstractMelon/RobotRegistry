#!/bin/bash
# Development script - runs backend and frontend concurrently

set -e

# Load environment variables
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

echo "Starting development servers..."
echo "Backend: http://localhost:${PORT:-8080}"
echo "Frontend: http://localhost:5173"
echo ""

# Run backend and frontend concurrently
trap 'kill 0' EXIT

cd backend && go run main.go &
BACKEND_PID=$!

cd frontend && pnpm dev &
FRONTEND_PID=$!

wait
