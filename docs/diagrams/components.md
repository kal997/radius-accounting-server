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
  - Store(ctx, record) error
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