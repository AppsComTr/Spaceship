package test

import (
	"encoding/json"
	"errors"
	"spaceship/server"
	"spaceship/socketapi"
)

type PTGame struct {}

var ptGameSpecs = server.GameSpecs{
	PlayerCount: 2,
	Mode: server.GAME_TYPE_ACTIVE_TURN_BASED,
}

const (
	PT_GAME_USER_STATE_WAITING_FOR_PLAY = iota
	PT_GAME_USER_STATE_COMPLETED
)

type PTGameUpdateData struct {
	FoundWordCount int
	FoundWordsLength int
	TotalDuration int
}

type PTGameUserData struct {
	UserID string
	State int
	FoundWordCount int
	FoundWordsLength int
	TotalDuration int
}

//Dummy struct for this example game
type PTGameData struct {
	Board string
	HomeUser *PTGameUserData
	AwayUser *PTGameUserData
}

func (tg *PTGame) GetName() string {
	//These value should be unique for each games
	return "testGame"
}

func (tg *PTGame) Init(gameData *socketapi.GameData) error {

	ptGameData := PTGameData{
		Board: "zzzxxxyyyaaabbbccc",
	}

	data, err := json.Marshal(ptGameData)
	if err != nil {
		return err
	}

	gameData.Metadata = string(data)

	return nil
}

func (tg *PTGame) Join(gameData *socketapi.GameData, session server.Session) error {

	var ptGameData PTGameData

	err := json.Unmarshal([]byte(gameData.Metadata), &ptGameData)
	if err != nil {
		return nil
	}

	if ptGameData.HomeUser == nil {
		ptGameData.HomeUser = &PTGameUserData{
			UserID: session.UserID(),
			State: PT_GAME_USER_STATE_WAITING_FOR_PLAY,
		}
	}else{
		ptGameData.AwayUser = &PTGameUserData{
			UserID: session.UserID(),
			State: PT_GAME_USER_STATE_WAITING_FOR_PLAY,
		}
	}

	ptGameRaw, err := json.Marshal(ptGameData)
	if err != nil {
		return err
	}

	gameData.Metadata = string(ptGameRaw)

	return nil
}

//func (tg *TestGame) Leave(gameID string, session server.Session) error {
//
//	return nil
//}

//Users should create their own metadata format. Ex: json string
func (tg *PTGame) Update(gameData *socketapi.GameData, session server.Session, metadata string) error {

	var ptGameUpdateData PTGameUpdateData
	err := json.Unmarshal([]byte(metadata), &ptGameUpdateData)
	if err != nil {
		return nil
	}

	var ptGameData PTGameData
	err = json.Unmarshal([]byte(gameData.Metadata), &ptGameData)
	if err != nil {
		return nil
	}

	if ptGameData.HomeUser != nil && ptGameData.HomeUser.UserID == session.UserID() {
		if ptGameData.HomeUser.State == PT_GAME_USER_STATE_WAITING_FOR_PLAY {
			ptGameData.HomeUser.FoundWordCount = ptGameUpdateData.FoundWordCount
			ptGameData.HomeUser.FoundWordsLength = ptGameUpdateData.FoundWordsLength
			ptGameData.HomeUser.TotalDuration = ptGameUpdateData.TotalDuration
			ptGameData.HomeUser.State = PT_GAME_USER_STATE_COMPLETED

			if ptGameData.AwayUser != nil && ptGameData.AwayUser.State == PT_GAME_USER_STATE_COMPLETED {
				//Game is finished

			}
		}else{
			return errors.New("This user was already sent update data for this game")
		}
	}else if ptGameData.AwayUser != nil && ptGameData.AwayUser.UserID == session.UserID() {
		if ptGameData.AwayUser.State == PT_GAME_USER_STATE_WAITING_FOR_PLAY {
			ptGameData.AwayUser.FoundWordCount = ptGameUpdateData.FoundWordCount
			ptGameData.AwayUser.FoundWordsLength = ptGameUpdateData.FoundWordsLength
			ptGameData.AwayUser.TotalDuration = ptGameUpdateData.TotalDuration
			ptGameData.AwayUser.State = PT_GAME_USER_STATE_COMPLETED

			if ptGameData.HomeUser != nil && ptGameData.HomeUser.State == PT_GAME_USER_STATE_COMPLETED {
				//Game is finished
			}
		}else{
			return errors.New("This user was already sent update data for this game")
		}
	}else{
		return errors.New("This user is not joined to this game")
	}

	data, err := json.Marshal(ptGameData)
	if err != nil {
		return err
	}

	gameData.Metadata = string(data)

	return nil
}

func (tg PTGame) GetGameSpecs() server.GameSpecs {
	return ptGameSpecs
}


