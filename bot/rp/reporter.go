package rp

import (
	"github.com/apex/log"
	"github.com/avarabyeu/gorp/gorp"
	"time"
)

//NewReporter creates new instance of Reporter
func NewReporter(client *gorp.Client) *Reporter {
	return &Reporter{
		rp: client,
	}
}

//Reporter simple wrapper over RP client
type Reporter struct {
	rp *gorp.Client
}

//StartTest starts new test in RP
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

//FinishTest finishes test in RP
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

//StartLaunch starts launch in report portal
func (r *Reporter) StartLaunch(name string) (string, error) {
	log.Debug("Starting launch in RP")
	rs, err := r.rp.StartLaunch(&gorp.StartLaunchRQ{
		StartRQ: gorp.StartRQ{
			Name: name,
			//Description: "test desc",
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

//FinishLaunch finishes launch in ReportPortal
func (r *Reporter) FinishLaunch(rpID string) error {
	_, err := r.rp.FinishLaunch(rpID, &gorp.FinishExecutionRQ{
		EndTime: gorp.Timestamp{
			Time: time.Now(),
		},
		//Status: "PASSED",
	})
	return err
}
