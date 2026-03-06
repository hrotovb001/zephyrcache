package node

import (
	"context"
	"log"
	"strings"

	discovery "github.com/ryandielhenn/zephyrcache/pkg/registry"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func BootstrapPeers(node *Node, cli *clientv3.Client) func() {
	// 1. Bootstrap peers into this ring
	resp, err := cli.Get(context.TODO(), "/zephyr/nodes", clientv3.WithPrefix())
	if err != nil {
		log.Fatal(err)
	}
	for _, kv := range resp.Kvs {
		nodeID := strings.TrimPrefix(string(kv.Key), "/zephyr/nodes/")
		peerHP := NormalizeHostPort(string(kv.Value), "8080")
		log.Printf("[Bootstrap] %s -> %s", nodeID, peerHP)
		node.addPeer(nodeID, peerHP)
	}

	log.Printf("[Boot] registering, %s : %s with etcd", node.id, node.addr)
	leaseId, cancel, err := discovery.RegisterNode(cli, node.id, node.addr, 10)
	if err != nil {
		log.Fatal(err)
	}

	cleanup := func() {
		cancel()
		_, _ = cli.Revoke(context.TODO(), leaseId)
	}
	return cleanup

}

func WatchPeers(node *Node, cli *clientv3.Client) {
		// 2. Watch for updates about peers
	log.Printf("[Boot] before watch peers")
	discovery.WatchPeers(cli, func(peers map[string]string) {
		normalizedPeers := make(map[string]string, len(peers))
		for id, addr := range peers {
			normalizedPeers[id] = NormalizeHostPort(addr, "8080")
		}
		node.syncPeers(normalizedPeers)
		log.Printf("[WatchPeers Callback] synced %d peers\n", len(peers))
	})
	log.Printf("[BOOT] after WatchPeers")
}
