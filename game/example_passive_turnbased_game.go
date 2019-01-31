package game

import (
	"encoding/json"
	"errors"
	"log"
	"spaceship/server"
	"spaceship/socketapi"
)

type ExampleGame struct {}

var exGameSpecs = server.GameSpecs{
	PlayerCount: 2,
	Mode: server.GAME_TYPE_PASSIVE_TURN_BASED,
}

const (
	EX_GAME_USER_STATE_WAITING_FOR_PLAY = iota
	EX_GAME_USER_STATE_COMPLETED
)

type EXGameUpdateData struct {
	FoundWordCount int
	FoundWordsLength int
	TotalDuration int
}

type EXGameUserData struct {
	UserID string
	State int
	FoundWordCount int
	FoundWordsLength int
	TotalDuration int
}

//Dummy struct for this example game
type EXGameData struct {
	Board string
	HomeUser *EXGameUserData
	AwayUser *EXGameUserData
}

func (tg *ExampleGame) GetName() string {
	//These value should be unique for each games
	return "exampleATGame"
}

func (tg *ExampleGame) Init(gameData *socketapi.GameData) error {

	ptGameData := EXGameData{
		Board: "zzzxxxyyyaaabbbccc",
	}

	data, err := json.Marshal(ptGameData)
	if err != nil {
		return err
	}

	gameData.Metadata = string(data)

	return nil
}

func (tg *ExampleGame) Join(gameData *socketapi.GameData, session server.Session) error {

	var ptGameData EXGameData

	err := json.Unmarshal([]byte(gameData.Metadata), &ptGameData)
	if err != nil {
		return nil
	}

	if ptGameData.HomeUser == nil {
		ptGameData.HomeUser = &EXGameUserData{
			UserID: session.UserID(),
			State: EX_GAME_USER_STATE_WAITING_FOR_PLAY,
		}
	}else{
		ptGameData.AwayUser = &EXGameUserData{
			UserID: session.UserID(),
			State: EX_GAME_USER_STATE_WAITING_FOR_PLAY,
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
func (tg *ExampleGame) Update(gameData *socketapi.GameData, session server.Session, metadata string, leaderboard *server.Leaderboard) (bool, error) {

	var ptGameUpdateData EXGameUpdateData
	err := json.Unmarshal([]byte(metadata), &ptGameUpdateData)
	if err != nil {
		return false, err
	}

	var ptGameData EXGameData
	err = json.Unmarshal([]byte(gameData.Metadata), &ptGameData)
	if err != nil {
		return false, err
	}

	isGameFinished := false

	if ptGameData.HomeUser != nil && ptGameData.HomeUser.UserID == session.UserID() {
		if ptGameData.HomeUser.State == EX_GAME_USER_STATE_WAITING_FOR_PLAY {
			ptGameData.HomeUser.FoundWordCount = ptGameUpdateData.FoundWordCount
			ptGameData.HomeUser.FoundWordsLength = ptGameUpdateData.FoundWordsLength
			ptGameData.HomeUser.TotalDuration = ptGameUpdateData.TotalDuration
			ptGameData.HomeUser.State = EX_GAME_USER_STATE_COMPLETED

			if ptGameData.AwayUser != nil && ptGameData.AwayUser.State == EX_GAME_USER_STATE_COMPLETED {
				//Game is finished
				isGameFinished = true

				if ptGameData.HomeUser.FoundWordCount > ptGameData.AwayUser.FoundWordCount {
					err = leaderboard.Score(ptGameData.HomeUser.UserID, tg.GetName(), 100)
					if err != nil {
						log.Println(err)
					}
					err = leaderboard.Score(ptGameData.AwayUser.UserID, tg.GetName(), 20)
					if err != nil {
						log.Println(err)
					}
				}else if ptGameData.HomeUser.FoundWordCount < ptGameData.AwayUser.FoundWordCount {
					err = leaderboard.Score(ptGameData.AwayUser.UserID, tg.GetName(), 100)
					if err != nil {
						log.Println(err)
					}
					err = leaderboard.Score(ptGameData.HomeUser.UserID, tg.GetName(), 20)
					if err != nil {
						log.Println(err)
					}
				}else{
					err = leaderboard.Score(ptGameData.HomeUser.UserID, tg.GetName(), 50)
					if err != nil {
						log.Println(err)
					}
					err = leaderboard.Score(ptGameData.AwayUser.UserID, tg.GetName(), 50)
					if err != nil {
						log.Println(err)
					}
				}
			}
		}else{
			return false, errors.New("This user was already sent update data for this game")
		}
	}else if ptGameData.AwayUser != nil && ptGameData.AwayUser.UserID == session.UserID() {
		if ptGameData.AwayUser.State == EX_GAME_USER_STATE_WAITING_FOR_PLAY {
			ptGameData.AwayUser.FoundWordCount = ptGameUpdateData.FoundWordCount
			ptGameData.AwayUser.FoundWordsLength = ptGameUpdateData.FoundWordsLength
			ptGameData.AwayUser.TotalDuration = ptGameUpdateData.TotalDuration
			ptGameData.AwayUser.State = EX_GAME_USER_STATE_COMPLETED

			if ptGameData.HomeUser != nil && ptGameData.HomeUser.State == EX_GAME_USER_STATE_COMPLETED {
				//Game is finished
				isGameFinished = true

				if ptGameData.HomeUser.FoundWordCount > ptGameData.AwayUser.FoundWordCount {
					err = leaderboard.Score(ptGameData.HomeUser.UserID, tg.GetName(), 100)
					if err != nil {
						log.Println(err)
					}
					err = leaderboard.Score(ptGameData.AwayUser.UserID, tg.GetName(), 20)
					if err != nil {
						log.Println(err)
					}
				}else if ptGameData.HomeUser.FoundWordCount < ptGameData.AwayUser.FoundWordCount {
					err = leaderboard.Score(ptGameData.AwayUser.UserID, tg.GetName(), 100)
					if err != nil {
						log.Println(err)
					}
					err = leaderboard.Score(ptGameData.HomeUser.UserID, tg.GetName(), 20)
					if err != nil {
						log.Println(err)
					}
				}else{
					err = leaderboard.Score(ptGameData.HomeUser.UserID, tg.GetName(), 50)
					if err != nil {
						log.Println(err)
					}
					err = leaderboard.Score(ptGameData.AwayUser.UserID, tg.GetName(), 50)
					if err != nil {
						log.Println(err)
					}
				}
			}
		}else{
			return false, errors.New("This user was already sent update data for this game")
		}
	}else{
		return false, errors.New("This user is not joined to this game")
	}

	data, err := json.Marshal(ptGameData)
	if err != nil {
		return false, err
	}

	gameData.Metadata = string(data)

	return isGameFinished, nil
}

func (tg *ExampleGame) Loop(gameData *socketapi.GameData, queuedDatas []socketapi.MatchUpdateQueue, leaderboard *server.Leaderboard) bool {

	return true

}

func (tg ExampleGame) GetGameSpecs() server.GameSpecs {
	return exGameSpecs
}


