```plantuml
@startuml simple_deployment
node "Docker Host" {
    rectangle "radius-controlplane" as radius {
        component "radius-controlplane "
    }
    
    rectangle "redis" as redis_container {
        component "Redis Server"
        note right: notify-keyspace-events KEA\n(via command line)
    }
    
    rectangle "redis-controlplane-logger" as logger {
        component "radius-controlplane-logger "
        file "radius_updates.log" as log_file
    }
    
    rectangle "radclient-test" as test {
        component "FreeRADIUS Utils"
        file "acct_start.txt" as start_file
        file "acct_stop.txt" as stop_file
    }
    
    folder "Host ./examples/" {
        file "acct_start.txt" as host_start
        file "acct_stop.txt" as host_stop
    }
    
    folder "Host ./logs/" {
        file "radius_updates.log" as host_log
    }
}

test --> radius : UDP 1813
radius --> redis_container
logger --> redis_container

host_start ..> start_file : mount to\n/examples/
host_stop ..> stop_file : mount to\n/examples/
host_log ..> log_file : mount to\n/var/log/

@enduml