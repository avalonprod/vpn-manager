package bot

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"gopkg.in/telebot.v4"
)

const (
	StartScreen                  = "start_screen"
	TrialAccessScreen            = "trial_access_screen"
	SubscriptionsScreen          = "subscriptions_screen"
	AppListScreen                = "app_list_screen"
	SuccessScreen                = "success_screen"
	PostImportInstructionScreen  = "post_import_instruction_screen"
	SetupInstructionScreen       = "setup_instruction_screen"
	ManualSetupScreen            = "manual_setup_screen"
	SuccessPaymentScreen         = "success_payment_screen"
	SubscriptionManagementScreen = "subscription_management_screen"
	CancelSubscriptionScreen     = "cancel_subscription_screen"
	TrialPeriodExpiryScreen      = "trial_period_expiry_screen"
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
	Context  string
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
	return b.stackStore.Push(context.Background(), userID, screen.Context)
}

func (b *Bot) EditMessage(c telebot.Context, screen *Screen) error {
	err := c.Edit(screen.Text, screen.Keyboard)
	if err != nil {
		return err
	}

	b.pushMessage(c.Sender().ID, c.Message())
	return b.stackStore.Push(context.Background(), c.Sender().ID, screen.Context)
}

func (b *Bot) BuildStartScreen(userID int64) *Screen {
	keyword := &telebot.ReplyMarkup{}

	keyword.Inline(
		telebot.Row{
			b.navigateButtonTo("Попробовать бесплатно", TrialAccessScreen),
		},
		telebot.Row{
			b.navigateButtonTo("Купить от 190р в месяц", SubscriptionsScreen),
		},
		telebot.Row{
			ofertaButton,
		},
		telebot.Row{
			privacyPolicyButton,
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
		Context:  StartScreen,
	}
}

func (b *Bot) BuildTrialAccessScreen(ctx context.Context, userID int64) (*Screen, error) {
	subscription, err := b.subscriptionsService.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	keyword := &telebot.ReplyMarkup{}
	keyword.Inline(
		telebot.Row{
			b.navigateButtonTo("Скачать приложение", AppListScreen),
		},
		telebot.Row{
			b.backButton(),
			b.navigateButtonTo("Продлить подписку", SubscriptionsScreen),
		},
	)

	text := fmt.Sprintf(`
🚀 <b>Ваша подписка активна до %s</b>

Бесплатный доступ: 3 дня

Вы можете пользоваться ВПН на всех ваших устройствах, без ограничений.

❗️<b>Для начала скачайте наше приложение:</b>
	`, subscription.ExpiresAt.Format(time.DateOnly))

	return &Screen{
		Text:     text,
		Keyboard: keyword,
		Context:  TrialAccessScreen,
	}, nil
}

func (b *Bot) BuildAppsListScreen(userID int64) *Screen {
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
			b.backButton(),
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
		Context:  AppListScreen,
	}
}

func (b *Bot) BuildSubscriptionsScreen(ctx context.Context, userID int64, args ...string) (*Screen, error) {
	keyword := &telebot.ReplyMarkup{}
	plans, err := b.plansService.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	rows := make([]telebot.Row, 0, len(plans))

	for _, plan := range plans {
		row := telebot.Row{
			{
				Text: plan.Title,
				URL:  fmt.Sprintf(`%s/subscribe?plan=%s&user_id=%d`, b.apiUrl, plan.ID, userID),
			},
		}
		rows = append(rows, row)
	}

	rows = append(rows, telebot.Row{
		b.backButton(args...),
		supportButton,
	})

	keyword.Inline(rows...)

	text := `
<b>Выберите срок подписки.</b>

Чем больше срок подписки, тем ниже стоимость одного месяца.
	`

	return &Screen{
		Text:     text,
		Keyboard: keyword,
		Context:  SubscriptionsScreen,
	}, nil
}

