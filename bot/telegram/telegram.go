package telegram

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"gitlab.com/avarabyeu/rpquiz/bot/engine"
	"gitlab.com/avarabyeu/rpquiz/bot/engine/ctx"
	"gopkg.in/telegram-bot-api.v4"
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

	log.Printf("Authorized on account %s", tBot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	go func() {
		updates, err := tBot.GetUpdatesChan(u)
		if nil != err {
			log.Panic(err)
		}

		for update := range updates {
			fmt.Println("OK")
			//fmt.Println(update.CallbackQuery.Message)
			//fmt.Println(update.CallbackQuery.Data)
			//fmt.Println(update.ChosenInlineResult.InlineMessageID)
			//fmt.Println(update.ChosenInlineResult.Query)
			fmt.Println("OK2")

			var message string
			var tMessage *tgbotapi.Message
			if update.Message != nil {
				message = update.Message.Text
				tMessage = update.Message
			} else if update.CallbackQuery != nil {
				message = update.CallbackQuery.Data
				tMessage = update.CallbackQuery.Message
			} else {
				continue
			}

			if nil != update.Message {
				log.Infof("[%s] %s", update.Message.From.UserName, message)
			}

			go func(update *tgbotapi.Message) {

				ctx, cancel := context.WithCancel(botctx.WithOriginalMessage(context.Background(), update))
				defer cancel()
				rs := b.Dispatcher.Dispatch(ctx, message)
				reply(tBot, update, rs)
			}(tMessage)

		}
	}()
	return nil

}

func reply(bot *tgbotapi.BotAPI, m *tgbotapi.Message, rs *bot.Response) {
	msg := tgbotapi.NewMessage(m.Chat.ID, rs.Text)
	msg.ReplyToMessageID = m.MessageID
	msg.ParseMode = "Markdown"

	buttonsCount := len(rs.Buttons)
	if buttonsCount > 0 {
		inlineBtns := make([]tgbotapi.InlineKeyboardButton, buttonsCount)
		for i, btn := range rs.Buttons {
			inlineBtns[i] = tgbotapi.NewInlineKeyboardButtonData(btn.Text, btn.Data)
		}

		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(inlineBtns)
	}

	if _, err := bot.Send(msg); nil != err {
		log.Error(err)
	}
}
