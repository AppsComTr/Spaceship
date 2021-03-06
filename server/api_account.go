package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"github.com/dgrijalva/jwt-go"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/satori/go.uuid"
	"google.golang.org/grpc/status"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"os"
	"spaceship/api"
	"spaceship/model"
	"strings"
	"time"
)

func (as *Server) AuthenticateFingerprint(context context.Context, request *api.AuthenticateFingerprint) (*api.Session, error) {

	user, err := AuthenticateFingerprint(request.Fingerprint, as.db)
	if err != nil {
		return nil, status.Error(400, err.Error())
	}

	token, _ := generateToken(user.Id.Hex(), user.Username, as.config)

	var session api.Session
	session = api.Session{
		Token: token,
		User: user.MapToPB(),
	}

	return &session, nil

}

func (as *Server) AuthenticateFacebook(context context.Context, request *api.AuthenticateFacebook) (*api.Session, error) {

	user, err := AuthenticateFacebook(request.Fingerprint, request.Token, as.db)
	if err != nil {
		return nil, status.Error(400, err.Error())
	}

	token, _ := generateToken(user.Id.Hex(), user.Username, as.config)

	var session api.Session
	session = api.Session{
		Token: token,
		User: user.MapToPB(),
	}

	return &session, nil

}

func (as *Server) UnlinkFacebook(context context.Context, request *empty.Empty) (*empty.Empty, error) {

	emptyS := &empty.Empty{}

	userID := context.Value(ctxUserIDKey{}).(string)

	user := &model.User{}

	conn := as.db.Copy()
	defer conn.Close()
	db := conn.DB("spaceship")
	err := db.C(user.GetCollectionName()).Find(bson.M{
		"_id": bson.ObjectIdHex(userID),
	}).One(user)
	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, status.Error(404, "User couldn't found")
		}else{
			as.logger.Errorw("Error while trying to fetch user from db", "userID", userID, "error", err)
			return nil, status.Error(500, "Internal server error")
		}
	}

	user.FacebookId = ""

	err = db.C(user.GetCollectionName()).Update(bson.M{
		"_id": bson.ObjectIdHex(userID),
	}, user)
	if err != nil {
		as.logger.Errorw("Error while updating user in db", "userID", userID, "error", err)
		return nil, status.Error(500, "Internal server error")
	}

	return emptyS, nil

}

func (as *Server) UpdateUser(context context.Context, request *api.UserUpdate) (*api.User, error) {

	userID := context.Value(ctxUserIDKey{}).(string)

	user := &model.User{}

	conn := as.db.Copy()
	defer conn.Close()
	db := conn.DB("spaceship")
	err := db.C(user.GetCollectionName()).Find(bson.M{
		"_id": bson.ObjectIdHex(userID),
	}).One(user)
	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, status.Error(404, "User couldn't found")
		}else{
			as.logger.Errorw("Error while trying to fetch user from db", "userID", userID, "error", err)
			return nil, status.Error(500, "Internal server error")
		}
	}

	user.Update(request)

	if request.Avatar != "" {
		reader := base64.NewDecoder(base64.StdEncoding, strings.NewReader(request.Avatar))

		buff := bytes.Buffer{}
		_, err = buff.ReadFrom(reader)
		if err != nil {
			as.logger.Errorw("Error while reading from base64 decoder", "error", err)
			return nil, status.Error(500, "Internal server error")
		}

		_, _, err := image.Decode(bytes.NewReader(buff.Bytes()))
		if err != nil {
			as.logger.Errorw("Error while trying to decode image from bytes", "error", err)
			return nil, status.Error(400, err.Error())
		}

		if _, err := os.Stat("/var/spaceshipassets"); os.IsNotExist(err) {
			err = os.Mkdir("/var/spaceshipassets", os.ModePerm)
			if err != nil {
				as.logger.Errorw("Error while creating directory for assets", "error", err)
				return nil, status.Error(500, "Internal server error")
			}
		}

		fileName := strings.Replace(uuid.NewV4().String(), "-", "", -1) + ".jpg"

		for {
			if _, err := os.Stat("/var/spaceshipassets/"+fileName); os.IsNotExist(err) {
				break
			}
			fileName = strings.Replace(uuid.NewV4().String(), "-", "", -1) + ".jpg"
		}

		err = ioutil.WriteFile("/var/spaceshipassets/"+fileName, buff.Bytes(), 0644)
		if err != nil {
			as.logger.Errorw("Error while writing image to file", "fileName", fileName, "error", err)
			return nil, status.Error(500, "Internal server error")
		}
		user.AvatarUrl = as.config.ApiURL + "/assets/" + fileName

	}

	err = db.C(user.GetCollectionName()).Update(bson.M{
		"_id": bson.ObjectIdHex(userID),
	}, user)
	if err != nil {
		as.logger.Errorw("Error while updating user in db", "userID", userID, "error", err)
		return nil, status.Error(500, "Internal server error")
	}

	return user.MapToPB(), nil

}

