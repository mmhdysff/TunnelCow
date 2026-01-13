package tunnel

import "encoding/json"

const (
	MsgTypeReqBind        = "REQ_BIND"
	MsgTypeReqUnbind      = "REQ_UNBIND"
	MsgTypeNewConn        = "NEW_CONN"
	MsgTypePing           = "PING"
	MsgTypeReqDomainMap   = "REQ_DOMAIN_MAP"
	MsgTypeReqDomainUnmap = "REQ_DOMAIN_UNMAP"
)

type ControlMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type ReqBindPayload struct {
	PublicPort int `json:"public_port"`
	LocalPort  int `json:"local_port"`
}

type ReqUnbindPayload struct {
	PublicPort int `json:"public_port"`
}

type ReqDomainMapPayload struct {
	Domain     string `json:"domain"`
	PublicPort int    `json:"public_port"`
	Mode       string `json:"mode"`
}

type ReqDomainUnmapPayload struct {
	Domain string `json:"domain"`
}
