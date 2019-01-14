package server

import (
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"log"
	"github.com/satori/go.uuid"
	"sync"
)

type session struct {
	sync.Mutex
	id uuid.UUID
	userID     string
	username   string
	expiry     int64
	clientIP   string
	clientPort string

	conn *websocket.Conn
}

func NewSession(userID string, username string, expiry int64, clientIP string, clientPort string, conn *websocket.Conn) Session {

	sessionID := uuid.Must(uuid.NewV4(), nil)

	return &session{
		id: sessionID,
		userID: userID,
		username: username,
		expiry: expiry,
		clientIP: clientIP,
		clientPort: clientPort,
		conn: conn,
	}

}

func (s *session) ID() uuid.UUID {
	return s.id
}

func (s *session) UserID() string {
	return s.userID
}

func (s *session) ClientIP() string {
	return s.clientIP
}

func (s *session) ClientPort() string {
	return s.clientPort
}

func (s *session) Username() string {
	return s.username
}

func (s *session) SetUsername(username string) {
	s.username = username
}

func (s *session) Expiry() int64 {
	return s.expiry
}

//func (s *session) Consume(processRequest func(session Session, envelope *rtapi.Envelope) bool) {
func (s *session) Consume() {

	s.conn.SetReadLimit(4096)
	//TODO: read deadline will be defined
	//TODO: pong handler will be defined

	//TODO: outgoing messages will be processed

	for {
		_, data, err := s.conn.ReadMessage()
		if err != nil {
			log.Println("Read message error", errors.WithStack(err))
			break
		}

		log.Println(data)

	}

}
