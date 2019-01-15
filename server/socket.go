package server

import (
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"log"
	"net"
	"net/http"
	"strings"
)

func NewSocketAcceptor(sessionHolder *SessionHolder) func(http.ResponseWriter, *http.Request) {
	upgrader := &websocket.Upgrader{
		ReadBufferSize: 4096,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	return func(w http.ResponseWriter, r *http.Request) {

		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "Invalid token", 401)
			return
		}

		//TODO: hmac key will readed from config
		userID, username, expiry, ok := parseToken([]byte("asdasdqweqasdqwwe"), token)
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
			log.Println("Could not extract client address from request.", errors.WithStack(err))
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Websocket upgrade was failed", errors.WithStack(err))
			return
		}

		s := NewSession(userID, username, expiry, clientIP, clientPort, conn)

		log.Println("New socket connection was established id: " + s.ID().String())

		sessionHolder.add(s)

		s.Consume()

	}
}
