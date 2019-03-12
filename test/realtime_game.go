package test

import (
	"encoding/json"
	"spaceship/server"
	"spaceship/socketapi"
)

type RTGame struct {}

var rtGameSpecs = server.GameSpecs{
	PlayerCount: 2,
	Mode: server.GAME_TYPE_REAL_TIME,
	TickInterval: 1000,
}

const (
	RT_GAME_STATE_CONTINUE = iota
	RT_GAME_STATE_FINISHED
)

type RTGameUpdateData struct {
	Damage int
}

//Dummy struct for this example game
type RTGameData struct {
	GameState int
	BossHealth int
	WinnerUserID *string
}

func (tg *RTGame) GetName() string {
	//These value should be unique for each games
	return "realtimeTestGame"
}

func (tg *RTGame) Init(gameData *socketapi.GameData, logger *server.Logger) error {

	rtGameData := RTGameData{
		GameState: RT_GAME_STATE_CONTINUE,
		BossHealth: 300,
	}

	data, err := json.Marshal(rtGameData)
	if err != nil {
		return err
	}

	gameData.Metadata = string(data)

	return nil
}

func (tg *RTGame) Join(gameData *socketapi.GameData, session server.Session, notification *server.Notification, logger *server.Logger) error {

	return nil

}

func (tg *RTGame) Leave(gameData *socketapi.GameData, userID string, logger *server.Logger) error {

	return nil
}

//Users should create their own metadata format. Ex: json string
func (tg *RTGame) Update(gameData *socketapi.GameData, session server.Session, metadata string, leaderboard *server.Leaderboard, notification *server.Notification, logger *server.Logger) (bool, error) {
	return false, nil
}

func (tg *RTGame) Loop(gameData *socketapi.GameData, queuedDatas []socketapi.GameUpdateQueue, leaderboard *server.Leaderboard, notification *server.Notification, logger *server.Logger) bool {

	var rtGameData RTGameData
	err := json.Unmarshal([]byte(gameData.Metadata), &rtGameData)
	if err != nil {
		logger.Error(err)
		return true
	}

	isFinished := false
	for _, queueItem := range queuedDatas {

		var updateData RTGameUpdateData
		err = json.Unmarshal([]byte(queueItem.Metadata), &updateData)
		if err != nil {
			logger.Error(err)
			return true
		}

		rtGameData.BossHealth -= updateData.Damage

		if rtGameData.BossHealth <= 0 {
			rtGameData.BossHealth = 0
			rtGameData.GameState = RT_GAME_STATE_FINISHED
			rtGameData.WinnerUserID = &queueItem.UserID
			isFinished = true
		}

	}

	rtGameDataS, err := json.Marshal(rtGameData)
	if err != nil {
		return true
	}
	gameData.Metadata = string(rtGameDataS)

	return isFinished

}

func (tg RTGame) GetGameSpecs() server.GameSpecs {
	return rtGameSpecs
}
