package gossip

type Message struct {
    Type      	MessageType 	`json:"type"`
	SubjectId   string	 		`json:"sub_id"`
	SourceId    string 			`json:"src_id"`
	OriginId    string 			`json:"orig_id"`
	Payload     *MessagePayload `json:"payload"`
}

type MessagePayload struct {
	Type      		PayloadType       `json:"type"`
	Peers     		map[string]string `json:"peers"`
	TransmitCount	int               `json:"-"`
}

type MessageType string

const (
	Ping   		MessageType = "ping"
	PingReq  	MessageType = "ping_request"
	PingAck  	MessageType = "ping_ack"
)

type PayloadType string

const (
	JoinRequest     PayloadType = "join_request"
	JoinResponse    PayloadType = "join_response"
    NewMember       PayloadType = "new_member"
	DeadMember      PayloadType = "dead_member"
)

func NewMessage(msgType MessageType, subjectId string, sourceId, originId string, payload *MessagePayload) *Message {
	return &Message{
 		Type: msgType,
		SubjectId: subjectId,
		SourceId: sourceId,
		OriginId: originId,
		Payload: payload,
	}
}

func NewPayload(payloadType PayloadType, peers map[string]string) *MessagePayload {
	var transmitCount int
	if payloadType == JoinResponse {
		transmitCount = 0
	} else {
		transmitCount = 1
	}
	return &MessagePayload{
		Type: payloadType,
		Peers: peers,
		TransmitCount: transmitCount,
	}
}

