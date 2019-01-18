package game

import (
	"spaceship/server"
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

func (tg *TestGame) Init(gameData *server.GameData) error {

	gameData.Metadata = "atatat"

	return nil
}

func (tg *TestGame) Join(gameID string, session server.Session) error {

	return nil
}

func (tg *TestGame) Leave(gameID string, session server.Session) error {

	return nil
}

//Users should create their own metadata format. Ex: json string
func (tg *TestGame) Update(gameData *server.GameData, session server.Session, metadata string) error {

	return nil
}

func (tg TestGame) GetGameSpecs() server.GameSpecs {
	return testGameSpecs
}

