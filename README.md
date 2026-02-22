# SignalStream

A real-time financial event pipeline that ingests market news and SEC filings, enriches them with semantic embeddings, and triggers alerts when incoming events match user-defined investment strategies via vector similarity search.

Users define strategies using natural language queries like "merger and acquisition activity in the technology sector" with optional source and ticker filters. The system continuously processes incoming events and alerts users when content semantically matches their strategies above a configurable similarity threshold — not keyword matching, but meaning-based matching using cosine similarity over OpenAI embeddings.

## Architecture

```
┌──────────────┐   ┌──────────────┐
│ Polygon.io   │   │ SEC EDGAR    │
│ News API     │   │ 8-K Filings  │
└──────┬───────┘   └──────┬───────┘
       │ rate-limited      │ 90s poll
       │ (1 req/12s)       │
       ▼                   ▼
   ┌──────────────────────────┐
   │   NormalizedEvent chan   │  buffered channel (cap 100)
   │       (fan-in)          │
   └────────────┬────────────┘
                │
        ┌───────┼───────┐
        ▼       ▼       ▼        4 worker goroutines (fan-out)
   ┌─────────────────────────┐
   │    Enrichment Service   │
   │  fetch article text     │
   │  → OpenAI embedding     │
   │  → Qdrant upsert        │
   └────────────┬────────────┘
                │
   ┌────────────▼────────────┐
   │     Alerter Service     │
   │  load active strategies │
   │  → pre-filter (source/  │
   │    ticker)              │
   │  → embed strategy query │
   │  → Qdrant cosine search │
   │  → threshold check      │
   └────────────┬────────────┘
                │
   ┌────────────▼────────────┐
   │    MultiNotifier        │
   │  → structured log       │
   │  → PostgreSQL persist   │
   │  → SSE broadcast        │
   └─────────────────────────┘
                │
        ┌───────┼───────┐
        ▼       ▼       ▼
   [Log]    [Database]  [Browser via SSE]
```

### Components

- **Ingestion** — Two producer goroutines poll Polygon.io (rate-limited via `x/time/rate`) and SEC EDGAR (Atom feed via `gofeed`). Both normalize events into a shared `NormalizedEvent` struct and push to a buffered channel.

- **Enrichment** — Workers fetch full article text using `go-readability`, generate 1536-dimensional embeddings via OpenAI `text-embedding-3-small`, and upsert into Qdrant with deterministic UUID v5 point IDs for idempotency.

- **Alerter** — For each enriched event, the alerter loads all active strategies from PostgreSQL, pre-filters by source/ticker overlap (avoiding unnecessary API calls), embeds the strategy query, performs a filtered cosine similarity search in Qdrant, and checks if the event appears in results above the user's threshold.

- **Notification** — A `MultiNotifier` fans out alert delivery: structured log output, PostgreSQL persistence, and SSE broadcast to connected browser clients.

- **API Server** — REST API with JWT authentication, AES-256-GCM encrypted API key storage, strategy CRUD, paginated alerts, and SSE streaming.

- **Frontend** — React + TypeScript dashboard with Zustand state management, TanStack Query for data fetching, and Tremor UI components.

## Data Flow

```
1. Polygon/SEC → poll → NormalizedEvent
2. NormalizedEvent → buffered channel (100)
3. Worker pool (4) → fetch article text (go-readability, 30s timeout)
4. Article text → OpenAI text-embedding-3-small → 1536-dim vector
5. Vector + metadata → Qdrant upsert (deterministic UUID v5)
6. Load active strategies (PostgreSQL JOIN users)
7. Pre-filter: source/ticker match check
8. Strategy query → OpenAI embedding
9. Qdrant cosine similarity search (filtered, top 10)
10. If event in results AND score ≥ threshold → trigger alert
11. Alert → log + PostgreSQL + SSE broadcast
```

## Key Features

- **Fan-in/fan-out pipeline** — 2 producers, buffered channel, 4 consumer workers with per-event context timeouts (2min) and graceful shutdown via OS signals
- **Semantic alerting** — Vector similarity search instead of keyword matching. Users define strategies in natural language
- **Retry with exponential backoff** — All OpenAI and Qdrant calls wrapped with configurable retry (3 attempts, jitter, context-aware)
- **Structured logging** — JSON-formatted `slog` output with event_id, strategy_id, latency_ms, and contextual fields throughout
- **Metrics** — Atomic counters for events_ingested, events_processed, events_enriched, alerts_triggered, errors_total, retry_attempts. Exposed via `/api/v1/metrics`
- **SSE real-time alerts** — Server-Sent Events endpoint streams alerts to connected clients, supporting multiple tabs per user
- **Security** — JWT (HS256, 24h expiry), bcrypt password hashing, AES-256-GCM encryption for stored API keys
- **Interface-driven design** — `Embedder`, `VectorSearcher`, `StrategyStore`, `Notifier`, `ContentFetcher` interfaces enable testing with mocks

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.24 |
| Database | PostgreSQL 16 (pgx/v5 driver, connection pool) |
| Vector DB | Qdrant (gRPC client, cosine similarity, 1536-dim) |
| Embeddings | OpenAI text-embedding-3-small |
| News Data | Polygon.io client-go |
| SEC Data | gofeed (Atom parser) + SEC EDGAR RSS |
| Auth | golang-jwt/v5 (HS256), bcrypt, AES-256-GCM |
| Frontend | React 18, TypeScript, Vite, TailwindCSS, Tremor, Zustand, TanStack Query |
| Infra | Docker Compose, GitHub Actions CI |