func (b *Bot) BuildSuccessScreen(ctx context.Context, userID int64, os string) (*Screen, error) {
	subscription, err := b.subscriptionsService.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	keyword := &telebot.ReplyMarkup{}

	keyword.Inline(
		telebot.Row{
			b.navigateButtonTo("Скачать приложение", AppListScreen),
		},
		telebot.Row{
			b.navigateButtonTo("Продлить доступ", SubscriptionsScreen, os),
		},
		telebot.Row{
			supportButton,
		},
		telebot.Row{
			ofertaButton,
		},
		telebot.Row{
			privacyPolicyButton,
		},
	)

	text := fmt.Sprintf(`
Если возникунт проблемы пишите нам

Подписка действует до %s
	`, subscription.ExpiresAt.Format(time.DateOnly))

	return &Screen{
		Text:     text,
		Keyboard: keyword,
		Context:  SuccessScreen,
	}, nil
}

func (b *Bot) BuildSetupScreen(userID int64, os string) *Screen {
	keyword := &telebot.ReplyMarkup{}

	keyword.Inline(
		telebot.Row{
			{Text: "Автонастройка", URL: fmt.Sprintf("%s/setup?user_id=%d&os=%s", b.apiUrl, userID, os)},
		},
		telebot.Row{
			b.backButton(),
		},
	)

	text := `
🔔 Скачали приложение? Возвращайтесь и запустите работу прокси.	
	`

	return &Screen{
		Text:     text,
		Keyboard: keyword,
		Context:  SetupInstructionScreen,
	}
}

func (b *Bot) BuildPostImportInstructionsScreen(userID int64, os string) *Screen {
	keyword := &telebot.ReplyMarkup{}

	keyword.Inline(
		telebot.Row{
			b.navigateButtonTo("Все заработало спасибо", SuccessScreen, os),
		},
		telebot.Row{
			b.navigateButtonTo("Ручная настройка", ManualSetupScreen, os),
		},
		telebot.Row{
			supportButton,
		},
		telebot.Row{
			b.backButton(os),
		},
	)

	text := `
✅ Все готово, нажмите на регион в приложении и VPN будет активирован.

Если нужна помощь, напишите нам в поддержку, мы оперативно поможем!
`

	return &Screen{
		Text:     text,
		Keyboard: keyword,
		Context:  PostImportInstructionScreen,
	}
}

