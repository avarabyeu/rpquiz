package db

import (
	"github.com/avarabyeu/rpquiz/bot/predefined"
)

//QuizSession DB model
type QuizSession struct {
	ID        string `storm:"id"`
	Questions []*predefined.Question
	LaunchID  string
	SuiteID   string
	TestID    string
	Results   map[int]bool
}