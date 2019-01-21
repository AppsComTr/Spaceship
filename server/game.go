package server

import (
	"errors"
	"github.com/mediocregopher/radix/v3"
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

//This should be used in matchmaker module
func NewGame(modeName string, holder *GameHolder, session Session) (*socketapi.GameData, error) {
	gameData := &socketapi.GameData{
		Id: uuid.NewV4().String(),
		Name: modeName,
		CreatedAt: time.Now().Unix(),
	}

	game := holder.Get(modeName)

	if game == nil {
		return nil, errors.New("Game couldn't found with given mode name")
	}

	err := game.Init(gameData)

	if err != nil {
		return nil, err
	}

	gameDataS, err := holder.jsonProtoMarshler.MarshalToString(gameData)
	if err != nil {
		return nil, err
	}

	err = holder.redis.Do(radix.Cmd(nil, "SET", gameData.Id, gameDataS))

	if err != nil {
		return nil, err
	}

	return gameData, nil
}

func UpdateGame(session Session, updateData *socketapi.MatchUpdate, fn func(gameData *socketapi.GameData, session Session, metadata string) error) (*socketapi.GameData, error) {

	//TODO: we should fetch the game from redis with id in updateData, I'll user dummy for now
	gameData := &socketapi.GameData{
		Id: uuid.NewV4().String(),
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