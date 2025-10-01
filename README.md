# MIT Service

Go service with inbox pattern for async CRUD operations. Resource limits: 2 CPU cores, 2GB RAM.

## Quick Start

```bash
# 1. Build and run
make docker-build
make docker-run

# 2. Test service  
curl http://localhost:8080/health

# 3. Load testing
pip3 install aiohttp
python3 scripts/load_test.py --rps 50 --tasks 100
```

## API Endpoints

- `POST /insert` - Create record (async)
- `POST /update` - Update record (async)  
- `POST /delete` - Delete record (async)
- `GET /get?id=<id>` - Get record (sync)
- `GET /health` - Health check
- `GET /metrics` - Performance metrics
- `GET /stats` - Task statistics

## Load Testing

```bash
# Basic test (5 RPS, 10 tasks)
python3 scripts/load_test.py --rps 5 --tasks 10

# Medium load (50 RPS, 200 tasks)  
python3 scripts/load_test.py --rps 50 --tasks 200

# High load (200 RPS, 500 tasks)
python3 scripts/load_test.py --rps 200 --tasks 500 --timeout 30
```

Each task: INSERT → UPDATE → GET  
ID = MD5(abcdefg + task_number)

## Monitoring

```bash
# Resource usage
docker stats --no-stream

# Service metrics
curl http://localhost:8080/performance
curl http://localhost:8080/stats
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `REPOSITORY_TYPE` | `postgres` | Repository type (`postgres`/`mock`) |
| `DB_HOST` | `postgres-main` | Main PostgreSQL host |
| `INBOX_DB_HOST` | `postgres-inbox` | Inbox PostgreSQL host |
| `INBOX_DB_PORT` | `5433` | Inbox PostgreSQL port |
| `INBOX_WORKER_COUNT` | `5` | Number of inbox workers |
| `INBOX_BATCH_SIZE` | `10` | Task batch size |

**Two separate databases:**
- `postgres-main:5432` - Business records (`mitservice` database)  
- `postgres-inbox:5433` - Inbox tasks (`mitservice_inbox` database)

## Example Usage

```bash
# Insert record
curl -X POST http://localhost:8080/insert \
  -H "Content-Type: application/json" \
  -d '{"id": "user_123", "value": {"name": "John", "age": 30}}'

# Get record
curl "http://localhost:8080/get?id=user_123"

# Check stats
curl "http://localhost:8080/stats"
```