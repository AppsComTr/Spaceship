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
	"log"
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
			log.Println(err)
			return nil, status.Error(500, "Internal server error")
		}
	}

	user.Update(request)

	if request.Avatar != "" {
		reader := base64.NewDecoder(base64.StdEncoding, strings.NewReader(request.Avatar))

		buff := bytes.Buffer{}
		_, err = buff.ReadFrom(reader)
		if err != nil {
			log.Println(err)
			return nil, status.Error(500, "Internal server error")
		}

		_, format, err := image.Decode(bytes.NewReader(buff.Bytes()))
		if err != nil {
			return nil, status.Error(400, err.Error())
		}
		log.Println(format)

		if _, err := os.Stat("/var/spaceshipassets"); os.IsNotExist(err) {
			err = os.Mkdir("/var/spaceshipassets", os.ModePerm)
			if err != nil {
				log.Println(err)
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
			log.Println(err)
			return nil, status.Error(500, "Internal server error")
		}
		user.AvatarUrl = as.config.ApiURL + "/assets/" + fileName

	}

	err = db.C(user.GetCollectionName()).Update(bson.M{
		"_id": bson.ObjectIdHex(userID),
	}, user)
	if err != nil {
		log.Println(err)
		return nil, status.Error(500, "Internal server error")
	}

	return user.MapToPB(), nil

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
