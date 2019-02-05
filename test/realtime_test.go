package test

import (
	"encoding/json"
	"math/rand"
	"spaceship/socketapi"
	"testing"
	"time"
)

func TestRTGame(t *testing.T) {

	server := NewServer(t)
	defer server.Stop()

	failChan := make(chan string)
	gameIDChan := make(chan string, 2)
	doneChan := make(chan struct{}, 2)

	//We need to create two seperate client in two seperate routines
	for j := 0; j < 2; j++ {
		go func(){

			session := CreateSessionChan(failChan)
			conn, onMessageChan := CreateSocketConnChan(failChan, session.Token)
			defer conn.Close()

			WriteMessage(failChan, conn, &socketapi.Envelope{Cid: "", Message: &socketapi.Envelope_MatchFind{
				MatchFind: &socketapi.MatchFind{
					GameName: "realtimeTestGame",
					QueueProperties: map[string]string{"player_count": "2"},
				},
			}})

			var message socketapi.Envelope

			var matchEntry *socketapi.MatchEntry
			for {
				message = ReadMessage(failChan, onMessageChan)

				matchEntry = message.GetMatchEntry()
				if matchEntry == nil {
					failChan <- "Expected message match entry but unrecognized message was returned"
					return
				}

				if matchEntry.State == int32(socketapi.MatchEntry_MATCH_AWAITING_PLAYERS) {
					break
				}
			}

			WriteMessage(failChan, conn, &socketapi.Envelope{Cid:"", Message: &socketapi.Envelope_MatchJoin{
				MatchJoin: &socketapi.MatchJoin{
					MatchId: matchEntry.MatchId,
				},
			}})

			message = ReadMessage(failChan, onMessageChan)
			for {
				if message.GetMatchStart() != nil {
					break
				}
				message = ReadMessage(failChan, onMessageChan)
			}

			matchStart := message.GetMatchStart()
			if matchStart == nil {
				failChan <- "Expected message match start but unrecognized message was returned"
				return
			}

			gameIDChan <- matchStart.GameData.Id

			go func(){

				for i := 0; i < 20; i++ {

					rand.Seed(time.Now().Unix())
					//Send match update data
					matchUpdateData := RTGameUpdateData{
						Damage: rand.Intn(30-10) + 10,
					}
					matchUpdateRaw, err := json.Marshal(matchUpdateData)
					if err != nil {
						failChan <- err.Error()
						return
					}
					WriteMessage(failChan, conn, &socketapi.Envelope{Cid: "", Message: &socketapi.Envelope_MatchUpdate{
						MatchUpdate: &socketapi.MatchUpdate{
							GameID: matchStart.GameData.Id,
							Metadata: string(matchUpdateRaw),
						},
					}})

				}

			}()

			for {
				message = ReadMessage(failChan, onMessageChan)

				for {
					if message.GetMatchUpdateResp() != nil {
						break
					}
					message = ReadMessage(failChan, onMessageChan)
				}

				matchUpdateResp := message.GetMatchUpdateResp()
				if matchUpdateResp == nil {
					failChan <- "Expected message match update resp but unrecognized message was returned"
					return
				}

				var gameData RTGameData
				err := json.Unmarshal([]byte(matchUpdateResp.GameData.Metadata), &gameData)
				if err != nil{
					failChan <- err.Error()
					return
				}

				if gameData.GameState == RT_GAME_STATE_FINISHED {
					doneChan <- struct{}{}
					return
				}

			}

		}()
	}

	doneCount := 0
	prevGameID := ""

	ParentLoop:
	for {
		select {
		case err := <- failChan:
			t.Fatal(err)
			break
		case <- doneChan:
			doneCount++
			if doneCount == 2{
				break ParentLoop
			}
			break
		case gameID := <-gameIDChan:
			if prevGameID == "" {
				prevGameID = gameID
			}else{
				newGameID := gameID
				if prevGameID != newGameID {
					t.Fatal("Game IDs are not equal test failed", prevGameID, newGameID)
				}
			}
			break
		}
	}

}
