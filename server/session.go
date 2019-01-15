package server

import (
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"log"
	"net"
	"sync"
	"time"
	"go.uber.org/atomic"
)

type session struct {
	sync.Mutex
	id uuid.UUID
	userID     string
	username   string
	expiry     int64
	clientIP   string
	clientPort string

	pingPeriodTime time.Duration
	pongWaitTime time.Duration
	writeWaitTime time.Duration

	config *Config
	conn *websocket.Conn

	receivedMsgDecrement int
	pingTimer *time.Timer
	pingTimerCas *atomic.Uint32
}

func NewSession(userID string, username string, expiry int64, clientIP string, clientPort string, conn *websocket.Conn, config *Config) Session {

	sessionID := uuid.Must(uuid.NewV4(), nil)

	return &session{
		id: sessionID,
		userID: userID,
		username: username,
		expiry: expiry,
		clientIP: clientIP,
		clientPort: clientPort,

		pingPeriodTime: time.Duration(config.SocketConfig.PingPeriodTime) * time.Millisecond,
		pongWaitTime: time.Duration(config.SocketConfig.PongWaitTime) * time.Millisecond,
		writeWaitTime: time.Duration(config.SocketConfig.WriteWaitTime) * time.Millisecond,

		config: config,
		conn: conn,

		receivedMsgDecrement: config.SocketConfig.ReceivedMessageDecrementCount,
		pingTimer: time.NewTimer(time.Duration(config.SocketConfig.PingPeriodTime) * time.Millisecond),
		pingTimerCas: atomic.NewUint32(1),
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
	if err := s.conn.SetReadDeadline(time.Now().Add(s.pongWaitTime)); err != nil {
		log.Println("Error occured while trying to set read deadline", errors.WithStack(err))
		return
	}
	s.conn.SetPongHandler(func(string) error {
		log.Println("pong received")
		s.resetPingTimer()
		return nil
	})

	go s.processOutgoing()

	for {
		_, data, err := s.conn.ReadMessage()

		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {
				log.Println("Socket connection was closed id: " + s.ID().String())
			}else if e, ok := err.(*net.OpError); !ok || e.Err.Error() != "use of closed network connection" {
				log.Println("Socket connection was closed id: " + s.ID().String())
			}else{
				log.Println("Error occured while reading message on socket connection", errors.WithStack(err))
			}
			//Even if connection was closed or error occured we should break the loop
			break
		}

		s.receivedMsgDecrement--
		if s.receivedMsgDecrement < 1 {
			s.receivedMsgDecrement = s.config.SocketConfig.ReceivedMessageDecrementCount
			if !s.resetPingTimer(){
				// We couldn't reset ping timer so there should be an error we need to close the loop
				return
			}
		}

		if err != nil {
			log.Println("Read message error", errors.WithStack(err))
			break
		}

		log.Println(data)

	}

}

func (s *session) resetPingTimer() bool {

	if !s.pingTimerCas.CAS(1, 0) {
		return true
	}
	defer s.pingTimerCas.CAS(0, 1)

	s.Lock()

	if !s.pingTimer.Stop() {
		select {
		case <-s.pingTimer.C:
		default:
		}
	}

	s.pingTimer.Reset(s.pingPeriodTime)
	err := s.conn.SetReadDeadline(time.Now().Add(s.pongWaitTime))
	s.Unlock()
	if err != nil {
		log.Println("Error while trying to set read deadline on socket connection", errors.WithStack(err))
		//TODO: socket connection or overall session should be closed here
		return false
	}
	return true
}

func (s *session) processOutgoing() {

	for {
		select {
		case <-s.pingTimer.C:
			if !s.pingNow() {
				return
			}
			break
		}
	}

}

func (s *session) pingNow() bool {
	s.Lock()
	//if s.stopped {
	//	s.Unlock()
	//	return false
	//}
	if err := s.conn.SetWriteDeadline(time.Now().Add(10*time.Second)); err != nil {
		s.Unlock()
		log.Println("Could not set write deadline to ping", err)
		return false
	}
	err := s.conn.WriteMessage(websocket.PingMessage, []byte{})
	s.Unlock()
	if err != nil {
		log.Println("Could not send ping", err)
		return false
	}

	return true
}