package server

import "sync"

type Game interface {
	GetName() string
	Start(session *Session) error
}

type GameHolder struct {
	sync.RWMutex
	games map[string]Game
}

func NewGameHolder() *GameHolder {
	return &GameHolder{
		games: make(map[string]Game),
	}
}

func (r *GameHolder) Get(gameName string) Game {
	var g Game
	r.RLock()
	g = r.games[gameName]
	r.RUnlock()
	return g
}

func (r *GameHolder) Add(g Game) {
	r.Lock()
	r.games[g.GetName()] = g
	r.Unlock()
}

func (r *GameHolder) Remove(gameName string) {
	r.Lock()
	delete(r.games, gameName)
	r.Unlock()
}
