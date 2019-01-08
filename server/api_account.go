package server

import (
	"context"
	"spaceship/api"
)

func (as *Server) AuthenticateFingerprint(context context.Context, request *api.AuthenticateFingerprint) (*api.Session, error) {

	var session api.Session
	session = api.Session{
		Token: request.Fingerprint,
	}

	return &session, nil

}
