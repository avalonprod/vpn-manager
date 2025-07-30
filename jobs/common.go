package jobs

type IBot interface {
	SendTrialSubscriptionsExpiryReminder(userID int64) error
}
