package server

import (
	"bytes"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"go.uber.org/atomic"
	"net"
	"spaceship/socketapi"
	"sync"
	"time"
)

type session struct {
	sync.Mutex
	id uuid.UUID
	userID     string
	username   string
	expiry     int64
	clientIP   string
	clientPort string

	format string

	pingPeriodTime time.Duration
	pongWaitTime time.Duration
	writeWaitTime time.Duration

	sessionHolder *SessionHolder
	config *Config
	stats *Stats
	logger *Logger
	conn *websocket.Conn

	jsonProtoMarshler *jsonpb.Marshaler
	jsonProtoUnmarshler *jsonpb.Unmarshaler

	receivedMsgDecrement int
	pingTimer *time.Timer
	pingTimerCas *atomic.Uint32

	gameHolder *GameHolder
	outgoingCh chan []byte

	closed bool
}

func NewSession(userID string, username string, expiry int64, clientIP string, clientPort string, format string, conn *websocket.Conn, config *Config, sessionHolder *SessionHolder, gameHolder *GameHolder, jsonProtoMarshler *jsonpb.Marshaler, jsonProtoUnmarshler *jsonpb.Unmarshaler, stats *Stats, logger *Logger) Session {

	sessionID := uuid.Must(uuid.NewV4(), nil)

	stats.IncrSocketConnection()

	return &session{
		id: sessionID,
		userID: userID,
		username: username,
		expiry: expiry,
		clientIP: clientIP,
		clientPort: clientPort,

		format: format,

		pingPeriodTime: time.Duration(config.SocketConfig.PingPeriodTime) * time.Millisecond,
		pongWaitTime: time.Duration(config.SocketConfig.PongWaitTime) * time.Millisecond,
		writeWaitTime: time.Duration(config.SocketConfig.WriteWaitTime) * time.Millisecond,

		config: config,
		conn: conn,
		sessionHolder: sessionHolder,
		stats: stats,
		logger: logger,

		jsonProtoMarshler: jsonProtoMarshler,
		jsonProtoUnmarshler: jsonProtoUnmarshler,

		receivedMsgDecrement: config.SocketConfig.ReceivedMessageDecrementCount,
		pingTimer: time.NewTimer(time.Duration(config.SocketConfig.PingPeriodTime) * time.Millisecond),
		pingTimerCas: atomic.NewUint32(1),

		gameHolder: gameHolder,
		outgoingCh: make(chan []byte, config.SocketConfig.OutgoingQueueSize),

		closed: false,
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
func (s *session) Consume(handlerFunc func(session Session, envelope *socketapi.Envelope) bool) {
	defer s.Close()
	s.conn.SetReadLimit(4096)
	if err := s.conn.SetReadDeadline(time.Now().Add(s.pongWaitTime)); err != nil {
		s.logger.Infow("Error occured while trying to set read deadline", "error", err)
		return
	}
	//When pong message is received from client for this session, we can reset ping timer
	s.conn.SetPongHandler(func(string) error {
		s.resetPingTimer()
		return nil
	})

	//The routine that will handle outgoing messages
	go s.processOutgoing()

	for {
		_, data, err := s.conn.ReadMessage()
		s.stats.IncrSocketRequest()

		//Closed connections can be detected at this point. Just need to check error type.
		//Anyway, if error will happen, need to break this loop
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {
				s.logger.Infow("Socket connection was closed", "id", s.ID().String())
			}else if e, ok := err.(*net.OpError); !ok || e.Err.Error() != "use of closed network connection" {
				s.logger.Infow("Socket connection was closed", "id", s.ID().String())
			}else{
				s.logger.Errorw("Error occured while reading message on socket connection", "error", err)
			}
			//Even if connection was closed or error occured we should break the loop
			break
		}

		//If enough message was received in reset period, timer can be reset
		//Because we know the connection is open, no need to send ping to keep alive
		s.receivedMsgDecrement--
		if s.receivedMsgDecrement < 1 {
			s.receivedMsgDecrement = s.config.SocketConfig.ReceivedMessageDecrementCount
			if !s.resetPingTimer(){
				// We couldn't reset ping timer so there should be an error we need to close the loop
				return
			}
		}

		request := &socketapi.Envelope{}

		if s.format == "proto" {
			err = proto.Unmarshal(data, request)
		}else{
			err = s.jsonProtoUnmarshler.Unmarshal(bytes.NewReader(data), request)
		}

		if err != nil {
			s.logger.Errorw("Read message error", "error", err)
			break
		}

		if !handlerFunc(s, request) {
			break
		}

	}

}

func (s *session) resetPingTimer() bool {

	if !s.pingTimerCas.CAS(1, 0) {
		return true
	}
	defer s.pingTimerCas.CAS(0, 1)

	s.Lock()
	if s.closed {
		s.Unlock()
		return false
	}

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
		s.logger.Error("Error while trying to set read deadline on socket connection", "error", err)
		s.Close()
		return false
	}
	return true
}

