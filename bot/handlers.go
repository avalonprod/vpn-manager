package bot

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"vpn-manager/subscriptions"
	"vpn-manager/users"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (tb *TGBot) handleStart(ctx context.Context, b *bot.Bot, update *models.Update) {
	if err := tb.usersService.Register(ctx, users.CreateUserInput{
		ID:        update.Message.From.ID,
		Username:  update.Message.From.Username,
		FirstName: update.Message.From.FirstName,
	}); err != nil {
		tb.logger.Error(err)
		tb.sendError(ctx, b, update.Message.Chat.ID, ErrDefault)
		return
	}

	activateTrialAccessButton := &models.InlineKeyboardButton{
		Text:         "Попробовать бесплатно",
		CallbackData: ActivateTrialAccessCallback,
	}

	buySubscriptionButton := &models.InlineKeyboardButton{
		Text:         "Купить от 190р в месяц",
		CallbackData: SubscribeCallback,
	}

	keyboard := models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{*activateTrialAccessButton},
			{*buySubscriptionButton},
		},
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        Msg.AlertStart,
		ReplyMarkup: &keyboard,
		ParseMode:   models.ParseModeHTML,
	})

}

func (tb *TGBot) handleSupport(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   Msg.AlertSupport,
	})
}

func (tb *TGBot) handleSupportCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.CallbackQuery.Message.Message.Chat.ID,
		Text:   Msg.AlertSupport,
	})

	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})
}

func (tb *TGBot) handleActivateTrialAccessCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	err := tb.subscriptionsService.CreateTrialSubscription(ctx, update.CallbackQuery.Message.Message.Chat.ID)
	if err != nil {
		if errors.Is(err, subscriptions.ErrTrialAccessAlreadyActivated) {
			tb.logger.Info(err)
			tb.sendError(ctx, b, update.CallbackQuery.Message.Message.Chat.ID, ErrTrialAccessAlreadyActivated)
			return
		}
		tb.logger.Error(err)
		tb.sendError(ctx, b, update.CallbackQuery.Message.Message.Chat.ID, ErrDefault)
		return
	}

	activateTrialAccessButton := &models.InlineKeyboardButton{
		Text:         "Выбрать регион",
		CallbackData: LocationListCallback,
	}

	buySubscriptionButton := &models.InlineKeyboardButton{
		Text:         "Продлить подписку",
		CallbackData: SubscribeCallback,
	}

	keyboard := models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{*activateTrialAccessButton},
			{*buySubscriptionButton},
		},
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.CallbackQuery.Message.Message.Chat.ID,
		Text:        Msg.AlertTrialAccess,
		ReplyMarkup: &keyboard,
		ParseMode:   models.ParseModeHTML,
	})

	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})
}

func (tb *TGBot) handleSubscribeCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	oneMonthSubscription := &models.InlineKeyboardButton{
		Text:         "💳 1 месяц - 500 ₽",
		CallbackData: AppListCallback,
	}

	sixMonthSubscription := &models.InlineKeyboardButton{
		Text:         "💳 6 мес. + 2 мес. 🎁 - 350 ₽ / мес",
		CallbackData: AppListCallback,
	}

	oneYearSubscription := &models.InlineKeyboardButton{
		Text:         "💳 1 год + 3 мес. 🎁 - 280 ₽ / мес",
		CallbackData: AppListCallback,
	}

	twoYearsSubscription := &models.InlineKeyboardButton{
		Text:         "💳 2 года + 6 мес. 🎁 - 190 ₽ / мес",
		CallbackData: AppListCallback,
	}

	supportButton := &models.InlineKeyboardButton{
		Text:         "❓ Поддержка",
		CallbackData: SupportCallback,
	}

	keyboard := models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{*oneMonthSubscription},
			{*sixMonthSubscription},
			{*oneYearSubscription},
			{*twoYearsSubscription},
			{*supportButton},
		},
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.CallbackQuery.Message.Message.Chat.ID,
		Text:        Msg.AlertSubscriptions,
		ReplyMarkup: &keyboard,
		ParseMode:   models.ParseModeHTML,
	})

	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})
}

func (tb *TGBot) handleConnectCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	const prefix = "connect_"
	if !strings.HasPrefix(update.CallbackQuery.Data, prefix) {
		return
	}

	serverID := strings.TrimPrefix(update.CallbackQuery.Data, prefix)
	confContent, err := tb.serversService.RegisterNewPeer(ctx, update.CallbackQuery.Message.Message.Chat.ID, serverID)
	if err != nil {
		tb.logger.Error(err)
		tb.sendError(ctx, b, update.CallbackQuery.Message.Message.Chat.ID, ErrDefault)
	}

	deepLinkButton := &models.InlineKeyboardButton{
		Text: "Автонастройка",
		URL:  confContent.ConfigUrl,
	}

	supportButton := &models.InlineKeyboardButton{
		Text:         "❓ Поддержка",
		CallbackData: SupportCallback,
	}

	keyboard := models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{*deepLinkButton},
			{*supportButton},
		},
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.CallbackQuery.Message.Message.Chat.ID,
		Text:        Msg.AlertListApp,
		ReplyMarkup: &keyboard,
		ParseMode:   models.ParseModeHTML,
	})

	if err != nil {
		tb.logger.Error(err)
		tb.sendError(ctx, b, update.CallbackQuery.Message.Message.Chat.ID, ErrDefault)
	}

	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})
}

func (tb *TGBot) handleLocationsList(ctx context.Context, b *bot.Bot, update *models.Update) {
	servers, err := tb.serversService.GetAllServers(ctx)
	if err != nil {
		tb.logger.Errorf("error get all servers: %v", err)
		tb.sendError(ctx, b, update.CallbackQuery.Message.Message.Chat.ID, ErrDefault)
	}

	buttons := make([][]models.InlineKeyboardButton, 0, len(servers))

	for _, server := range servers {
		serverButton := &models.InlineKeyboardButton{
			Text:         server.Location,
			CallbackData: fmt.Sprintf("connect_%s", server.ID),
		}

		buttons = append(buttons, []models.InlineKeyboardButton{*serverButton})
	}

	keyboard := models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.CallbackQuery.Message.Message.Chat.ID,
		Text:        Msg.AlertLocations,
		ReplyMarkup: &keyboard,
	})

	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})
}
