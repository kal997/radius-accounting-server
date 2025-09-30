# RADIUS Accounting System v1

A production-ready RADIUS accounting server implementation in Go that processes RADIUS accounting packets, stores them in Redis, and provides real-time event notifications through a subscriber service.

> **Version**: 1.0.0  
> **Status**: Production Ready  
> **Purpose**: Monitor and collect mirrored RADIUS accounting traffic for analysis

## Overview

This system implements a complete RADIUS accounting solution following RFC 2866, designed to receive mirrored/copied RADIUS traffic for monitoring purposes (not performing authentication). It consists of three main services:

- **radius-controlplane**: RADIUS server that processes accounting packets
- **redis-controlplane-logger**: Event subscriber that logs all accounting activities
- **redis**: Storage backend with keyspace notifications enabled
- **radclient-test**: Test Container with freeradius-utils and out of the box sample tests


## Features

### Core Functionality
- RADIUS Accounting protocol (RFC 2866) support
- Processes Start, Stop, and Interim-Update accounting types
- Shared secret authentication for packet verification
- Redis storage with configurable TTL
- Real-time event notifications via Redis keyspace notifications
- Persistent logging of all accounting events

### Technical Features
- Database-agnostic storage interface
- Generic event notification system
- Graceful shutdown handling
- Health check endpoints
- Comprehensive error handling
- Thread-safe logging
- Docker containerization
- Docker Compose orchestration

## Architecture

![High-Level Architecture](./docs/diagrams/architecture.png)

For detailed architecture diagrams, see [docs/architecture.md](docs/architecture.md).

## Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.22+ (for local development)
- Bash shell

### Running the System

1. **Create environment configuration**:
```bash
cp .env.example .env
# Edit .env with your configuration
```

Required environment variables:
```bash
RADIUS_SHARED_SECRET=your-secret-key-minimum-8-chars
REDIS_HOST=redis
RECORD_TTL_HOURS=24
LOG_LEVEL=info
LOG_FILE=./radius_accounting.log
LOG_FILE_CONTAINER=/var/log/radius_updates.log
```

2. **Start all services**:
```bash
./run.sh complete
```

3. **Interactive testing** (start services + radclient shell):
```bash
./run.sh full
```

4. **Stop all services**:
```bash
./run.sh down
```

## Testing

### Using radclient (Recommended)

The system includes pre-configured test files in the `examples/` directory.

**Option 1: Entire Solution**
```bash
./run.sh complete
```

**Option 2: Interactive Shell**
```bash
./run.sh full
# Inside the container:
./main.sh  # Sends both start and stop requests
```

**Option 3: Manual Testing**
```bash
# Start services, this will start the radius-controlplanner, radius-controlplane-logger, and redis services
./run.sh full

# Send Accounting-Start in the radclient interactive shell
radclient -x $RADIUS_SERVER:1813 acct $RADIUS_SHARED_SECRET < /requests/acct_start.txt
# Send Accounting-Stop in the radclient interactive shell
radclient -x $RADIUS_SERVER:1813 acct $RADIUS_SHARED_SECRET < /requests/acct_stop.txt
```

### Verifying Results

1. **Check RADIUS server logs**:
```bash
docker logs radius-controlplane
```

Expected output:
```
Stored accounting record: radius:acct:testuser:session12345:2024-01-15T10:30:45Z
```

2. **Check Redis data**:
```bash
docker exec -it redis redis-cli
> KEYS radius:acct:*
> GET radius:acct:testuser:session12345:2024-01-15T10:30:45Z
```

3. **Check subscriber logs**:
```bash
cat radius_accounting.log
```

Expected format:
```
2024-01-15 10:30:45.123456 - Received update for key: radius:acct:testuser:session12345:2024-01-15T10:30:45Z, Operation: set
```

## Project Structure

```
.
├── cmd/
│   ├── radclient-test/          # Test container with freeradius-utils
│   ├── radius-controlplane/     # Main RADIUS server
│   └── radius-controlplane-logger/ # Event subscriber service
├── internal/
│   ├── config/                  # Configuration management
│   ├── logger/                  # logging interface and file logging implementation
│   ├── models/                  # Data models (AccountingRecord)
│   ├── notifier/                # Event notification interface and redis implementation
│   └── storage/                 # Storage interface and Redis implementation
├── examples/                    # Sample RADIUS packets
├── docs/                        # Architecture documentation
├── docker-compose.yml           # Container orchestration
└── run.sh                       # Convenience script
```

## Data Model

### Accounting Record Structure
```json
{
  "username": "testuser",
  "nas_ip_address": "192.168.1.1",
  "nas_port": 0,
  "acct_status_type": 1,
  "acct_session_id": "session12345",
  "framed_ip_address": "10.0.0.100",
  "calling_station_id": "123-456-7890",
  "called_station_id": "987-654-3210",
  "timestamp": "2024-01-15T10:30:45.123456789Z",
  "client_ip": "192.168.1.100",
  "packet_type": "Accounting-Request"
}
```

