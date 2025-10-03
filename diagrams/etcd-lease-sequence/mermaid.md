# Etcd Member Registration and Lease Lifecycle

Use mermaid editor to render/edit e.g. https://www.eraser.io

```
sequenceDiagram
    participant C as Client (node1)
    participant E as etcd cluster
    participant P as Peers / Watchers

    %% Phase 1: Registration
    C->>E: LeaseGrant(TTL=10s)
    E-->>C: LeaseGrantResp(leaseID=0x1234, TTL=10s)
    C->>E: Put(/members/node1, value=..., lease=0x1234)
    E-->>C: OK

    %% Phase 2: Heartbeat
    Note over C,E: Bi-di stream: LeaseKeepAlive(0x1234)
    loop KeepAlive every ~TTL/3
        C->>E: KeepAlive(0x1234)
        E-->>C: KeepAliveResp(TTL≈10s)
    end

    %% Phase 3: Failure detection
    Note over C: crash / network partition / exit
    E->>E: TTL counts down
    E->>E: EXPIRE lease 0x1234
    E->>E: DELETE /members/node1
    E-->>P: WATCH EVENT → DELETE (reason=EXPIRED)

    %% Phase 4: Graceful revoke
    opt Graceful shutdown
        C->>E: LeaseRevoke(0x1234)
        E-->>C: OK
        E->>E: DELETE /members/node1
        E-->>P: WATCH EVENT → DELETE (reason=REVOKED)
    end
```