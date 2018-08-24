package df

import (
	"google.golang.org/genproto/googleapis/cloud/dialogflow/v2"
)

//FulfillmentBuilder builder for API V2 fulfillment responses
type FulfillmentBuilder struct {
	rs *dialogflow.WebhookResponse
}

//NewBuilder creates new instance of builder
func NewBuilder() *FulfillmentBuilder {
	return &FulfillmentBuilder{
		rs: &dialogflow.WebhookResponse{},
	}
}

//AddMessage adds new message to fulfillment response
func (ff *FulfillmentBuilder) Defaults(qr *dialogflow.QueryResult) *FulfillmentBuilder {
	ff.rs.FulfillmentMessages = qr.FulfillmentMessages
	ff.rs.FulfillmentText = qr.FulfillmentText
	return ff
}

//AddMessage adds new message to fulfillment response
func (ff *FulfillmentBuilder) AddMessage(msg *dialogflow.Intent_Message) *FulfillmentBuilder {
	ff.rs.FulfillmentMessages = append(ff.rs.FulfillmentMessages, msg)
	return ff
}

//AddTextMessage adds new text message to fulfillment response
func (ff *FulfillmentBuilder) AddTextMessage(text ...string) *FulfillmentBuilder {
	ff.rs.FulfillmentMessages = append(ff.rs.FulfillmentMessages, &dialogflow.Intent_Message{
		Message: &dialogflow.Intent_Message_Text_{
			Text: &dialogflow.Intent_Message_Text{
				Text: text,
			},
		},
	})
	return ff
}

//Build builds fulfillment response
func (ff *FulfillmentBuilder) Build() *dialogflow.WebhookResponse {
	return ff.rs
}
