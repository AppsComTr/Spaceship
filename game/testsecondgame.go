package game

import (
	"log"
	"spaceship/server"
)

type TestSecondGame struct {}

func (tg *TestSecondGame) GetName() string {
	//These value should be unique for each games
	return "testSecondGame"
}

func (tg *TestSecondGame) Start(session server.Session) error {

	log.Println("testSecondGame start")

	return nil
}

