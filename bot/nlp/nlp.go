package nlp

import (
	log "github.com/sirupsen/logrus"
	"gopkg.in/resty.v1"
	"net/http"
)

type (
	//IntentParser is NLP processor client
	IntentParser struct {
		s *resty.Client
	}

	//Intent represents parsed intent from user input
	Intent struct {
		Conf    float64           `json:"conf"`
		Matches map[string]string `json:"matches"`
		Name    string            `json:"name"`
		Sent    string            `json:"sent"`
	}
)

//NewIntentParser creates new instance of IntentParser
func NewIntentParser(url string) *IntentParser {
	c := http.Client{}
	return &IntentParser{
		s: resty.NewWithClient(&c).SetHostURL(url),
	}
}

//Parse parses intent based natural language
func (n *IntentParser) Parse(q string) *Intent {
	var rs Intent
	resp, err := n.s.NewRequest().SetBody(map[string]string{"q": q}).SetResult(&rs).Post("")
	if nil != err || nil == &rs {
		log.Errorf("Error executing request. Status code: %d. %s", resp.StatusCode(), err)
	}
	return &rs
}
