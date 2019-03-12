package test

import (
	"encoding/json"
	"github.com/pkg/errors"
	"spaceship/server"
	"spaceship/socketapi"
)

type ATGame struct {}

var ATGameSpecs = server.GameSpecs{
	PlayerCount: 2,
	Mode: server.GAME_TYPE_ACTIVE_TURN_BASED,
}

const (
	AT_GAME_USER_STATE_WAITING_FOR_PLAY = iota
	AT_GAME_USER_STATE_COMPLETED
)

type ATGameUpdateData struct {
	FoundWordCount int
	FoundWordsLength int
	TotalDuration int
}

type ATGameUserData struct {
	UserID string
	State int
	FoundWordCount int
	FoundWordsLength int
	TotalDuration int
}

//Dummy struct for this example game
type ATGameData struct {
	Board string
	HomeUser *ATGameUserData
	AwayUser *ATGameUserData
}

func (tg *ATGame) GetName() string {
	//These value should be unique for each games
	return "ATGame"
}

func (tg *ATGame) Init(gameData *socketapi.GameData, logger *server.Logger) error {

	ptGameData := ATGameData{
		Board: "zzzxxxyyyaaabbbccc",
	}

	data, err := json.Marshal(ptGameData)
	if err != nil {
		return err
	}

	gameData.Metadata = string(data)

	return nil
}

func (tg *ATGame) Join(gameData *socketapi.GameData, session server.Session, notification *server.Notification, logger *server.Logger) error {

	var ptGameData ATGameData

	err := json.Unmarshal([]byte(gameData.Metadata), &ptGameData)
	if err != nil {
		return nil
	}

	if ptGameData.HomeUser == nil {
		ptGameData.HomeUser = &ATGameUserData{
			UserID: session.UserID(),
			State: AT_GAME_USER_STATE_WAITING_FOR_PLAY,
		}
	}else{
		ptGameData.AwayUser = &ATGameUserData{
			UserID: session.UserID(),
			State: AT_GAME_USER_STATE_WAITING_FOR_PLAY,
		}
	}

	ptGameRaw, err := json.Marshal(ptGameData)
	if err != nil {
		return err
	}

	gameData.Metadata = string(ptGameRaw)

	return nil
}

func (tg *ATGame) Leave(gameData *socketapi.GameData, userID string, logger *server.Logger) error {

	return nil
}

//Users should create their own metadata format. Ex: json string
func (tg *ATGame) Update(gameData *socketapi.GameData, session server.Session, metadata string, leaderboard *server.Leaderboard, notification *server.Notification, logger *server.Logger) (bool, error) {

	var ptGameUpdateData ATGameUpdateData
	err := json.Unmarshal([]byte(metadata), &ptGameUpdateData)
	if err != nil {
		return false, err
	}

	var ptGameData ATGameData
	err = json.Unmarshal([]byte(gameData.Metadata), &ptGameData)
	if err != nil {
		return false, err
	}

	isGameFinished := false

	if ptGameData.HomeUser != nil && ptGameData.HomeUser.UserID == session.UserID() {
		if ptGameData.HomeUser.State == AT_GAME_USER_STATE_WAITING_FOR_PLAY {
			ptGameData.HomeUser.FoundWordCount = ptGameUpdateData.FoundWordCount
			ptGameData.HomeUser.FoundWordsLength = ptGameUpdateData.FoundWordsLength
			ptGameData.HomeUser.TotalDuration = ptGameUpdateData.TotalDuration
			ptGameData.HomeUser.State = AT_GAME_USER_STATE_COMPLETED

			if ptGameData.AwayUser != nil && ptGameData.AwayUser.State == AT_GAME_USER_STATE_COMPLETED {
				//Game is finished
				isGameFinished = true

				if ptGameData.HomeUser.FoundWordCount > ptGameData.AwayUser.FoundWordCount {
					err = leaderboard.Score(ptGameData.HomeUser.UserID, tg.GetName(), 100)
					if err != nil {
						logger.Error(err)
					}
					err = leaderboard.Score(ptGameData.AwayUser.UserID, tg.GetName(), 20)
					if err != nil {
						logger.Error(err)
					}
				}else if ptGameData.HomeUser.FoundWordCount < ptGameData.AwayUser.FoundWordCount {
					err = leaderboard.Score(ptGameData.AwayUser.UserID, tg.GetName(), 100)
					if err != nil {
						logger.Error(err)
					}
					err = leaderboard.Score(ptGameData.HomeUser.UserID, tg.GetName(), 20)
					if err != nil {
						logger.Error(err)
					}
				}else{
					err = leaderboard.Score(ptGameData.HomeUser.UserID, tg.GetName(), 50)
					if err != nil {
						logger.Error(err)
					}
					err = leaderboard.Score(ptGameData.AwayUser.UserID, tg.GetName(), 50)
					if err != nil {
						logger.Error(err)
					}
				}
			}
		}else{
			return false, errors.New("This user was already sent update data for this game")
		}
	}else if ptGameData.AwayUser != nil && ptGameData.AwayUser.UserID == session.UserID() {
		if ptGameData.AwayUser.State == AT_GAME_USER_STATE_WAITING_FOR_PLAY {
			ptGameData.AwayUser.FoundWordCount = ptGameUpdateData.FoundWordCount
			ptGameData.AwayUser.FoundWordsLength = ptGameUpdateData.FoundWordsLength
			ptGameData.AwayUser.TotalDuration = ptGameUpdateData.TotalDuration
			ptGameData.AwayUser.State = AT_GAME_USER_STATE_COMPLETED

			if ptGameData.HomeUser != nil && ptGameData.HomeUser.State == AT_GAME_USER_STATE_COMPLETED {
				//Game is finished
				isGameFinished = true

				if ptGameData.HomeUser.FoundWordCount > ptGameData.AwayUser.FoundWordCount {
					err = leaderboard.Score(ptGameData.HomeUser.UserID, tg.GetName(), 100)
					if err != nil {
						logger.Error(err)
					}
					err = leaderboard.Score(ptGameData.AwayUser.UserID, tg.GetName(), 20)
					if err != nil {
						logger.Error(err)
					}
				}else if ptGameData.HomeUser.FoundWordCount < ptGameData.AwayUser.FoundWordCount {
					err = leaderboard.Score(ptGameData.AwayUser.UserID, tg.GetName(), 100)
					if err != nil {
						logger.Error(err)
					}
					err = leaderboard.Score(ptGameData.HomeUser.UserID, tg.GetName(), 20)
					if err != nil {
						logger.Error(err)
					}
				}else{
					err = leaderboard.Score(ptGameData.HomeUser.UserID, tg.GetName(), 50)
					if err != nil {
						logger.Error(err)
					}
					err = leaderboard.Score(ptGameData.AwayUser.UserID, tg.GetName(), 50)
					if err != nil {
						logger.Error(err)
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

func (tg *ATGame) Loop(gameData *socketapi.GameData, queuedDatas []socketapi.GameUpdateQueue, leaderboard *server.Leaderboard, notification *server.Notification, logger *server.Logger) bool {

	return true

}

func (tg ATGame) GetGameSpecs() server.GameSpecs {
	return ATGameSpecs
}




