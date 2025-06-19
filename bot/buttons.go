package bot

import "gopkg.in/telebot.v4"

var (
	selector = &telebot.ReplyMarkup{}

	trialAccessButton       = selector.Data("Попробовать бесплатно", "trial_access")
	appsListButton          = selector.Data("Скачать приложение", "apps_list")
	subscribeButton         = selector.Data("Купить от 190р в месяц", "subscribe")
	renewSubscriptionButton = selector.Data("Продлить подписку", "renew_subscribe")
	supportButton           = selector.URL("Поддержка", "https://t.me/neonguard_support")
	manualSettingsButton    = selector.Data("Ручные настройки", "manual_settings")
	successButton           = selector.Data("Все заработало спасибо", "success")
	backButton              = selector.Data("Назад", "back")
)
