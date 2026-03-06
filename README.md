# ZephyrCache

## Quick start (gossip)
```bash
make up                  # start seed node with 3 peers (default)
make up NODES=5          # start seed node with 5 peers
make seed                # start stand-alone seed node
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
