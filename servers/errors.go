package servers

import "errors"

var (
	ErrServerNotFound  = errors.New("server not found")
	ErrServerInactive  = errors.New("server is inactive")
	ErrInvalidServerID = errors.New("invalid server id")
	ErrInvalidInput    = errors.New("invalid server input")
)
