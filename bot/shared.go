package bot

import "gopkg.in/telebot.v4"

const (
	ErrDefault = "Что-то пошло не так. Мы уже работаем над этим!"
)

var (
	selector            = &telebot.ReplyMarkup{}
	supportButton       = selector.URL("Поддержка", "t.me/neonguard_support")
	ofertaButton        = selector.URL("Оферта", "https://neonguard.ru/oferta")
	privacyPolicyButton = selector.URL("Политика конфиденциальности", "https://neonguard.ru/popd")
)

func (b *Bot) replyError(c telebot.Context, msg string) error {
	return c.Send(msg)
}

func (b *Bot) backButton(args ...string) telebot.Btn {
	return selector.Data("Назад", "back", args...)
}

func (b *Bot) navigateButtonTo(text, screen string, args ...string) telebot.Btn {
	return selector.Data(text, screen, args...)
}
