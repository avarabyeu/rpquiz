package telegram

import (
	"context"
	"github.com/apex/log"
	"github.com/avarabyeu/rpquiz/bot/engine"
	"github.com/avarabyeu/rpquiz/bot/engine/ctx"
	"gopkg.in/telegram-bot-api.v4"
	"strconv"
)

//Bot is telegram bot abstraction
type Bot struct {
	Token      string
	Dispatcher *bot.Dispatcher
}

//Start connects to telegram servers and starts listening
func (b *Bot) Start() error {
	tBot, err := tgbotapi.NewBotAPI(b.Token)
	if err != nil {
		return err
	}

	//tBot.Debug = true

	log.Debugf("Authorized on account %s", tBot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	go func() {
		updates, err := tBot.GetUpdatesChan(u)
		if nil != err {
			log.WithError(err).Error(err.Error())
		}

		for update := range updates {

			var message string
			var tMessage *tgbotapi.Message
			var user string
			var userID string

			callback := false

			if update.Message != nil {
				message = update.Message.Text
				tMessage = update.Message
				user = update.Message.From.UserName
				userID = strconv.Itoa(update.Message.From.ID)
			} else if update.CallbackQuery != nil {
				message = update.CallbackQuery.Data
				tMessage = update.CallbackQuery.Message
				user = update.CallbackQuery.From.UserName
				userID = strconv.Itoa(update.CallbackQuery.From.ID)
				callback = true
			} else {
				continue
			}

			if "" != user {
				log.Debugf("[%s] %s", user, message)
			}

			go func(update *tgbotapi.Message) {

				ctx := botctx.WithOriginalMessage(context.Background(), update)
				ctx = botctx.WithUserName(ctx, user)
				ctx = botctx.WithUserID(ctx, userID)
				ctx, cancel := context.WithCancel(ctx)
				defer cancel()
				rss := b.Dispatcher.Dispatch(ctx, message, callback)
				reply(tBot, update, rss)
			}(tMessage)

		}
	}()
	return nil

}

func reply(bot *tgbotapi.BotAPI, m *tgbotapi.Message, rss []*bot.Response) {
	for _, rs := range rss {
		msg := tgbotapi.NewMessage(m.Chat.ID, rs.Text)
		//msg.ReplyToMessageID = m.MessageID
		msg.ParseMode = "Markdown"

		buttonsCount := len(rs.Buttons)
		if buttonsCount > 0 {
			inlineBtns := make([][]tgbotapi.InlineKeyboardButton, buttonsCount)
			for i, btn := range rs.Buttons {
				inlineBtns[i] = []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(btn.Text, btn.Data)}
			}

			msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(inlineBtns...)
		}

		if _, err := bot.Send(msg); nil != err {
			log.WithError(err).Error(err.Error())
		}
	}

}
