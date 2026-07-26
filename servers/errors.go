package servers

import "errors"

var (
	ErrServerNotFound       = errors.New("server not found")
	ErrServerInactive       = errors.New("server is inactive")
	ErrInvalidServerID      = errors.New("invalid server id")
	ErrInvalidInput         = errors.New("invalid server input")
	ErrPanelUnauthorized    = errors.New("panel rejected the auth token")
	ErrPanelEndpointMissing = errors.New("panel endpoint not found")
	ErrUserBlocked          = errors.New("user is blocked")
	ErrNoServersAvailable   = errors.New("no server accepted the client")
)
