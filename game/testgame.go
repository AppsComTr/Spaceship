package game

import (
	"log"
	"spaceship/server"
)

type TestGame struct {}

func (tg *TestGame) GetName() string {
	//These value should be unique for each games
	return "testGame"
}

func (tg *TestGame) Start(session server.Session) error {

	log.Println("testGame start")

	return nil
}
