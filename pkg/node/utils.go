package node

import (
	"net"
	"strings"
)

// NormalizeHostPort removes the http:// or https:// prefix from the address
// and sets the port to port (replacing any existing port).
func NormalizeHostPort(addr, port string) string {
	if rest, ok := strings.CutPrefix(addr, "http://"); ok {
		addr = rest
	} else if rest, ok := strings.CutPrefix(addr, "https://"); ok {
		addr = rest
	}

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}

	return net.JoinHostPort(host, port)
}

// ownerForKey looks up the owner for a key and normalizes the address of the owner
func (s *Node) OwnerForKey(key string) (ownerHP, selfHP string, ok bool) {
	ownerID := s.ring.Lookup([]byte(key)) // e.g. "Node3"
	ownerAddr, ok := s.ring.Addr(ownerID) // e.g. "Node3:8080" (what you stored)
	if !ok || ownerAddr == "" {
		return "", "", false
	}
	return NormalizeHostPort(ownerAddr, "8080"), NormalizeHostPort(s.addr, "8080"), true
}
