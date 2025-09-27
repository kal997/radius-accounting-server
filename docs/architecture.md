# RADIUS Accounting Server - Architecture & Design

## Table of Contents
- [Overview](#overview)
- [System Architecture](#system-architecture)
- [Component Design](#component-design)
- [Interface Design](#interface-design)
- [Data Flow](#data-flow)
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
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   RADIUS        │    │     Redis       │    │   Subscriber    │
│  Accounting     │───▶│   Storage &     │───▶│    Service      │
│   Server        │    │   Pub/Sub       │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
    UDP Packets              Key-Value Store         Log Files
```

### Component Overview
1. **radius-controlplane**: Receives and processes RADIUS packets
2. **redis**: Provides storage and pub/sub messaging
3. **redis-controlplane-logger**: Subscribes to events and logs them
4. **radclient-test**: Testing and simulation container

## Component Design

### radius-controlplane
**Responsibilities:**
- Listen on UDP port 1813 for RADIUS Accounting packets
- Validate packet integrity using shared secret
- Extract accounting attributes using layeh/radius library
- Store accounting records in Redis with TTL
- Send RADIUS Accounting-Response packets

**Key Components:**
- RADIUS Handler: Packet processing logic
- Storage Interface: Database abstraction layer
- Redis Client: Concrete storage implementation

### redis-controlplane-logger
**Responsibilities:**
- Subscribe to Redis keyspace notifications
- Convert Redis events to application events
- Log accounting activities to persistent storage
- Handle connection failures gracefully

**Key Components:**
- Subscriber Handler: Event processing coordination
- Notifier Interface: Event subscription abstraction
- Log Interface: Logging abstraction
- File Logger: Persistent file-based logging

### Architecture Patterns Used
- **Interface Segregation**: Small, focused interfaces
- **Dependency Injection**: Interfaces injected into components
- **Repository Pattern**: Storage interface abstracts data access

## Interface Design

### Storage Interface (Database Abstraction)
```go
type Storage interface {
    Store(ctx context.Context, record *AccountingRecord) error
    HealthCheck(ctx context.Context) error
    Close() error
}
```

**Design Rationale:**
- **Database Independence**: Can swap Redis for PostgreSQL, MongoDB, etc.
- **Context Support**: Enables timeouts and cancellation
- **Health Monitoring**: Supports service health checks
- **Resource Management**: Proper cleanup with Close()

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
    EventType EventType
    Timestamp time.Time
    Metadata  map[string]string
}
```

**Design Rationale:**
- **Generic Events**: Not tied to Redis-specific messages
- **Rich Context**: Events include type, timestamp, and metadata
- **Channel-Based**: Non-blocking, concurrent event processing
- **Pattern Flexibility**: Support multiple subscription patterns

### Log Interface (Output Abstraction)
```go
type Log interface {
    Log(ctx context.Context, message string) error
    Close() error
}
```

**Design Rationale:**
- **Output Independence**: File, syslog, or remote logging
- **Context Support**: Enables cancellation and timeouts
- **Simple API**: Easy to implement and test

## Data Flow

### RADIUS Packet Processing Flow
1. **Packet Reception**: UDP packet arrives on port 1813
2. **Validation**: Verify shared secret and packet integrity
3. **Attribute Extraction**: Parse RADIUS attributes to AccountingRecord
4. **Storage**: Store record in Redis with generated key and TTL
5. **Response**: Send RADIUS Accounting-Response
6. **Notification**: Redis auto-publishes keyspace notification

### Event Processing Flow
1. **Keyspace Notification**: Redis publishes "__keyspace@0__:radius:acct:*"
2. **Event Conversion**: Notifier converts to StorageEvent
3. **Event Distribution**: Event sent via channel to subscriber
4. **Logging**: Subscriber formats and logs event to file

### Data Structures
```go
type AccountingRecord struct {
    Username         string    `json:"username"`
    NASIPAddress     string    `json:"nas_ip_address"`
    AcctStatusType   string    `json:"acct_status_type"`
    AcctSessionID    string    `json:"acct_session_id"`
    // ... other RADIUS attributes
    Timestamp        time.Time `json:"timestamp"`
    ClientIP         string    `json:"client_ip"`
}
```

## Design Decisions

### 1. Database-Agnostic Interfaces
**Decision**: Use generic interfaces instead of Redis-specific APIs

**Rationale:**
- **Flexibility**: Easy to change storage backends
- **Testability**: Mock interfaces for unit testing
- **Maintenance**: Easier to update implementations
- **Reusability**: Interfaces can be used in other projects

**Trade-off**: Slight performance overhead vs. flexibility

### 2. Channel-Based Event System
**Decision**: Return channels from Subscribe() instead of callbacks

**Rationale:**
- **Control Flow**: Caller controls processing loop
- **Concurrency**: Natural Go concurrency patterns
- **Backpressure**: Buffered channels provide flow control
- **Cancellation**: Context cancellation works naturally

**Trade-off**: More complex implementation vs. better control

### 3. Always Send RADIUS Response
**Decision**: Send response even if Redis storage fails

**Rationale:**
- RADIUS protocol compliance
- Prevents client timeouts
- Storage failures shouldn't block protocol flow

## Technical Considerations

### Error Handling
- Storage failures are logged but don't block responses
- Redis connection failures handled gracefully
- Context cancellation for clean shutdown

### Performance
- Buffered channels (100 events) prevent blocking
- Single goroutine per subscription pattern
- Redis operations use context timeouts

### Security
- RADIUS shared secret validation
- Input validation on all attributes

## Conclusion

This architecture provides the foundation for a RADIUS accounting system with the following key benefits:

1. **Modularity**: Clear separation of concerns with well-defined interfaces
2. **Flexibility**: Database-agnostic design enables easy technology swaps
3. **Reliability**: Robust error handling and graceful degradation
4. **Maintainability**: Clean code structure with comprehensive testing

