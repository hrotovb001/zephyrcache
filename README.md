# ZephyrCache

## Quick start (gossip)
```bash
make seed                # start seed node
make up                  # start 3 peers (default)
make up NODES=5          # start 5 peers
```
## Quick start (etcd)
```bash
make up DISCOVERY=etcd            # etcd mode, 3 nodes
make up DISCOVERY=etcd NODES=5    # etcd mode, 5 nodes
```

## Teardown
```bash
make down                # tear down
```

## Check Status/Logs
```bash
make logs                # tail all logs
make status              # show running containers
```

## Format code to pass CI
```bash
make format              # format all code
```