## Running the Project

### Prerequisites

- Go 1.24+
- Docker and Docker Compose
- [Polygon.io](https://polygon.io/) API key (free tier works)
- [OpenAI](https://platform.openai.com/) API key

### Setup

```sh
git clone https://github.com/Atrix21/signalstream.git
cd signalstream

# Configure environment
cp .env.example .env
# Edit .env with your API keys, JWT secret (≥16 chars), and encryption key (exactly 32 bytes)

# Start infrastructure (PostgreSQL + Qdrant)
docker-compose up -d

# Install dependencies
go mod tidy

# Run the backend
go run ./cmd/signalstream/

# Run the frontend (separate terminal)
cd frontend && npm install && npm run dev
```

### Running Tests

```sh
go test ./internal/... -race -count=1
```

## API Overview

All routes under `/api/v1/`. Authenticated routes require `Authorization: Bearer <jwt>`.

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/auth/register` | No | Create account, returns JWT |
| POST | `/auth/login` | No | Authenticate, returns JWT |
| GET | `/auth/me` | Yes | Current user profile |
| GET | `/strategies` | Yes | List user strategies |
| POST | `/strategies` | Yes | Create strategy |
| DELETE | `/strategies?id=` | Yes | Delete strategy |
| PATCH | `/strategies/toggle?id=` | Yes | Toggle strategy active/paused |
| GET | `/alerts?limit=&offset=` | Yes | Paginated alert list |
| PATCH | `/alerts/read?id=` | Yes | Mark alert as read |
| GET | `/alerts/stream` | Yes | SSE — real-time alert stream |
| GET | `/keys` | Yes | List API key status (has_key booleans) |
| POST | `/keys` | Yes | Store encrypted API key |
| DELETE | `/keys?provider=` | Yes | Remove API key |
| GET | `/metrics` | No | JSON metrics counters |

## Design Decisions

**Go channels instead of Kafka/Redis** — The event throughput (2 sources, ~5 events/minute) doesn't justify external messaging infrastructure. A buffered Go channel provides the same fan-in/fan-out pattern with zero operational overhead. The `NormalizedEvent` struct acts as the contract between producers and consumers.

**Qdrant for vector storage** — Purpose-built for filtered vector search. Supports combining cosine similarity with metadata filters (source, tickers) in a single query. The gRPC client provides efficient communication compared to REST-based alternatives.

**OpenAI embeddings for semantic matching** — `text-embedding-3-small` provides 1536-dimensional vectors that capture semantic meaning. This enables matching events to strategies based on conceptual similarity rather than keyword overlap. A user strategy like "earnings surprises in semiconductor companies" will match articles about unexpected chip revenue without requiring those exact words.

**Deterministic UUID v5 for point IDs** — `uuid.NewSHA1(uuid.Nil, []byte(event.ID))` ensures the same event always maps to the same Qdrant point ID, making upserts idempotent across restarts.

**MultiNotifier pattern** — Fan-out notifications to multiple backends (log, database, SSE) through a single `Notifier` interface. Adding a new delivery channel (email, Slack) requires implementing one method.

## Project Structure

```
cmd/signalstream/        # Application entry point and worker orchestration
internal/
  alerter/               # Strategy evaluation engine with vector search
  api/                   # HTTP server, routes, JWT middleware, SSE handler
  auth/                  # JWT tokens, bcrypt passwords, AES-GCM encryption
  config/                # Environment-based configuration with validation
  database/              # PostgreSQL repository layer, embedded migrations
  embedding/             # Shared Embedder interface + OpenAI implementation
  enrichment/            # Article fetching, embedding, Qdrant upsert pipeline
  ingestion/             # Polygon.io and SEC EDGAR polling producers
  metrics/               # Atomic counters and /metrics HTTP handler
  notification/          # Notifier interface, log/database/multi implementations
  platform/              # NormalizedEvent — canonical event struct
  retry/                 # Exponential backoff with jitter, context-aware
  sec/                   # SEC CIK-to-ticker mapping (singleton, thread-safe)
  sse/                   # Server-Sent Events broker (pub/sub, per-user channels)
frontend/                # React + TypeScript dashboard
tools/qdrant-inspector/  # Standalone Qdrant collection diagnostic CLI
```
