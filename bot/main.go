package main

import (
	"context"
	"fmt"
	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/asdine/storm"
	"github.com/avarabyeu/gorp/gorp"
	"github.com/avarabyeu/rpquiz/bot/db"
	"github.com/avarabyeu/rpquiz/bot/engine"
	"github.com/avarabyeu/rpquiz/bot/engine/ctx"
	"github.com/avarabyeu/rpquiz/bot/intents"
	"github.com/avarabyeu/rpquiz/bot/nlp"
	"github.com/avarabyeu/rpquiz/bot/rp"
	"github.com/avarabyeu/rpquiz/bot/telegram"
	"github.com/caarlos0/env"
	"github.com/go-chi/chi"
	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
	"go.uber.org/fx"
	"net/http"
	"os"
	"strings"
)

type (
	conf struct {
		LoggingLevel string `env:"LOGGING_LEVEL" envDefault:"info"`
		Port         int    `env:"PORT" envDefault:"4200"`
		RpUUID       string `env:"RP_UUID,required"`
		RpProject    string `env:"RP_PROJECT,required"`
		RpHost       string `env:"RP_HOST" envDefault:"https://rp.epam.com"`

		//DB settings
		DbFile string `env:"DB_FILE" envDefault:"qabot.db"`

		//NLP settings
		NlpURL string `env:"NLP_URL" envDefault:"http://localhost:5000"`

		//Telegram
		TelegramToken string `env:"TG_TOKEN,required"`
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

	app.Run()
}

func newConf() (*conf, error) {
	cfg := conf{}
	err := env.Parse(&cfg)
	return &cfg, err
}

func initLogger(c *conf) error {
	level, err := log.ParseLevel(c.LoggingLevel)
	if err != nil {
		return err
	}
	log.SetHandler(cli.New(os.Stdout))
	log.SetLevel(level)
	return nil
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

func newSessionRepo(lc fx.Lifecycle, cfg *conf) (db.SessionRepo, error) {
	bdb, err := storm.Open(cfg.DbFile, storm.BoltOptions(0600, &bolt.Options{}))
	if err != nil {
		log.WithError(err).Error("Cannot open DB")
		return nil, err
	}
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return bdb.Close()
		},
	})

	return db.NewStormSessionRepo(bdb)
}

func newIntentDispatcher(nlp *nlp.IntentParser, repo db.SessionRepo, rp *rp.Reporter) *bot.Dispatcher {
	d := &bot.Dispatcher{
		NLP: nlp,
		Handler: bot.IntentNameDispatcher(map[string]bot.Handler{
			"exit.intent":  intents.NewExitQuizHandler(repo, rp),
			"start.intent": intents.NewStartQuizHandler(repo, rp),
		}, intents.NewQuizIntentHandler(repo, rp), bot.NewHandlerFunc(func(ctx context.Context, rq bot.Request) ([]*bot.Response, error) {
			return bot.Respond(bot.NewResponse().WithText("What...??? I don't know how to handle that!")), nil
		})),
		ErrHandler: bot.ErrorHandlerFunc(func(ctx context.Context, err error) []*bot.Response {
			logErr(err)
			return bot.Respond(bot.NewResponse().WithText(fmt.Sprintf("Sorry, error has occured: %s", err)))
		}),
	}
	d.Use(func(next bot.Handler) bot.Handler {
		return bot.NewHandlerFunc(func(ctx context.Context, rq bot.Request) ([]*bot.Response, error) {
			sessionID := botctx.GetUserID(ctx)
			if "" == sessionID {
				return nil, errors.Errorf("User ID isn't recognized")
			}
			session, err := loadSession(repo, sessionID)
			if nil == err && nil != session {
				ctx = botctx.WithSession(ctx, session)
			}
			return next.Handle(ctx, rq)
		})
	})

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

func register(mux chi.Router, bot *telegram.Bot) {
	mux.Get("/health", func(w http.ResponseWriter, rq *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"status" : "ok"}`)); nil != err {
			log.WithError(err).Error("health check response error")
		}
		w.WriteHeader(http.StatusOK)
	})
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

func loadSession(repo db.SessionRepo, id string) (*db.QuizSession, error) {
	var session db.QuizSession
	err := repo.Load(id, &session)
	if err != nil {
		return nil, err
	}
	return &session, nil
}
