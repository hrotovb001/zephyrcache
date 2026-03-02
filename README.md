# ZephyrCache

## Quick start
```bash
# Run with default configuration (1 cache node, 1 etcd node, default docker network)
docker-compose -f deploy/docker-compose.yml up -d

# Test the cluster
curl localhost:8080/healthz
curl -X PUT localhost:8080/kv/foo -d 'bar'
curl localhost:8080/kv/foo

# Scale to more cache nodes
docker-compose -f deploy/docker-compose.yml up -d --scale node=10
docker-compose -f deploy/docker-compose.yml up -d --scale node=50
```


## Viewing logs
```bash
# View logs from all nodes in real-time
docker-compose -f deploy/docker-compose.yml logs -f node

# View logs from all services (etcd + nodes)
docker-compose -f deploy/docker-compose.yml logs -f

# View logs from a specific container
docker logs -f deploy-node-1

# View last 100 lines
docker-compose -f deploy/docker-compose.yml logs --tail 100 node
```

## Stopping the cluster
```bash
# Stop all containers
docker-compose -f deploy/docker-compose.yml down
```