func (b *Bot) BuildManualSetupScreen(userID int64, os string) *Screen {
	keyword := &telebot.ReplyMarkup{}

	keyword.Inline(
		telebot.Row{
			b.backButton(os),
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
		Context:  ManualSetupScreen,
	}
}

func (b *Bot) BuildSuccessPaymentScreen(ctx context.Context, userID int64) (*Screen, error) {
	subscription, err := b.subscriptionsService.GetByUserID(context.Background(), userID)
	if err != nil {
		return nil, err
	}

	keyword := &telebot.ReplyMarkup{}

	keyword.Inline(
		telebot.Row{
			b.navigateButtonTo("Скачать приложение", AppListScreen),
		},
		telebot.Row{
			b.navigateButtonTo("Управление подпиской", SubscriptionManagementScreen),
		},
		telebot.Row{
			supportButton,
		},
		telebot.Row{
			ofertaButton,
		},
		telebot.Row{
			privacyPolicyButton,
		},
	)

	text := fmt.Sprintf(`
Ваша подписка активна до %s
Вы можете пользоваться ВПН на всех ваших устройствах, без ограничений.
	`, subscription.ExpiresAt.Format(time.DateOnly))

	return &Screen{
		Text:     text,
		Keyboard: keyword,
		Context:  SuccessPaymentScreen,
	}, nil
}

func (b *Bot) BuildSubscriptionManagementScreen(ctx context.Context, userID int64) (*Screen, error) {
	subscription, err := b.subscriptionsService.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	plan, err := b.plansService.GetByID(ctx, subscription.PlanID)
	if err != nil {
		return nil, err
	}

	keyword := &telebot.ReplyMarkup{}

	keyword.Inline(
		telebot.Row{
			b.navigateButtonTo("Отменить подписку", CancelSubscriptionScreen),
		},
		telebot.Row{},
		telebot.Row{
			b.backButton(),
		},
	)

	text := fmt.Sprintf(
		`
Управление подпиской
Подписка продлевается автоматически.

Ваш тариф: %s
	`, plan.SubTitle)

	return &Screen{
		Text:     text,
		Keyboard: keyword,
		Context:  SubscriptionManagementScreen,
	}, nil
}

func (b *Bot) BuildCancelSubscriptionScreen(ctx context.Context, userID int64) (*Screen, error) {
	subscription, err := b.subscriptionsService.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	keyword := &telebot.ReplyMarkup{}

	keyword.Inline(
		telebot.Row{
			b.navigateButtonTo("Возобновить подписку", SubscriptionsScreen),
		},
		telebot.Row{
			b.backButton(),
		},
	)

	text := fmt.Sprintf(`
Управление подпиской

Вы отменили подписку.

Можете пользоваться VPN до: %s
	`, subscription.ExpiresAt.Format(time.DateOnly))

	return &Screen{
		Text:     text,
		Keyboard: keyword,
		Context:  CancelSubscriptionScreen,
	}, nil
}

func (b *Bot) BuildTrialPeriodExpiryScreen(ctx context.Context, userID int64) *Screen {
	keyword := &telebot.ReplyMarkup{}

	keyword.Inline(
		telebot.Row{
			b.navigateButtonTo("Продлить доступ", SubscriptionsScreen),
		},
		telebot.Row{
			b.backButton(),
		},
	)

	text := `
🔔 Ну как вам? Тестовый период подошел концу. 

Если все понравилось, вы можете продлить доступ и получить до полугода в подарок. ⬇️
	`

	return &Screen{
		Text:     text,
		Keyboard: keyword,
		Context:  TrialPeriodExpiryScreen,
	}
}

func (b *Bot) handleBack(c telebot.Context) error {
	user := c.Sender()
	_ = c.Respond()

	args := c.Args()
	lenArgs := len(args)

	screen, err := b.stackStore.PopAndPeek(context.Background(), user.ID)
	if err != nil {
		return err
	}

	switch screen {
	case StartScreen:
		screen := b.BuildStartScreen(user.ID)
		return c.Edit(screen.Text, screen.Keyboard)
	case TrialAccessScreen:
		screen, err := b.BuildTrialAccessScreen(context.Background(), user.ID)
		if err != nil {
			return err
		}
		return c.Edit(screen.Text, screen.Keyboard)
	case AppListScreen:
		screen := b.BuildAppsListScreen(user.ID)
		return c.Edit(screen.Text, screen.Keyboard)
	case SuccessScreen:
		if lenArgs != 0 {
			screen, err := b.BuildSuccessScreen(context.Background(), user.ID, args[0])
			if err != nil {
				return err
			}
			return c.Edit(screen.Text, screen.Keyboard)
		}
	case SetupInstructionScreen:
		if lenArgs != 0 {
			screen := b.BuildSetupScreen(user.ID, args[0])
			return c.Edit(screen.Text, screen.Keyboard)
		}
	case PostImportInstructionScreen:
		if lenArgs != 0 {
			screen := b.BuildPostImportInstructionsScreen(user.ID, args[0])
			return c.Edit(screen.Text, screen.Keyboard)
		}
	case SuccessPaymentScreen:
		screen, err := b.BuildSuccessPaymentScreen(context.Background(), user.ID)
		if err != nil {
			return err
		}
		return c.Edit(screen.Text, screen.Keyboard)
	case SubscriptionManagementScreen:
		screen, err := b.BuildSubscriptionManagementScreen(context.Background(), user.ID)
		if err != nil {
			return err
		}
		return c.Edit(screen.Text, screen.Keyboard)
	case CancelSubscriptionScreen:
		screen, err := b.BuildCancelSubscriptionScreen(context.Background(), user.ID)
		if err != nil {
			return err
		}
		return c.Edit(screen.Text, screen.Keyboard)
	default:
		return nil
	}

	return nil
}
