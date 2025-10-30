# Coyote - Code Execution Service

A microservice for executing code submissions in isolated Docker containers. It listens to a Redis queue for submission requests, runs code against test cases, and stores results in PostgreSQL.

## How It Works

1. **Receives Submissions**: Listens to a Redis pub/sub channel for code submission messages
2. **Executes Code**: Runs submissions in isolated Docker containers with resource limits
3. **Returns Results**: Stores execution results (verdict, stdout, stderr) in PostgreSQL

## Supported Languages

- C (compiled with gcc, C17 standard)
- Python 3

## Configuration

Configure via YAML file (see `config/` directory):

```yaml
env: "dev" # local, dev or prod

http:
  port: "7197" # port for accepting HTTP requests (purely for healthchecking now)

postgres:
  host: "localhost"
  port: "5432"
  user: "postgres"
  name: "postgres"
  password: "password"
  sslmode: "disable"

redis:
  addr: "localhost:6379"
  password: "password"
  db: 0

runner:
  docker_image: "ghcr.io/voidcontests/runner" # runner image, to isolate submisisons execution
  channel: "submissions" # channel where to listen for submissions
```

## Submission Format

The service expects submission messages from Redis with the following structure:

```go
type Submission struct {
    ID        int
    EntryID   int
    ProblemID int
    Code      string
    Language  string
}
```

## Execution

Each submission:
- Runs in an isolated Docker container with limits:
  - 0.5 CPU cores
  - 128MB memory
  - 50 process limit
  - No network access
- Has a configurable time limit per test case
- Executes against multiple test cases stored in the database

## Response

Results are stored in the database with:
- `verdict`: Execution verdict (AC, WA, TLE, RE, CE, etc.)
- `stdout`: Program output
- `stderr`: Error messages
- `passed_tests_count`: Number of test cases passed
- `answer`: Actual output for failed tests

## Running

### Native

```bash
# Build
go build -o coyote cmd/coyote/main.go

# Run with config
CONFIG_PATH=./config/dev.yaml ./coyote
```

### Docker

```bash
# Build image
docker build -t coyote .

# Run container
docker compose up -d
```

**Note**: The service requires access to the Docker socket (`/var/run/docker.sock`) to run code in isolated containers.

## Healthcheck

HTTP endpoint available at `/api/healthcheck` for service health monitoring.

```bash
curl -i localhost:7197/api/healthcheck
```
