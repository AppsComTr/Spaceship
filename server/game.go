package server

import (
	"errors"
	"github.com/kayalardanmehmet/redsync-radix"
	"github.com/mediocregopher/radix/v3"
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
func NewGame(matchID string, gameName string, holder *GameHolder, pipeline *Pipeline, session Session, logger *Logger, matchmaker Matchmaker) (*socketapi.GameData, error) {
	gameData := &socketapi.GameData{
		Id: matchID,
		Name: gameName,
		CreatedAt: time.Now().Unix(),
		UserIDs: make([]string, 0),
	}

	game := holder.Get(gameName)

	if game == nil {
		return nil, errors.New("Game couldn't found with given game name")
	}

	gameData.GameName = game.GetName()

	err := game.Init(gameData, logger)

	if err != nil {
		logger.Errorw("Error in game controllers init method", "error", err)
		return nil, err
	}

	gameDataS, err := holder.jsonProtoMarshler.MarshalToString(gameData)
	if err != nil {
		logger.Errorw("Error while trying to marshaling game data", "error", err)
		return nil, err
	}

	err = holder.redis.Do(radix.Cmd(nil, "SET", "game-" + gameData.Id, gameDataS))
	if err != nil {
		logger.Errorw("Redis error", "command", "SET", "key", "game-" + gameData.Id, "val", gameDataS, "error", err)
		return nil, err
	}

	if game.GetGameSpecs().TickInterval > 0 {
		//We should start timer
		ticker := time.NewTicker(time.Millisecond * time.Duration(game.GetGameSpecs().TickInterval))

		go func(){

			for {
				select {
				case <- ticker.C:

					if !loopGame(holder, gameData.Id, game, ticker, pipeline, logger, matchmaker){
						return
					}

					break
				}
			}

		}()
	}

	return gameData, nil
}

func loopGame(holder *GameHolder, gameID string, game GameController, ticker *time.Ticker, pipeline *Pipeline, logger *Logger, matchmaker Matchmaker) bool {

	rs := redsyncradix.New([]radix.Client{holder.redis})
	mutex := rs.NewMutex("lock|" + gameID)
	if err := mutex.Lock(); err != nil {
		logger.Errorw("Error while using lock", "key", "lock|" + gameID, "error", err)
	} else {
		defer mutex.Unlock()
	}

	var gameDataS string
	err := holder.redis.Do(radix.Cmd(&gameDataS, "GET", "game-" + gameID))
	if err != nil {
		logger.Errorw("Redis error", "command", "GET", "key", "game-" + gameID, "error", err)
		return false
	}

	var rGameData socketapi.GameData
	err = holder.jsonProtoUnmarshler.Unmarshal(strings.NewReader(gameDataS), &rGameData)
	if err != nil {
		logger.Errorw("Error while unmarshaling game data", "gameData", gameDataS, "error", err)
		return false
	}

	queuedDatasS := make([]string, 0)
	err = holder.redis.Do(radix.Cmd(&queuedDatasS, "LRANGE", "game|update|" + gameID, "0", "-1"))
	if err != nil {
		logger.Errorw("Redis error", "command", "LRANGE", "key", "game|update|" + gameID, "error", err)
		return false
	}

	queuedDatas := make([]socketapi.GameUpdateQueue, 0)
	for _, queuedDataS := range queuedDatasS {

		var queuedData socketapi.GameUpdateQueue
		err = holder.jsonProtoUnmarshler.Unmarshal(strings.NewReader(queuedDataS), &queuedData)
		if err != nil {
			logger.Errorw("Error while unmarshling queued data", "data", queuedDatas, "error", err)
			return false
		}

		queuedDatas = append(queuedDatas, queuedData)

	}

	isFinished := game.Loop(&rGameData, queuedDatas, holder.leaderboard, holder.notification, logger)

	rGameData.UpdatedAt = time.Now().Unix()

	if isFinished {

		gameDataDB := model.GameData{}
		gameDataDB.MapFromPB(&rGameData)

		conn := pipeline.db.Copy()
		defer conn.Close()
		db := conn.DB("spaceship")

		err = db.C(gameDataDB.GetCollectionName()).Insert(gameDataDB)
		if err != nil {
			logger.Errorw("Error while inserting game data to db", "data", gameDataDB, "error", err)
			return false
		}

		err = holder.redis.Do(radix.Cmd(nil, "DEL", "game-" + rGameData.Id))
		if err != nil {
			logger.Errorw("Redis error", "command", "DEL", "key", "game-" + rGameData.Id, "error", err)
		}
		matchmaker.ClearMatch(rGameData.Id)

		ticker.Stop()

	}else{

		gameDataS, err = holder.jsonProtoMarshler.MarshalToString(&rGameData)
		if err != nil {
			logger.Errorw("Error while marshling game date", "gameData", rGameData, "error", err)
			return false
		}

		err = holder.redis.Do(radix.Cmd(nil, "SET", "game-" + rGameData.Id, gameDataS))
		if err != nil {
			logger.Errorw("Redis error", "command", "SET", "key", "game-" + rGameData.Id, "error", err)
		}

	}

	_ = holder.redis.Do(radix.Cmd(nil, "DEL", "game|update|" + gameID))

	pipeline.broadcastGame(&rGameData)

	return !isFinished

}

func JoinGame(gameID string, holder *GameHolder, session Session, logger *Logger) (*socketapi.GameData, error) {
	gameData := &socketapi.GameData{}

	var gameDataS string

	err := holder.redis.Do(radix.Cmd(&gameDataS, "GET", "game-" + gameID))
	if err != nil {
		logger.Errorw("Redis error", "command", "GET", "key", "game-" + gameID, "error", err)
		return nil, err
	}

	err = holder.jsonProtoUnmarshler.Unmarshal(strings.NewReader(gameDataS), gameData)
	if err != nil {
		return nil, err
	}

	game := holder.Get(gameData.GameName)
	if game == nil {
		return nil, errors.New("Game couldn't found with given game name")
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

	err = game.Join(gameData, session, holder.notification, logger)
	if err != nil {
		logger.Errorw("Error in game controllers join method", "error", err)
		return nil, err
	}

	gameData.UpdatedAt = time.Now().Unix()

	gameDataS, err = holder.jsonProtoMarshler.MarshalToString(gameData)
	if err != nil {
		logger.Errorw("Error while trying to marshaling game data", "error", err)
		return nil, err
	}

	err = holder.redis.Do(radix.Cmd(nil, "SET", "game-" + gameID, gameDataS))
	if err != nil {
		logger.Errorw("Redis error", "command", "game-" + gameID, "val", gameDataS, "error", err)
		return nil, err
	}

	return gameData, nil
}

func LeaveGame(gameID string, holder *GameHolder, userID string, logger *Logger) (*socketapi.GameData, error) {

	gameData := &socketapi.GameData{}

	var gameDataS string

	err := holder.redis.Do(radix.Cmd(&gameDataS, "GET", "game-" + gameID))
	if err != nil {
		logger.Errorw("Redis error", "command", "GET", "key", "game-" + gameID, "error", err)
		return nil, err
	}

	err = holder.jsonProtoUnmarshler.Unmarshal(strings.NewReader(gameDataS), gameData)
	if err != nil {
		logger.Errorw("Error while trying to unmarshal game data", "gameData", gameDataS, "error", err)
		return nil, err
	}

	game := holder.Get(gameData.GameName)
	if game == nil {
		return nil, errors.New("Game couldn't found with given game name")
	}

	userIndex := -1
	for i, uID := range gameData.UserIDs {
		if uID == userID {
			userIndex = i
			break
		}
	}

	if userIndex == -1 {
		return nil, errors.New("This user is already not in this game")
	}

	gameData.UserIDs = append(gameData.UserIDs[:userIndex], gameData.UserIDs[userIndex+1:]...)

	err = game.Leave(gameData, userID, logger)
	if err != nil {
		logger.Errorw("Error in game controllers leave method", "error", err)
		return nil, err
	}

	gameData.UpdatedAt = time.Now().Unix()

	gameDataS, err = holder.jsonProtoMarshler.MarshalToString(gameData)
	if err != nil {
		logger.Errorw("Error while trying to marshal game data", "error", err)
		return nil, err
	}

	err = holder.redis.Do(radix.Cmd(nil, "SET", "game-" + gameData.Id, gameDataS))
	if err != nil {
		logger.Errorw("Redis error", "command", "SET", "key", "game-" + gameData.Id, "val", gameDataS, "error", err)
		return nil, err
	}

	return gameData, nil

}

func UpdateGame(holder *GameHolder, session Session, pipeline *Pipeline, updateData *socketapi.GameUpdate, logger *Logger, matchmaker Matchmaker) (*socketapi.GameData, error) {

	gameData := &socketapi.GameData{}

	var gameDataS string

	err := holder.redis.Do(radix.Cmd(&gameDataS, "GET", "game-" + updateData.GameID))
	if err != nil {
		logger.Errorw("Redis error", "command", "GET", "key", "game-" + updateData.GameID)
		return nil, err
	}

	err = holder.jsonProtoUnmarshler.Unmarshal(strings.NewReader(gameDataS), gameData)
	if err != nil {
		logger.Errorw("Erroro while trying to unmarshal game data", "gameData", gameDataS, "error", err)
		return nil, err
	}

	game := holder.Get(gameData.GameName)
	if game == nil {
		return nil, errors.New("Game couldn't found with given game name")
	}

	if game.GetGameSpecs().TickInterval > 0 {

		queueUpdateData := &socketapi.GameUpdateQueue{
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
			logger.Errorw("Redis error", "command", "RPUSH", "key", "game|update|" + gameData.Id, "data", updateDataS, "error", err)
			return nil, err
		}

		return gameData, nil

	}else{

		isFinished, err := game.Update(gameData, session, updateData.Metadata, holder.leaderboard, holder.notification, logger)
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
				logger.Errorw("Error while trying to insert game data to database", "data", gameDataDB, "error", err)
				return nil, err
			}

			err = holder.redis.Do(radix.Cmd(nil, "DEL", "game-" + gameData.Id))
			if err != nil {
				logger.Errorw("Redis error", "command", "DEL", "key", "game-" + gameData.Id, "error", err)
			}
			matchmaker.ClearMatch(gameData.Id)

		}else{

			gameDataS, err = holder.jsonProtoMarshler.MarshalToString(gameData)
			if err != nil {
				logger.Errorw("Error while trying to marshal game data", "gameData", gameData, "error", err)
				return nil, err
			}

			err = holder.redis.Do(radix.Cmd(nil, "SET", "game-" + gameData.Id, gameDataS))
			if err != nil {
				logger.Errorw("Redis error", "command", "SET", "key", "game-" + gameData.Id, "val", gameDataS, "error", err)
				return nil, err
			}

		}

		pipeline.broadcastGame(gameData)

		return gameData, nil

	}

}