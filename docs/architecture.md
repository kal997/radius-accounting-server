# RADIUS Accounting Server - Architecture & Design

## Table of Contents
- [Overview](#overview)
- [System Architecture](#system-architecture)
- [Component Design](#component-design)
- [Interface Design](#interface-design)
- [Data Flow](#data-flow)
- [Sequence Diagrams](#sequence-diagrams)
- [Deployment Architecture](#deployment-architecture)
- [Design Decisions](#design-decisions)
- [Technical Considerations](#technical-considerations)
- [Conclusion](#conclusion)

## Overview

The RADIUS Accounting Server is a monitoring and data collection system designed to receive mirrored RADIUS accounting traffic, store it for analysis, and provide real-time notifications of accounting events. The system follows a microservices architecture with clear separation of concerns and database-agnostic interfaces.

### Key Requirements Addressed
- Process RADIUS Accounting-Request packets (RFC 2866)
- Store accounting data in Redis with configurable TTL
- Real-time notification system using Redis keyspace events
- Persistent logging of all accounting activities
- Containerized deployment with Docker Compose
- Testing capabilities with radclient integration

## System Architecture

### High-Level Architecture

![High-Level Architecture](./diagrams/architecture.png)

The system consists of four main containerized services that work together to process, store, and log RADIUS accounting events. The architecture follows a microservices pattern with clear separation between the RADIUS protocol handler, data storage, event notification, and testing components.

### Component Overview

1. **radius-controlplane**: Receives and processes RADIUS packets
2. **redis**: Provides storage and pub/sub messaging
3. **redis-controlplane-logger**: Subscribes to events and logs them
4. **radclient-test**: Testing and simulation container

## Component Design

![Component Diagram](./diagrams/components.png)

### radius-controlplane

**Responsibilities:**

- Listen on UDP port 1813 for RADIUS Accounting packets.
- Validate packet integrity using a shared secret.
- Parse and validate the packet into a strongly‑typed accounting event.
- Store accounting events in Redis with a TTL.
- Send RADIUS Accounting‑Response packets.

**Key Components:**

- **RADIUS Handler**: Packet processing logic.
- **Storage Interface**: Database abstraction layer.
- **Redis Client**: Concrete storage implementation.

**Implementation Details:**

- Uses the `layeh.com/radius` library for RADIUS protocol handling.
- Implements graceful shutdown via context cancellation.
- Always sends Accounting‑Response even if storage fails (protocol compliance).

### redis‑controlplane‑logger

**Responsibilities:**

- Subscribe to Redis keyspace notifications.
- Convert Redis events to application events.
- Log accounting activities to persistent storage.
- Handle connection failures gracefully.

**Key Components:**

- **Subscriber Handler**: Event processing coordination.
- **Notifier Interface**: Event subscription abstraction.
- **Log Interface**: Logging abstraction.
- **File Logger**: Persistent file‑based logging.

**Implementation Details:**
- Subscribes to pattern `radius:acct:*` via Redis keyspace notifications
- Uses buffered channels (100 events) for backpressure control
- Thread-safe file writing with mutex locks
- Syncs writes to disk for durability

### Architecture Patterns Used

- **Interface Segregation**: Small, focused interfaces (Storage, Notifier, Logger).
- **Dependency Injection**: Interfaces injected into components.
- **Repository Pattern**: Storage interface abstracts data access.
- **Publisher‑Subscriber**: Redis keyspace notifications for event distribution.
- **Graceful Degradation**: System continues operation despite storage failures.

## Interface Design

### Storage Interface (Database Abstraction)

```go
type Storage interface {
    // Store saves an accounting event.
    Store(ctx context.Context, event AccountingEvent) error
    // HealthCheck verifies storage connectivity.
    HealthCheck(ctx context.Context) error
    // Close closes the storage connection.
    Close() error
}
```

**Design Rationale:**

- **Database Independence**: Can swap Redis for PostgreSQL, MongoDB, etc.
- **Context Support**: Enables timeouts and cancellation.
- **Health Monitoring**: Supports service health checks.
- **Resource Management**: Proper cleanup with `Close()`.

**Current Implementation**: `RedisStorage` in `internal/storage/redis.go`

### Notifier Interface (Event System)

```go
type Notifier interface {
    Subscribe(ctx context.Context, patterns []string) (<-chan StorageEvent, error)
    Unsubscribe(patterns []string) error
    HealthCheck(ctx context.Context) error
    Close() error
}

type StorageEvent struct {
    Key       string
    Operation string
    Timestamp time.Time
}
```

**Design Rationale:**
- **Generic Events**: Not tied to Redis-specific messages
- **Rich Context**: Events include key, operation type, and timestamp
- **Channel-Based**: Non-blocking, concurrent event processing
- **Pattern Flexibility**: Support multiple subscription patterns

**Current Implementation**: `RedisNotifier` in `internal/notifier/redis.go`

### Logger Interface (Output Abstraction)

```go
type Logger interface {
    Log(ctx context.Context, message string) error
    Close() error
}
```

**Design Rationale:**

- **Output Independence**: File, syslog, or remote logging.
- **Context Support**: Enables cancellation and timeouts.
- **Simple API**: Easy to implement and test.

**Current Implementation**: `FileLogger` in `internal/logger/file.go`.

### Accounting Data Model

RADIUS accounting supports multiple record types.  Instead of a single `AccountingRecord` struct, the server defines a small interface and a set of concrete record types.

- **AccRecordType enumeration**: identifies the type of accounting event.

  ```go
  type AccRecordType int

  const (
      Start  AccRecordType = iota + 1 // RADIUS uses 1,2,3
      Stop
      Interim
  )
  ```

- **AccountingEvent interface**:

  ```go
  type AccountingEvent interface {
      Validate() error
      GenerateRedisKey() string
      GetType() AccRecordType
  }
  ```

- **BaseAccountingRecord** holds attributes common to all events:

  ```go
  type BaseAccountingRecord struct {
      // The username from the RADIUS request (User‑Name attribute)
      Username string `json:"username"`
      // The IP address of the Network Access Server (NAS‑IP‑Address attribute)
      NASIPAddress string `json:"nas_ip_address"`
      // The port number on the NAS (NAS‑Port attribute)
      NASPort int `json:"nas_port"`
      // Unique identifier for the session (Acct‑Session‑Id attribute)
      AcctSessionID string `json:"acct_session_id"`
      // The caller's identifier (Calling‑Station‑Id attribute)
      CallingStationID string `json:"calling_station_id"`
      // The called party's identifier (Called‑Station‑Id attribute)
      CalledStationID string `json:"called_station_id"`
      // The IP address of the client making the request
      ClientIP string `json:"client_ip"`
      // When the accounting request was received
      Timestamp string `json:"timestamp"`
  }
  ```

- **StartRecord** extends `BaseAccountingRecord` with `FramedIPAddress`.
- **StopRecord** extends `BaseAccountingRecord` with `SessionTime`, `TerminateCause`, `InputOctets`, and `OutputOctets`.
- **InterimRecord** extends `BaseAccountingRecord` with `SessionTime`, `InputOctets`, and `OutputOctets`.

Each concrete type implements `Validate()`, `GetType()` and `GenerateRedisKey()`.  The `GenerateRedisKey()` method prepends a prefix based on the event type to a key composed of the username, session ID and timestamp.  For example:

- `StartRecord` keys look like `radius:acct:{username} {acct_session_id}:{timestamp}:start`.
- `StopRecord` keys look like `radius:acct:{username} {acct_session_id}:{timestamp}:stop`.
- `InterimRecord` keys look like `radius:acct:{username} {acct_session_id}:{timestamp}:interim`.

## Data Flow

![Data Flow Diagram](./diagrams/dataflow.png)

### RADIUS Packet Processing Flow

1. **Packet Reception**: UDP packet arrives on port 1813.
2. **Validation**: Verify the shared secret and packet integrity (handled by `layeh/radius`).
3. **Parsing & Validation**: Parse the packet to an `AccountingEvent` using `ParseRADIUSPacket()` and validate the resulting `StartRecord`, `StopRecord` or `InterimRecord`.
4. **Storage**: Store the event in Redis with its generated key and a TTL.
5. **Response**: Send a RADIUS Accounting‑Response regardless of storage success.
6. **Notification**: Redis auto‑publishes a keyspace notification when the key is set.

### Event Processing Flow

1. **Keyspace Notification**: Redis publishes `__keyspace@0__:radius:acct:*`
2. **Event Conversion**: Notifier converts to `StorageEvent`
3. **Event Distribution**: Event sent via channel to subscriber
4. **Logging**: Subscriber formats and logs event to file

## Sequence Diagrams

### System‑Level Packet Processing

![Sequence Diagram](./diagrams/sequence.png)

This diagram illustrates the end‑to‑end flow from packet arrival to event logging, including error handling.

### Logger Service – Normal Operation

![Logger Normal Operation](./diagrams/logger_normal_operation.png)

Shows the initialization sequence and normal event processing loop for the subscriber service.

### Logger Service – Graceful Shutdown

![Logger Shutdown Sequence](./diagrams/logger_shutdown_sequence.png)

Demonstrates how the logger service handles `SIGINT`/`SIGTERM` signals and performs clean resource cleanup.

## Deployment Architecture

![Deployment Diagram](./diagrams/simple_deployment.png)

The deployment uses Docker Compose to orchestrate four containers on a single Docker host:

- **radius‑controlplane**: Exposes UDP port 1813 and connects to Redis.
- **redis**: Redis 7 Alpine with keyspace notifications enabled.
- **redis‑controlplane‑logger**: Connects to Redis and writes to a volume‑mounted log file.
- **radclient‑test**: Testing container with freeradius‑utils.

All containers communicate via a custom bridge network (`radius‑network`). Redis data persists via a named volume, and logs persist via a host volume mount.

## Design Decisions

### 1. Database‑Agnostic Interfaces

**Decision**: Use generic interfaces instead of Redis‑specific APIs.

**Rationale:**

- **Flexibility**: Easy to change storage backends (PostgreSQL, MongoDB).
- **Testability**: Mock interfaces for unit testing without Redis.
- **Maintenance**: Easier to update implementations independently.
- **Reusability**: Interfaces can be used in other projects.

**Trade‑off**: Slight performance overhead vs. flexibility and maintainability.

### 2. Channel‑Based Event System

**Decision**: Return channels from `Subscribe()` instead of callbacks.

**Rationale:**

- **Control Flow**: Caller controls the processing loop and can break at any time.
- **Concurrency**: Natural Go concurrency patterns with `select` statements.
- **Backpressure**: Buffered channels provide flow control (100‑event buffer).
- **Cancellation**: Context cancellation works naturally with channel selects.

**Trade‑off**: More complex implementation vs. better control and Go idiomatic patterns.

### 3. Always Send RADIUS Response

**Decision**: Send an Accounting‑Response even if Redis storage fails.

**Rationale:**

- **RADIUS Protocol Compliance**: RFC 2866 requires responses.
- **Prevents Client Timeouts**: NAS devices expect responses.
- **Separation of Concerns**: Storage failures shouldn't block protocol flow.
- **Graceful Degradation**: System remains operational even with storage issues.

**Implementation**: Error handling logs storage failures but doesn't prevent response transmission.

### 4. Separate Logger Service

**Decision**: Create an independent subscriber service instead of logging in the main server.

**Rationale:**

- **Separation of Concerns**: RADIUS server focuses on protocol; logger focuses on persistence.
- **Independent Scaling**: Can scale the logger separately if log volume increases.
- **Fault Isolation**: Logger crashes don't affect RADIUS packet processing.
- **Flexibility**: Can add multiple subscribers for different purposes (metrics, analytics, etc.).

**Trade‑off**: Additional service complexity vs. better modularity.

## Technical Considerations

### Error Handling

- **Storage Failures**: Logged but don't block RADIUS responses.
- **Redis Connection Failures**: Handled gracefully with health checks.
- **Context Cancellation**: Clean shutdown for all goroutines.
- **Parse Errors**: Invalid packets logged with details.
- **Validation Errors**: Rejected packets logged with reasons.

### Performance

- **Buffered Channels**: 100‑event buffer prevents blocking on slow consumers.
- **Single Goroutine per Subscription**: Minimal resource usage.
- **Redis Context Timeouts**: 5‑second timeout prevents hanging operations.
- **Connection Pooling**: Redis client uses a connection pool internally.

### Concurrency Safety

- **Thread‑Safe Logging**: Mutex protects file writes.
- **Race Detection**: All tests run with the `-race` flag.
- **Channel Ownership**: Clear ownership prevents data races.
- **Context Propagation**: Proper context usage throughout.

### Security

- **RADIUS Shared Secret**: Minimum 8 characters, validated at startup.
- **Input Validation**: All extracted attributes validated before storage.
- **No Secrets in Logs**: Sensitive data not logged.

### Observability

- **Structured Logging**: Consistent log format with timestamps.
- **Health Checks**: Both services implement health check methods.
- **Connection Status**: Redis connectivity verified at startup.
- **Storage Confirmation**: Successful storage logged with the Redis key.

## Conclusion

This architecture provides the foundation for a production‑ready RADIUS accounting system with the following key benefits:

1. **Modularity**: Clear separation of concerns with well‑defined interfaces.
2. **Flexibility**: Database‑agnostic design enables easy technology swaps.
3. **Reliability**: Robust error handling and graceful degradation.
4. **Maintainability**: Clean code structure with comprehensive testing.
5. **Scalability**: Stateless design allows horizontal scaling.
6. **Observability**: Comprehensive logging and health checks.

The interface‑based architecture makes it straightforward to extend the system with additional storage backends, notification channels, or logging destinations without modifying core business logic.