package model

import (
	"github.com/globalsign/mgo/bson"
	"spaceship/socketapi"
)

type GameData struct {
	Id bson.ObjectId `bson:"_id,omitempty"`
	GameID string `bson:"game_id"`
	Name                 string   `bson:"name"`
	Metadata             string   `bson:"metadata"`
	CreatedAt            int64    `bson:"created_at"`
	UpdatedAt            int64    `bson:"updated_at"`
	ModeName             string   `bson:"mode_name"`
	UserIDs              []string `bson:"user_ids"`
}

func (gd *GameData) MapFromPB(pData *socketapi.GameData) {
	gd.GameID = pData.Id
	gd.Name = pData.Name
	gd.Metadata = pData.Metadata
	gd.CreatedAt = pData.CreatedAt
	gd.UpdatedAt = pData.UpdatedAt
	gd.ModeName = pData.ModeName
	gd.UserIDs = pData.UserIDs
}

func (gd GameData) GetCollectionName() string {
	return "games"
}