### Redis Key Format
```
radius:acct:{username}:{acct_session_id}:{timestamp}
```

Example:
```
radius:acct:testuser:session12345:2024-01-15T10:30:45.123456789Z
```

## Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `RADIUS_SHARED_SECRET` | RADIUS shared secret (min 8 chars) | - | Yes |
| `REDIS_HOST` | Redis hostname | - | Yes |
| `RECORD_TTL_HOURS` | TTL for Redis records in hours | - | Yes |
| `LOG_LEVEL` | Logging level (debug/info/warn/error) | - | Yes |
| `LOG_FILE` | Host path for log file | - | Yes |
| `LOG_FILE_CONTAINER` | Container path for log file | - | Yes |
| `ENV` | Environment (prod/dev) | dev | No |

### Redis Configuration

Redis is configured with keyspace notifications enabled:
```bash
notify-keyspace-events KEA
```

This enables the subscriber service to receive real-time notifications when keys are created, modified, or expired.

## Development

### Running Locally (Without Docker)

1. **Start Redis**:
```bash
redis-server --notify-keyspace-events KEA
```

2. **Create .env file**:
```bash
RADIUS_SHARED_SECRET=testing123
REDIS_HOST=localhost
RECORD_TTL_HOURS=24
LOG_LEVEL=debug
LOG_FILE=./radius_accounting.log
```

3. **Run RADIUS server**:
```bash
go run cmd/radius-controlplane/main.go
```

4. **Run subscriber** (in another terminal):
```bash
go run cmd/radius-controlplane-logger/main.go
```

5. **Send test packets**:
```bash
radclient -x localhost:1813 acct testing123 < examples/acct_start.txt
```

### Running Tests

The project includes comprehensive unit and integration tests with >80% code coverage.

**Quick test commands via run.sh:**

```bash
# Run all tests (unit + integration + Overall Coverage)
./run.sh test

# Run only unit tests (fast, no dependencies)
./run.sh test-unit

# Run only integration tests (requires Redis)
./run.sh test-integration
```

**Manual test execution:**

```bash
# Unit tests (no Redis required)
go test -v -race ./internal/config
go test -v -race ./internal/models
go test -v -race ./internal/notifier

# Integration tests (requires Redis running)
docker compose up -d redis
go test -v -race ./internal/storage
go test -v -race ./internal/logger
```

**Test Coverage:**

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./internal/...
go tool cover -html=coverage.out
```

**Test Packages:**
- `internal/config`: Configuration loading and validation tests
- `internal/models`: Accounting record parsing and validation tests
- `internal/notifier`: Redis keyspace notification parsing tests
- `internal/storage`: Redis storage integration tests
- `internal/logger`: File logging integration tests

All tests include race detection (`-race` flag) to catch concurrency issues.

## Design Decisions

### Database-Agnostic Interfaces
The system uses interface-based design for storage and notifications, making it easy to swap Redis for other databases (PostgreSQL, MongoDB, etc.) without changing business logic.

### Always Send RADIUS Response
The server always sends an Accounting-Response, even if Redis storage fails. This ensures RADIUS protocol compliance and prevents client timeouts.

### Channel-Based Event System
The notification system uses Go channels for event distribution, providing natural concurrency patterns and backpressure control.

### Graceful Shutdown
Both services handle SIGINT/SIGTERM signals gracefully, ensuring proper cleanup of Redis connections and file handles through context propagation.

## Performance Considerations

- **Buffered Channels**: Event channels use 100-item buffers to prevent blocking
- **Redis TTL**: Automatic expiration prevents unbounded storage growth
- **File Sync**: Log writes are synced to disk for durability

## Security Notes

- RADIUS shared secret must be at least 8 characters
- All RADIUS packets are validated using the shared secret

## Known Limitations (v1)

- File-based logging only (no remote logging, but can be extended using logging interface)
- Limited RADIUS attributes extracted
- Basic error recovery (no retry logic)
- Test covarage < 80% (currently 55.2%)

## References

- [RFC 2866 - RADIUS Accounting](https://tools.ietf.org/html/rfc2866)
- [layeh.com/radius - Go RADIUS Library](https://github.com/layeh/radius)
- [Redis Keyspace Notifications](https://redis.io/docs/latest/develop/pubsub/keyspace-notifications/)
- [FreeRADIUS radclient Documentation](https://freeradius.org/radiusd/man/radclient.html)


## Support

For issues and questions, please refer to the documentation in the `docs/` directory or contact [me](mailto:khaled.soliman97@gmail.com) directly.