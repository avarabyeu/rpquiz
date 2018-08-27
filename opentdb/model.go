package opentdb

import (
	"gopkg.in/resty.v1"
	"strconv"
)

const opebTdbURL = "https://opentdb.com"

type (
	Response struct {
		Code    int         `json:"response_code,omitempty"`
		Results []*Question `json:"results,omitempty"`
	}

	Question struct {
		Category         string   `json:"category,omitempty"`
		Type             string   `json:"type,omitempty"`
		Difficulty       string   `json:"difficulty,omitempty"`
		Question         string   `json:"question,omitempty"`
		CorrectAnswer    string   `json:"correct_answer,omitempty"`
		IncorrectAnswers []string `json:"incorrect_answers,omitempty"`
	}

	Client struct {
		http *resty.Client
	}
)

func NewClient() *Client {
	return &Client{
		http: resty.New().SetHostURL(opebTdbURL),
	}
}

func (c Client) GetQuestions(count int) ([]*Question, error) {
	var q Response
	_, err := c.http.
		NewRequest().
		SetQueryParam("amount", strconv.Itoa(count)).
		SetQueryParam("encode", "url3986").
		SetResult(&q).
		Get("/api.php")
	return q.Results, err
}
