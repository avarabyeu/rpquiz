package telegram

import (
	"context"
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
			if update.Message == nil {
				continue
			}
			go func(update tgbotapi.Update) {
				log.Infof("[%s] %s", update.Message.From.UserName, update.Message.Text)

				ctx, cancel := context.WithCancel(botctx.WithOriginalMessage(context.Background(), update))
				defer cancel()
				rs := b.Dispatcher.Dispatch(ctx, update.Message.Text)
				reply(tBot, update.Message, rs.Text)
			}(update)

		}
	}()
	return nil

}

func reply(bot *tgbotapi.BotAPI, m *tgbotapi.Message, text string) {
	msg := tgbotapi.NewMessage(m.Chat.ID, text)
	msg.ReplyToMessageID = m.MessageID
	msg.ParseMode = "Markdown"
	if _, err := bot.Send(msg); nil != err {
		log.Error(err)
	}
}
