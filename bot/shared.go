package bot

import "gopkg.in/telebot.v4"

const (
	ErrDefault = "Что-то пошло не так. Мы уже работаем над этим!"
)

var (
	selector      = &telebot.ReplyMarkup{}
	supportButton = selector.URL("Поддержка", "t.me/neonguard_support")
)

func (b *Bot) backButtonTo(screen string, data ...string) telebot.Btn {
	args := []string{screen}
	args = append(args, data...)
	return selector.Data("Назад", "back", args...)
}

func (b *Bot) navigateButtonTo(text, screen string, context string, args ...string) telebot.Btn {
	data := []string{
		context,
	}
	data = append(data, args...)
	return selector.Data(text, screen, data...)
}

func (b *Bot) GetScreenFromCtx(c telebot.Context) string {
	args := c.Args()
	if len(args) > 0 {
		return args[0]
	}

	return ""
}

func (b *Bot) GetArgsFromCtx(c telebot.Context) []string {
	args := c.Args()
	if len(args) > 1 {
		return args[1:]
	}

	return []string{}
}
