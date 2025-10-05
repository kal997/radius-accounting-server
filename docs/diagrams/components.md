# RADIUS Accounting Server - Component Design

## Component Diagram
```plantuml
@startuml components
package "radius-controlplane " {
    [radius-controlplane] --> [Storage Interface]
    [Storage Interface] <|-- [Redis Client]
}

package "redis-controlplane-logger " {
    [radius-controlplane-logger] --> [Notifier Interface]
    [radius-controlplane-logger] --> [Log Interface]
    [Notifier Interface] <|-- [Redis Subscriber]
    [Log Interface] <|-- [File Logger]
}

package "radclient-test" {
    [freeradius-utils]
    [Test Files] --> [freeradius-utils]
}

package "redis" {
    database "Redis DB" {
        [Key-Value Store]
        [Pub/Sub Engine]
    }
}

[freeradius-utils] --> [radius-controlplane] : UDP 1813
[Redis Client] --> [Redis DB] : TCP (Store Only)
[Redis Subscriber] --> [Redis DB] : TCP (Subscribe Only)

note right of [Storage Interface]
  Database-agnostic storage
  - Store(ctx, event) error
  - HealthCheck(ctx) error
  - Close() error
end note

note right of [Notifier Interface]
  Generic event notifications
  - Subscribe(ctx, patterns) (<-chan StorageEvent, error)
  - Unsubscribe(patterns) error
  - HealthCheck(ctx) error
  - Close() error
end note

note right of [Log Interface]
  Interface for logging
  - Log(ctx, message) error
  - Close() error
end note

note right of [radius-controlplane]
  Receives accounting packets
  Flow: receive -> extract -> store
  Redis auto-publishes keyspace events
end note

@enduml
```
This component diagram reflects the current (v2) implementation of the RADIUS accounting system. The `radius-controlplane` service parses incoming packets into strongly typed `AccountingEvent` objects and stores them via the `Storage` interface. Note that the `Store` method now accepts an `event` rather than a (v1)`record` and that keys generated in Redis are postfixed with `start:`, `stop:` or `interim:` depending on the accounting event type.