package notifier

import (
	"context"

	"gopkg.in/telebot.v4"
)

type notifier struct {
	bot *telebot.Bot
}

func NewNotifier(bot *telebot.Bot) *notifier {
	return &notifier{
		bot: bot,
	}
}

func (n *notifier) Notify(ctx context.Context, userID int64, msg string) error {
	user := &telebot.User{ID: userID}

	_, err := n.bot.Send(user, msg)

	return err
}
