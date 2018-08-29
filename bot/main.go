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
	"github.com/pkg/errors"
	"gitlab.com/avarabyeu/rpquiz/bot/db"
	"gitlab.com/avarabyeu/rpquiz/bot/engine"
	"gitlab.com/avarabyeu/rpquiz/bot/intents"
	"gitlab.com/avarabyeu/rpquiz/bot/nlp"
	"gitlab.com/avarabyeu/rpquiz/bot/rp"
	"gitlab.com/avarabyeu/rpquiz/bot/telegram"
	"go.uber.org/fx"
	"net/http"
	"os"
	"strings"
)

type (
	conf struct {
		Port      int    `env:"PORT" envDefault:"4200"`
		RpUUID    string `env:"RP_UUID" envDefault:"a47d5107-edc0-46b9-9258-4e1f8fcfc0ef"`
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
			newConf,
			newMux,
			newSessionRepo,
			newRPReporter,
			newTelegramBot,
			newIntentDispatcher,
			newIntentParser,
		),
		fx.Invoke(initLogger, register),
	)

	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		log.WithError(err).Fatal("Cannot start application")
	}
	<-app.Done()

}

func newConf() (*conf, error) {
	cfg := conf{}
	err := env.Parse(&cfg)
	return &cfg, err
}

func initLogger() {
	log.SetHandler(cli.New(os.Stdout))
	log.SetLevel(log.DebugLevel)
}

func newMux(lc fx.Lifecycle, cfg *conf) chi.Router {
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

func newSessionRepo(lc fx.Lifecycle) (db.SessionRepo, error) {
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

func newIntentDispatcher(nlp *nlp.IntentParser, repo db.SessionRepo, rp *rp.Reporter) *bot.Dispatcher {
	d := &bot.Dispatcher{
		NLP: nlp,
		Handler: bot.IntentNameDispatcher(map[string]bot.Handler{
			"exit.intent":  intents.NewExitQuizHandler(repo, rp),
			"start.intent": intents.NewStartQuizHandler(repo, rp),
		}, intents.NewQuizIntentHandler(repo, rp)),
		ErrHandler: bot.ErrorHandlerFunc(func(ctx context.Context, err error) []*bot.Response {
			logErr(err)
			return bot.Respond(bot.NewResponse().WithText(fmt.Sprintf("Sorry, error has occured: %s", err)))
		}),
	}
	//d.Use(func(next bot.Handler) bot.Handler {
	//	return bot.NewHandlerFunc(func(ctx context.Context, rq *bot.Request) (*bot.Response, error) {
	//		if upd, ok := botctx.GetOriginalMessage(ctx).(*tgbotapi.Message); ok {
	//			return next.Handle(botctx.WithUser(ctx, upd.From.UserName), rq)
	//		}
	//		return next.Handle(ctx, rq)
	//	})
	//})

	return d
}

func newIntentParser(cfg *conf) *nlp.IntentParser {
	return nlp.NewIntentParser(cfg.NlpURL)
}

func newRPReporter(cfg *conf) *rp.Reporter {
	return rp.NewReporter(gorp.NewClient(cfg.RpHost, cfg.RpProject, cfg.RpUUID))
}

func newTelegramBot(lc fx.Lifecycle, cfg *conf, dispatcher *bot.Dispatcher) *telegram.Bot {
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

func register(tb *telegram.Bot) {
}

func logErr(err error) {
	if err != nil {
		if err, ok := err.(stackTracer); ok {
			stackTrace := make([]string, len(err.StackTrace()))
			for i, f := range err.StackTrace() {
				stackTrace[i] = fmt.Sprintf("%+v", f)
			}
			fmt.Println(strings.Join(stackTrace, "\n"))
		} else {
			log.Errorf("%s", err)
		}
	}
}

// stackTracer interface.
type stackTracer interface {
	StackTrace() errors.StackTrace
}
