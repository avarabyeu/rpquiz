package intents

import (
	"context"
	"fmt"
	"github.com/apex/log"
	"github.com/pkg/errors"
	"gitlab.com/avarabyeu/rpquiz/bot/db"
	"gitlab.com/avarabyeu/rpquiz/bot/engine"
	"gitlab.com/avarabyeu/rpquiz/bot/engine/ctx"
	"gitlab.com/avarabyeu/rpquiz/opentdb"
	"gitlab.com/avarabyeu/rpquiz/rp"
	"google.golang.org/genproto/googleapis/cloud/dialogflow/v2"
	"net/url"
	"strings"
)

func NewStartQuizHandler(repo db.SessionRepo, rp *rp.Reporter) bot.Handler {
	return bot.NewHandlerFunc(func(ctx context.Context, rq *bot.Request) (*bot.Response, error) {
		sessionID := botctx.GetUser(ctx)

		log.Infof("Starting new quiz for %s", sessionID)
		//handle start, first question
		rpID, err := rp.StartLaunch()
		if err != nil {
			return nil, err
		}
		count := 5
		questions, err := opentdb.NewClient().GetQuestions(count)
		if err != nil {
			return nil, err
		}

		question, err := url.PathUnescape(questions[0].Question)
		if nil != err {
			return nil, err
		}

		testID, err := rp.StartTest(rpID, question)
		if nil != err {
			return nil, err
		}

		err = repo.Save(sessionID, &QuizSession{
			Questions: questions,
			Results:   map[int]bool{},
			LaunchID:  rpID,
			TestID:    testID,
		})
		if err != nil {
			return nil, err
		}

		return bot.NewResponse().WithText(fmt.Sprintf("We are starting a quiz! I'll be asked %d questions.\n%s", count, question)), nil
	})
}

func NewExitQuizHandler(repo db.SessionRepo, rp *rp.Reporter) bot.Handler {
	return bot.NewHandlerFunc(func(ctx context.Context, rq *bot.Request) (*bot.Response, error) {
		sessionID := botctx.GetUser(ctx)
		if rq.Confidence >= 0.8 && "quit" == rq.Intent {

			session, err := loadSession(repo, sessionID)
			if err != nil {
				return nil, err
			}

			//handle quit
			if err := rp.FinishLaunch(sessionID); nil != err {
				return nil, err
			}

			if err := repo.Delete(sessionID); err != nil {
				return nil, err
			}

			if err := rp.FinishLaunch(session.LaunchID); nil != err {
				return nil, err
			}

		}
		return bot.NewResponse().WithText("Thanks for quiizing!"), nil

	})
}

type QuizIntentHandler struct {
	repo db.SessionRepo
	rp   *rp.Reporter
}

func NewQuizIntentHandler(repo db.SessionRepo, rp *rp.Reporter) *QuizIntentHandler {
	return &QuizIntentHandler{repo: repo, rp: rp}
}

func (h *QuizIntentHandler) Handle(ctx context.Context, rq *bot.Request) (*bot.Response, error) {
	sessionID := botctx.GetUser(ctx)

	session, err := loadSession(h.repo, sessionID)
	if nil != err {
		return nil, err
	}
	if nil == session {
		return nil, errors.New("Session is nil!")
	}
	if currQuestion := len(session.Results); currQuestion >= 0 {

		answer := rq.Raw
		if nil == session.Results {
			session.Results = map[int]bool{}
		}
		correctAnswer, err := url.PathUnescape(session.Questions[currQuestion].CorrectAnswer)
		if nil != err {
			return nil, err
		}

		passed := strings.EqualFold(answer, correctAnswer)
		session.Results[currQuestion] = passed
		if err := h.rp.FinishTest(session.TestID, passed); nil != err {
			return nil, err
		}

		if currQuestion < len(session.Questions)-1 {
			log.Info("Handling question")

			newQuestion, err := url.PathUnescape(session.Questions[currQuestion+1].Question)
			if nil != err {
				return nil, err
			}

			testID, err := h.rp.StartTest(session.LaunchID, newQuestion)
			if nil != err {
				return nil, err
			}
			session.TestID = testID

			if err := h.repo.Save(sessionID, session); nil != err {
				return nil, err
			}

			return bot.NewResponse().WithText(newQuestion), nil
			// handle question
		} else {
			log.Info("Handling last question")

			if err := h.repo.Delete(sessionID); nil != err {
				return nil, err
			}
			if err := h.rp.FinishLaunch(session.LaunchID); nil != err {
				return nil, err
			}
			return bot.NewResponse().WithText("Thank you!. You passed a quiz!"), nil

			// handle last question. close session
		}
	}

	return bot.NewResponse().WithText("hm.."), nil
}

func (h *QuizIntentHandler) findContext(rq *dialogflow.WebhookRequest) bool {
	for _, ctx := range rq.GetQueryResult().GetOutputContexts() {
		if strings.HasSuffix(ctx.Name, "/quiz_dialog_context") {
			return true
		}
	}
	return false
}

func loadSession(repo db.SessionRepo, id string) (*QuizSession, error) {
	var session QuizSession
	err := repo.Load(id, &session)
	if err != nil {
		return nil, err
	}
	return &session, nil
}
