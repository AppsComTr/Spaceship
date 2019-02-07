package server

import (
	"cirello.io/goherokuname"
	"errors"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	fb "github.com/huandu/facebook"
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

func AuthenticateFacebook(fingerprint string, fbToken string, conn *mgo.Session) (*model.User, error) {

	if fingerprint == "" || fbToken == "" {
		return nil, errors.New("fingerprint or token couldn't be empty")
	}

	cConn := conn.Copy()
	defer cConn.Close()
	db := cConn.DB("spaceship")

	fbRes, err := fb.Get("/me", fb.Params{
		"fields": "first_name,email",
		"access_token": fbToken,
	})

	if err != nil {
		return nil, errors.New("error while accesing facebook api " + err.Error())
	}

	//First check if user exists with given facebook id
	user := &model.User{}

	err = db.C(user.GetCollectionName()).Find(bson.M{
		"facebook_id": fbRes["id"],
	}).One(user)

	if err != nil {
		if err == mgo.ErrNotFound {

			err = db.C(user.GetCollectionName()).Find(bson.M{
				"fingerprint": fingerprint,
			}).One(user)
			if err != nil{
				if err == mgo.ErrNotFound {

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
						DisplayName: fbRes["first_name"].(string),
						FacebookId: fbRes["id"].(string),
						AvatarUrl: "http://graph.facebook.com/" + fbRes["id"].(string) + "/picture?type=square&type=normal",
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

				user.DisplayName = fbRes["first_name"].(string)
				user.FacebookId = fbRes["id"].(string)
				user.AvatarUrl = "http://graph.facebook.com/" + fbRes["id"].(string) + "/picture?type=square&type=normal"

				err = db.C(user.GetCollectionName()).UpdateId(user.Id, user)
				if err != nil{
					return nil, errors.New("error while updating user " + err.Error())
				}

				return user, nil
			}

		}else{
			return nil, errors.New("error while search user in db " + err.Error())
		}
	}else{
		//Fingerprint also should be updated for this user
		if user.Fingerprint != fingerprint {

			user.Fingerprint = fingerprint

			err = db.C(user.GetCollectionName()).UpdateId(user.Id, user)
			if err != nil{
				return nil, errors.New("error while updating user " + err.Error())
			}

		}
		return user, nil
	}

}
