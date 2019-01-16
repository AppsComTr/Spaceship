package server

import (
	"github.com/satori/go.uuid"
	"sync"
)

type GameController interface {
	GetName() string
	Create() uuid.UUID
	Join(gameID uuid.UUID, session Session) error
	Leave(gameID uuid.UUID, session Session) error
	Update()
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
