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



	fbApp := fb.New("", "")
	fbSession := fbApp.Session(fbToken)
	fbRes, err := fbSession.Get("/me", fb.Params{
		"fields": "first_name,email",
	})

	if err != nil {
		return nil, errors.New("error while accesing facebook api " + err.Error())
	}

	//Fetch user friends
	friendsRes, err := fbSession.Get("/me/friends", fb.Params{
		"limit": "500",
	})
	if err != nil {
		return nil, errors.New("error while accesing facebook api for friends " + err.Error())
	}

	friendsPaging, err := friendsRes.Paging(fbSession)
	if err != nil {
		return nil, errors.New("error while paging fb result" + err.Error())
	}

	var allResults []fb.Result
	allResults = append(allResults, friendsPaging.Data()...)

	for{
		noMore, err := friendsPaging.Next()
		if err != nil {
			return nil, errors.New("error while paging fb result" + err.Error())
		}
		if noMore {
			break
		}

		allResults = append(allResults, friendsPaging.Data()...)
	}

	var allFriendsFBIDs []string

	for _, friendRes := range allResults {
		allFriendsFBIDs = append(allFriendsFBIDs, friendRes["id"].(string))
	}

	var friendUsers []model.User
	var friendUserIDs []bson.ObjectId
	err = db.C(model.User{}.GetCollectionName()).Find(bson.M{
		"facebookID": bson.M{
			"$in": allFriendsFBIDs,
		},
	}).All(&friendUsers)
	if err != nil {
		return nil, errors.New("error while fetching all friends of user from db")
	}

	for _, user := range friendUsers {
		friendUserIDs = append(friendUserIDs, user.Id)
	}

	//First check if user exists with given facebook id
	user := &model.User{}

	err = db.C(user.GetCollectionName()).Find(bson.M{
		"facebookID": fbRes["id"],
	}).One(user)

	var loggedInUserID bson.ObjectId

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

					loggedInUserID = bson.NewObjectId()

					user = &model.User{
						Id: loggedInUserID,
						Username: username,
						Fingerprint: fingerprint,
						DisplayName: fbRes["first_name"].(string),
						FacebookId: fbRes["id"].(string),
						AvatarUrl: "http://graph.facebook.com/" + fbRes["id"].(string) + "/picture?type=square&type=normal",
						Friends: friendUserIDs,
					}

					err = db.C(user.GetCollectionName()).Insert(&user)
					if err != nil {
						return nil, err
					}

				}else{
					return nil, err
				}
			}else{

				loggedInUserID = user.Id

				user.DisplayName = fbRes["first_name"].(string)
				user.FacebookId = fbRes["id"].(string)
				user.AvatarUrl = "http://graph.facebook.com/" + fbRes["id"].(string) + "/picture?type=square&type=normal"

				//We should add friends to existing ones if not exist
				friendsMap := make(map[bson.ObjectId]struct{}, 0)
				for _, id := range user.Friends {
					friendsMap[id] = struct{}{}
				}

				for _, id := range friendUserIDs {
					if _, ok := friendsMap[id]; !ok {
						user.Friends = append(user.Friends, id)
					}
				}

				err = db.C(user.GetCollectionName()).UpdateId(user.Id, user)
				if err != nil{
					return nil, errors.New("error while updating user " + err.Error())
				}

			}

		}else{
			return nil, errors.New("error while search user in db " + err.Error())
		}
	}else{
		loggedInUserID = user.Id
		//Fingerprint also should be updated for this user
		if user.Fingerprint != fingerprint {

			user.Fingerprint = fingerprint

			err = db.C(user.GetCollectionName()).UpdateId(user.Id, user)
			if err != nil{
				return nil, errors.New("error while updating user " + err.Error())
			}

		}
	}

	//Now we should also add this user to other users as friend
	if len(friendUserIDs) > 0 {
		_ = db.C(user.GetCollectionName()).Update(bson.M{
			"_id": bson.M{
				"$in": friendUserIDs,
			},
		}, bson.M{"$addToSet": bson.M{
			"friends": loggedInUserID,
		}})
	}

	return user, nil

}
