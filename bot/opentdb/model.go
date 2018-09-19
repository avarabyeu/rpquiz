package opentdb

import (
	"gopkg.in/resty.v1"
	"strconv"
	"io/ioutil"
	"encoding/json"
	"os"
	"math/rand"
	"time"
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

//GetPredefinedQuestions get number of random questions
func GetPredefinedQuestions(count int) ([]*Question, error) {
	var res response
	dir := "go/src/github.com/avarabyeu/rpquiz/bot/opentdb/rpQuestions.json"
	jsonFile, err := os.Open(dir)

	if err != nil {
		return nil, err
	}

	byteValue, err := ioutil.ReadAll(jsonFile)
	json.Unmarshal(byteValue, &res)

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(res.Results), func(i, j int) { res.Results[i], res.Results[j] = res.Results[j], res.Results[i] })
	return res.Results[:count], err
}