func (as *Server) AddNotificationToken(context context.Context, request *api.NotificationTokenUpdate) (*empty.Empty, error) {

	emptyS := &empty.Empty{}
	userID := context.Value(ctxUserIDKey{}).(string)
	token := strings.TrimSpace(request.Token)

	if token == "" {
		return nil, status.Error(400, "Token couldn't be empty")
	}

	modelS := model.NotificationToken{
		UserID: bson.ObjectIdHex(userID),
		Token: token,
	}

	conn := as.db.Copy()
	defer conn.Close()
	db := conn.DB("spaceship")

	count, err := db.C(modelS.GetCollectionName()).Find(bson.M{
		"userID": bson.ObjectIdHex(userID),
		"token": token,
	}).Count()
	if err != nil {
		return nil, status.Error(500, "DB error: " + err.Error())
	}
	//This entry already exists in db so we don't need to create it again
	if count > 0 {
		return emptyS, nil
	}

	err = db.C(modelS.GetCollectionName()).Insert(modelS)
	if err != nil {
		return nil, status.Error(500, "DB error: " + err.Error())
	}

	return emptyS, nil
}

func (as *Server) UpdateNotificationToken(context context.Context, request *api.NotificationTokenUpdate) (*empty.Empty, error) {

	emptyS := &empty.Empty{}
	userID := context.Value(ctxUserIDKey{}).(string)
	token := strings.TrimSpace(request.Token)
	oldToken := strings.TrimSpace(request.OldToken)

	if token == "" || oldToken == "" {
		return nil, status.Error(400, "token or old_token couldn't be empty")
	}

	modelS := model.NotificationToken{}

	conn := as.db.Copy()
	defer conn.Close()
	db := conn.DB("spaceship")

	err := db.C(modelS.GetCollectionName()).Find(bson.M{
		"userID": bson.ObjectIdHex(userID),
		"token": oldToken,
	}).One(&modelS)
	if err != nil {
		if err == mgo.ErrNotFound{
			return nil, status.Error(400, "Cannot find token with given values")
		}else{
			as.logger.Errorw("Error while notification token user from db", "userID", userID, "oldToken", oldToken, "error", err)
			return nil, status.Error(500, "DB error: " + err.Error())
		}
	}

	modelS.Token = token

	err = db.C(modelS.GetCollectionName()).Update(bson.M{
		"_id": modelS.Id,
	}, modelS)
	if err != nil {
		return nil, status.Error(500, "DB error: " + err.Error())
	}

	return emptyS, nil
}

func (as *Server) DeleteNotificationToken(context context.Context, request *api.NotificationTokenUpdate) (*empty.Empty, error) {

	emptyS := &empty.Empty{}
	userID := context.Value(ctxUserIDKey{}).(string)
	token := strings.TrimSpace(request.Token)

	if token == "" {
		return nil, status.Error(400, "Token couldn't be empty")
	}

	conn := as.db.Copy()
	defer conn.Close()
	db := conn.DB("spaceship")

	err := db.C(model.NotificationToken{}.GetCollectionName()).Remove(bson.M{
		"userID": bson.ObjectIdHex(userID),
		"token": token,
	})
	if err != nil {
		return nil, status.Error(500, "DB error: " + err.Error())
	}

	return emptyS, nil
}

