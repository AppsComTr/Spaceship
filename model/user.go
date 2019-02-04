package model

import (
	"github.com/globalsign/mgo/bson"
	"spaceship/api"
	"strings"
)

type User struct {
	Id bson.ObjectId `bson:"_id,omitempty"`
	Username string `bson:"username"`
	Fingerprint string `bson:"fingerprint"`
	DisplayName string `bson:"displayName"`
	AvatarUrl string `bson:"avatarURL"`
	Metadata string `bson:"metadata"`
	FacebookId string `bson:"facebookID"`
	Online bool `bson:"isOnline"`
}

func (u User) MapToPB() *api.User {
	return &api.User{
		Id: u.Id.Hex(),
		Username: u.Username,
		Fingerprint: u.Fingerprint,
		DisplayName: u.DisplayName,
		AvatarUrl: u.AvatarUrl,
		Metadata: u.Metadata,
		FacebookId: u.FacebookId,
		Online: u.Online,
	}
}

func (u *User) MapFromPB(pbUser api.User) {
	u.Id = bson.ObjectIdHex(pbUser.Id)
	u.Username =  pbUser.Username
	u.Fingerprint = pbUser.Fingerprint
	u.DisplayName = pbUser.DisplayName
	u.AvatarUrl = pbUser.AvatarUrl
	u.Metadata = pbUser.Metadata
	u.FacebookId = pbUser.FacebookId
	u.Online = pbUser.Online
}

func (u *User) Update(updateData *api.UserUpdate) {
	updateData.DisplayName = strings.TrimSpace(updateData.DisplayName)
	if updateData.DisplayName != "" {
		u.DisplayName = updateData.DisplayName
	}
}

func (u User) GetCollectionName() string {
	return "users"
}
