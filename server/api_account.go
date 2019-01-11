package server

import (
	"context"
	"github.com/dgrijalva/jwt-go"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/status"
	"spaceship/api"
	"time"
)

func (as *Server) AuthenticateFingerprint(context context.Context, request *api.AuthenticateFingerprint) (*api.Session, error) {

	user, err := AuthenticateFingerprint(request.Fingerprint, as.db)
	if err != nil {
		return nil, status.Error(400, err.Error())
	}

	token, _ := generateToken(user.Id.Hex(), user.Username)

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

	token, _ := generateToken(user.Id.Hex(), user.Username)

	var session api.Session
	session = api.Session{
		Token: token,
		User: user.MapToPB(),
	}

	return &session, nil

}

func (as *Server) TestEcho(context context.Context, empty *empty.Empty) (*api.Session, error){
	return &api.Session{
		Token: "at",
	}, nil
}

func generateToken(userID, username string) (string, int64) {
	//TODO: Duration parameter should be retrieved from config
	exp := time.Now().UTC().Add(time.Duration(86400) * time.Second).Unix()
	return generateTokenWithExpiry(userID, username, exp)
}

func generateTokenWithExpiry(userID, username string, exp int64) (string, int64) {
	//TODO: Encryption key should be retrieved from config
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid": userID,
		"exp": exp,
		"usn": username,
	})
	signedToken, _ := token.SignedString([]byte("asdasdqweqasdqwwe"))
	return signedToken, exp
}