func (as *Server) GetFriends(context context.Context, empty *empty.Empty) (*api.UserFriends, error) {

	resp := &api.UserFriends{
		Friends: make([]*api.User, 0),
	}

	userID := context.Value(ctxUserIDKey{}).(string)

	user := &model.User{}

	conn := as.db.Copy()
	defer conn.Close()
	db := conn.DB("spaceship")
	err := db.C(user.GetCollectionName()).Find(bson.M{
		"_id": bson.ObjectIdHex(userID),
	}).One(user)
	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, status.Error(404, "User couldn't found")
		}else{
			as.logger.Errorw("Error while trying to fetch user from db", "userID", userID, "error", err)
			return nil, status.Error(500, "Internal server error")
		}
	}

	if len(user.Friends) > 0 {
		var friends []model.User

		err = db.C(user.GetCollectionName()).Find(bson.M{
			"_id": bson.M{
				"$in": user.Friends,
			},
		}).All(&friends)
		if err != nil {
			return nil, status.Error(500, "Error while trying to fetch friends from db")
		}

		for _, friend := range friends {
			resp.Friends = append(resp.Friends, friend.MapToPB())
		}
	}

	return resp, nil

}

func (as *Server) AddFriend(context context.Context, request *api.FriendRequest) (*empty.Empty, error) {

	friendUser := &model.User{}

	userID := context.Value(ctxUserIDKey{}).(string)
	friendID := strings.TrimSpace(request.UserId)
	friendUsername := strings.TrimSpace(request.Username)

	conn := as.db.Copy()
	defer conn.Close()
	db := conn.DB("spaceship")

	query := bson.M{}

	if friendID != "" {
		if len(friendID) != 24 {
			return nil, status.Error(400, "user_id is not valid")
		}
		query = bson.M{
			"_id": bson.ObjectIdHex(friendID),
		}
	}else{
		query = bson.M{
			"username": friendUsername,
		}
	}

	err := db.C(friendUser.GetCollectionName()).Find(query).One(friendUser)
	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, status.Error(404, "user couldn't be found with given parameters")
		}
		return nil, status.Error(500, "Error while trying to fetch user from db")
	}

	err = db.C(friendUser.GetCollectionName()).Update(bson.M{
		"_id": bson.ObjectIdHex(userID),
	}, bson.M{
		"$addToSet": bson.M{
			"friends": friendUser.Id,
		},
	})
	if err != nil {
		as.logger.Errorw("Error while trying to update friends list", "userID", userID, "friendID", friendID, "error", err)
		return nil, status.Error(500, "Error while trying to update friends list")
	}

	return &empty.Empty{}, nil
}

func (as *Server) DeleteFriend(context context.Context, request *api.FriendRequest) (*empty.Empty, error) {

	friendUser := &model.User{}

	userID := context.Value(ctxUserIDKey{}).(string)
	friendID := strings.TrimSpace(request.UserId)
	friendUsername := strings.TrimSpace(request.Username)

	conn := as.db.Copy()
	defer conn.Close()
	db := conn.DB("spaceship")

	query := bson.M{}

	if friendID != "" {
		if len(friendID) != 24 {
			return nil, status.Error(400, "user_id is not valid")
		}
		query = bson.M{
			"_id": bson.ObjectIdHex(friendID),
		}
	}else{
		query = bson.M{
			"username": friendUsername,
		}
	}

	err := db.C(friendUser.GetCollectionName()).Find(query).One(friendUser)
	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, status.Error(404, "user couldn't be found with given parameters")
		}
		return nil, status.Error(500, "Error while trying to fetch user from db")
	}

	err = db.C(model.User{}.GetCollectionName()).Update(bson.M{
		"_id": bson.ObjectIdHex(userID),
	}, bson.M{
		"$pull": bson.M{
			"friends": friendUser.Id,
		},
	})
	if err != nil {
		as.logger.Errorw("Error while trying to update friends list", "userID", userID, "friendID", friendID, "error", err)
		return nil, status.Error(500, "Error while trying to update friends list")
	}

	return &empty.Empty{}, nil
}

func (as *Server) TestEcho(context context.Context, empty *empty.Empty) (*api.Session, error){
	return &api.Session{
		Token: "at",
	}, nil
}

func generateToken(userID, username string, config *Config) (string, int64) {
	exp := time.Now().UTC().Add(time.Duration(config.AuthConfig.TokenExpireTime) * time.Second).Unix()
	return generateTokenWithExpiry(userID, username, exp, config)
}

func generateTokenWithExpiry(userID, username string, exp int64, config *Config) (string, int64) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid": userID,
		"exp": exp,
		"usn": username,
	})
	signedToken, _ := token.SignedString([]byte(config.AuthConfig.JWTSecret))
	return signedToken, exp
}
