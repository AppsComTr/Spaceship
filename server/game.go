package server

import (
	"errors"
	"github.com/mediocregopher/radix/v3"
	"github.com/satori/go.uuid"
	"spaceship/socketapi"
	"strings"
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

	gameData.ModeName = game.GetName()

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

func JoinGame(gameID string, holder *GameHolder, session Session) (*socketapi.GameData, error) {
	gameData := &socketapi.GameData{}

	var gameDataS string

	err := holder.redis.Do(radix.Cmd(&gameDataS, "GET", gameID))
	if err != nil {
		return nil, err
	}

	err = holder.jsonProtoUnmarshler.Unmarshal(strings.NewReader(gameDataS), gameData)
	if err != nil {
		return nil, err
	}

	game := holder.Get(gameData.ModeName)
	if game == nil {
		return nil, errors.New("Game couldn't found with given mode name")
	}

	//Check if user is already joined to this game
	userExists := false
	for _, userID := range gameData.UserIDs {
		if userID == session.UserID() {
			userExists = true
			break
		}
	}
	if userExists {
		return nil, errors.New("This user already joined to this game")
	}

	gameData.UserIDs = append(gameData.UserIDs, session.UserID())

	err = game.Join(gameData, session)
	if err != nil {
		return nil, err
	}

	gameData.UpdatedAt = time.Now().Unix()

	gameDataS, err = holder.jsonProtoMarshler.MarshalToString(gameData)
	if err != nil {
		return nil, err
	}

	err = holder.redis.Do(radix.Cmd(nil, "SET", gameData.Id, gameDataS))
	if err != nil {
		return nil, err
	}

	//TODO: maybe we need to broadcast join data to inform clients
	return gameData, nil
}

func UpdateGame(holder *GameHolder, session Session, pipeline *Pipeline, updateData *socketapi.MatchUpdate) (*socketapi.GameData, error) {

	gameData := &socketapi.GameData{}

	var gameDataS string

	err := holder.redis.Do(radix.Cmd(&gameDataS, "GET", updateData.GameID))
	if err != nil {
		return nil, err
	}

	err = holder.jsonProtoUnmarshler.Unmarshal(strings.NewReader(gameDataS), gameData)
	if err != nil {
		return nil, err
	}

	game := holder.Get(gameData.ModeName)
	if game == nil {
		return nil, errors.New("Game couldn't found with given mode name")
	}

	err = game.Update(gameData, session, updateData.Metadata)
	if err != nil {
		return nil, err
	}

	gameData.UpdatedAt = time.Now().Unix()

	gameDataS, err = holder.jsonProtoMarshler.MarshalToString(gameData)
	if err != nil {
		return nil, err
	}

	err = holder.redis.Do(radix.Cmd(nil, "SET", gameData.Id, gameDataS))
	if err != nil {
		return nil, err
	}

	pipeline.broadcastGame(gameData)

	return gameData, nil

}