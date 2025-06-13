package servers

import "errors"

var (
	ErrServerNotFound = errors.New("server not found")
	ErrServerInactive = errors.New("server is inactive")
)
