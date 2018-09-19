package intents

import (
	"context"
	"fmt"
	"github.com/apex/log"
	"github.com/avarabyeu/rpquiz/bot/db"
	"github.com/avarabyeu/rpquiz/bot/engine"
	"github.com/avarabyeu/rpquiz/bot/engine/ctx"
	"github.com/avarabyeu/rpquiz/bot/rp"
	"github.com/pkg/errors"
	"math/rand"
	"net/url"
	"strings"
	"github.com/avarabyeu/rpquiz/bot/opentdb"
)

const questionsCount = 6

//NewStartQuizHandler creates new start intent handler - greeting and first question
func NewStartQuizHandler(repo db.SessionRepo, rp *rp.Reporter) bot.Handler {
	return bot.NewHandlerFunc(func(ctx context.Context, rq bot.Request) ([]*bot.Response, error) {
		userID := botctx.GetUserID(ctx)
		if "" == userID {
			return nil, errors.Errorf("User ID isn't recognized")
		}

		//if old session is still started, quit it gracefully.
		if oldSession, ok := botctx.GetSession(ctx); ok && "" != oldSession.LaunchID {
			if err := quiteSessionGracefully(repo, rp, oldSession); nil != err {
				return nil, err
			}
		}

		userName := botctx.GetUserName(ctx)

		log.Infof("Starting new quiz for %s[%s]", userName, userID)
		//handle start, first question

		questions, err := opentdb.GetPredefinedQuestions(questionsCount)
		if err != nil {
			return nil, err
		}
		if len(questions) < 1 {
			return nil, errors.New("Questions for a quiz cannot be retrieved")
		}

		session := &db.QuizSession{
			ID:        userID,
			Questions: questions,
			Results:   map[int]bool{},
		}
		err = repo.Save(session)
		if err != nil {
			return nil, err
		}

		//grab the very first question
		q := askQuestion(questions[0])

		//start launch and root suite in RP
		rp.StartLaunch(fmt.Sprintf("Quiz by %s", userName), func(launchID, sID string, e error) error {
			if err != nil {
				return err
			}
			//start test in RP
			rp.StartTest(launchID, sID, q.Text, func(testID string, e error) {
				repo.Update(&db.QuizSession{
					ID:       userID,
					TestID:   testID,
					SuiteID:  sID,
					LaunchID: launchID,
				})
				if err != nil {
					log.WithError(err).Error("Cannot create test suite")
				}
			})

			return nil

		})

		return bot.Respond(bot.NewResponse().WithText(fmt.Sprintf("Hi %s! We are starting new quiz!", userName)), q), nil
	})
}

//NewExitQuizHandler creates new intent handler that processes quit from quiz
func NewExitQuizHandler(repo db.SessionRepo, rp *rp.Reporter) bot.Handler {
	return bot.NewHandlerFunc(func(ctx context.Context, rq bot.Request) ([]*bot.Response, error) {
		if irq, ok := rq.(*bot.IntentRequest); ok && irq.Confidence >= 0.8 {

			session, ok := botctx.GetSession(ctx)
			if !ok {
				return nil, errors.Errorf("Quiz for user %s not found", botctx.GetUserName(ctx))
			}

			if err := quiteSessionGracefully(repo, rp, session); nil != err {
				return nil, err
			}
			return bot.Respond(bot.NewResponse().WithText("Thanks for quizzing!")), nil

		}
		return []*bot.Response{}, nil
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
func (h *QuizIntentHandler) Handle(ctx context.Context, rq bot.Request) ([]*bot.Response, error) {

	session, ok := botctx.GetSession(ctx)
	if !ok {
		return nil, errors.Errorf("Quiz for user %s isn't started", botctx.GetUserName(ctx))
	}

	if currQuestion := len(session.Results); currQuestion >= 0 {

		//handle answer to the previous question
		var text string
		var err error
		if text, err = h.handleAnswer(rq, session, currQuestion); nil != err {
			log.WithError(err).Error("Answer handling error")
			return nil, errors.WithStack(err)
		}

		// not a last question. Ask next one
		if currQuestion < len(session.Questions)-1 {
			newQuestion, err := h.handleNewQuestion(session, currQuestion)
			if nil != err {
				return nil, err
			}
			return bot.Respond(bot.NewResponse().WithText(text), newQuestion), nil
		}

		// handle last question. close session
		log.Debug("Handling last question")
		if err := h.repo.Delete(session.ID); nil != err {
			return nil, err
		}
		h.rp.FinishLaunch(session.LaunchID, session.SuiteID, true, func(err error) {
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

func (h *QuizIntentHandler) handleNewQuestion(session *db.QuizSession, currQuestion int) (*bot.Response, error) {
	log.Debug("Handling question")

	newQuestion := askQuestion(session.Questions[currQuestion+1])

	h.rp.StartTest(session.LaunchID, session.SuiteID, newQuestion.Text, func(testID string, err error) {
		if nil != err {
			log.WithError(err).Error("Cannot start test in RP")
			return
		}

		if err := h.repo.Update(&db.QuizSession{
			ID:      session.ID,
			SuiteID: session.SuiteID,
			TestID:  testID,
		}); nil != err {
			log.WithError(err).Error("Cannot update session in DB")
		}
	})
	//testID, err := h.rp.StartTest(session.LaunchID, newQuestion.Text)

	return newQuestion, nil
}

func (h *QuizIntentHandler) handleAnswer(rq bot.Request, session *db.QuizSession, currQuestion int) (string, error) {
	answer := rq.GetRaw()
	if nil == session.Results {
		session.Results = map[int]bool{}
	}
	correctAnswer, err := url.PathUnescape(session.Questions[currQuestion].CorrectAnswer)
	if nil != err {
		return "", err
	}

	passed := strings.EqualFold(answer, correctAnswer)
	session.Results[currQuestion] = passed
	h.repo.Update(&db.QuizSession{
		ID:      session.ID,
		Results: session.Results,
		SuiteID: session.SuiteID,
	})

	h.rp.FinishTest(session.TestID, passed, func(err error) {
		if err != nil {
			log.WithError(err).Error("Cannot finish Test")
		} else {
			log.Debugf("Test %s has been finished", session.TestID)
		}
	})

	return getAnswerText(passed, correctAnswer), nil

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

func quiteSessionGracefully(repo db.SessionRepo, rp *rp.Reporter, session *db.QuizSession) error {
	if err := repo.Delete(session.ID); err != nil {
		return err
	}

	rp.FinishLaunch(session.LaunchID, session.SuiteID, false, func(err error) {
		if nil != err {
			log.WithError(err).Error("Cannot finish launch")
		}
	})
	return nil
}

func getAnswerText(passed bool, correctAnswer string) (text string) {

	if passed {
		text = "That's correct!\n"
	} else {
		text = "Wrong answer! "
	}

	if !passed {
		text = fmt.Sprintf("%sCorrect answer is '%s'", text, correctAnswer)
	}

	return
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
