package server

import (
	"errors"
	"github.com/kayalardanmehmet/redsync-radix"
	"github.com/mediocregopher/radix/v3"
	"log"
	"spaceship/model"
	"spaceship/socketapi"
	"strings"
	"time"
)

const (
	GAME_TYPE_PASSIVE_TURN_BASED int = iota
	GAME_TYPE_ACTIVE_TURN_BASED
	GAME_TYPE_REAL_TIME
)

type GameSpecs struct {
	PlayerCount int
	Mode int
	TickInterval int
}

//This should be used in matchmaker module
func NewGame(matchID string, modeName string, holder *GameHolder, pipeline *Pipeline, session Session) (*socketapi.GameData, error) {
	gameData := &socketapi.GameData{
		Id: matchID,
		Name: modeName,
		CreatedAt: time.Now().Unix(),
	}

	game := holder.Get(modeName)

	if game == nil {
		return nil, errors.New("Game couldn't found with given mode name")
	}

	gameData.ModeName = game.GetName()

	err := game.Init(gameData)

	if err != nil {
		return nil, err
	}

	gameDataS, err := holder.jsonProtoMarshler.MarshalToString(gameData)
	if err != nil {
		return nil, err
	}

	err = holder.redis.Do(radix.Cmd(nil, "SET", "game-" + gameData.Id, gameDataS))
	if err != nil {
		return nil, err
	}

	if game.GetGameSpecs().TickInterval > 0 {
		//We should start timer
		ticker := time.NewTicker(time.Millisecond * time.Duration(game.GetGameSpecs().TickInterval))

		go func(){

			for {
				select {
				case <- ticker.C:

					if !loopGame(holder, gameData, game, ticker, pipeline){
						return
					}

					break
				}
			}

		}()
	}

	return gameData, nil
}

func loopGame(holder *GameHolder, gameData *socketapi.GameData, game GameController, ticker *time.Ticker, pipeline *Pipeline) bool {

	rs := redsyncradix.New([]radix.Client{holder.redis})
	mutex := rs.NewMutex("lock|" + gameData.Id)
	if err := mutex.Lock(); err != nil {
		log.Println(err.Error())
	} else {
		defer mutex.Unlock()
	}

	var gameDataS string
	err := holder.redis.Do(radix.Cmd(&gameDataS, "GET", "game-" + gameData.Id))
	if err != nil {
		log.Println(err)
		return false
	}

	var rGameData *socketapi.GameData
	err = holder.jsonProtoUnmarshler.Unmarshal(strings.NewReader(gameDataS), rGameData)
	if err != nil {
		log.Println(err)
		return false
	}

	queuedDatasS := make([]string, 0)
	err = holder.redis.Do(radix.Cmd(&queuedDatasS, "LRANGE", "game|update|" + gameData.Id, "0", "-1"))
	if err != nil {
		log.Println(err)
		return false
	}

	queuedDatas := make([]socketapi.MatchUpdateQueue, 0)
	for _, queuedDataS := range queuedDatasS {

		var queuedData socketapi.MatchUpdateQueue
		err = holder.jsonProtoUnmarshler.Unmarshal(strings.NewReader(queuedDataS), &queuedData)
		if err != nil {
			log.Println(err)
			return false
		}

		queuedDatas = append(queuedDatas, queuedData)

	}

	isFinished := game.Loop(rGameData, queuedDatas)

	rGameData.UpdatedAt = time.Now().Unix()

	if isFinished {

		gameDataDB := model.GameData{}
		gameDataDB.MapFromPB(gameData)

		conn := pipeline.db.Copy()
		defer conn.Close()
		db := conn.DB("spaceship")

		err = db.C(gameDataDB.GetCollectionName()).Insert(gameDataDB)
		if err != nil {
			log.Println(err)
			return false
		}

		err = holder.redis.Do(radix.Cmd(nil, "DEL", "game-" + rGameData.Id))
		if err != nil {
			log.Println(err)
		}

		ticker.Stop()

	}else{

		gameDataS, err = holder.jsonProtoMarshler.MarshalToString(gameData)
		if err != nil {
			log.Println(err)
			return false
		}

		err = holder.redis.Do(radix.Cmd(nil, "SET", "game-" + rGameData.Id, gameDataS))
		if err != nil {
			log.Println(err)
		}

	}

	_ = holder.redis.Do(radix.Cmd(nil, "DEL", "game|update|" + gameData.Id))

	pipeline.broadcastGame(gameData)

	return !isFinished

}

