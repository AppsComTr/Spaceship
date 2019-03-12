package model

type PubSubMessage struct {
	UserIDs []string `json:"userIDs"`
	Data interface{} `json:"data"`
}
