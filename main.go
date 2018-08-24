package main

import (
	"context"
	"fmt"
	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/avarabyeu/gorp/gorp"
	"github.com/avarabyeu/rpquiz/db"
	"github.com/avarabyeu/rpquiz/df"
	"github.com/avarabyeu/rpquiz/intents"
	"github.com/caarlos0/env"
	"github.com/coreos/bbolt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"go.uber.org/fx"
	"net/http"
	"os"
)

type (
	conf struct {
		Port      int    `env:"PORT" envDefault:"4200"`
		RpUuid    string `env:"RP_UUID" envDefault:"a47d5107-edc0-46b9-9258-4e1f8fcfc0ef"`
		RpProject string `env:"RP_PROJECT" envDefault:"andrei_varabyeu_personal"`
		RpHost    string `env:"RP_HOST" envDefault:"https://rp.epam.com"`
	}
)

func main() {
	app := fx.New(
		fx.Provide(
			NewConf,
			NewMux,
			NewDFDispatcher,
			NewSessionRepo,
			NewRPClient,
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

func NewDFDispatcher() *df.Dispatcher {
	return df.NewIntentDispatcher()
}

func NewRPClient(cfg *conf) *gorp.Client {
	fmt.Println(cfg.RpUuid)
	return gorp.NewClient(cfg.RpHost, cfg.RpProject, cfg.RpUuid)
}

func Register(mux chi.Router, dispatcher *df.Dispatcher, repo db.SessionRepo, rp *gorp.Client) {
	dispatcher.SetHandler(df.IntentNameDispatcher(map[string]df.IntentHandler{
		"quiz": intents.NewQuizIntentHandler(repo, rp),
		"q1":   intents.Q1Func(),
	}))

	mux.Use(middleware.Logger)
	mux.Use(middleware.Recoverer)
	mux.Post("/", dispatcher.HTTPHandler())
}
