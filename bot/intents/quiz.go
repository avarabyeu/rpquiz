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
		sessionID := botctx.GetUserID(ctx)
		if "" == sessionID {
			return nil, errors.Errorf("User ID isn't recognized")
		}
		userName := botctx.GetUserName(ctx)

		log.Infof("Starting new quiz for %s", sessionID)
		//handle start, first question

		questions, err := opentdb.NewClient().GetQuestions(questionsCount)
		if err != nil {
			return nil, err
		}

		session := &db.QuizSession{
			ID:        sessionID,
			Questions: questions,
			Results:   map[int]bool{},
		}
		err = repo.Save(session)
		if err != nil {
			return nil, err
		}

		q := askQuestion(questions[0])

		rp.StartLaunch(fmt.Sprintf("Quiz by %s", userName), func(launchID string, e error) error {
			err := repo.Update(&db.QuizSession{
				ID:       sessionID,
				LaunchID: launchID,
			})
			if err != nil {
				return err
			}

			rp.StartTest(launchID, q.Text, func(testID string, e error) {
				repo.Update(&db.QuizSession{
					ID:     sessionID,
					TestID: testID,
				})

			})
			return nil

		})

		return bot.Respond(bot.NewResponse().WithText(fmt.Sprintf("Hi %s! We are starting new quiz!", userName)), q), nil
	})
}

//NewExitQuizHandler creates new intent handler that processes quit from quiz
func NewExitQuizHandler(repo db.SessionRepo, rp *rp.Reporter) bot.Handler {
	return bot.NewHandlerFunc(func(ctx context.Context, rq *bot.Request) ([]*bot.Response, error) {
		sessionID := botctx.GetUserID(ctx)
		if "" == sessionID {
			return nil, errors.Errorf("User ID isn't recognized")
		}

		if rq.Confidence >= 0.8 {

			session, err := loadSession(repo, sessionID)
			if err != nil {
				return nil, err
			}

			if err := repo.Delete(sessionID); err != nil {
				return nil, err
			}

			//TODO finish active test (if any)

			rp.FinishLaunch(session.LaunchID, func(err error) {
				if nil != err {
					log.WithError(err).Error("Cannot finish launch")
				}
			})

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
	sessionID := botctx.GetUserID(ctx)
	if "" == sessionID {
		return nil, errors.Errorf("User ID isn't recognized")
	}

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
			return h.handleNewQuestion(sessionID, session, currQuestion)
		}

		//if previous question was answered
		text := getAnswerText(session.Results[currQuestion])

		// handle last question. close session
		log.Debug("Handling last question")
		if err := h.repo.Delete(sessionID); nil != err {
			return nil, err
		}
		h.rp.FinishLaunch(session.LaunchID, func(err error) {
			if err != nil {
				log.WithError(err).Error("Cannot finish launch")
			}
		})

		return bot.Respond(bot.NewResponse().WithText(text), bot.NewResponse().
			WithText(fmt.Sprintf("Thank you! You passed a quiz! Your score is %d", calculateScore(session))),
			bot.NewResponse().
				WithText(fmt.Sprintf("Don't forget to star us!\n%s",
					markdownLink("https://github.com/avarabyeu/rpquiz"))),
			bot.NewResponse().WithText(markdownLink("https://github.com/reportportal/reportportal"))), nil

	}

	//should never happen :)
	return bot.Respond(bot.NewResponse().WithText("hm..")), nil
}

func (h *QuizIntentHandler) handleNewQuestion(sessionID string, session *db.QuizSession, currQuestion int) ([]*bot.Response, error) {
	log.Debug("Handling question")

	newQuestion := askQuestion(session.Questions[currQuestion+1])

	h.rp.StartTest(session.LaunchID, newQuestion.Text, func(testID string, err error) {
		if nil != err {
			return
		}

		if err := h.repo.Update(&db.QuizSession{
			ID:     sessionID,
			TestID: testID,
		}); nil != err {
			//return nil, err
		}
	})
	//testID, err := h.rp.StartTest(session.LaunchID, newQuestion.Text)

	//if previous question was answered
	text := getAnswerText(session.Results[currQuestion])

	return bot.Respond(bot.NewResponse().WithText(text), newQuestion), nil
	// handle question
}

func (h *QuizIntentHandler) handleAnswer(rq *bot.Request, session *db.QuizSession, currQuestion int) error {
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
	h.repo.Update(&db.QuizSession{
		ID:      session.ID,
		Results: session.Results,
	})

	h.rp.FinishTest(session.TestID, passed, func(err error) {
		if err != nil {
			log.WithError(err).Error("Cannot finish Test")
		} else {
			log.Debugf("Test %s has been finished", session.TestID)
		}
	})

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

func loadSession(repo db.SessionRepo, id string) (*db.QuizSession, error) {
	var session db.QuizSession
	err := repo.Load(id, &session)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func calculateScore(s *db.QuizSession) int {
	score := 0
	for _, success := range s.Results {
		if success {
			score++
		}
	}
	return score
}

func markdownLink(url string) string {
	return fmt.Sprintf("[%s](%s)", url, url)
}
