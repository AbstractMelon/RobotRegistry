# Robot Registry

A modern, fast, and user-friendly website for browsing robot combat events, bots, teams, and rankings.

## Quick Start

### Prerequisites

- Go 1.21 or higher
- Node.js 18 or higher
- pnpm

**Backend:**

```bash
cd backend
go mod tidy
go run main.go
```

**Frontend:**

```bash
cd frontend
pnpm install
pnpm dev
```

The backend will run on http://localhost:8080 and the frontend on http://localhost:5173.

## API Endpoints

- `GET /api/events` - List events with filtering and pagination
- `GET /api/events/:id` - Get event details
- `GET /api/competitions/:id` - Get competition details
- `GET /api/bots` - List bots with filtering and pagination
- `GET /api/bots/:id` - Get bot details
- `GET /api/teams` - List teams with pagination
- `GET /api/teams/:id` - Get team details
- `GET /api/rankings` - Get rankings by year and weight class
- `GET /api/rankings/years` - Get available years
- `GET /api/rankings/weight-classes` - Get available weight classes
- `GET /api/search?q=query` - Global search

## Production Build

**Backend:**

```bash
cd backend
go build -o robotregistry main.go
./robotregistry
```

**Frontend:**

```bash
cd frontend
pnpm build
```

The backend will automatically serve the frontend build in production.

## Acknowledgments

Data sourced from Robot Combat Events