package intents

import (
	"context"
	"fmt"
	"github.com/apex/log"
	"github.com/avarabyeu/gorp/gorp"
	"github.com/avarabyeu/rpquiz/db"
	"github.com/avarabyeu/rpquiz/df"
	"google.golang.org/genproto/googleapis/cloud/dialogflow/v2"
	"strings"
	"time"
)

type QuizIntentHandler struct {
	repo db.SessionRepo
	rp   *gorp.Client
}

func NewQuizIntentHandler(repo db.SessionRepo, rp *gorp.Client) *QuizIntentHandler {
	return &QuizIntentHandler{repo: repo, rp: rp}
}

func (h *QuizIntentHandler) Handle(ctx context.Context, rq *dialogflow.WebhookRequest) (*dialogflow.WebhookResponse, error) {
	//fmt.Println(rq.GetSession())
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
		}
	}()
	rpSession, err := h.repo.Find(rq.GetSession())
	if nil != err {
		log.WithError(err).Error("Cannot find session in DB")
	}

	//Quiz is just started. Create Launch in RP
	if rq.QueryResult.AllRequiredParamsPresent == false && h.findContext(rq) && nil == rpSession {
		if rpSession, err = h.handleLaunchStart(ctx, rq); nil != err {
			return nil, err
		}

		//All quiz questions are answered. Finishing launch in RP
	} else if rq.QueryResult.AllRequiredParamsPresent {
		if err := h.handleLaunchFinish(ctx, rq, rpSession); nil != err {
			return nil, err
		}

	} else {
		//report result
		h.reportResult(ctx, rq, rpSession)
	}

	rs := df.NewBuilder().Defaults(rq.QueryResult).Build()
	rs.FulfillmentText = rs.FulfillmentText + "PROMPT!"
	return rs, nil
}

func (h *QuizIntentHandler) findContext(rq *dialogflow.WebhookRequest) bool {
	for _, ctx := range rq.GetQueryResult().GetOutputContexts() {
		if strings.HasSuffix(ctx.Name, "/quiz_dialog_context") {
			return true
		}
	}
	return false
}

func (h *QuizIntentHandler) hasDialogParamsCtx(rq *dialogflow.WebhookRequest) bool {
	for _, ctx := range rq.GetQueryResult().GetOutputContexts() {
		if strings.Contains(ctx.Name, "/quiz_dialog_params_") {
			return true
		}
	}
	return false
}

func (h *QuizIntentHandler) handleLaunchStart(ctx context.Context, rq *dialogflow.WebhookRequest) (*db.RPSession, error) {
	log.Info("Starting launch in RP")
	rs, err := h.rp.StartLaunch(&gorp.StartLaunchRQ{
		StartRQ: gorp.StartRQ{
			Name:        "bot",
			Description: "test desc",
			StartTime: gorp.Timestamp{
				Time: time.Now(),
			},
		},
	})
	if err != nil {
		return nil, err
	}

	s := &db.RPSession{
		DfID: rq.Session,
		RpID: rs.ID,
	}
	return s, h.repo.Save(s)
}

func (h *QuizIntentHandler) reportResult(ctx context.Context, rq *dialogflow.WebhookRequest, session *db.RPSession) error {
	rs, err := h.rp.StartTest(&gorp.StartTestRQ{
		LaunchID: session.RpID,
		Type:     "TEST",
		StartRQ: gorp.StartRQ{
			StartTime: gorp.Timestamp{Time: time.Now()},
			Name:      rq.QueryResult.GetQueryText(),
		},
	})
	if err != nil {
		return err
	}

	h.rp.FinishTest(rs.ID, &gorp.FinishTestRQ{
		Retry: false,
		FinishExecutionRQ: gorp.FinishExecutionRQ{
			Status:  "PASSED",
			EndTime: gorp.Timestamp{Time: time.Now()},
		},
	})
	return nil
}

func (h *QuizIntentHandler) handleLaunchFinish(ctx context.Context, rq *dialogflow.WebhookRequest, session *db.RPSession) error {
	_, err := h.rp.FinishLaunch(session.RpID, &gorp.FinishExecutionRQ{
		EndTime: gorp.Timestamp{
			Time: time.Now(),
		},
		Status: "PASSED",
	})
	if nil != err {
		return err
	}
	return h.repo.Delete(rq.GetSession())
}

func Q1Func() df.HandlerFunc {
	return func(ctx context.Context, rq *dialogflow.WebhookRequest) (*dialogflow.WebhookResponse, error) {
		build := df.NewBuilder().Defaults(rq.QueryResult).Build()

		fmt.Println(rq.GetQueryResult().GetOutputContexts())

		build.FollowupEventInput = &dialogflow.EventInput{
			Name: "q1.q1-custom",
			//Parameters: &structpb.Struct{},
		}
		fmt.Println(build)

		return build, nil
	}
}
