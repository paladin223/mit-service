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
python3 -m venv venv
source venv/bin/activate
pip3 install -r scripts/requirements.txt
python scripts/populate_db.py --rps 100 --duration 600 --start 1220000
python scripts/read_load_test.py --rps 100 --start-id 1220000

# Статистика по памяти и загрузке процессора
docker stats --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}"
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
# Basic test (5 RPS, 10 tasks, max 60s)
python3 scripts/load_test.py --rps 5 --tasks 10 --max-duration 60

# Medium load (50 RPS, 200 tasks, max 120s)  
python3 scripts/load_test.py --rps 50 --tasks 200 --max-duration 120

# High load (200 RPS, 500 tasks, max 300s)
python3 scripts/load_test.py --rps 200 --tasks 500 --max-duration 300
```

Each task: INSERT → UPDATE → GET  
INSERT/UPDATE: ID = MD5(abcdefg + (1000000 + task_number))  
GET: ID = MD5(abcdefg + (1 + task_number % 100000))

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