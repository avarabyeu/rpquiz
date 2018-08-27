package main

import (
	"context"
	"fmt"
	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/avarabyeu/gorp/gorp"
	"github.com/caarlos0/env"
	"github.com/coreos/bbolt"
	"github.com/go-chi/chi"
	"gitlab.com/avarabyeu/rpquiz/bot/db"
	"gitlab.com/avarabyeu/rpquiz/bot/engine"
	"gitlab.com/avarabyeu/rpquiz/bot/engine/ctx"
	"gitlab.com/avarabyeu/rpquiz/bot/nlp"
	"gitlab.com/avarabyeu/rpquiz/bot/telegram"
	"gitlab.com/avarabyeu/rpquiz/intents"
	"gitlab.com/avarabyeu/rpquiz/rp"
	"go.uber.org/fx"
	"gopkg.in/telegram-bot-api.v4"
	"net/http"
	"os"
)

type (
	conf struct {
		Port      int    `env:"PORT" envDefault:"4200"`
		RpUuid    string `env:"RP_UUID" envDefault:"a47d5107-edc0-46b9-9258-4e1f8fcfc0ef"`
		RpProject string `env:"RP_PROJECT" envDefault:"andrei_varabyeu_personal"`
		RpHost    string `env:"RP_HOST" envDefault:"https://rp.epam.com"`

		//DB settings
		DbFile string `env:"DB_FILE" envDefault:"qabot.db"`

		//NLP settings
		NlpURL string `env:"NLP_URL" envDefault:"http://localhost:5000"`

		//Telegram
		TelegramToken string `env:"TG_TOKEN" envDefault:"597153786:AAGw8YF-LJh9V0aP9Cq-yWheMM9dPhiVjAU"`
	}
)

func main() {
	app := fx.New(
		fx.Provide(
			NewConf,
			NewMux,
			//NewDFDispatcher,
			NewSessionRepo,
			NewRPReporter,
			NewTelegramBot,
			NewIntentDispatcher,
			NewIntentParser,
		),
		fx.Invoke(InitLogger, Register),
	)

	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		log.WithError(err).Fatal("Cannot start application")
	}
	<-app.Done()

}

func NewConf() (*conf, error) {
	cfg := conf{}
	err := env.Parse(&cfg)
	return &cfg, err
}

func InitLogger() {
	log.SetHandler(cli.New(os.Stdout))
	log.SetLevel(log.DebugLevel)
}

func NewMux(lc fx.Lifecycle, cfg *conf) chi.Router {
	mux := chi.NewRouter()

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: mux,
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Infof("Starting HTTP server on port %d", cfg.Port)

			go server.ListenAndServe()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Info("Stopping HTTP server.")
			return server.Shutdown(ctx)
		},
	})
	return mux
}

func NewSessionRepo(lc fx.Lifecycle) (db.SessionRepo, error) {
	bdb, err := bolt.Open("my.db", 0600, nil)
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return bdb.Close()
		},
	})

	return db.NewBoltSessionRepo(bdb)
}

//func NewDFDispatcher() *df.Dispatcher {
//	return df.NewIntentDispatcher()
//}

func NewIntentDispatcher(nlp *nlp.IntentParser, repo db.SessionRepo, rp *rp.Reporter) *bot.Dispatcher {
	d := &bot.Dispatcher{
		NLP: nlp,
		Handler: bot.IntentNameDispatcher(map[string]bot.Handler{
			"exit.intent":  intents.NewExitQuizHandler(repo, rp),
			"start.intent": intents.NewStartQuizHandler(repo, rp),
		}, intents.NewQuizIntentHandler(repo, rp)),
		//Fallback: bot.NewHandlerFunc(func(ctx context.Context, rq *bot.Request) (*bot.Response, error) {
		//	return bot.NewResponse().WithText("What...?"), nil
		//}),
		ErrHandler: bot.ErrorHandlerFunc(func(ctx context.Context, err error) *bot.Response {
			return bot.NewResponse().WithText(fmt.Sprintf("Sorry, error has occured: %s", err))
		}),
	}
	d.Use(func(next bot.Handler) bot.Handler {
		return bot.NewHandlerFunc(func(ctx context.Context, rq *bot.Request) (*bot.Response, error) {
			if upd, ok := botctx.GetOriginalMessage(ctx).(tgbotapi.Update); ok {
				return next.Handle(botctx.WithUser(ctx, upd.Message.From.UserName), rq)
			}
			return next.Handle(ctx, rq)
		})
	})

	//d.Use(func(next bot.Handler) bot.Handler {
	//	return bot.NewHandlerFunc(func(ctx context.Context, rq *bot.Request) (*bot.Response, error) {
	//		if user := botctx.GetUser(ctx); "" != user {
	//			var s map[string]string
	//			if err := repo.Load(user, &s); nil != err {
	//				return next.Handle(botctx.WithSession(ctx, s), rq)
	//			}
	//		}
	//		return next.Handle(ctx, rq)
	//	})
	//})
	return d
}

func NewIntentParser(cfg *conf) *nlp.IntentParser {
	return nlp.NewIntentParser(cfg.NlpURL)
}

func NewRPReporter(cfg *conf) *rp.Reporter {
	return rp.NewReporter(gorp.NewClient(cfg.RpHost, cfg.RpProject, cfg.RpUuid))
}

func NewTelegramBot(lc fx.Lifecycle, cfg *conf, dispatcher *bot.Dispatcher) *telegram.Bot {
	tBot := &telegram.Bot{
		Token:      cfg.TelegramToken,
		Dispatcher: dispatcher,
	}
	lc.Append(fx.Hook{
		OnStart: func(ctc context.Context) error {
			return tBot.Start()
		},
	})
	return tBot
}

func Register(tb *telegram.Bot) {
}
