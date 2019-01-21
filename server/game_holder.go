package server

import (
	"github.com/golang/protobuf/jsonpb"
	"github.com/mediocregopher/radix/v3"
	"spaceship/socketapi"
	"sync"
)

type GameController interface {
	GetName() string
	Init(gameData *socketapi.GameData) error
	Join(gameID string, session Session) error
	Leave(gameID string, session Session) error
	Update(gameData *socketapi.GameData, session Session, metadata string) error
	GetGameSpecs() GameSpecs
}

type GameHolder struct {
	sync.RWMutex
	games map[string]GameController
	redis *radix.Pool
	jsonProtoMarshler *jsonpb.Marshaler
	jsonProtoUnmarshler *jsonpb.Unmarshaler
}

func NewGameHolder(redis *radix.Pool, jsonpbMarshler *jsonpb.Marshaler, jsonpbUnmarshaler *jsonpb.Unmarshaler) *GameHolder {
	return &GameHolder{
		games: make(map[string]GameController),
		redis: redis,
		jsonProtoMarshler: jsonpbMarshler,
		jsonProtoUnmarshler: jsonpbUnmarshaler,
	}
}

func (r *GameHolder) Get(gameName string) GameController {
	var g GameController
	r.RLock()
	g := r.games[gameName]
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
