package server

import (
	"github.com/golang/protobuf/jsonpb"
	"github.com/mediocregopher/radix/v3"
	"spaceship/socketapi"
	"sync"
)

//Interface that controls game logic
//When new game is created, Init method will be triggered
//If an user joins to game by match maker, Join method will be triggered
//If defined game is not a real time game, update method will be triggered when clients send data about this game
//Else Loop method will be triggered with defined interval in gamespecs. Datas which clients send will be buffered until this method works
type GameController interface {
	GetName() string
	Init(gameData *socketapi.GameData, logger *Logger) error
	Join(gameData *socketapi.GameData, session Session, notification *Notification, logger *Logger) error
	//Leave(gameData *socketapi.GameData, session Session) error
	//Should return true if game is finished, so framework can remove gamedata from redis and store it in db
	Update(gameData *socketapi.GameData, session Session, metadata string, leaderboard *Leaderboard, notification *Notification, logger *Logger) (bool, error)
	//This will be called instead of update if game is realtime game with tick rate
	Loop(gameData *socketapi.GameData, queuedDatas []socketapi.MatchUpdateQueue, leaderboard *Leaderboard, notification *Notification, logger *Logger) bool
	GetGameSpecs() GameSpecs
}

type GameHolder struct {
	sync.RWMutex
	games map[string]GameController
	redis radix.Client
	jsonProtoMarshler *jsonpb.Marshaler
	jsonProtoUnmarshler *jsonpb.Unmarshaler
	leaderboard *Leaderboard
	notification *Notification
}

func NewGameHolder(redis radix.Client, jsonpbMarshler *jsonpb.Marshaler, jsonpbUnmarshaler *jsonpb.Unmarshaler, leaderboard *Leaderboard, notification *Notification) *GameHolder {
	return &GameHolder{
		games: make(map[string]GameController),
		redis: redis,
		jsonProtoMarshler: jsonpbMarshler,
		jsonProtoUnmarshler: jsonpbUnmarshaler,
		leaderboard: leaderboard,
		notification: notification,
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
