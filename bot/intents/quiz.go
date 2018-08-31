package intents

import (
	"context"
	"fmt"
	"github.com/apex/log"
	"github.com/avarabyeu/rpquiz/bot/db"
	"github.com/avarabyeu/rpquiz/bot/engine"
	"github.com/avarabyeu/rpquiz/bot/engine/ctx"
	"github.com/avarabyeu/rpquiz/bot/opentdb"
	"github.com/avarabyeu/rpquiz/bot/rp"
	"github.com/pkg/errors"
	"math/rand"
	"net/url"
	"strings"
)

const questionsCount = 5

//NewStartQuizHandler creates new start intent handler - greeting and first question
func NewStartQuizHandler(repo db.SessionRepo, rp *rp.Reporter) bot.Handler {
	return bot.NewHandlerFunc(func(ctx context.Context, rq *bot.Request) ([]*bot.Response, error) {
		sessionID := botctx.GetUser(ctx)

		log.Infof("Starting new quiz for %s", sessionID)
		//handle start, first question
		rpID, err := rp.StartLaunch()
		if err != nil {
			return nil, err
		}
		questions, err := opentdb.NewClient().GetQuestions(questionsCount)
		if err != nil {
			return nil, err
		}

		q := askQuestion(questions[0])
		testID, err := rp.StartTest(rpID, q.Text)
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

		return bot.Respond(bot.NewResponse().WithText(fmt.Sprintf("Hi %s! We are starting new quiz!", sessionID)), q), nil
	})
}

//NewExitQuizHandler creates new intent handler that processes quit from quiz
func NewExitQuizHandler(repo db.SessionRepo, rp *rp.Reporter) bot.Handler {
	return bot.NewHandlerFunc(func(ctx context.Context, rq *bot.Request) ([]*bot.Response, error) {
		sessionID := botctx.GetUser(ctx)

		if rq.Confidence >= 0.8 {

			session, err := loadSession(repo, sessionID)
			if err != nil {
				return nil, err
			}

			if err := repo.Delete(sessionID); err != nil {
				return nil, err
			}

			//TODO finish active test (if any)

			if err := rp.FinishLaunch(session.LaunchID); nil != err {
				return nil, err
			}

		}
		return bot.Respond(bot.NewResponse().WithText("Thanks for quizzing!")), nil

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
func (h *QuizIntentHandler) Handle(ctx context.Context, rq *bot.Request) ([]*bot.Response, error) {
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

		//if previous question was answered
		text := getAnswerText(session.Results[currQuestion])

		// not a last question. Ask next one
		if currQuestion < len(session.Questions)-1 {
			log.Debug("Handling question")

			newQuestion := askQuestion(session.Questions[currQuestion+1])

			testID, err := h.rp.StartTest(session.LaunchID, newQuestion.Text)
			if nil != err {
				return nil, err
			}
			session.TestID = testID

			if err := h.repo.Save(sessionID, session); nil != err {
				return nil, err
			}

			return bot.Respond(bot.NewResponse().WithText(text), newQuestion), nil
			// handle question
		}

		// handle last question. close session
		log.Debug("Handling last question")
		if err := h.repo.Delete(sessionID); nil != err {
			return nil, err
		}
		if err := h.rp.FinishLaunch(session.LaunchID); nil != err {
			return nil, err
		}
		return bot.Respond(bot.NewResponse().WithText(text), bot.NewResponse().
			WithText(fmt.Sprintf("Thank you! You passed a quiz! Your score is %d", calculateScore(session))),
			bot.NewResponse().WithText(`Don't forget to star us!\nhttps://github.com/avarabyeu/rpquiz\nhttps://github.com/reportportal/reportportal`)), nil

	}

	//should never happen :)
	return bot.Respond(bot.NewResponse().WithText("hm..")), nil
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
	var btns []*bot.Button
	if len(q.IncorrectAnswers) > 0 {
		btns = make([]*bot.Button, len(q.IncorrectAnswers)+1)
		for i, btn := range q.IncorrectAnswers {
			btnText, _ := url.PathUnescape(btn)
			btns[i] = &bot.Button{
				Data: btnText,
				Text: btnText,
			}
		}
		rs.WithButtons(btns...)
	}
	if len(btns) == 0 {
		btns = make([]*bot.Button, 1)
	}
	correctAnswerText, _ := url.PathUnescape(q.CorrectAnswer)

	btns[len(btns)-1] = &bot.Button{
		Data: correctAnswerText,
		Text: correctAnswerText,
	}

	//shuffle the array
	rand.Shuffle(len(btns), func(i, j int) {
		btns[i], btns[j] = btns[j], btns[i]
	})

	return rs
}

func getAnswerText(success bool) (text string) {
	if success {
		text = "That'a correct!\n"
	} else {
		text = "Wrong answer!\n"
	}
	return
}

func loadSession(repo db.SessionRepo, id string) (*QuizSession, error) {
	var session QuizSession
	err := repo.Load(id, &session)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func calculateScore(s *QuizSession) int {
	score := 0
	for _, success := range s.Results {
		if success {
			score++
		}
	}
	return score
}
