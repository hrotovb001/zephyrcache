## Request Forwarding
Clients can send requests to any node in the cluster without needing to know which node owns the data.

![Request Forwarding](diagrams/request-forwarding/diagram.png)

## Consistent Hashing
Each node uses consistent hashing to route requests to the correct owner:

**How it works:**
- The hash ring spans positions from 0 to 2^m-1
- Each node is assigned one or more token positions (t1, t2, t3, t4) on the ring
- When a key arrives, hash(k) maps it to a position p on the ring
- The owner is the first node found traveling clockwise from position p

**Example:**
![Consistent Hashing Example](diagrams/consistent_hashing/diagram.png)
