package servers

type Security struct {
	PublicKey string `bson:"public_key"`
	ShortID   string `bson:"short_id"`
	SNI       string `bson:"SNI"`
}

type Server struct {
	ID         string   `bson:"_id"`
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
