# Etcd Member Registration and Lease Lifecycle

Use mermaid editor to render/edit e.g. https://www.eraser.io

```
title Etcd Member Registration and Lease Lifecycle
autoNumber nested

Cache Node [icon: azure-support-center-blue, color: blue]
Etcd Cluster [icon: azure-api-center, color: purple]
Peers [icon: azure-consortium, color: green]

// Phase 1: Registration
Cache Node > Etcd Cluster: Request short-term lease (e.g., 10s)
activate Cache Node
Etcd Cluster --> Cache Node: Grant lease with ID and TTL
Cache Node > Etcd Cluster: Register self with lease ID
Etcd Cluster --> Cache Node: Registration confirmation

// Phase 2: Heartbeat
loop [label: Keep lease alive, icon: refresh, color: blue] {
  Cache Node > Etcd Cluster: Send keep-alive for lease
  Etcd Cluster --> Cache Node: Respond with updated TTL
}

// Phase 3: Failure detection
break [label: Cache Node failure, icon: alert-triangle, color: red] {
  Cache Node > Cache Node: Crash, network partition, or exit
  Etcd Cluster > Etcd Cluster: Lease TTL counts down
  Etcd Cluster > Etcd Cluster: Lease expires
  Etcd Cluster > Etcd Cluster: Remove client registration
  Etcd Cluster --> Peers: Notify peers of removal (reason: expired)
}

// Phase 4: Graceful shutdown
opt [label: Graceful shutdown, icon: log-out, color: orange] {
  Cache Node > Etcd Cluster: Request lease revocation
  Etcd Cluster --> Cache Node: Confirm revocation
  Etcd Cluster > Etcd Cluster: Remove client registration
  Etcd Cluster --> Peers: Notify peers of removal (reason: revoked)
}
deactivate Cache Node
```
