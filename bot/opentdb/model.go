package opentdb

import (
	"gopkg.in/resty.v1"
	"strconv"
	"io/ioutil"
	"encoding/json"
	"os"
)

const openTdbURL = "https://opentdb.com"

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
		http *resty.Client
	}
)

//NewClient creates new OpenTDB client
func NewClient() *Client {
	return &Client{
		http: resty.New().SetHostURL(openTdbURL),
	}
}

//GetQuestions retrieves given amount of questions
func (c Client) GetQuestions(count int) ([]*Question, error) {
	var q response
	_, err := c.http.
		NewRequest().
		SetQueryParam("amount", strconv.Itoa(count)).
		SetQueryParam("encode", "url3986").
		SetResult(&q).
		Get("/api.php")
	return q.Results, err
}

//GetPredefinedQuestions retrieves given amount of predefined questions
func (c Client) GetPredefinedQuestions(count int) ([]*Question, error) {
	jsonFile, err := os.Open("rpQuestions.json")
	byteValue, err := ioutil.ReadAll(jsonFile)
	var q response
	json.Unmarshal(byteValue, &q)
	return q.Results, err
}
