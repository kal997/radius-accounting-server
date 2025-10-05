# Data Flow Diagram

```plantuml
@startuml dataflow
start
  :RADIUS packet arrives on UDP 1813;
  note right: layeh/radius handles\nUDP server setup;
  :Parse packet to AccountingEvent;
  note right: Determine StartRecord,\nStopRecord or InterimRecord;
  :Validate AccountingEvent;
  :Generate Redis key;
  note right: radius:acct:username:session:timestamp:start/stop/interim;
  if (Store in Redis?) then (success)
    :Redis SET with TTL;
    note right: Redis auto‑publishes\nkeyspace notification;
    :Log success;
  else (error)
    :Log storage error;
    note right: Continue processing\nwon't fail the request;
  endif
  :Send RADIUS Accounting Response;
  note right: Always send response\nregardless of storage result;
  fork
    :Redis keyspace notification;
    note right: __keyspace@0__:radius:acct:*;
    :Subscriber receives notification;
    :Convert to StorageEvent;
    note right: Extract key and operation;
    :Format log message;
    note right: Received update for key: <key>, Operation: <operation>;
    :Write to log file;
    note right: /var/log/radius_updates.log;
  endfork
stop
@enduml
```

This data‑flow diagram updates the original to account for the new `AccountingEvent` model.  Instead of creating a single `AccountingRecord` (v1), the server parses each incoming RADIUS packet into a specific event type (`StartRecord`, `StopRecord` or `InterimRecord`). The generated Redis key is postfixed with `start:`, `stop:` or `interim:` accordingly.