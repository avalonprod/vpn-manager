package notifier

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type notifier struct {
	bot *bot.Bot
}

func NewNotifier(bot *bot.Bot) *notifier {
	return &notifier{
		bot: bot,
	}
}

func (n *notifier) Notify(ctx context.Context, chatId int64, msg string) error {
	_, err := n.bot.SendMessage(
		ctx,
		&bot.SendMessageParams{
			ChatID:    chatId,
			Text:      msg,
			ParseMode: models.ParseModeHTML,
		},
	)

	return err
}
