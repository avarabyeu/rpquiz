package db

import (
	"encoding/json"
	"testing"
)

func TestSession(t *testing.T) {
	s := `{"questions":[{"category":"General Knowledge","type":"boolean","difficulty":"easy","question":"The Great Wall of China is visible from the moon.","correct_answer":"False","incorrect_answers":["True"]},{"category":"Entertainment: Music","type":"multiple","difficulty":"easy","question":"The Red Hot Chili Pepper song \u0026quot;Give It Away\u0026quot; is from what album?","correct_answer":"Blood Sugar Sex Magik","incorrect_answers":["One Hot Minute","By the Way","Stadium Arcadium"]},{"category":"Animals","type":"boolean","difficulty":"easy","question":"Kangaroos keep food in their pouches next to their children.","correct_answer":"False","incorrect_answers":["True"]},{"category":"Science \u0026 Nature","type":"multiple","difficulty":"hard","question":"Which horizon in a soil profile consists of bedrock?","correct_answer":"R","incorrect_answers":["O","B","D"]},{"category":"Animals","type":"multiple","difficulty":"hard","question":"What scientific suborder does the family Hyaenidae belong to?","correct_answer":"Feliformia","incorrect_answers":["Haplorhini","Caniformia","Ciconiiformes"]}],"rp_id":"5b8138bcadbe1d0001e9abd3"}`

	var session *QuizSession
	json.Unmarshal([]byte(s), session)
}
