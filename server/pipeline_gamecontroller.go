package server

import (
	"spaceship/socketapi"
)

func (p Pipeline) broadcastGame(gameData *socketapi.GameData) {
	//Need to fetch all users session by their ids from gameData and send them msg
	message := &socketapi.Envelope{Cid: "", Message: &socketapi.Envelope_GameUpdateResp{GameUpdateResp: &socketapi.GameUpdateResp{GameData: gameData}}}

	_ = p.pubSub.Send(&socketapi.PubSubMessage{
		UserIDs: gameData.UserIDs,
		Data: message,
	})
	//for _, userID := range gameData.UserIDs {
	//	session := p.sessionHolder.GetByUserID(userID)
	//	err := session.Send(false, 0, message)
	//	if err != nil {
	//		p.logger.Errorw("Error occured while trying to broadcast over game", "gameData", gameData, "error", err)
	//	}
	//}
}
