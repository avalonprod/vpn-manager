package bot

import (
	"context"
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

	if err := b.clearMessages(user.ID); err != nil {
		return err
	}

	screen := b.GenerateStartScreen(user.ID)
	return c.Send(screen.Text, screen.Keyboard)
}

func (b *Bot) handleSupport(c telebot.Context) error {
	user := c.Sender()

	screen := b.GenerateSupportScreen(user.ID)
	return c.Send(screen.Text)
}

func (b *Bot) handleTrialAccess(c telebot.Context) error {
	user := c.Sender()
	err := b.subscriptionsService.CreateTrialSubscription(context.Background(), c.Sender().ID)
	if err != nil {
		b.logger.Error(err)
		return b.replyError(c, ErrDefault)
	}

	screen := b.GenerateTrialAccessScreen(user.ID)
	return c.Edit(screen.Text, screen.Keyboard)
}

func (b *Bot) handleAppsList(c telebot.Context) error {
	user := c.Sender()

	screen := b.GenerateAppsListScreen(user.ID)
	return c.Edit(screen.Text, screen.Keyboard)
}

func (b *Bot) handleSubscribe(c telebot.Context) error {
	user := c.Sender()

	screen := b.GenerateSubscriptionsScreen(user.ID)
	return c.Edit(screen.Text, screen.Keyboard)
}

func (b *Bot) handleRenewSubscribe(c telebot.Context) error {
	user := c.Sender()

	screen := b.GenerateRenewSubscriptionsScreen(user.ID)
	return c.Edit(screen.Text, screen.Keyboard)
}

func (b *Bot) handleSuccess(c telebot.Context) error {
	user := c.Sender()

	screen := b.GenerateSuccessScreen(user.ID)
	return c.Edit(screen.Text, screen.Keyboard)
}

func (b *Bot) SendSetupInstruction(userID int64) error {
	user := &telebot.User{ID: userID}

	if err := b.clearMessages(user.ID); err != nil {
		return err
	}

	screen := b.GenerateSetupScreen(userID)
	msg, err := b.bot.Send(user, screen.Text, screen.Keyboard)
	b.pushMessage(userID, msg)
	return err
}

func (b *Bot) SendPostImportInstructions(userID int64) error {
	user := &telebot.User{ID: userID}

	if err := b.clearMessages(user.ID); err != nil {
		return err
	}

	screen := b.GeneratePostImportInstructionsScreen(userID)
	msg, err := b.bot.Send(user, screen.Text, screen.Keyboard)
	b.pushMessage(userID, msg)
	return err
}
