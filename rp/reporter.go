package rp

import (
	"github.com/apex/log"
	"github.com/avarabyeu/gorp/gorp"
	"time"
)

func NewReporter(client *gorp.Client) *Reporter {
	return &Reporter{
		rp: client,
	}
}

type Reporter struct {
	rp *gorp.Client
}

type RpEvent struct {
	launchID string
	rq       interface{}
}

func (r *Reporter) StartTest(launchID, question string) (string, error) {
	log.Debug("Reporting new question to RP")

	rs, err := r.rp.StartTest(&gorp.StartTestRQ{
		LaunchID: launchID,
		Type:     "test",
		StartRQ: gorp.StartRQ{
			StartTime: gorp.Timestamp{Time: time.Now()},
			Name:      question,
		}})
	if nil != err {
		return "", err
	}
	return rs.ID, nil

}
func (r *Reporter) FinishTest(testID string, pass bool) error {
	log.Debug("Reporting new question to RP")

	var status string
	if pass {
		status = "PASSED"
	} else {
		status = "FAILED"
	}
	_, err := r.rp.FinishTest(testID, &gorp.FinishTestRQ{
		FinishExecutionRQ: gorp.FinishExecutionRQ{
			Status:  status,
			EndTime: gorp.Timestamp{Time: time.Now()},
		},
	})
	return err
}

func (r *Reporter) StartLaunch() (string, error) {
	log.Info("Starting launch in RP")
	rs, err := r.rp.StartLaunch(&gorp.StartLaunchRQ{
		StartRQ: gorp.StartRQ{
			Name:        "bot",
			Description: "test desc",
			StartTime: gorp.Timestamp{
				Time: time.Now(),
			},
		},
	})
	if err != nil {
		return "", err
	}

	return rs.ID, nil
}

func (r *Reporter) FinishLaunch(rpID string) error {
	_, err := r.rp.FinishLaunch(rpID, &gorp.FinishExecutionRQ{
		EndTime: gorp.Timestamp{
			Time: time.Now(),
		},
		//Status: "PASSED",
	})
	return err
}
