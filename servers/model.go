package servers

type Security struct {
	PublicKey string `bson:"public_key"`
	ShortID   string `bson:"short_id"`
	SNI       string `bson:"SNI"`
}

type Server struct {
	ID         string   `bson:"_id,omitempty"`
	Location   string   `bson:"location"`
	Username   string   `bson:"username"`
	Host       string   `bson:"host"`
	Port       int      `bson:"port"`
	Ip         string   `bson:"ip"`
	ApiUrl     string   `bson:"api_url"`
	InBoundID  int      `bson:"inbound_id"`
	MaxClients int      `bson:"max_clients,omitempty"`
	IsActive   bool     `bson:"is_active"`
	Security   Security `bson:"security"`
}

// CreateInput — параметры создания сервера из админ-панели.
type CreateInput struct {
	Location   string
	Username   string
	Host       string
	Port       int
	Ip         string
	ApiUrl     string
	InBoundID  int
	MaxClients int
	IsActive   bool
	Security   Security
}

// UpdateInput — частичное обновление сервера: применяются только не-nil поля.
type UpdateInput struct {
	Location   *string
	Username   *string
	Host       *string
	Port       *int
	Ip         *string
	ApiUrl     *string
	InBoundID  *int
	MaxClients *int
	IsActive   *bool
	Security   *Security
}

// Health — результат проверки доступности панели сервера.
type Health struct {
	ServerID  string `json:"server_id"`
	Location  string `json:"location"`
	Reachable bool   `json:"reachable"`
	LatencyMs int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}
