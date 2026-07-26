package servers

type Security struct {
	PublicKey   string `bson:"public_key"`
	ShortID     string `bson:"short_id"`
	SNI         string `bson:"SNI"`
	Fingerprint string `bson:"fingerprint,omitempty"`
}

type Server struct {
	ID       string `bson:"_id,omitempty"`
	Location string `bson:"location"`

	AuthToken  string   `bson:"auth_token"`
	Host       string   `bson:"host"`
	Port       int      `bson:"port"`
	Ip         string   `bson:"ip"`
	ApiUrl     string   `bson:"api_url"`
	InBoundID  int      `bson:"inbound_id"`
	MaxClients int      `bson:"max_clients,omitempty"`
	IsActive   bool     `bson:"is_active"`
	Security   Security `bson:"security"`
}

type CreateInput struct {
	Location   string
	AuthToken  string
	Host       string
	Port       int
	Ip         string
	ApiUrl     string
	InBoundID  int
	MaxClients int
	IsActive   bool
	Security   Security
}

type UpdateInput struct {
	Location   *string
	AuthToken  *string
	Host       *string
	Port       *int
	Ip         *string
	ApiUrl     *string
	InBoundID  *int
	MaxClients *int
	IsActive   *bool
	Security   *Security
}

type Health struct {
	ServerID  string `json:"server_id"`
	Location  string `json:"location"`
	Reachable bool   `json:"reachable"`
	LatencyMs int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}
