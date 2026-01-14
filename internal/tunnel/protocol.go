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
	AuthUser   string `json:"auth_user,omitempty"`
	AuthPass   string `json:"auth_pass,omitempty"`
}

type ReqDomainUnmapPayload struct {
	Domain string `json:"domain"`
}

const (
	MsgTypeInspectData = "INSPECT_DATA"
)

type InspectPayload struct {
	ID         string            `json:"id"`
	Timestamp  int64             `json:"timestamp"`
	Method     string            `json:"method"`
	URL        string            `json:"url"`
	ReqHeaders map[string]string `json:"req_headers"`
	ReqBody    string            `json:"req_body"`
	Status     int               `json:"status"`
	ResHeaders map[string]string `json:"res_headers"`
	ResBody    string            `json:"res_body"`
	DurationMs int64             `json:"duration_ms"`
	ClientIP   string            `json:"client_ip"`
	PublicPort int               `json:"public_port"`
}
