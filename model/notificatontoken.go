package model

import (
	"github.com/globalsign/mgo/bson"
)

type NotificationToken struct {
	Id bson.ObjectId `bson:"_id,omitempty"`
	UserID bson.ObjectId `bson:"userID"`
	Token string `bson:"token"`
}

func (n NotificationToken) GetCollectionName() string {
	return "notificationTokens"
}