func (s *session) processOutgoing() {
	defer s.Close()
	//This method starts infinite loop to detect outgoing ping or payload messages from relevant channels
	//Send method writes data to a channel don't send any data directly. This datas is being catched with this loop
	for {
		select {
		case <-s.pingTimer.C:
			if !s.pingNow() {
				return
			}
			break
		case payload := <-s.outgoingCh:
			s.Lock()

			if s.closed {
				s.Unlock()
				return
			}

			// Process the outgoing message queue.
			s.conn.SetWriteDeadline(time.Now().Add(10*time.Second))
			if err := s.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				s.Unlock()
				s.logger.Errorw("Could not write message", "error", err)
				return
			}
			s.Unlock()
		}
	}

}

func (s *session) pingNow() bool {
	s.Lock()
	if s.closed {
		s.Unlock()
		return false
	}
	if err := s.conn.SetWriteDeadline(time.Now().Add(10*time.Second)); err != nil {
		s.Unlock()
		s.logger.Errorw("Could not set write deadline to ping", "error", err)
		return false
	}
	err := s.conn.WriteMessage(websocket.PingMessage, []byte{})
	s.Unlock()
	if err != nil {
		s.logger.Errorw("Could not send ping", "error", err)
		return false
	}

	return true
}


func (s *session) Send(isStream bool, mode uint8, envelope *socketapi.Envelope) error {
	var payload []byte
	var err error
	var buf bytes.Buffer
	if s.format == "proto" {
		payload, err = proto.Marshal(envelope)
	}else{
		if err = s.jsonProtoMarshler.Marshal(&buf, envelope); err == nil {
			payload = buf.Bytes()
		}
	}
	if err != nil {
		s.logger.Errorw("Could not marshal envelope", "envelope", envelope, "error", err)
		return err
	}

	return s.SendBytes(isStream, mode, []byte(payload))
}

func (s *session) SendBytes(isStream bool, mode uint8, payload []byte) error {
	s.Lock()
	if s.closed {
		s.Unlock()
		return nil
	}

	if isStream {
		s.outgoingCh <- payload
		s.Unlock()
		return nil
	}

	// By default attempt to queue messages and observe failures.
	select {
	case s.outgoingCh <- payload:
		s.Unlock()
		return nil
	default:
		// The outgoing queue is full, likely because the remote client can't keep up.
		// Terminate the connection immediately because the only alternative that doesn't block the server is
		// to start dropping messages, which might cause unexpected behaviour.
		s.Unlock()
		s.logger.Warn("Could not write message, session outgoing queue full")
		s.Close()
		return errors.New("outgoing queue full")
	}
}

func (s *session) Close() {

	s.Lock()
	//This method can be triggered from many places. closed flag is being used to detect if socket connection is already closed
	//If connection is already closed, don't need to run again
	if s.closed {
		s.Unlock()
		return
	}
	s.closed = true
	s.Unlock()

	s.stats.DecrSocketConnection()

	s.sessionHolder.leave(s.id)
	s.sessionHolder.remove(s.id)

	s.pingTimer.Stop()
	close(s.outgoingCh)

	if err := s.conn.WriteControl(websocket.CloseMessage, []byte{}, time.Now().Add(s.writeWaitTime)); err != nil {
		s.logger.Error("Couldn't send close message to client")
	}

	if err := s.conn.Close(); err != nil {
		s.logger.Errorw("Couldn't close socket connection", "sessionID", s.id.String(), "error", err)
	}

}

func (s session) IsClosed() bool {
	return s.closed
}