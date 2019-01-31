package model

import "github.com/globalsign/mgo/bson"

type LeaderboardModel struct {
	Id bson.ObjectId `bson:"_id,omitempty"`
	Type string `bson:"type"` //day, week, month, overall
	TypeID *string `bson:"typeID"` // could be nil for overall
	Mode *string `bson:"mode"` // could be nil or given mode name
	UserID bson.ObjectId `bson:"userID"`
	Score int64 `bson:"score"`
}

func (_ LeaderboardModel) GetCollectionName() string {
	return "scores"
}