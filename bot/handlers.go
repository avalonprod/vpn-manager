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
	if err := b.SendMessage(user.ID, screen); err != nil {
		b.logger.Error(err)
		return err
	}
	return nil
}

func (b *Bot) handleTrialAccess(c telebot.Context) error {
	ctx := context.Background()
	user := c.Sender()
	err := b.subscriptionsService.CreateTrialSubscription(ctx, c.Sender().ID)
	if err != nil {
		b.logger.Error(err)
		return b.replyError(c, ErrDefault)
	}

	if err := b.peersService.ActivatePeer(ctx, user.ID); err != nil {
		b.logger.Error(err)
		return b.replyError(c, ErrDefault)
	}

	screen, err := b.BuildTrialAccessScreen(ctx, user.ID)
	if err != nil {
		b.logger.Error(err)
		return err
	}

	if err := b.EditMessage(c, screen); err != nil {
		b.logger.Error(err)
		return err
	}

	return nil
}

func (b *Bot) handleAppsList(c telebot.Context) error {
	user := c.Sender()

	screen := b.BuildAppsListScreen(user.ID)
	if err := b.EditMessage(c, screen); err != nil {
		b.logger.Error(err)
		return err
	}
	return nil
}

func (b *Bot) handleSubscribe(c telebot.Context) error {
	user := c.Sender()

	screen, err := b.BuildSubscriptionsScreen(context.Background(), user.ID, c.Args()...)
	if err != nil {
		b.logger.Error(err)
		return b.replyError(c, ErrDefault)
	}
	if err := b.EditMessage(c, screen); err != nil {
		b.logger.Error(err)
		return err
	}
	return nil
}

func (b *Bot) handleSuccess(c telebot.Context) error {
	user := c.Sender()

	var os string
	args := c.Args()
	if len(args) > 0 {
		os = args[0]
	}

	screen, err := b.BuildSuccessScreen(context.Background(), user.ID, os)
	if err != nil {
		b.logger.Error(err)
		return err
	}

	if err := b.EditMessage(c, screen); err != nil {
		b.logger.Error(err)
		return err
	}
	return nil
}

func (b *Bot) handleCancelSubscription(c telebot.Context) error {
	user := c.Sender()

	screen, err := b.BuildCancelSubscriptionScreen(context.Background(), user.ID)
	if err != nil {
		b.logger.Error(err)
		return b.replyError(c, ErrDefault)
	}

	if err := b.EditMessage(c, screen); err != nil {
		b.logger.Error(err)
		return err
	}

	return nil
}

func (b *Bot) handleSubscriptionManagement(c telebot.Context) error {
	user := c.Sender()

	screen, err := b.BuildSubscriptionManagementScreen(context.Background(), user.ID)
	if err != nil {
		b.logger.Error(err)
		return b.replyError(c, ErrDefault)
	}

	if err := b.EditMessage(c, screen); err != nil {
		b.logger.Error(err)
		return err
	}
	return nil
}

func (b *Bot) SendSetupInstruction(userID int64, os string) error {
	user := &telebot.User{ID: userID}

	screen := b.BuildSetupScreen(userID, os)
	if err := b.SendMessage(user.ID, screen); err != nil {
		b.logger.Error(err)
		return err
	}
	return nil
}

func (b *Bot) SendPostImportInstructions(userID int64, os string) error {
	user := &telebot.User{ID: userID}

	screen := b.BuildPostImportInstructionsScreen(userID, os)
	if err := b.SendMessage(user.ID, screen); err != nil {
		b.logger.Error(err)
		return err
	}
	return nil
}

func (b *Bot) SendSuccessPayment(userID int64) error {
	user := &telebot.User{ID: userID}

	screen, err := b.BuildSuccessPaymentScreen(context.Background(), user.ID)
	if err != nil {
		return err
	}

	if err := b.SendMessage(user.ID, screen); err != nil {
		b.logger.Error(err)
		return err
	}
	return nil
}

func (b *Bot) handleManualSetup(c telebot.Context) error {
	user := c.Sender()

	var os string
	args := c.Args()
	if len(args) > 0 {
		os = args[0]
	}

	screen := b.BuildManualSetupScreen(user.ID, os)
	if err := b.EditMessage(c, screen); err != nil {
		b.logger.Error(err)
		return err
	}
	return nil
}
