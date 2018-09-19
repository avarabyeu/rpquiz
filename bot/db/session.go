package db

import (
	"github.com/avarabyeu/rpquiz/bot/opentdb"
)

//QuizSession DB model
type QuizSession struct {
	ID        string `storm:"id"`
	Questions []*opentdb.Question
	LaunchID  string
	SuiteID   string
	TestID    string
	Results   map[int]bool
}