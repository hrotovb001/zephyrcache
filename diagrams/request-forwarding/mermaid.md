# Request Forwarding

Client [icon: azure-administrative-units, color: blue]
Ingress Node (2) [icon: azure-load-balancers, color: orange]
Ring [icon:azure-confidential-ledgers, color: teal]
Node 3 [icon: azure-support-center-blue, color: green]

Client->Ingress Node (2): HTTP PUT /kv/k v
Ingress Node (2)->Ring: ring.successor(hash(k))
Ring-->Ingress Node (2): owner = N3
Ingress Node (2)->Node 3: RPC Write(k,v)
activate Node 3
Node 3-->Ingress Node (2): OK
deactivate Node 3
Ingress Node (2)-->Client: 200 OK
