package bot

import (
	"fmt"
	"strconv"

	"gopkg.in/telebot.v4"
)

var store = map[int64][]telebot.StoredMessage{}

func (b *Bot) clearMessages(userID int64) error {
	messages := store[userID]

	editables := make([]telebot.Editable, len(messages))
	for i, msg := range messages {
		editables[i] = msg
	}

	if len(editables) == 0 {
		return nil
	}

	store[userID] = []telebot.StoredMessage{}

	return b.bot.DeleteMany(editables)
}

func (b *Bot) pushMessage(userID int64, msg *telebot.Message) {
	if _, ok := store[userID]; !ok {
		store[userID] = []telebot.StoredMessage{}
	}

	store[userID] = append(store[userID], telebot.StoredMessage{
		ChatID:    msg.Chat.ID,
		MessageID: strconv.Itoa(msg.ID),
	})
}

type Screen struct {
	Text     string
	Keyboard *telebot.ReplyMarkup
}

func (b *Bot) GenerateStartScreen(userID int64) *Screen {
	keyword := &telebot.ReplyMarkup{}

	keyword.Inline(
		telebot.Row{
			trialAccessButton,
		},
		telebot.Row{
			subscribeButton,
		},
	)

	text := `
🔥 <b>NeonGuard</b> — твой интернет без ограничений

📡 Мгновенное подключение. Без логов. Без следов.

🛫 Доступ к зарубежным сайтам и приложениям.

🚀 Настройка занимает меньше минуты — всё готово.

<b>NeonGuard</b> - Это быстрый, безопасный и простой способ подключиться к миру.	
	`

	return &Screen{
		Text:     text,
		Keyboard: keyword,
	}
}

func (b *Bot) GenerateTrialAccessScreen(userID int64) *Screen {
	keyword := &telebot.ReplyMarkup{}
	keyword.Inline(
		telebot.Row{
			appsListButton,
		},
		telebot.Row{
			b.backButtonTo("start"),
			renewSubscriptionButton,
		},
	)

	text := `
🚀 <b>Ваша подписка активна до 07.06.2025</b>

Бесплатный доступ: 3 дня

Вы можете пользоваться ВПН на всех ваших устройствах, без ограничений.

❗️<b>Для начала скачайте наше приложение:</b>
	`

	return &Screen{
		Text:     text,
		Keyboard: keyword,
	}
}

func (b *Bot) GenerateSupportScreen(userID int64) *Screen {
	text := `
Если что-то не работает, не получается подключиться или возникли вопросы — мы на связи.

📩 Напиши нам прямо сюда:
@neonguard_support
	`

	return &Screen{
		Text:     text,
		Keyboard: nil,
	}
}

func (b *Bot) GenerateAppsListScreen(userID int64) *Screen {
	keyword := &telebot.ReplyMarkup{}

	keyword.Inline(
		telebot.Row{
			{Text: "📱 iPhone / iPad", URL: fmt.Sprintf("%s/apps?user_id=%d&os=ios", b.apiUrl, userID)},
			{Text: "💻 MacOs", URL: fmt.Sprintf("%s/apps?user_id=%d&os=macos", b.apiUrl, userID)},
		},
		telebot.Row{
			b.backButtonTo("trial_access"),
			supportButton,
		},
	)

	text := `
Выберите устройство.

Вы можете одновременно использовать ВПН на 4-х девайсах.
	`

	return &Screen{
		Text:     text,
		Keyboard: keyword,
	}
}

func (b *Bot) GenerateSubscriptionsScreen(userID int64) *Screen {
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
			b.backButtonTo("start"),
			supportButton,
		},
	)

	text := `
<b>Выберите срок подписки.</b>

Чем больше срок подписки, тем ниже стоимость одного месяца.
	`

	return &Screen{
		Text:     text,
		Keyboard: keyword,
	}
}

func (b *Bot) GenerateRenewSubscriptionsScreen(userID int64) *Screen {
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
			b.backButtonTo("trial_access"),
			supportButton,
		},
	)

	text := `
<b>Выберите срок подписки.</b>

Чем больше срок подписки, тем ниже стоимость одного месяца.
	`

	return &Screen{
		Text:     text,
		Keyboard: keyword,
	}
}

func (b *Bot) GenerateSuccessScreen(userID int64) *Screen {
	keyword := &telebot.ReplyMarkup{}

	keyword.Inline(
		telebot.Row{
			renewSubscriptionButton,
			supportButton,
		},
	)

	text := `
Если возникунт проблемы пишите нам

Подписка действует до 07.06.2025
	`

	return &Screen{
		Text:     text,
		Keyboard: keyword,
	}
}

func (b *Bot) GenerateSetupScreen(userID int64) *Screen {
	keyword := &telebot.ReplyMarkup{}

	keyword.Inline(
		telebot.Row{
			{Text: "Автонастройка", URL: fmt.Sprintf("%s/setup?user_id=%d&os=ios", b.apiUrl, userID)},
		},
		telebot.Row{
			b.backButtonTo("apps_list"),
		},
	)

	text := `
Если скачали приложение, нажмите Автонастройка и все настроится автоматически.		
	`

	return &Screen{
		Text:     text,
		Keyboard: keyword,
	}
}

func (b *Bot) GeneratePostImportInstructionsScreen(userID int64) *Screen {
	keyword := &telebot.ReplyMarkup{}

	keyword.Inline(
		telebot.Row{
			successButton,
		},
		telebot.Row{
			supportButton,
		},
	)

	text := `
✅ Все готово, нажмите на регион в приложении и VPN будет активирован.

Если нужна помощь, напишите нам в поддержку, мы оперативно поможем!
`

	return &Screen{
		Text:     text,
		Keyboard: keyword,
	}
}

func (b *Bot) handleBack(c telebot.Context) error {
	user := c.Sender()
	_ = c.Respond()

	screen := c.Data()

	switch screen {
	case "start":
		screen := b.GenerateStartScreen(user.ID)
		return c.Edit(screen.Text, screen.Keyboard)
	case "trial_access":
		screen := b.GenerateTrialAccessScreen(user.ID)
		return c.Edit(screen.Text, screen.Keyboard)
	case "apps_list":
		screen := b.GenerateAppsListScreen(user.ID)
		return c.Edit(screen.Text, screen.Keyboard)
	}

	return nil
}