func JoinGame(gameID string, holder *GameHolder, session Session) (*socketapi.GameData, error) {
	gameData := &socketapi.GameData{}

	var gameDataS string

	err := holder.redis.Do(radix.Cmd(&gameDataS, "GET", "game-" + gameID))
	if err != nil {
		return nil, err
	}

	err = holder.jsonProtoUnmarshler.Unmarshal(strings.NewReader(gameDataS), gameData)
	if err != nil {
		return nil, err
	}

	game := holder.Get(gameData.ModeName)
	if game == nil {
		return nil, errors.New("Game couldn't found with given mode name")
	}

	//Check if user is already joined to this game
	userExists := false
	for _, userID := range gameData.UserIDs {
		if userID == session.UserID() {
			userExists = true
			break
		}
	}
	if userExists {
		return nil, errors.New("This user already joined to this game")
	}

	gameData.UserIDs = append(gameData.UserIDs, session.UserID())

	err = game.Join(gameData, session)
	if err != nil {
		return nil, err
	}

	gameData.UpdatedAt = time.Now().Unix()

	gameDataS, err = holder.jsonProtoMarshler.MarshalToString(gameData)
	if err != nil {
		return nil, err
	}

	err = holder.redis.Do(radix.Cmd(nil, "SET", "game-" + gameData.Id, gameDataS))
	if err != nil {
		return nil, err
	}

	//TODO: maybe we need to broadcast join data to inform clients
	return gameData, nil
}

//func LeaveGame(gameID string, holder *GameHolder, session Session, pipeline *Pipeline) (*socketapi.GameData, error) {
//
//	gameData := &socketapi.GameData{}
//
//	var gameDataS string
//
//	err := holder.redis.Do(radix.Cmd(&gameDataS, "GET", gameID))
//	if err != nil {
//		return nil, err
//	}
//
//	err = holder.jsonProtoUnmarshler.Unmarshal(strings.NewReader(gameDataS), gameData)
//	if err != nil {
//		return nil, err
//	}
//
//	game := holder.Get(gameData.ModeName)
//	if game == nil {
//		return nil, errors.New("Game couldn't found with given mode name")
//	}
//
//	userIndex := -1
//	for i, userID := range gameData.UserIDs {
//		if userID == session.UserID() {
//			userIndex = i
//			break
//		}
//	}
//
//	if userIndex == -1 {
//		return nil, errors.New("This user is already not in this game")
//	}
//
//	gameData.UserIDs = append(gameData.UserIDs[:userIndex], gameData.UserIDs[userIndex+1:]...)
//
//	err = game.Leave(gameData, session)
//	if err != nil {
//		return nil, err
//	}
//
//	gameData.UpdatedAt = time.Now().Unix()
//
//	gameDataS, err = holder.jsonProtoMarshler.MarshalToString(gameData)
//	if err != nil {
//		return nil, err
//	}
//
//	err = holder.redis.Do(radix.Cmd(nil, "SET", gameData.Id, gameDataS))
//	if err != nil {
//		return nil, err
//	}
//
//	//TODO: maybe we need to broadcast updated data to inform clients
//	return gameData, nil
//
//}

func UpdateGame(holder *GameHolder, session Session, pipeline *Pipeline, updateData *socketapi.MatchUpdate) (*socketapi.GameData, error) {

	gameData := &socketapi.GameData{}

	var gameDataS string

	err := holder.redis.Do(radix.Cmd(&gameDataS, "GET", "game-" + updateData.GameID))
	if err != nil {
		return nil, err
	}

	err = holder.jsonProtoUnmarshler.Unmarshal(strings.NewReader(gameDataS), gameData)
	if err != nil {
		return nil, err
	}

	game := holder.Get(gameData.ModeName)
	if game == nil {
		return nil, errors.New("Game couldn't found with given mode name")
	}

	//TODO: we need to check if game exists with given game id
	if game.GetGameSpecs().TickInterval > 0 {

		queueUpdateData := &socketapi.MatchUpdateQueue{
			GameID: updateData.GameID,
			UserID: session.UserID(),
			Metadata: updateData.Metadata,
		}

		updateDataS, err := holder.jsonProtoMarshler.MarshalToString(queueUpdateData)
		if err != nil {
			return nil, err
		}

		err = holder.redis.Do(radix.Cmd(nil, "RPUSH", "game|update|" + gameData.Id, updateDataS))
		if err != nil {
			return nil, err
		}

		return gameData, nil

	}else{

		isFinished, err := game.Update(gameData, session, updateData.Metadata)
		if err != nil {
			return nil, err
		}

		gameData.UpdatedAt = time.Now().Unix()

		if isFinished {

			gameDataDB := model.GameData{}
			gameDataDB.MapFromPB(gameData)

			conn := pipeline.db.Copy()
			defer conn.Close()
			db := conn.DB("spaceship")

			err = db.C(gameDataDB.GetCollectionName()).Insert(gameDataDB)
			if err != nil {
				return nil, err
			}

			err = holder.redis.Do(radix.Cmd(nil, "DEL", "game-" + gameData.Id))
			if err != nil {
				log.Println(err)
			}

		}else{

			gameDataS, err = holder.jsonProtoMarshler.MarshalToString(gameData)
			if err != nil {
				return nil, err
			}

			err = holder.redis.Do(radix.Cmd(nil, "SET", "game-" + gameData.Id, gameDataS))
			if err != nil {
				return nil, err
			}

		}

		pipeline.broadcastGame(gameData)

		return gameData, nil

	}

}