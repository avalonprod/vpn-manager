package bot

import (
	"context"
	"fmt"
	"vpn-manager/users"

	"gopkg.in/telebot.v4"
)

func (b *Bot) handleStart(c telebot.Context) error {
	user := c.Sender()
	if err := b.usersService.Register(context.Background(), users.CreateUserInput{
		ID:        user.ID,
		Username:  user.Username,
		FirstName: user.FirstName,
	}); err != nil {
		b.logger.Error(err)
		return b.replyError(c, ErrDefault)
	}

	keyword := &telebot.ReplyMarkup{}

	keyword.Inline(
		telebot.Row{
			trialAccessButton,
		},
		telebot.Row{
			subscribeButton,
		},
	)

	b.pushState(c.Sender().ID, Screen{
		Text:     Msg.AlertStart,
		Keyboard: keyword,
	})

	return c.Send(Msg.AlertStart, keyword)
}

func (b *Bot) handleTrialAccess(c telebot.Context) error {
	err := b.subscriptionsService.CreateTrialSubscription(context.Background(), c.Sender().ID)
	if err != nil {
		b.logger.Error(err)
		return b.replyError(c, ErrDefault)
	}

	keyword := &telebot.ReplyMarkup{}
	keyword.Inline(
		telebot.Row{
			appsListButton,
		},
		telebot.Row{
			backButton,
			renewSubscriptionButton,
		},
	)

	b.pushState(c.Sender().ID, Screen{
		Text:     Msg.AlertTrialAccess,
		Keyboard: keyword,
	})

	return c.Edit(Msg.AlertTrialAccess, keyword)
}

func (b *Bot) handleAppsList(c telebot.Context) error {
	userID := c.Sender().ID
	keyword := &telebot.ReplyMarkup{}

	keyword.Inline(
		telebot.Row{
			{Text: "📱 iPhone / iPad", URL: fmt.Sprintf("%s/apps?user_id=%d&os=ios", b.apiUrl, userID)},
			{Text: "💻 MacOs", URL: fmt.Sprintf("%s/apps?user_id=%d&os=macos", b.apiUrl, userID)},
		},
		telebot.Row{
			backButton,
			supportButton,
		},
	)

	b.pushState(c.Sender().ID, Screen{
		Text:     Msg.AlertListApp,
		Keyboard: keyword,
	})

	return c.Edit(Msg.AlertListApp, keyword)
}

func (b *Bot) handleSubscribe(c telebot.Context) error {
	keyword := &telebot.ReplyMarkup{}

	keyword.Inline(
		telebot.Row{
			{Text: "💳 1 месяц - 500 ₽"},
		},
		telebot.Row{
			{Text: "💳 6 мес. + 2 мес. 🎁 - 350 ₽ / мес"},
		},
		telebot.Row{
			{Text: "💳 1 год + 3 мес. 🎁 - 280 ₽ / мес"},
		},
		telebot.Row{
			{Text: "💳 2 года + 6 мес. 🎁 - 190 ₽ / мес"},
		},
		telebot.Row{
			backButton,
			supportButton,
		},
	)

	b.pushState(c.Sender().ID, Screen{
		Text:     Msg.AlertSubscriptions,
		Keyboard: keyword,
	})

	return c.Edit(Msg.AlertSubscriptions, keyword)
}

func (b *Bot) handleSuccess(c telebot.Context) error {
	keyword := &telebot.ReplyMarkup{}

	keyword.Inline(
		telebot.Row{
			renewSubscriptionButton,
			supportButton,
		},
	)

	return c.Edit(Msg.AlertSuccess, keyword)
}

func (b *Bot) SendSetupInstruction(userID int64) error {
	keyword := &telebot.ReplyMarkup{}

	keyword.Inline(
		telebot.Row{
			{Text: "Автонастройка", URL: fmt.Sprintf("%s/setup?user_id=%d&os=ios", b.apiUrl, userID)},
		},
	)

	b.pushState(userID, Screen{
		Text:     Msg.AlertSetupInstruction,
		Keyboard: keyword,
	})

	user := &telebot.User{ID: userID}
	_, err := b.bot.Send(user, Msg.AlertSetupInstruction, keyword)

	return err
}

func (b *Bot) SendPostImportInstructions(userID int64) error {
	keyword := &telebot.ReplyMarkup{}

	keyword.Inline(
		telebot.Row{
			successButton,
		},
		telebot.Row{
			supportButton,
		},
	)

	user := &telebot.User{ID: userID}
	_, err := b.bot.Send(user, Msg.AlertPostImportInstructions, keyword)

	return err
}
