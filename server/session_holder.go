package server

import (
	"github.com/satori/go.uuid"
	"sync"
)

type Session interface {
	ID() uuid.UUID
	UserID() string
	ClientIP() string
	ClientPort() string

	Username() string
	SetUsername(string)

	Expiry() int64
	//Consume(func(session Session, envelope *rtapi.Envelope) bool)
	Consume()

	//Send(isStream bool, mode uint8, envelope *rtapi.Envelope) error
	//SendBytes(isStream bool, mode uint8, payload []byte) error

	//Close()
}

// SessionRegistry maintains a thread-safe list of sessions to their IDs.
type SessionHolder struct {
	sync.RWMutex
	sessions map[uuid.UUID]Session
	config *Config
}

func NewSessionHolder(config *Config) *SessionHolder {
	return &SessionHolder{
		sessions: make(map[uuid.UUID]Session),
		config: config,
	}
}

func (r *SessionHolder) Stop() {}

func (r *SessionHolder) Get(sessionID uuid.UUID) Session {
	var s Session
	r.RLock()
	s = r.sessions[sessionID]
	r.RUnlock()
	return s
}

func (r *SessionHolder) add(s Session) {
	r.Lock()
	r.sessions[s.ID()] = s
	r.Unlock()
}

func (r *SessionHolder) remove(sessionID uuid.UUID) {
	r.Lock()
	delete(r.sessions, sessionID)
	r.Unlock()
}
