package server

import (
	"github.com/satori/go.uuid"
	"spaceship/socketapi"
	"time"
)

const (
	GAME_TYPE_PASSIVE_TURN_BASED int = iota
	GAME_TYPE_ACTIVE_TURN_BASED
	GAME_TYPE_REAL_TIME
)

type GameSpecs struct {
	PlayerCount int
	Mode int
}

type GameData struct {
	ID string
	Name string
	Metadata string
	CreatedAt int64
	UpdatedAt int64
}

func (gd GameData) GetID() string {
	return gd.ID
}

func (gd GameData) GetName() string {
	return gd.Name
}

func (gd GameData) GetCreatedAt() int64 {
	return gd.CreatedAt
}

func (gd GameData) GetUpdatedAt() int64 {
	return gd.UpdatedAt
}

func (gd *GameData) SetID(id string) {
	gd.ID = id
}

func (gd *GameData) SetName(gameName string){
	gd.Name = gameName
}

func (gd *GameData) SetCreatedAt(createdAt int64) {
	gd.CreatedAt = createdAt
}

//This should be used in matchmaker module
func NewGame(modeName string, fn func(gameData *GameData) error) (*GameData, error) {
	gameData := &GameData{
		ID: uuid.NewV4().String(),
		Name: modeName,
		CreatedAt: time.Now().Unix(),
	}

	err := fn(gameData)

	if err != nil {
		return nil, err
	}

	//TODO: we should save game to redis. But I don't know should we save it in here or should we do it on another layer ?

	return gameData, nil
}

func UpdateGame(session Session, updateData *socketapi.MatchUpdate, fn func(gameData *GameData, session Session, metadata string) error) (*GameData, error) {

	//TODO: we should fetch the game from redis with id in updateData, I'll user dummy for now
	gameData := &GameData{
		ID: uuid.NewV4().String(),
		Name: "atatat",
		CreatedAt: time.Now().Unix(),
	}

	err := fn(gameData, session, updateData.Metadata)

	if err != nil {
		return nil, err
	}

	gameData.UpdatedAt = time.Now().Unix()

	//TODO: again we should save
	//TODO: and we should also broadcast to updated data to users of this game
	return gameData, nil

}