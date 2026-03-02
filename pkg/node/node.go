package node

import (
	"math"
	"math/rand"
	"time"
	"log"

	"github.com/ryandielhenn/zephyrcache/pkg/gossip"
	"github.com/ryandielhenn/zephyrcache/pkg/kv"
	"github.com/ryandielhenn/zephyrcache/pkg/ring"
)

type Node struct {
	kv          *kv.Store
	ring        *ring.HashRing
	gossipQueue []*gossip.MessagePayload
	suspectPeer string
	peers 		map[string]string
	id    		string
	addr  		string
	timeout     *time.Timer
}

func NewNode(store *kv.Store, r *ring.HashRing, id string, addr string) *Node {
	return &Node{
		kv: store,
		ring: r,
		gossipQueue: make([]*gossip.MessagePayload, 0),
		suspectPeer: "",
		peers: make(map[string]string),
		id: id,
		addr: addr,
	}
}

func (n *Node) addGossip(msg *gossip.MessagePayload) {
	n.gossipQueue = append(n.gossipQueue, msg)
}

func (n *Node) removeGossip() *gossip.MessagePayload {
	if len(n.gossipQueue) == 0 {
		return nil
	}
	msg := n.gossipQueue[0]
	n.gossipQueue = n.gossipQueue[1:]
	count := int(math.Floor(3 * math.Log2(float64(len(n.peers)))))
	if msg.TransmitCount > 0 && msg.TransmitCount <= count {
		msg.TransmitCount += 1
		n.gossipQueue = append(n.gossipQueue, msg)
	}
	return msg
}

func (n *Node) addPeer(id string, hostport string) {
	n.ring.Add(id, hostport)
	n.peers[id] = hostport
	log.Printf("%+v", n.peers)
}

func (n *Node) removePeer(id string) {
	n.ring.Remove(id)
	delete(n.peers, id)
	log.Printf("%+v", n.peers)
}

func (n *Node) clearPeers() {
	n.ring.Clear()
	n.peers = make(map[string]string)
}

func (n *Node) getRandomPeer() string {
	if len(n.peers) == 0 {
		return ""
	}
	keys := make([]string, 0, len(n.peers))
	for key := range n.peers {
		keys = append(keys, key)
	}
	return keys[rand.Intn(len(keys))]
}

func (n *Node) getKRandomPeers(k int) []string {
	if len(n.peers) == 0 {
		return make([]string, 0)
	}
	keys := make([]string, 0, len(n.peers))
	rand.Shuffle(len(n.peers), func(i, j int) {
		keys[i], keys[j] = keys[j], keys[i]
	})
	if k > len(n.peers) {
		k = len(n.peers)
	}
	return keys[:k]
}

