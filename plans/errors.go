package plans

import "errors"

var (
	ErrPlanNotFound      = errors.New("plan not found")
	ErrPlanAlreadyExists = errors.New("plan already exists")
	ErrInvalidPlanID     = errors.New("invalid plan id")
	ErrInvalidInput      = errors.New("invalid plan input")
)
