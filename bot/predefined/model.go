package predefined

import (
	"os"
	"io/ioutil"
	"encoding/json"
	"math/rand"
	"time"
)

type (
	response struct {
		Code    int         `json:"response_code,omitempty"`
		Results []*Question `json:"results,omitempty"`
	}

	//Question represents one question in openTDB
	Question struct {
		Category         string   `json:"category,omitempty"`
		Type             string   `json:"type,omitempty"`
		Difficulty       string   `json:"difficulty,omitempty"`
		Question         string   `json:"question,omitempty"`
		CorrectAnswer    string   `json:"correct_answer,omitempty"`
		IncorrectAnswers []string `json:"incorrect_answers,omitempty"`
	}

	//Client is the OpenTDB client
	Client struct {
		res response
	}
)

//NewClient initialize list of predefined questions
func NewClient() *Client {
	var q response
	jsonFile, _ := os.Open("rpQuestions.json")
	byteValue, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(byteValue, &q)
	return &Client{
		res: q,
	}
}

//GetPredefinedQuestions get number of random questions
func (c Client) GetPredefinedQuestions(count int) ([]*Question, error) {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(c.res.Results), func(i, j int) { c.res.Results[i], c.res.Results[j] = c.res.Results[j], c.res.Results[i] })
	return c.res.Results[:count], nil
}
