package server

import (
	"cirello.io/goherokuname"
	"errors"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"spaceship/model"
)

func AuthenticateFingerprint(fingerprint string, conn *mgo.Session) (*model.User, error) {

	if fingerprint == "" {
		return nil, errors.New("fingerprint couldn't be empty")
	}

	cConn := conn.Copy()
	defer cConn.Close()
	db := cConn.DB("spaceship")

	//First check if user exists with given fingerprint
	user := &model.User{}

	err := db.C(user.GetCollectionName()).Find(bson.M{
		"fingerprint": fingerprint,
	}).One(user)
	if err != nil{
		if err.Error() == mgo.ErrNotFound.Error() {

			username := goherokuname.HaikunateCustom("-", 4, "DfWx9873214560jzrl")

			//Generate user name until find one that doesn't exists in db
			for {
				count, err := db.C(user.GetCollectionName()).Find(bson.M{"username": username}).Count()
				if err != nil {
					return nil, err
				}
				if count == 0 {
					break
				}
				username = goherokuname.HaikunateCustom("-", 4, "DfWx9873214560jzrl")
			}

			user := &model.User{
				Id: bson.NewObjectId(),
				Username: username,
				Fingerprint: fingerprint,
				DisplayName: username,
				AvatarUrl: "http://api.adorable.io/avatars/150/" + username + ".png",
			}

			err = db.C(user.GetCollectionName()).Insert(&user)
			if err != nil {
				return nil, err
			}

			return user, nil

		}else{
			return nil, err
		}
	}else{
		return user, nil
	}

}
