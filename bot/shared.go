package bot

import (
	"context"

	"github.com/go-telegram/bot"
)

var Msg = struct {
	AlertStart         string
	AlertSupport       string
	AlertListApp       string
	AlertTrialAccess   string
	AlertSubscriptions string
	AlertLocations     string
}{
	AlertStart: `
🔥 <b>NeonGuard</b> — твой интернет без ограничений

📡 Мгновенное подключение. Без логов. Без следов.

🛫 Доступ к зарубежным сайтам и приложениям.

🚀 Настройка занимает меньше минуты — всё готово.

<b>NeonGuard</b> - Это быстрый, безопасный и простой способ подключиться к миру.
`,
	AlertSupport: `
Если что-то не работает, не получается подключиться или возникли вопросы — мы на связи.

📩 Напиши нам прямо сюда:
@neonguard_support
`,
	AlertTrialAccess: `
<b>Ваша подписка активна до 07.06.2025</b>

Ваша подписка: 3 дня
`,

	AlertListApp: `
Вы можете скачать Streisand из <a href="https://apps.apple.com/us/app/streisand/id6450534064">App Store</a>
Нажмите Автонастройка, и подключитесь к ВПН.
`,
	AlertSubscriptions: `
<b>Выберите срок подписки.</b>

Чем больше срок подписки, тем ниже стоимость одного месяца.
`,

	AlertLocations: "Выберите регион к которому хотите подключиться:",
}

const (
	ActivateTrialAccessCallback = "activate_trial_access"
	SupportCallback             = "support"
	ConnectCallback             = "connect_"
	SubscribeCallback           = "extend_subscribe"
	AppListCallback             = "app_list"
	LocationListCallback        = "location_list"
)

const (
	StartCommand   = "/start"
	SupportCommand = "/support"
)

const (
	ErrDefault                     = "Oops something wrong..."
	ErrTrialAccessAlreadyActivated = "Вы уже активировали пробный период"
)

func (tb *TGBot) sendError(ctx context.Context, b *bot.Bot, chatID int64, msg string) {
	_, err := b.SendMessage(ctx,
		&bot.SendMessageParams{
			ChatID: chatID,
			Text:   msg,
		})

	if err != nil {
		tb.logger.Errorf("failed to send error: %v", err)
	}
}
