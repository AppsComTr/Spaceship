package server

import (
	"github.com/golang/protobuf/jsonpb"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"net"
	"net/http"
	"strings"
)

func NewSocketAcceptor(sessionHolder *SessionHolder, config *Config, gameHolder *GameHolder, jsonProtoMarshler *jsonpb.Marshaler, jsonProtoUnmarshler *jsonpb.Unmarshaler, pipeline *Pipeline, stats *Stats, logger *Logger) func(http.ResponseWriter, *http.Request) {
	upgrader := &websocket.Upgrader{
		ReadBufferSize: 4096,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	return func(w http.ResponseWriter, r *http.Request) {

		//Connections which are made for this endpoint, will be upgraded to websocket connection if token is valid
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "Invalid token", 401)
			return
		}

		userID, username, expiry, ok := parseToken([]byte(config.AuthConfig.JWTSecret), token)
		if !ok {
			http.Error(w, "Invalid token", 401)
			return
		}

		clientAddr := ""
		clientIP := ""
		clientPort := ""
		if ips := r.Header.Get("x-forwarded-for"); len(ips) > 0 {
			clientAddr = strings.Split(ips, ",")[0]
		} else {
			clientAddr = r.RemoteAddr
		}

		clientAddr = strings.TrimSpace(clientAddr)
		if host, port, err := net.SplitHostPort(clientAddr); err == nil {
			clientIP = host
			clientPort = port
		} else if addrErr, ok := err.(*net.AddrError); ok && addrErr.Err == "missing port in address" {
			clientIP = clientAddr
		} else {
			logger.Warnw("Could not extract client address from request.", "error", errors.WithStack(err))
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Errorw("Websocket upgrade was failed", "error", errors.WithStack(err))
			return
		}

		s := NewSession(userID, username, expiry, clientIP, clientPort, conn, config, sessionHolder, gameHolder, jsonProtoMarshler, jsonProtoUnmarshler, stats, logger)

		logger.Infow("New socket connection was established", "id", s.ID().String())

		sessionHolder.add(s)

		//Incoming requests will be handled in sessions Consume method and will be passed to pipeline to run logic part of the each request
		s.Consume(pipeline.handleSocketRequests)

	}
}
