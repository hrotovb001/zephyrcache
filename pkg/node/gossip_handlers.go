package node

import (
	"encoding/json"
	"log/slog"
	"net"
	"time"

	"github.com/ryandielhenn/zephyrcache/pkg/gossip"
	"github.com/ryandielhenn/zephyrcache/pkg/peer"
)

func (n *Node) handleGossip(msg *gossip.Message, addr string) {
	if msg == nil {
		return
	}

	slog.Debug("Received Message", "message", *msg)

	if msg.Payload != nil {
		n.handlePayload(msg.Payload, msg.SourceId)
	}

	switch msg.Type {
	case gossip.Ping:
		n.handlePing(msg)
	case gossip.PingReq:
		n.handlePingReq(msg)
	case gossip.PingAck:
		n.handlePingAck(msg)
	}
}

func (n *Node) handlePing(msg *gossip.Message) {
	peerBody, ok := n.peers[msg.SourceId]
	if !ok {
		return
	}
	payload := n.removeGossip()
	message := gossip.NewMessage(
		gossip.PingAck,
		msg.SubjectId,
		n.id,
		msg.OriginId,
		payload,
	)
	n.sendGossip(message, peerBody.Addr)
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
	// handle ack at node that requested it
	if msg.OriginId == n.id {
		if n.suspectPeer == msg.SubjectId {
			if n.timeout != nil {
				n.timeout.Stop()
				n.timeout = nil
			}
			n.suspectPeer = ""
		}
		return
	}
	
	// handle forwarding ack when ping req
	peerBody, ok := n.peers[msg.OriginId]
	if !ok {
		return
	}
	payload := n.removeGossip()
	message := gossip.NewMessage(
		gossip.PingAck,
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

	slog.Debug("Received Message Payload", "payload", *msg)

	for id, updatedPeer := range msg.Peers {
		switch updatedPeer.Status {
		case peer.Alive:
			n.handleAliveStatus(id, updatedPeer, sourceId)
		case peer.Suspected:
			n.handleSuspectedStatus(id, updatedPeer)
		case peer.Dead:
			n.handleDeadStatus(id, updatedPeer)
		}
	}
}

func (n *Node) handleAliveStatus(id string, updatedPeer peer.Peer, sourceId string) {
	// drop payloads about yourself
	if id == n.id {
		return
	}

	// handle join requests
	// when new nodes sends alive status for itself respond with peers
	currentPeer, ok := n.peers[id]
	if !ok && id == sourceId {
		peers := n.getPeerMap()
		peers[n.id] = peer.Peer{
			Addr:        n.addr,
			Status:      peer.Alive,
			Incarnation: n.incarnation,
		}
		delete(peers, id)
		payload := gossip.NewPayload(peers, false)
		n.prependGossip(payload)
	}

	// determine whether message is stale or not
	// update peer status if not stale and propagate update to other nodes
	shouldUpdate := !ok || (updatedPeer.Incarnation > currentPeer.Incarnation)
	if shouldUpdate {
		n.setPeer(id, updatedPeer)
		peers := map[string]peer.Peer{
			id: updatedPeer,
		}
		payload := gossip.NewPayload(peers, true)
		n.addGossip(payload)
	}
}

func (n *Node) handleSuspectedStatus(id string, updatedPeer peer.Peer) {
	// drop payloads about yourself
	if id == n.id {
		// refute updates saying you are suspected
		if updatedPeer.Incarnation == n.incarnation {
			n.incarnation += 1
			peers := map[string]peer.Peer{
				n.id: peer.Peer{
					Addr: n.addr,
					Status: peer.Alive,
					Incarnation: n.incarnation,
				},
			}
			payload := gossip.NewPayload(peers, true)
			n.addGossip(payload)
		}
		return
	}

	// determine whether message is stale or not
	// update peer status if not stale and propagate update to other nodes
	// suspected status has precedence over alive messages for equal incarnation
	currentPeer, ok := n.peers[id]
	shouldUpdate := !ok || (updatedPeer.Incarnation > currentPeer.Incarnation ||
		updatedPeer.Incarnation == currentPeer.Incarnation && currentPeer.Status == peer.Alive)
	if shouldUpdate {
		n.setPeer(id, updatedPeer)
		peers := map[string]peer.Peer{
			id: updatedPeer,
		}
		payload := gossip.NewPayload(peers, true)
		n.addGossip(payload)
	}
}

func (n *Node) handleDeadStatus(id string, updatedPeer peer.Peer) {
	// drop payloads about yourself
	if id == n.id {
		return
	}

	// determine whether message is stale or not
	// update peer status if not stale and propagate update to other nodes
	// dead status has precedence over alive and suspected messages for equal incarnation
	currentPeer, ok := n.peers[id]
	shouldUpdate := !ok || (updatedPeer.Incarnation > currentPeer.Incarnation ||
		updatedPeer.Incarnation == currentPeer.Incarnation && currentPeer.Status != peer.Dead)
	if shouldUpdate {
		n.setPeer(id, updatedPeer)
		peers := map[string]peer.Peer{
			id: updatedPeer,
		}
		payload := gossip.NewPayload(peers, true)
		n.addGossip(payload)
	}
}

func (n *Node) sendGossip(msg *gossip.Message, addr string) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	udpAddr, err := net.ResolveUDPAddr("udp", OverrideHostPort(addr, n.gossipPort))
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

func (n *Node) attemptConnectToCluster(addr string) {
	peers := map[string]peer.Peer{
		n.id: {
			Addr:        n.addr,
			Status:      peer.Alive,
			Incarnation: 0,
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

func (n *Node) ConnectToCluster(addr string, attemptPeriod time.Duration) {
	ticker := time.NewTicker(attemptPeriod)
	for range ticker.C {
		if len(n.peers) > 0 {
			break
		}
		n.attemptConnectToCluster(addr)
	}
}

func StartGossipListener(node *Node) {
	address := net.JoinHostPort("", node.gossipPort)

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
		// propagate SUSPECTED if has not been acked since last ping
		if node.suspectPeer != "" {
			peerBody, ok := node.peers[node.suspectPeer] 
			if ok && peerBody.Status != peer.Dead {
				peerBody.Status = peer.Suspected
				node.peers[node.suspectPeer] = peerBody
				peers := map[string]peer.Peer{
					node.suspectPeer: peerBody,
				}
				payload := gossip.NewPayload(peers, true)
				node.addGossip(payload)

				// set timeout to declare dead if SUSPECTED for long enough
				suspectPeer := node.suspectPeer
				time.AfterFunc(600 * time.Millisecond, func() {
					peerBody, ok := node.peers[suspectPeer]
					if !ok || peerBody.Status != peer.Suspected {
						return
					}
					peerBody.Status = peer.Dead
					node.setPeer(suspectPeer, peerBody)
					peers := map[string]peer.Peer{
						suspectPeer: peerBody,
					}
					payload := gossip.NewPayload(peers, true)
					node.addGossip(payload)
				})
			}
		}

		// send ping to new random suspected peer
		node.suspectPeer = node.getRandomPeer()
		peerBody, ok := node.peers[node.suspectPeer]
		if !ok {
			continue
		}
		payload := node.removeGossip()
		message := gossip.NewMessage(
			gossip.Ping,
			node.suspectPeer,
			node.id,
			node.id,
			payload,
		)
		node.sendGossip(message, peerBody.Addr)

		// send ping req to k random peers after timeout
		suspectPeer := node.suspectPeer
		node.timeout = time.AfterFunc(cfg.timeout, func() {
			for _, id := range node.getKRandomPeers(cfg.k) {
				if id == suspectPeer {
					continue
				}
				peerBody, ok := node.peers[id]
				if !ok {
					continue
				}
				payload := node.removeGossip()
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
