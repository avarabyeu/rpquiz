package intents

import (
	"context"
	"github.com/apex/log"
	"github.com/pkg/errors"
	"gitlab.com/avarabyeu/rpquiz/bot/db"
	"gitlab.com/avarabyeu/rpquiz/bot/engine"
	"gitlab.com/avarabyeu/rpquiz/bot/engine/ctx"
	"gitlab.com/avarabyeu/rpquiz/opentdb"
	"gitlab.com/avarabyeu/rpquiz/rp"
	"net/url"
	"strings"
)

//NewStartQuizHandler creates new start intent handler - greeting and first question
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

		q := questions[0]
		testID, err := rp.StartTest(rpID, q.Question)
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

		q.Question = "We are starting new quiz!\n" + q.Question
		return askQuestion(q), nil
	})
}

//NewExitQuizHandler creates new intent handler that processes quit from quiz
func NewExitQuizHandler(repo db.SessionRepo, rp *rp.Reporter) bot.Handler {
	return bot.NewHandlerFunc(func(ctx context.Context, rq *bot.Request) (*bot.Response, error) {
		sessionID := botctx.GetUser(ctx)
		if rq.Confidence >= 0.8 && "quit" == rq.Intent {

			session, err := loadSession(repo, sessionID)
			if err != nil {
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

//QuizIntentHandler handles answer to a question
type QuizIntentHandler struct {
	repo db.SessionRepo
	rp   *rp.Reporter
}

//NewQuizIntentHandler creates new instance of a handler
func NewQuizIntentHandler(repo db.SessionRepo, rp *rp.Reporter) *QuizIntentHandler {
	return &QuizIntentHandler{repo: repo, rp: rp}
}

//Handle handles answer to a question
func (h *QuizIntentHandler) Handle(ctx context.Context, rq *bot.Request) (*bot.Response, error) {
	sessionID := botctx.GetUser(ctx)

	session, err := loadSession(h.repo, sessionID)
	if nil != err || nil == session {
		return nil, errors.Errorf("session for user %s not found", sessionID)
	}

	if currQuestion := len(session.Results); currQuestion >= 0 {

		//handle answer to the previous question
		if err := h.handleAnswer(rq, session, currQuestion); nil != err {
			return nil, errors.WithStack(err)
		}

		// not a last question. Ask next one
		if currQuestion < len(session.Questions)-1 {
			log.Info("Handling question")

			newQuestion := askQuestion(session.Questions[currQuestion+1])

			testID, err := h.rp.StartTest(session.LaunchID, newQuestion.Text)
			if nil != err {
				return nil, err
			}
			session.TestID = testID

			if err := h.repo.Save(sessionID, session); nil != err {
				return nil, err
			}

			return newQuestion, nil
			// handle question
		}

		// handle last question. close session
		log.Info("Handling last question")
		if err := h.repo.Delete(sessionID); nil != err {
			return nil, err
		}
		if err := h.rp.FinishLaunch(session.LaunchID); nil != err {
			return nil, err
		}
		return bot.NewResponse().WithText("Thank you! You passed a quiz!"), nil

	}

	return bot.NewResponse().WithText("hm.."), nil
}

func (h *QuizIntentHandler) handleAnswer(rq *bot.Request, session *QuizSession, currQuestion int) error {
	answer := rq.Raw
	if nil == session.Results {
		session.Results = map[int]bool{}
	}
	correctAnswer, err := url.PathUnescape(session.Questions[currQuestion].CorrectAnswer)
	if nil != err {
		return err
	}

	passed := strings.EqualFold(answer, correctAnswer)
	session.Results[currQuestion] = passed
	if err := h.rp.FinishTest(session.TestID, passed); nil != err {
		return err
	}
	return nil

}

func askQuestion(q *opentdb.Question) *bot.Response {
	qText, _ := url.PathUnescape(q.Question)
	rs := bot.NewResponse().WithText(qText)
	if len(q.IncorrectAnswers) > 0 {
		btns := make([]*bot.Button, len(q.IncorrectAnswers))
		for i, btn := range q.IncorrectAnswers {
			btnText, _ := url.PathUnescape(btn)
			btns[i] = &bot.Button{
				Data: btnText,
				Text: btnText,
			}
		}
		rs.WithButtons(btns...)
	}
	return rs
}

func loadSession(repo db.SessionRepo, id string) (*QuizSession, error) {
	var session QuizSession
	err := repo.Load(id, &session)
	if err != nil {
		return nil, err
	}
	return &session, nil
}
