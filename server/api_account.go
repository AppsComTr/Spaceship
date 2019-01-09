package server

import (
	"context"
	"google.golang.org/grpc/status"
	"spaceship/api"
)

func (as *Server) AuthenticateFingerprint(context context.Context, request *api.AuthenticateFingerprint) (*api.Session, error) {

	user, err := AuthenticateFingerprint(request.Fingerprint, as.db)
	if err != nil {
		return nil, status.Error(400, err.Error())
	}

	var session api.Session
	session = api.Session{
		Token: request.Fingerprint,
		User: user.MapToPB(),
	}

	return &session, nil

}
