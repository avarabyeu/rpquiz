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
func (r *Reporter) StartTest(launchID, sID, question string, callback func(string, error)) {
	go func() {
		log.Debug("Reporting new question to RP")

		rs, err := r.rp.StartChildTest(sID, &gorp.StartTestRQ{
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

		_, err := r.rp.FinishTest(testID, &gorp.FinishTestRQ{
			FinishExecutionRQ: gorp.FinishExecutionRQ{
				Status:  asStatus(pass),
				EndTime: gorp.Timestamp{Time: time.Now()},
			},
		})
		callback(err)
	}()

}

//StartLaunch starts launch in report portal
func (r *Reporter) StartLaunch(name string, callback func(string, string, error) error) {
	go func() {
		log.Debug("Starting launch in RP")
		rs, err := r.rp.StartLaunch(&gorp.StartLaunchRQ{
			StartRQ: gorp.StartRQ{
				Name: name,
				//Description: "test desc",
				StartTime: gorp.Timestamp{Time: time.Now()},
			},
		})
		if err != nil {
			log.WithError(err).Error("Cannot start launch in rp")
			callback("", "", err)
			return
		}

		sRS, err := r.rp.StartTest(&gorp.StartTestRQ{
			LaunchID: rs.ID,
			Type:     "SUITE",
			StartRQ: gorp.StartRQ{
				StartTime: gorp.Timestamp{Time: time.Now()},
				Name:      name,
			},
		})
		if err != nil {
			log.WithError(err).Error("Cannot start launch in rp")
			callback("", "", err)
			return
		}

		callback(rs.ID, sRS.ID, nil)
	}()

}

//FinishLaunch finishes launch in ReportPortal
func (r *Reporter) FinishLaunch(rpID, sID string, needRetry bool, callback func(error)) {
	go func() {
		//due to RP constant that all child items should be finished,
		//we use retry here.
		//since reporting is implemented in async fashion, not all items
		//may be finished at the time when launch finish is triggered

		var err error

		// finish can be retried if needed (to make sure all children are finished)
		if needRetry {

			//retry finishing of root test suite
			_, err = retry(5, 3*time.Second, func() (interface{}, error) {
				return r.rp.FinishTest(sID, &gorp.FinishTestRQ{
					FinishExecutionRQ: gorp.FinishExecutionRQ{
						EndTime: gorp.Timestamp{Time: time.Now()},
					},
				})
			})

			//if finished successfully, finish launch
			if nil == err {
				_, err = retry(5, 3*time.Second, func() (interface{}, error) {
					return r.finishLaunchInternally(rpID)
				})
			}

		} else {
			//finish without retries
			_, err = r.finishLaunchInternally(rpID)
		}

		//if finish haven't passed successfully, ЖЕСТАЧАЙШЕ execute force finish
		if nil != err {
			log.Warnf("Cannot finish launch %s. Forcing stop...", rpID)
			_, err = r.rp.StopLaunch(rpID)
		}
		callback(err)
	}()
}

func (r *Reporter) finishLaunchInternally(rpID string) (interface{}, error) {
	return r.rp.FinishLaunch(rpID, &gorp.FinishExecutionRQ{
		EndTime: gorp.Timestamp{
			Time: time.Now(),
		},
		//Status: "PASSED",
	})
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

func asStatus(pass bool) string {
	var status string

	if pass {
		status = "PASSED"
	} else {
		status = "FAILED"
	}

	return status
}

func boolPtr(b bool) *bool {
	return &b
}
