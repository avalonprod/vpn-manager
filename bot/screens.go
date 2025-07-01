package bot

import (
	"fmt"
	"strconv"

	"gopkg.in/telebot.v4"
)

const (
	StartScreen                 = "start_screen"
	TrialAccessScreen           = "trial_access_screen"
	SubscriptionsScreen         = "subscriptions_screen"
	AppListScreen               = "app_list_screen"
	SuccessScreen               = "success_screen"
	PostImportInstructionScreen = "post_import_instruction_screen"
	SetupInstructionScreen      = "setup_instruction_screen"
	ManualSetupScreen           = "manual_setup_screen"
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

func (b *Bot) SendMessage(userID int64, screen *Screen) error {
	msg, err := b.bot.Send(&telebot.User{ID: userID}, screen.Text, screen.Keyboard)
	if err != nil {
		return err
	}

	if err := b.clearMessages(userID); err != nil {
		return err
	}

	b.pushMessage(userID, msg)
	return nil
}

func (b *Bot) GenerateStartScreen(userID int64) *Screen {
	keyword := &telebot.ReplyMarkup{}

	keyword.Inline(
		telebot.Row{
			b.navigateButtonTo("Попробовать бесплатно", TrialAccessScreen, StartScreen),
		},
		telebot.Row{
			b.navigateButtonTo("Купить от 190р в месяц", SubscriptionsScreen, StartScreen),
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
			b.navigateButtonTo("Скачать приложение", AppListScreen, TrialAccessScreen),
		},
		telebot.Row{
			b.backButtonTo(StartScreen),
			b.navigateButtonTo("Продлить подписку", SubscriptionsScreen, TrialAccessScreen),
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

func (b *Bot) GenerateAppsListScreen(userID int64) *Screen {
	keyword := &telebot.ReplyMarkup{}

	keyword.Inline(
		telebot.Row{
			{Text: "💻 MacOs", URL: fmt.Sprintf("%s/apps?user_id=%d&os=macos", b.apiUrl, userID)},
		},
		telebot.Row{
			{Text: "📱 iPhone / iPad", URL: fmt.Sprintf("%s/apps?user_id=%d&os=ios", b.apiUrl, userID)},
			{Text: "📱 Android", URL: fmt.Sprintf("%s/apps?user_id=%d&os=android", b.apiUrl, userID)},
		},
		telebot.Row{
			b.backButtonTo(TrialAccessScreen),
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

func (b *Bot) GenerateSubscriptionsScreen(userID int64, context string, args ...string) *Screen {
	keyword := &telebot.ReplyMarkup{}

	keyword.Inline(
		telebot.Row{
			{Text: "💳 1 месяц - 500 ₽", URL: "https://c.cloudpayments.ru/payments/1981efead06a4baab02a1bc2f23ca920"},
		},
		telebot.Row{
			{Text: "💳 6 мес. + 2 мес. 🎁 - 350 ₽ / мес", URL: "https://c.cloudpayments.ru/payments/a0eb9ffcef944777a668c071530486ff"},
		},
		telebot.Row{
			{Text: "💳 1 год + 3 мес. 🎁 - 280 ₽ / мес", URL: "https://c.cloudpayments.ru/payments/5d6126e5ca4c4b7689bdc8693d78c9be"},
		},
		telebot.Row{
			{Text: "💳 2 года + 6 мес. 🎁 - 190 ₽ / мес", URL: "https://c.cloudpayments.ru/payments/1588814b133d465f859639af353101db"},
		},
		telebot.Row{
			b.backButtonTo(context, args...),
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

func (b *Bot) GenerateSuccessScreen(userID int64, os string) *Screen {
	keyword := &telebot.ReplyMarkup{}

	keyword.Inline(
		telebot.Row{
			b.navigateButtonTo("Продлить доступ", SubscriptionsScreen, SuccessScreen, os),
			supportButton,
		},
		telebot.Row{
			b.backButtonTo(PostImportInstructionScreen, os),
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

func (b *Bot) GenerateSetupScreen(userID int64, os string) *Screen {
	keyword := &telebot.ReplyMarkup{}

	keyword.Inline(
		telebot.Row{
			{Text: "Автонастройка", URL: fmt.Sprintf("%s/setup?user_id=%d&os=%s", b.apiUrl, userID, os)},
		},
		telebot.Row{
			b.backButtonTo(AppListScreen),
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

func (b *Bot) GeneratePostImportInstructionsScreen(userID int64, os string) *Screen {
	keyword := &telebot.ReplyMarkup{}

	keyword.Inline(
		telebot.Row{
			b.navigateButtonTo("Все заработало спасибо", SuccessScreen, PostImportInstructionScreen, os),
		},
		telebot.Row{
			b.navigateButtonTo("Ручная настройка", ManualSetupScreen, PostImportInstructionScreen, os),
		},
		telebot.Row{
			supportButton,
		},
		telebot.Row{
			b.backButtonTo(SetupInstructionScreen, os),
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

func (b *Bot) GenerateManualSetupScreen(userID int64, os string) *Screen {
	keyword := &telebot.ReplyMarkup{}

	keyword.Inline(
		telebot.Row{
			b.backButtonTo(PostImportInstructionScreen, os),
		},
	)

	url := fmt.Sprintf("%s/subs?user_id=%d&name=%s", b.apiUrl, userID, "NeonGuard")

	text := fmt.Sprintf(`
Ручная настройка очень простая и займет меньше минуты:

1. Скопируй эту ссылку: %s

2. Зайди в приложение Streisand на любом устройстве.

3. Нажми на "+" как на первом скриншоте.

4. И выбери Import from clipboard

5. Если возникли сложности — пиши нам в поддержку, мы оперативно поможем.
`, url)

	return &Screen{
		Text:     text,
		Keyboard: keyword,
	}
}

func (b *Bot) handleBack(c telebot.Context) error {
	user := c.Sender()
	_ = c.Respond()

	args := c.Args()
	if len(args) == 0 {
		return nil
	}

	screen := args[0]
	data := args[1:]

	switch screen {
	case StartScreen:
		screen := b.GenerateStartScreen(user.ID)
		return c.Edit(screen.Text, screen.Keyboard)
	case TrialAccessScreen:
		screen := b.GenerateTrialAccessScreen(user.ID)
		return c.Edit(screen.Text, screen.Keyboard)
	case AppListScreen:
		screen := b.GenerateAppsListScreen(user.ID)
		return c.Edit(screen.Text, screen.Keyboard)
	case SuccessScreen:
		if len(data) != 0 {
			screen := b.GenerateSuccessScreen(user.ID, data[0])
			return c.Edit(screen.Text, screen.Keyboard)
		}
	case SetupInstructionScreen:
		if len(data) != 0 {
			screen := b.GenerateSetupScreen(user.ID, data[0])
			return c.Edit(screen.Text, screen.Keyboard)
		}
	case PostImportInstructionScreen:
		if len(data) != 0 {
			screen := b.GeneratePostImportInstructionsScreen(user.ID, data[0])
			return c.Edit(screen.Text, screen.Keyboard)
		}
	default:
		return nil
	}

	return nil
}
