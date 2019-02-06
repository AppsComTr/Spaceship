package server

import (
	"github.com/satori/go.uuid"
	"spaceship/socketapi"
	"sync"
	"time"
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
	Consume(func(session Session, envelope *socketapi.Envelope) bool)

	Send(isStream bool, mode uint8, envelope *socketapi.Envelope) error
	SendBytes(isStream bool, mode uint8, payload []byte) error

	IsClosed() bool

	//Close()
}

// SessionRegistry maintains a thread-safe list of sessions to their IDs.
type SessionHolder struct {
	sync.RWMutex
	sessions map[uuid.UUID]Session
	sessionsPerUserID map[string]Session
	config *Config

	leaveListener func(userID string) error
}

func NewSessionHolder(config *Config) *SessionHolder {
	return &SessionHolder{
		sessions: make(map[uuid.UUID]Session),
		sessionsPerUserID: make(map[string]Session),
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

func (r *SessionHolder) GetByUserID(userID string) Session {
	var s Session
	r.RLock()
	s = r.sessionsPerUserID[userID]
	r.RUnlock()
	return s
}

func (r *SessionHolder) add(s Session) {
	r.Lock()
	r.sessions[s.ID()] = s
	r.sessionsPerUserID[s.UserID()] = s
	r.Unlock()
}

func (r *SessionHolder) remove(sessionID uuid.UUID) {
	r.Lock()
	s := r.sessions[sessionID]
	delete(r.sessionsPerUserID, s.UserID())
	delete(r.sessions, sessionID)
	r.Unlock()
}

func (r *SessionHolder) leave(sessionID uuid.UUID) {

	session := r.Get(sessionID)

	if session != nil {

		//Check if user still has session after 5 seconds to eliminate network switch problems
		go func(userID string, r *SessionHolder, sessionID uuid.UUID){

			time.Sleep(time.Duration(5)*time.Second)

			session := r.GetByUserID(userID)

			if session == nil || session.IsClosed() {
				if r.leaveListener != nil {
					_ = r.leaveListener(userID)
				}

			}

		}(session.UserID(), r, sessionID)

	}
}

func (r *SessionHolder) SetLeaveListener(fn func(userID string) error) {
	r.Lock()
	r.leaveListener = fn
	r.Unlock()
}

