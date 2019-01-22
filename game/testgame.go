package game

import (
	"spaceship/server"
	"spaceship/socketapi"
)

type TestGame struct {}

var testGameSpecs = server.GameSpecs{
	PlayerCount: 2,
	Mode: server.GAME_TYPE_ACTIVE_TURN_BASED,
}

//Dummy struct for this example game
type TestGameMeta struct {
	Board string
}

func (tg *TestGame) GetName() string {
	//These value should be unique for each games
	return "testGame"
}

func (tg *TestGame) Init(gameData *socketapi.GameData) error {

	gameData.Metadata = "atatat"

	return nil
}

func (tg *TestGame) Join(gameData *socketapi.GameData, session server.Session) error {

	return nil
}

//func (tg *TestGame) Leave(gameID string, session server.Session) error {
//
//	return nil
//}

//Users should create their own metadata format. Ex: json string
func (tg *TestGame) Update(gameData *socketapi.GameData, session server.Session, metadata string) error {

	return nil
}

func (tg TestGame) GetGameSpecs() server.GameSpecs {
	return testGameSpecs
}

