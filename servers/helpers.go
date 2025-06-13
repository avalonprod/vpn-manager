package servers

import (
	"fmt"
)

func buildClientConf(priv string, r RegisterResponse) string {
	return fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s/32
DNS = %s

[Peer]
PublicKey = %s
Endpoint = %s
AllowedIPs = %s
PersistentKeepalive = 25
`, priv, r.IP, r.DNS, r.ServerPublicKey, r.Endpoint, r.AllowedIPs)
}
