package node

import (
	"encoding/json"
	"net"
	"time"
	"log"
	"maps"

	"github.com/ryandielhenn/zephyrcache/pkg/gossip"
)

func (n *Node) handleGossip(msg *gossip.Message, addr string) {
	if msg == nil {
		return
	}

	// log.Printf("Receiving Message")
	// log.Printf("%+v", *msg)

	if msg.Payload != nil {
		n.handlePayload(msg.Payload)
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

func (n *Node) handlePing(msg *gossip.Message, addr string){
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

func (n *Node) handlePingReq(msg *gossip.Message){
	addr, ok := n.peers[msg.SubjectId]
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
	n.sendGossip(message, addr)
}

func (n *Node) handlePingAck(msg *gossip.Message){
	if msg.OriginId == n.id && n.suspectPeer == msg.SubjectId {
		if n.timeout != nil {
			n.timeout.Stop()
			n.timeout = nil
		}
		n.suspectPeer = ""
		return
	}
	addr, ok := n.peers[msg.OriginId]
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
	n.sendGossip(message, addr)
}

func (n *Node) handlePayload(msg *gossip.MessagePayload){
	if msg == nil {
	  	return
	}

	// log.Printf("%+v", *msg)

	switch msg.Type {
	case gossip.JoinRequest:
		n.handleJoinRequest(msg)
	case gossip.JoinResponse:
		n.handleJoinResponse(msg)
	case gossip.NewMember:
		n.handleNewMember(msg)
	case gossip.DeadMember:
		n.handleDeadMember(msg)
	}
}

func (n *Node) handleJoinRequest(msg *gossip.MessagePayload){
	n.handleNewMember(msg)
	var id string
	for k, _ := range msg.Peers {
		id = k
		break
	}
	peers := maps.Clone(n.peers)
	peers[n.id] = n.addr
	delete(peers, id)
	message := gossip.NewPayload(
		gossip.JoinResponse,
		peers,
	)
	n.gossipQueue = append([]*gossip.MessagePayload{message}, n.gossipQueue...)
}

func (n *Node) handleJoinResponse(msg *gossip.MessagePayload){
	n.peers = msg.Peers
	log.Printf("%+v", n.peers)
}

func (n *Node) handleNewMember(msg *gossip.MessagePayload){
	if len(msg.Peers) == 0 {
		return
	}
	var id, addr string
	for k, v := range msg.Peers {
		id, addr = k, v
		break
	}
	if id == n.id {
		return
	}
	if value, ok := n.peers[id]; ok && value == addr {
		return
	}
	n.addPeer(id, addr)
	newPeer := map[string]string {
		id: addr,
	}
	message := gossip.NewPayload(
		gossip.NewMember,
		newPeer,
	)
	n.addGossip(message)
}

func (n *Node) handleDeadMember(msg *gossip.MessagePayload){
	if len(msg.Peers) == 0 {
		return
	}
	var id string
	for k, _ := range msg.Peers {
		id = k
		break
	}
	if _, ok := n.peers[id]; !ok {
		return
	}
	n.removePeer(id)
	deadPeer := map[string]string {
		id: "",
	}
	message := gossip.NewPayload(
		gossip.DeadMember,
		deadPeer,
	)
	n.addGossip(message)
}

func (n *Node) sendGossip(msg *gossip.Message, addr string) {
	// log.Printf("Sending Message")
	// log.Printf("%+v", *msg)
	// if msg.Payload != nil {
	// 	log.Printf("%+v", *msg.Payload)
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
	peer := map[string]string{
        n.id: n.addr,
    }
	payload := gossip.NewPayload(
		gossip.JoinRequest,
        peer,
	)
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
			node.removePeer(node.suspectPeer)
			deadPeer := map[string]string {
				node.suspectPeer: "",
			}
			payload := gossip.NewPayload(
				gossip.DeadMember,
				deadPeer,
			)
			node.addGossip(payload)
		}
		payload := node.removeGossip()
		node.suspectPeer = node.getRandomPeer()
		addr, ok := node.peers[node.suspectPeer]
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
		node.sendGossip(message, addr)
		node.timeout = time.AfterFunc(cfg.timeout, func() {
			for _, id := range node.getKRandomPeers(cfg.k) {
				if id == node.suspectPeer {
					continue
				}
				addr, ok = node.peers[id]
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
				node.sendGossip(message, addr)
			}
		})
    }
}

