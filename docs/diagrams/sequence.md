# Sequence Diagram
```plantuml
@startuml sequence
actor "RADIUS Client" as client
participant "radius-controlplane" as radius
participant "Storage Interface" as storage
participant "Redis" as redis
participant "Redis Keyspace" as keyspace
participant "redis-controlplane-logger" as sub
participant "Notifier Interface" as notifier
participant "File Logger" as logger

client -> radius: RADIUS Accounting Request (UDP 1813)
activate radius

radius -> radius: Extract attributes
note right: Username, Session-ID,\nNAS-IP, etc.\nCreate AccountingRecord

radius -> storage: Store(ctx, record)
activate storage

storage -> redis: SET radius:acct:user:session:timestamp (with TTL)
activate redis

alt Redis available
    redis -> storage: OK
    redis -> keyspace: Auto-publish keyspace notification
    activate keyspace
    keyspace -> notifier: __keyspace@0__:radius:acct:*
    activate notifier
    notifier -> notifier: Convert to StorageEvent
    note right: Key: "radius:acct:user:session:timestamp"\nEventType: EventCreated\nTimestamp: time.Now()\nMetadata: {"source": "redis", "operation": "set"}
    notifier -> sub: StorageEvent via channel
    activate sub
    sub -> logger: Log(ctx, formatted_message)
    activate logger
    logger -> logger: Write to /var/log/radius_updates.log
    note right: Format: "YYYY-MM-DD HH:MM:SS.ffffff\nReceived update for key: <key>"
    deactivate logger
    deactivate sub
    deactivate notifier
    deactivate keyspace
    
    storage -> radius: Success
    deactivate storage
    radius -> client: RADIUS Accounting Response
    
else Redis unavailable
    redis -> storage: Connection Error
    deactivate redis
    storage -> radius: Error
    deactivate storage
    radius -> radius: Log error
    radius -> client: RADIUS Accounting Response
    note right: Still send response\nper RADIUS protocol
end

deactivate radius
@enduml