package peers

import "errors"

var (
	ErrInvalidId     = errors.New("invalid ID format")
	ErrPeerNotFound  = errors.New("peer not found")
	ErrPeerNotActive = errors.New("peer not active")
)
