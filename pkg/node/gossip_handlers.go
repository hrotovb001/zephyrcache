package node

import (
	"encoding/json"
	"net"
	"time"

	"github.com/ryandielhenn/zephyrcache/pkg/gossip"
	"github.com/ryandielhenn/zephyrcache/pkg/peer"
)

func (n *Node) handleGossip(msg *gossip.Message, addr string) {
	if msg == nil {
		return
	}

	// log.Printf("Receiving Message")
	// log.Printf("%+v", *msg)

	if msg.Payload != nil {
		n.handlePayload(msg.Payload, msg.SourceId)
	}

	switch msg.Type {
	case gossip.Ping:
		n.handlePing(msg, addr)
	case gossip.PingReq:
		n.handlePingReq(msg)
	case gossip.PingAck:
		n.handlePingAck(msg)
	}
}

func (n *Node) handlePing(msg *gossip.Message, addr string) {
	payload := n.removeGossip()
	message := gossip.NewMessage(
		gossip.PingAck,
		msg.SubjectId,
		n.id,
		msg.OriginId,
		payload,
	)
	n.sendGossip(message, NormalizeHostPort(addr, "4000"))
}

func (n *Node) handlePingReq(msg *gossip.Message) {
	peerBody, ok := n.peers[msg.SubjectId]
	if !ok {
		return
	}
	payload := n.removeGossip()
	message := gossip.NewMessage(
		gossip.Ping,
		msg.SubjectId,
		n.id,
		msg.OriginId,
		payload,
	)
	n.sendGossip(message, peerBody.Addr)
}

func (n *Node) handlePingAck(msg *gossip.Message) {
	if msg.OriginId == n.id && n.suspectPeer == msg.SubjectId {
		if n.timeout != nil {
			n.timeout.Stop()
			n.timeout = nil
		}
		n.suspectPeer = ""
		return
	}
	peerBody, ok := n.peers[msg.OriginId]
	if !ok {
		return
	}
	payload := n.removeGossip()
	message := gossip.NewMessage(
		gossip.Ping,
		msg.SubjectId,
		n.id,
		msg.OriginId,
		payload,
	)
	n.sendGossip(message, peerBody.Addr)
}

func (n *Node) handlePayload(msg *gossip.MessagePayload, sourceId string) {
	if msg == nil {
		return
	}

	// log.Printf("%+v", *msg)

	for id, peerBody := range msg.Peers {
		switch peerBody.Status {
		case peer.Alive:
			if id == n.id {
				continue
			}
			p, ok := n.peers[id]
			if !ok && id == sourceId && peerBody.Incarnation == 0 {
				peers := n.getPeerMap()
				peers[n.id] = peer.Peer{
					n.addr,
					peer.Alive,
					n.incarnation,
				}
				delete(peers, id)
				payload := gossip.NewPayload(peers, false)
				n.prependGossip(payload)
			}
			shouldUpdate := !ok || (peerBody.Incarnation > p.Incarnation)
			if shouldUpdate {
				n.setPeer(id, peerBody)
				peers := map[string]peer.Peer{
					id: peerBody,
				}
				payload := gossip.NewPayload(peers, true)
				n.addGossip(payload)
			}
		case peer.Dead:
			if id == n.id {
				peers := map[string]peer.Peer{
					id: peer.Peer{
						n.addr,
						peer.Alive,
						n.incarnation + 1,
					},
				}
				payload := gossip.NewPayload(peers, true)
				n.addGossip(payload)
				continue
			}
			p, ok := n.peers[id]
			shouldUpdate := !ok || (peerBody.Incarnation > p.Incarnation ||
				peerBody.Incarnation == p.Incarnation && p.Status == peer.Alive)
			if shouldUpdate {
				n.setPeer(id, peerBody)
				peers := map[string]peer.Peer{
					id: peerBody,
				}
				payload := gossip.NewPayload(peers, true)
				n.addGossip(payload)
			}
		}
	}
}

func (n *Node) sendGossip(msg *gossip.Message, addr string) {
	// log.Printf("Sending Message")
	// log.Printf("%+v", *msg)
	// if msg.Payload != nil {
	//  log.Printf("%+v", *msg.Payload)
	// }

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return
	}
	defer conn.Close()

	_, err = conn.Write(data)
	if err != nil {
		return
	}
}

func (n *Node) ConnectToCluster(addr string) {
	peers := map[string]peer.Peer{
		n.id: peer.Peer{
			n.addr,
			peer.Alive,
			0,
		},
	}
	payload := gossip.NewPayload(peers, true)
	message := gossip.NewMessage(
		gossip.Ping,
		"",
		n.id,
		n.id,
		payload,
	)
	n.sendGossip(message, addr)
}

func StartGossipListener(port string, node *Node) {
	address := net.JoinHostPort("", port)

	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return
	}
	defer conn.Close()

	buffer := make([]byte, 1024)
	for {
		n, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			continue
		}

		data := make([]byte, n)
		copy(data, buffer[:n])

		var msg gossip.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		node.handleGossip(&msg, addr.String())
	}
}

type pingerConfig struct {
	period  time.Duration
	timeout time.Duration
	k       int
}

type pingerOption func(*pingerConfig)

func WithPeriod(period time.Duration) pingerOption {
	return func(c *pingerConfig) {
		c.period = period
	}
}

func WithTimeout(timeout time.Duration) pingerOption {
	return func(c *pingerConfig) {
		c.timeout = timeout
	}
}

func WithK(k int) pingerOption {
	return func(c *pingerConfig) {
		c.k = k
	}
}

func StartGossipPinger(node *Node, opts ...pingerOption) {
	cfg := &pingerConfig{
		period:  1 * time.Second,
		timeout: 500 * time.Millisecond,
		k:       3,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	ticker := time.NewTicker(cfg.period)
	defer ticker.Stop()

	for range ticker.C {
		if node.suspectPeer != "" {
			peerBody, ok := node.peers[node.suspectPeer]
			if ok {
				peerBody.Status = peer.Dead
				peers := map[string]peer.Peer{
					node.suspectPeer: peerBody,
				}
				payload := gossip.NewPayload(peers, true)
				node.addGossip(payload)
				node.setPeer(node.suspectPeer, peerBody)
			}
		}
		payload := node.removeGossip()
		node.suspectPeer = node.getRandomPeer()
		peerBody, ok := node.peers[node.suspectPeer]
		if !ok {
			continue
		}
		message := gossip.NewMessage(
			gossip.Ping,
			node.suspectPeer,
			node.id,
			node.id,
			payload,
		)
		node.sendGossip(message, peerBody.Addr)
		node.timeout = time.AfterFunc(cfg.timeout, func() {
			for _, id := range node.getKRandomPeers(cfg.k) {
				if id == node.suspectPeer {
					continue
				}
				peerBody, ok := node.peers[id]
				if !ok {
					continue
				}
				message := gossip.NewMessage(
					gossip.PingReq,
					id,
					node.id,
					node.id,
					payload,
				)
				node.sendGossip(message, peerBody.Addr)
			}
		})
	}
}
