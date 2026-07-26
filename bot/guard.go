package bot

import (
	"context"

	"gopkg.in/telebot.v4"
)

func (b *Bot) accessToken(userID int64) string {
	token, err := b.peersService.EnsureAccessToken(context.Background(), userID)
	if err != nil {
		b.logger.Errorf("bot: failed to resolve access token for user %d: %v", userID, err)
		return ""
	}

	return token
}

func (b *Bot) blockGuard(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		sender := c.Sender()
		if sender == nil {
			return next(c)
		}

		blocked, err := b.usersService.IsBlocked(context.Background(), sender.ID)
		if err != nil {
			b.logger.Errorf("bot: failed to check block status for user %d: %v", sender.ID, err)
			return b.replyError(c, ErrDefault)
		}

		if blocked {
			b.logger.Warnf("bot: blocked user %d tried to interact", sender.ID)
			return b.replyError(c, ErrBlocked)
		}

		return next(c)
	}
}
