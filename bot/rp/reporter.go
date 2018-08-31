package rp

import (
	"fmt"
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
func (r *Reporter) StartTest(launchID, question string, callback func(string, error)) {
	go func() {
		log.Debug("Reporting new question to RP")

		rs, err := r.rp.StartTest(&gorp.StartTestRQ{
			LaunchID: launchID,
			Type:     "test",
			StartRQ: gorp.StartRQ{
				StartTime: gorp.Timestamp{Time: time.Now()},
				Name:      question,
			}})
		if nil != err {
			callback("", err)
		}
		callback(rs.ID, nil)
	}()

}

//FinishTest finishes test in RP
func (r *Reporter) FinishTest(testID string, pass bool, callback func(error)) {
	go func() {
		log.Debugf("Finishing test %s in RP", testID)

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
		callback(err)
	}()

}

//StartLaunch starts launch in report portal
func (r *Reporter) StartLaunch(name string, callback func(string, error) error) {
	go func() {
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
			log.WithError(err).Error("Cannot start launch in rp")
			callback("", err)
			return
		}

		callback(rs.ID, nil)
	}()

}

//FinishLaunch finishes launch in ReportPortal
func (r *Reporter) FinishLaunch(rpID string, callback func(error)) {
	go func() {
		//due to RP constant that all child items should be finished,
		//we use retry here.
		//since reporting is implemented in async fashion, not all items
		//may be finished at the time when launch finish is triggered
		_, err := retry(5, 3*time.Second, func() (interface{}, error) {
			return r.rp.FinishLaunch(rpID, &gorp.FinishExecutionRQ{
				EndTime: gorp.Timestamp{
					Time: time.Now(),
				},
				//Status: "PASSED",
			})
		})
		callback(err)
	}()

}

//retry executes callback func until it executes successfully
func retry(attempts int, timeout time.Duration, callback func() (interface{}, error)) (interface{}, error) {
	var err error
	for i := 0; i < attempts; i++ {
		var res interface{}
		res, err = callback()
		if err == nil {
			return res, nil
		}

		<-time.After(timeout)
		log.Infof("Retrying... Attempt: %d. Left: %d", i+1, attempts-1-i)
	}
	return nil, fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}
