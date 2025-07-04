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

	if err := b.serversService.RegisterNewPeers(context.Background(), user.ID); err != nil {
		b.logger.Error(err)
		return b.replyError(c, ErrDefault)
	}

	screen := b.BuildStartScreen(user.ID)
	return b.SendMessage(user.ID, screen)
}

func (b *Bot) handleTrialAccess(c telebot.Context) error {
	user := c.Sender()
	err := b.subscriptionsService.CreateTrialSubscription(context.Background(), c.Sender().ID)
	if err != nil {
		b.logger.Error(err)
		return b.replyError(c, ErrDefault)
	}

	if err := b.peersService.ActivatePeer(context.Background(), user.ID); err != nil {
		b.logger.Error(err)
		return b.replyError(c, ErrDefault)
	}

	screen := b.BuildTrialAccessScreen(user.ID)
	return c.Edit(screen.Text, screen.Keyboard)
}

func (b *Bot) handleAppsList(c telebot.Context) error {
	user := c.Sender()

	screen := b.BuildAppsListScreen(user.ID)
	return c.Edit(screen.Text, screen.Keyboard)
}

func (b *Bot) handleSubscribe(c telebot.Context) error {
	user := c.Sender()

	screen, err := b.BuildSubscriptionsScreen(context.Background(), user.ID, b.GetScreenFromCtx(c), b.GetArgsFromCtx(c)...)
	if err != nil {
		b.logger.Error(err)
		return b.replyError(c, ErrDefault)
	}
	return c.Edit(screen.Text, screen.Keyboard)
}

func (b *Bot) handleSuccess(c telebot.Context) error {
	user := c.Sender()

	var os string
	args := b.GetArgsFromCtx(c)
	if len(args) > 0 {
		os = args[0]
	}

	screen := b.BuildSuccessScreen(user.ID, os)
	return c.Edit(screen.Text, screen.Keyboard)
}

func (b *Bot) SendSetupInstruction(userID int64, os string) error {
	user := &telebot.User{ID: userID}

	screen := b.BuildSetupScreen(userID, os)
	return b.SendMessage(user.ID, screen)
}

func (b *Bot) SendPostImportInstructions(userID int64, os string) error {
	user := &telebot.User{ID: userID}

	screen := b.BuildPostImportInstructionsScreen(userID, os)
	return b.SendMessage(user.ID, screen)
}

func (b *Bot) handleManualSetup(c telebot.Context) error {
	user := c.Sender()

	var os string
	args := b.GetArgsFromCtx(c)
	if len(args) > 0 {
		os = args[0]
	}

	screen := b.BuildManualSetupScreen(user.ID, os)
	return c.Edit(screen.Text, screen.Keyboard)
}
