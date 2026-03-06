package peer

type Peer struct {
	Addr        string     `json:"addr"`
	Status      PeerStatus `json:"status"`
	Incarnation int        `json:"incarnation"`
}

type PeerStatus string

const (
	Alive PeerStatus = "alive"
	Dead  PeerStatus = "dead"
)
