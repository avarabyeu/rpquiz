package intents

import "gitlab.com/avarabyeu/rpquiz/bot/opentdb"

//QuizSession DB model
type QuizSession struct {
	Questions []*opentdb.Question `json:"questions,omitempty"`
	LaunchID  string              `json:"rp_launch_id,omitempty"`
	TestID    string              `json:"rp_test_id,omitempty"`
	Results   map[int]bool        `json:"results,omitempty"`
}
