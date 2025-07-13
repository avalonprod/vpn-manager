package subscriptions

import "errors"

var (
	ErrSubscriptionNotFound        = errors.New("user subscription not found")
	ErrTrialAccessAlreadyActivated = errors.New("trial access already activated")
	ErrInactiveSubscription        = errors.New("subscription inactive")
)
