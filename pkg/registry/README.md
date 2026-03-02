## etcd Lease Sequence
Registering all nodes in etcd on startup enables critical features such as request forwarding and health monitoring.

![etcd Lease Sequence](diagrams/etcd-lease-sequence/diagram.png)

- Each node registers its membership information in etcd under a single lease (e.g., `leaseID 0x1234`)
- All ephemeral membership keys for that node are attached to this lease, ensuring atomic cleanup on failure
- `KeepAlive` operates as a long-lived bidirectional gRPC stream implemented in the etcd client; nodes send heartbeat pings approximately every TTL/3 to maintain the lease
- When a lease expires (no heartbeat received within TTL) or is explicitly revoked, etcd automatically deletes all bound keys and emits watch events to all peers monitoring `/zephyr/nodes`
- Peer nodes receive these watch events and update their local membership view accordingly

### Known Limitations
- **Single Point of Failure**: etcd cluster outages prevent new nodes from joining and may cause cascading failures
- **Network Sensitivity**: Transient network issues can trigger false-positive failure detections
- **Scalability**: Watch event fanout becomes expensive with large clusters (>100 nodes)

### Planned solution
- Gossip protocol for cluster membership.
- Phi accrual failure detection.
