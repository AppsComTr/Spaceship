package server

import (
	"sync"
)

type GameController interface {
	GetName() string
	Init(gameData *GameData) error
	Join(gameID string, session Session) error
	Leave(gameID string, session Session) error
	Update(gameData *GameData, session Session, metadata string) error
	GetGameSpecs() GameSpecs
}

type GameHolder struct {
	sync.RWMutex
	games map[string]GameController
}

func NewGameHolder() *GameHolder {
	return &GameHolder{
		games: make(map[string]GameController),
	}
}

func (r *GameHolder) Get(gameName string) GameController {
	var g GameController
	r.RLock()
	g = r.games[gameName]
	r.RUnlock()
	return g
}

func (r *GameHolder) Add(g GameController) {
	r.Lock()
	r.games[g.GetName()] = g
	r.Unlock()
}

func (r *GameHolder) Remove(gameName string) {
	r.Lock()
	delete(r.games, gameName)
	r.Unlock()
}
