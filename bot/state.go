package bot

import (
	"reflect"

	"gopkg.in/telebot.v4"
)

type Screen struct {
	Text     string
	Keyboard *telebot.ReplyMarkup
}

var state = map[int64][]Screen{}

func (b *Bot) pushState(userID int64, screen Screen) {
	userState := state[userID]

	if len(userState) > 0 {
		last := userState[len(userState)-1]
		if last.Text == screen.Text && reflect.DeepEqual(last.Keyboard, screen.Keyboard) {
			return
		}
	}
	state[userID] = append(state[userID], screen)
}

func (b *Bot) popScreen(userID int64) *Screen {
	userState := state[userID]
	if len(userState) <= 1 {
		return &userState[0]
	}

	userState = userState[:len(userState)-1]
	state[userID] = userState

	return &userState[len(userState)-1]
}

func renderScreen(c telebot.Context, screen *Screen) error {
	return c.Edit(screen.Text, screen.Keyboard)
}

func (b *Bot) handleBack(c telebot.Context) error {
	_ = c.Respond()
	prev := b.popScreen(c.Sender().ID)
	return renderScreen(c, prev)
}
