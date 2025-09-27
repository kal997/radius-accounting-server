# Data Flow Diagram
```plantuml
@startuml dataflow
start

:RADIUS packet arrives on UDP 1813;
note right: layeh/radius handles\nUDP server setup

:Extract accounting attributes;
note right: Username, NAS-IP,\nAcct-Status-Type, etc.

:Create AccountingRecord struct;

:Generate Redis key;
note right: radius:acct:user:session:timestamp

if (Store in Redis?) then (success)
    :Redis SET with TTL;
    note right: Redis auto-publishes\nkeyspace notification
    :Log success;
else (error)
    :Log storage error;
    note right: Continue processing\ndon't fail the request
endif

:Send RADIUS Accounting Response;
note right: Always send response\nregardless of storage result

fork
    :Redis keyspace notification;
    note right: __keyspace@0__:radius:acct:*
    
    :Subscriber receives notification;
    
    :Extract key from notification;
    
    :Format log message;
    note right: YYYY-MM-DD HH:MM:SS.ffffff\nReceived update for key: <key>
    
    :Write to log file;
    note right: /var/log/radius_updates.log
    
    stop
endfork

stop
@enduml