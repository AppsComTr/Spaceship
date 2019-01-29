package server

import (
	"log"
	"spaceship/socketapi"
)

func (p Pipeline) broadcastGame(gameData *socketapi.GameData) {
	//Need to fetch all users session by their ids from gameData and send them msg
	message := &socketapi.Envelope{Cid: "", Message: &socketapi.Envelope_MatchUpdateResp{MatchUpdateResp: &socketapi.MatchUpdateResp{GameData: gameData}}}
	for _, userID := range gameData.UserIDs {
		session := p.sessionHolder.GetByUserID(userID)
		err := session.Send(false, 0, message)
		if err != nil {
			log.Println("Error occured while trying to broadcast over game")
		}
	}
}
