# Request Forwarding

sequenceDiagram
  C->N2: HTTP PUT /kv/k v
  N2->Ring: ring.successor(hash(k))
  Ring-->N2: owner = N3
  N2->N3: RPC Write(k,v)
  activate N3
  N3-->N2: OK
  deactivate N3
  N2-->C: 200 OK
