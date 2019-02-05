package server

import (
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/tbalthazar/onesignal-go"
	"spaceship/model"
)

type Notification struct {
	db *mgo.Session
	config *Config
	client *onesignal.Client
	logger *Logger
}

func NewNotificationService(db *mgo.Session, config *Config, logger *Logger) *Notification {

	client := onesignal.NewClient(nil)
	client.AppKey = config.NotificationConfig.AppKey

	return &Notification{
		db: db,
		config: config,
		client: client,
		logger: logger,
	}

}

func (n Notification) SendNotificationWithUserIDs(headings map[string]string, body map[string]string, sUserIDs ...string) {

	userIDs := make([]bson.ObjectId, 0)
	for _, id := range sUserIDs {
		userIDs = append(userIDs, bson.ObjectIdHex(id))
	}

	conn := n.db.Copy()
	defer conn.Close()
	db := conn.DB("spaceship")

	notificationTokens := make([]model.NotificationToken, 0)

	err := db.C(model.NotificationToken{}.GetCollectionName()).Find(bson.M{
		"userID": bson.M{
			"$in": userIDs,
		},
	}).All(&notificationTokens)
	if err != nil {
		n.logger.Errorw("Error while fetching all notification tokens belongs to given user ids", "userIDs", userIDs, "error", err)
		return
	}

	tokens := make([]string, 0)
	for _, token := range notificationTokens {
		tokens = append(tokens, token.Token)
	}

	n.SendNotificationWithTokens(headings, body, tokens)

}

func (n Notification) SendNotificationWithTokens(headings map[string]string, body map[string]string, tokens []string){

	loopCount := len(tokens) / 2000

	if len(tokens)%2000 > 0 {
		loopCount = loopCount + 1
	}

	for i := 0; i < loopCount; i++ {
		limit := (i + 1) * 2000

		if limit > len(tokens) {
			limit = len(tokens)
		}

		notificationReq := &onesignal.NotificationRequest{
			AppID:            n.config.NotificationConfig.AppID,
			Headings: 		  headings,
			Contents:         body,
			IncludePlayerIDs: tokens[i*2000 : limit],
		}

		_, _, err := n.client.Notifications.Create(notificationReq)

		if err != nil {
			n.logger.Errorw("Error while creating notification request", "headings", headings, "contents", body, "error", err)
			return
		}
	}

}