package game

import (
	"github.com/satori/go.uuid"
	"spaceship/server"
)

type TestGame struct {}

var testGameSpecs = server.GameSpecs{
	PlayerCount: 2,
	Mode: server.GAME_TYPE_ACTIVE_TURN_BASED,
}

func (tg *TestGame) GetName() string {
	//These value should be unique for each games
	return "testGame"
}

func (tg *TestGame) Create() uuid.UUID {
	gameID := uuid.NewV4()

	//TODO: we should use specific game struct for modes. But this struct can use a struct for common fields like id, game name etc...
	return gameID
}

func (tg *TestGame) Join(gameID uuid.UUID, session server.Session) error {

	return nil
}

func (tg *TestGame) Leave(gameID uuid.UUID, session server.Session) error {

	return nil
}

func (tg *TestGame) Update() {

}

func (tg TestGame) GetGameSpecs() server.GameSpecs {
	return testGameSpecs
}

