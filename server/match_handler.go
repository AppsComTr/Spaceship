package server

import (
	"github.com/satori/go.uuid"
	"sync"
)

//Manage Matches
//Join adds player/s to match by matchEntryID
//Leave removes player from a match
//LeaveAll remove player from only active and realtime matches
//List lists matches of player
//Broadcast publish match states to users

type Match struct {
	ID string //gameID
	Properties map[string]string
	Users map[string]MatchUserEntity

	//User MatchUserEntity self
}

type MatchUserEntity struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	Username  string
}

type MatchHandler struct {
	sync.RWMutex
}

func NewMatchHandler() *MatchHandler{
	return &MatchHandler{}
}

func (mh *MatchHandler) Join(){
	//create game
	//adds players to matchhandler
}

func (mh *MatchHandler) Leave(){

}

func (mh *MatchHandler) LeaveAll(){

}

func (mh *MatchHandler) List(){

}

func (mh *MatchHandler) broadcast(){

}