package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/kayalardanmehmet/redsync-radix"
	"github.com/mediocregopher/radix/v3"
	"github.com/satori/go.uuid"
	"spaceship/socketapi"
	"sync"
	"time"
)

//Adds and removes users from matchmaking queue
type Matchmaker interface {
	Find(session Session,gameName string, queueProperties map[string]string) (*socketapi.MatchEntry, error)
	Join(pipeline *Pipeline, session Session, matchID string) (*socketapi.GameData, error)
	Leave(session Session, matchID string) error
	LeaveActiveGames(userID string) error
	SetPipeline(pipeline *Pipeline)
	ClearMatch(matchID string)
}

type LocalMatchmaker struct {
	sync.RWMutex
	redis radix.Client
	gameHolder *GameHolder
	sessionHolder *SessionHolder
	notification *Notification
	logger *Logger
	config *Config
	pipeline *Pipeline

	context context.Context

	entries map[string]*socketapi.MatchEntry
}

func NewLocalMatchMaker(redis radix.Client, gameHolder *GameHolder, sessionHolder *SessionHolder, notification *Notification, logger *Logger, config *Config, context context.Context) Matchmaker {
	return &LocalMatchmaker{
		redis: redis,
		gameHolder: gameHolder,
		entries: make(map[string]*socketapi.MatchEntry),
		sessionHolder: sessionHolder,
		notification: notification,
		logger: logger,
		config: config,
		context: context,
	}
}

func (m *LocalMatchmaker) SetPipeline(pipeline *Pipeline){
	m.pipeline = pipeline
}

func (m *LocalMatchmaker) Find(session Session, gameName string, queueProperties map[string]string) (*socketapi.MatchEntry, error){
	game := m.gameHolder.Get(gameName)
	if game == nil {
		return nil, errors.New("can't find game for this request")
	}

	queueKey := m.generateQueueKey(gameName, queueProperties)

	rs := redsyncradix.New([]radix.Client{m.redis})
	mutex := rs.NewMutex("lock|gamequeue|" + queueKey)
	if err := mutex.Lock(); err != nil {
		m.logger.Errorw("Error while using lock", "key", "lock|gamequeue|" + queueKey, "error", err)
	} else {
		defer mutex.Unlock()
	}

	playerCount := game.GetGameSpecs().PlayerCount

	if game.GetGameSpecs().Mode == GAME_TYPE_PASSIVE_TURN_BASED {
		var matchID string
		err := m.redis.Do(radix.Cmd(&matchID, "LINDEX", queueKey, "0"))
		if err != nil {
			m.logger.Errorw("Redis error", "command", "LINDEX", "queueKey", queueKey, "error", err)
			return nil, err
		}

		if matchID == "" {//Create match
			matchID = uuid.NewV4().String()

			users := make([]*socketapi.MatchEntry_MatchUser, 0, playerCount)
			users = append(users, &socketapi.MatchEntry_MatchUser{
				UserId:session.UserID(),
				Username: session.Username(),
				State: int32(socketapi.MatchEntry_MatchUser_NOT_READY),
			})

			matchEntry := &socketapi.MatchEntry{
				MatchId: matchID,
				MaxCount: int32(playerCount),
				ActiveCount: 1,
				Users: users,
				GameName: gameName,
				Queuekey: queueKey,
				State: int32(socketapi.MatchEntry_MATCH_FINDING_PLAYERS),
			}

			err = m.redis.Do(radix.Cmd(nil, "LPUSH", queueKey, matchID))
			if err != nil {
				m.logger.Errorw("Redis error", "command", "LPUSH", "queueKey", queueKey, "matchID", matchID, "error", err)
				return nil, err
			}

			err = m.redis.Do(radix.Cmd(nil, "SADD", matchID, session.UserID()))
			if err != nil {
				m.logger.Errorw("Redis error", "command", "SADD", "matchID", matchID, "userID", session.UserID(), "error", err)
				return nil, err
			}

			err = m.redis.Do(radix.Cmd(nil, "SADD", "pm:" + session.UserID(), matchID))
			if err != nil {
				m.logger.Errorw("Redis error", "command", "SADD", "key", "pm:" + session.UserID(), "matchID", matchID, "error", err)
				return nil, err
			}

			m.entries[matchID] = matchEntry

			return matchEntry, nil
		}else {//Join existing match
			var isMember int
			err = m.redis.Do(radix.Cmd(&isMember, "SISMEMBER", matchID, session.UserID()))
			if err != nil {
				m.logger.Errorw("Redis error", "command", "SISMEMBER", "matchID", matchID, "userID", session.UserID(), "error", err)
				return nil, err
			}
			matchEntry, _ := m.entries[matchID]

			if isMember != 1 && matchEntry.ActiveCount < int32(playerCount){
				matchEntry.ActiveCount++
				matchEntry.Users = append(matchEntry.Users, &socketapi.MatchEntry_MatchUser{
					UserId:session.UserID(),
					State: int32(socketapi.MatchEntry_MatchUser_NOT_READY),
				})
				m.entries[matchID] = matchEntry

				err = m.redis.Do(radix.Cmd(nil, "SADD", "pm:" + session.UserID(), matchID))
				if err != nil {
					m.logger.Errorw("Redis error", "command", "SADD", "key", "pm:" + session.UserID(), "matchID", matchID, "error", err)
					return nil, err
				}

				if (matchEntry.ActiveCount) == int32(playerCount) {
					err = m.redis.Do(radix.Cmd(nil, "LREM", queueKey, "0", matchID))
					if err != nil {
						m.logger.Errorw("Redis error", "command", "LREM", "key", queueKey, "matchID", matchID)
						return nil, err
					}
					err = m.redis.Do(radix.Cmd(nil, "DEL", matchID))
					if err != nil {
						m.logger.Errorw("Redis error", "command", "DEL", "key", matchID, "error", err)
						return nil, err
					}
				}
			}

			return matchEntry, nil
		}
	}else if game.GetGameSpecs().Mode == GAME_TYPE_ACTIVE_TURN_BASED || game.GetGameSpecs().Mode == GAME_TYPE_REAL_TIME {
		var matchID string
		err := m.redis.Do(radix.Cmd(&matchID, "LINDEX", queueKey, "0"))
		if err != nil {
			m.logger.Errorw("Redis error", "command", "LINDEX", "key", queueKey, "error", err)
			return nil, err
		}

		if matchID == "" {
			matchID = uuid.NewV4().String()

			users := make([]*socketapi.MatchEntry_MatchUser, 0, playerCount)
			users = append(users, &socketapi.MatchEntry_MatchUser{
				UserId: session.UserID(),
				Username: session.Username(),
				State: int32(socketapi.MatchEntry_MatchUser_NOT_READY),
			})

			matchEntry := &socketapi.MatchEntry{
				MatchId: matchID,
				MaxCount: int32(playerCount),
				ActiveCount: 1,
				Users: users,
				GameName: gameName,
				Queuekey: queueKey,
				State: int32(socketapi.MatchEntry_MATCH_FINDING_PLAYERS),
			}

			err = m.redis.Do(radix.Cmd(nil, "LPUSH", queueKey, matchID))
			if err != nil {
				m.logger.Errorw("Redis error", "command", "LPUSH", "key", queueKey, "matchID", matchID, "error", err)
				return nil, err
			}

			err = m.redis.Do(radix.Cmd(nil, "SADD", matchID, session.UserID()))
			if err != nil {
				m.logger.Errorw("Redis error", "command", "SADD", "key", matchID, "userID", session.UserID(), "error", err)
				return nil, err
			}

			err = m.redis.Do(radix.Cmd(nil, "SADD", "pm:" + session.UserID(), matchID))
			if err != nil {
				m.logger.Errorw("Redis error", "command", "SADD", "key", "pm:" + session.UserID(), "matchID", matchID, "error", err)
				return nil, err
			}

			err = m.redis.Do(radix.Cmd(nil, "SADD", "pam:" + session.UserID(), matchID))
			if err != nil {
				m.logger.Errorw("Redis error", "command", "SADD", "key", "pam:" + session.UserID(), "matchID", matchID, "error", err)
				return nil, err
			}

			m.entries[matchID] = matchEntry

			go func(){
				ctx, cancel := context.WithCancel(m.context)

				watchChan := Watcher(ctx, m.redis, matchID, m.logger, m.config)
				ticker := time.NewTicker(time.Second * time.Duration(30))
				latestPlayerCount := -1
				var ok bool

				for {
					select{
					case latestPlayerCount, ok = <- watchChan:
						if !ok {
							m.logger.Warn("Match search failed")
							err := errors.New("match search failed")
							m.broadcastMatch(session, nil, "", err, int32(socketapi.MatchError_MATCH_INTERNAL_ERROR))
							m.clearMatch(queueKey, matchID, matchEntry.Users)
							return
						}else{
							if latestPlayerCount == -1 {
								m.logger.Warn("Match watcher send -1, error happened inside watcher")

								err := errors.New("match search failed")
								m.broadcastMatch(session, nil, "", err, int32(socketapi.MatchError_MATCH_INTERNAL_ERROR))
								m.clearMatch(queueKey, matchID, matchEntry.Users)
								return
							}else if latestPlayerCount == playerCount {
								m.logger.Infow("Found enough players", "matchID", matchID)

								matchEntry.State = int32(socketapi.MatchEntry_MATCH_AWAITING_PLAYERS)
								m.entries[matchID] = matchEntry

								m.broadcastMatch(session, matchEntry, "", nil, 0)

								err = m.redis.Do(radix.Cmd(nil, "LREM", queueKey, "0", matchID))
								if err != nil {
									m.logger.Errorw("Redis error", "command", "LREM", "key", queueKey, "matchID", matchID, "error", err)
								}
								return
							}else{
								m.logger.Infow("Waiting players", "matchID", matchID)
								m.broadcastMatch(session, matchEntry, "", nil, 0)
							}
						}
					case <- ticker.C:
						m.logger.Info("Match search timeout")
						err := errors.New("match search timeout")
						m.broadcastMatch(session, nil, "", err, int32(socketapi.MatchError_MATCH_TIMEOUT))
						m.clearMatch(queueKey, matchID, matchEntry.Users)
						cancel()
						return
					}
				}
			}()

			return matchEntry, nil
		}else{
			var isMember int
			err = m.redis.Do(radix.Cmd(&isMember, "SISMEMBER", matchID, session.UserID()))
			if err != nil {
				m.logger.Errorw("Redis error", "command", "SISMEMBER", "matchID", matchID, "userID", session.UserID(), "error", err)
				return nil, err
			}
			matchEntry, _ := m.entries[matchID]

			if isMember != 1 && matchEntry.ActiveCount < int32(playerCount){
				matchEntry.ActiveCount++
				matchEntry.Users = append(matchEntry.Users, &socketapi.MatchEntry_MatchUser{
					UserId:session.UserID(),
					State: int32(socketapi.MatchEntry_MatchUser_NOT_READY),
				})

				err = m.redis.Do(radix.Cmd(nil, "SADD", matchID, session.UserID()))
				if err != nil {
					m.logger.Errorw("Redis error", "command", "SADD", "matchID", matchID, "userID", session.UserID(), "error", err)
					return nil, err
				}

				err = m.redis.Do(radix.Cmd(nil, "SADD", "pm:" + session.UserID(), matchID))
				if err != nil {
					m.logger.Errorw("Redis error", "command", "SADD", "key", "pm:" + session.UserID(), "matchID", matchID, "error", err)
					return nil, err
				}

				err = m.redis.Do(radix.Cmd(nil, "SADD", "pam:" + session.UserID(), matchID))
				if err != nil {
					m.logger.Errorw("Redis error", "command", "SADD", "key", "pam:" + session.UserID(), "matchID", matchID, "error", err)
					return nil, err
				}

				m.entries[matchID] = matchEntry
			}else if isMember != 1 && matchEntry.ActiveCount >= int32(playerCount) {
				m.logger.Info("Ignore user request return not found and try again")
				err = errors.New("not suitable for this match")
				return nil,err
			}

			return matchEntry, nil
		}
	}

	return nil, nil//bad pattern: return error
}

func (m *LocalMatchmaker) Join(pipeline *Pipeline, session Session, matchID string) (*socketapi.GameData,error) {
	rs := redsyncradix.New([]radix.Client{m.redis})
	mutex := rs.NewMutex("lock|" + matchID)
	if err := mutex.Lock(); err != nil {
		m.logger.Errorw("Error while using lock", "key", "lock|" + matchID, "error", err)
	} else {
		defer mutex.Unlock()
	}

	var err error
	var gameData *socketapi.GameData

	matchEntry, ok := m.entries[matchID]
	if !ok {
		return nil, errors.New("matchID not found!")
	}

	game := m.gameHolder.Get(matchEntry.GameName)
	if game == nil {
		return nil, errors.New("can't find game for this request")
	}

	if game.GetGameSpecs().Mode == GAME_TYPE_PASSIVE_TURN_BASED {
		switch matchEntry.State {
		case int32(socketapi.MatchEntry_MATCH_FINDING_PLAYERS):
			gameData, err = NewGame(matchID, matchEntry.GameName, m.gameHolder, pipeline, session, m.logger, m)
			if err != nil {
				return nil, err
			}

			matchEntry.Game = gameData.Id
			matchEntry.State = int32(socketapi.MatchEntry_GAME_CREATED)

			gameData, err = JoinGame(matchEntry.Game, m.gameHolder, session, m.logger)
			if err != nil {
				return nil, err
			}

			for _,user := range matchEntry.Users {
				if user.UserId == session.UserID() {
					user.State = int32(socketapi.MatchEntry_MatchUser_READY)
				}
			}

			m.entries[matchID] = matchEntry

			break
		case int32(socketapi.MatchEntry_GAME_CREATED):
			//TODO we have enough player to start match, matchEntry data need to be validated via game.Matchmaked?

			for _,user := range matchEntry.Users {
				if user.UserId == session.UserID() && user.State == int32(socketapi.MatchEntry_MatchUser_NOT_READY){
					user.State = int32(socketapi.MatchEntry_MatchUser_READY)
					m.entries[matchID] = matchEntry

					//TODO game.Matchmaked validate every user join
					m.broadcastMatch(session, matchEntry, session.UserID(), nil, 0)

					//We should trigger relevant game controllers methods
					gameData, err = JoinGame(matchEntry.Game, m.gameHolder, session, m.logger)
					if err != nil {
						return nil, err
					}

					break
				}
			}
			break
		}

		return gameData,nil
	}else if game.GetGameSpecs().Mode == GAME_TYPE_ACTIVE_TURN_BASED || game.GetGameSpecs().Mode == GAME_TYPE_REAL_TIME {
		if matchEntry.State == int32(socketapi.MatchEntry_MATCH_AWAITING_PLAYERS) {
			for _,user := range matchEntry.Users {
				if user.UserId == session.UserID() && user.State == int32(socketapi.MatchEntry_MatchUser_NOT_READY) {
					user.State = int32(socketapi.MatchEntry_MatchUser_READY)

					err = m.redis.Do(radix.Cmd(nil, "SADD", matchID+":joins", session.UserID()))
					if err != nil {
						m.logger.Errorw("Redis error", "command", "SADD", "key", matchID+":joins", "userID", session.UserID(), "error", err)
						return nil, err
					}

					gameData, err = NewGame(matchID, matchEntry.GameName, m.gameHolder, pipeline, session, m.logger, m)
					if err != nil {
						m.broadcastMatch(session, nil, "", err, int32(socketapi.MatchError_MATCH_INTERNAL_ERROR))
						m.clearMatch(matchEntry.Queuekey, matchID, matchEntry.Users)
					}
					matchEntry.Game = gameData.Id

					gameData, err = JoinGame(matchEntry.Game, m.gameHolder, session, m.logger)
					if err != nil {
						m.broadcastMatch(session, nil, "", err, int32(socketapi.MatchError_MATCH_INTERNAL_ERROR))
						m.clearMatch(matchEntry.Queuekey, matchID, matchEntry.Users)
					}

					matchEntry.State = int32(socketapi.MatchEntry_MATCH_JOINING_PLAYERS)
					m.entries[matchID] = matchEntry
					break
				}
			}

			go func(){
				ctx, cancel := context.WithCancel(m.context)

				watchChan := Watcher(ctx, m.redis, matchID+":joins", m.logger, m.config)
				ticker := time.NewTicker(time.Second * time.Duration(30))
				latestPlayerCount := -1
				var ok bool

				for{
					select {
					case latestPlayerCount,ok = <- watchChan:
						if !ok {
							m.logger.Warn("match join failed")
							err := errors.New("match join failed")
							m.broadcastMatch(session, nil, "", err, int32(socketapi.MatchError_MATCH_INTERNAL_ERROR))
							m.clearMatch(matchEntry.Queuekey, matchID, matchEntry.Users)
							return
						}else{
							if latestPlayerCount == -1 {
								m.logger.Warn("Match join watcher send -1, error happened inside watcher")
								err := errors.New("match join failed")
								m.broadcastMatch(session, nil, "", err, int32(socketapi.MatchError_MATCH_INTERNAL_ERROR))
								m.clearMatch(matchEntry.Queuekey, matchID, matchEntry.Users)
								return
							}else if int32(latestPlayerCount) == matchEntry.MaxCount {
								m.logger.Info("All players joined, state sync, publish game")

								matchEntry.State = int32(socketapi.MatchEntry_GAME_CREATED)
								m.entries[matchID] = matchEntry

								m.broadcastMatch(session,matchEntry, "", nil, 0)
								return
							}else {
								m.logger.Info("Waiting players to join")
								m.broadcastMatch(session, matchEntry, "", nil, 0)
							}
						}
					case <- ticker.C:
						m.logger.Warn("Match join timeout")
						err := errors.New("match join timeout")
						m.broadcastMatch(session, nil, "", err, int32(socketapi.MatchError_MATCH_TIMEOUT))
						m.clearMatch(matchEntry.Queuekey, matchID, matchEntry.Users)
						cancel()
						return
					}
				}
			}()

		}else if matchEntry.State == int32(socketapi.MatchEntry_MATCH_JOINING_PLAYERS) {
			for _,user := range matchEntry.Users {
				if user.UserId == session.UserID() && user.State == int32(socketapi.MatchEntry_MatchUser_NOT_READY) {
					user.State = int32(socketapi.MatchEntry_MatchUser_READY)

					err = m.redis.Do(radix.Cmd(nil, "SADD", matchID+":joins", session.UserID()))
					if err != nil {
						m.logger.Errorw("Redis error", "command", "SADD", "key", matchID + ":joins", "userID", session.UserID(), "error", err)
						return nil, err
					}

					gameData, err = JoinGame(matchEntry.Game, m.gameHolder, session, m.logger)
					if err != nil {
						m.broadcastMatch(session, nil, "", err, int32(socketapi.MatchError_MATCH_INTERNAL_ERROR))
						m.clearMatch(matchEntry.Queuekey, matchID, matchEntry.Users)
					}

					m.entries[matchID] = matchEntry
					break
				}
			}
		}

		return gameData,nil
	}

	return nil, nil//bad pattern: return error
}

func (m *LocalMatchmaker) Leave(session Session, matchID string) error{
	matchEntry, ok := m.entries[matchID]
	if !ok {
		return errors.New("can't find match for this request")
	}
	game := m.gameHolder.Get(matchEntry.GameName)
	if game == nil {
		return errors.New("can't find game for this request")
	}

	rs := redsyncradix.New([]radix.Client{m.redis})
	mutex := rs.NewMutex("lock|" + matchID)
	if err := mutex.Lock(); err != nil {
		m.logger.Errorw("Error while using lock", "key", "lock|"+matchID)
	} else {
		defer mutex.Unlock()
	}

	var redisKey string
	if game.GetGameSpecs().Mode == GAME_TYPE_PASSIVE_TURN_BASED {
		redisKey = "pm:"+session.UserID()
	}else{
		redisKey = "pam:"+session.UserID()
	}

	var isMember int
	err := m.redis.Do(radix.Cmd(&isMember, "SISMEMBER", redisKey))
	if err != nil {
		return err
	}

	if isMember == 1 {
		if len(matchEntry.Users) == 1 {

			err := m.redis.Do(radix.Cmd(nil, "SREM", redisKey, matchID))
			if err != nil {
				return err
			}
			err = m.redis.Do(radix.Cmd(nil, "REM", matchID))
			if err != nil {
				return err
			}

			m.clearMatch(matchEntry.Queuekey, matchEntry.MatchId, matchEntry.Users)

			if matchEntry.State == int32(socketapi.MatchEntry_MATCH_JOINING_PLAYERS) || matchEntry.State == int32(socketapi.MatchEntry_GAME_CREATED) {
				_, _ = LeaveGame(matchEntry.MatchId, m.gameHolder, session.UserID(), m.logger)
			}

		}else{
			err := m.redis.Do(radix.Cmd(nil, "SREM", redisKey, matchID))
			if err != nil {
				return err
			}
			err = m.redis.Do(radix.Cmd(nil, "SREM", matchID, session.UserID()))
			if err != nil {
				return err
			}

			users := make([]*socketapi.MatchEntry_MatchUser, 0, matchEntry.MaxCount)
			for _, user := range matchEntry.Users {
				if user.UserId != session.UserID() {
					users = append(users, user)
				}
			}
			matchEntry.Users = users
			m.entries[matchID] = matchEntry

			if matchEntry.State == int32(socketapi.MatchEntry_MATCH_JOINING_PLAYERS) || matchEntry.State == int32(socketapi.MatchEntry_GAME_CREATED) {
				gameData, _ := LeaveGame(matchEntry.MatchId, m.gameHolder, session.UserID(), m.logger)
				m.pipeline.broadcastGame(gameData)
			}

		}
	}else{
		return  errors.New("match can not found")
	}

	return nil
}

func (m *LocalMatchmaker) LeaveActiveGames(userID string) error {
	m.Lock()
	defer m.Unlock()

	redisKey := "pam:"+userID

	var pams []string
	err := m.redis.Do(radix.Cmd(&pams, "SMEMBERS", redisKey))
	if err != nil {
		m.logger.Errorw("Redis error", "command", "SMEMBERS", "key", redisKey, "error", err)
		return err
	}

	for _,matchID := range pams {
		match := m.entries[matchID]
		var userIndex int
		for i,user := range match.Users {
			if user.UserId == userID {
				userIndex = i
				break
			}
		}

		if len(match.Users) == 1 {
			err = m.redis.Do(radix.Cmd(nil, "SREM", redisKey, matchID))
			if err != nil {
				m.logger.Errorw("Redis error", "command", "SREM", "key", redisKey, "matchID", matchID, "error", err)
				return err
			}
			err = m.redis.Do(radix.Cmd(nil, "REM", matchID))
			if err != nil {
				m.logger.Errorw("Redis error", "command", "REM", "key", matchID, "error", err)
				return err
			}

			m.clearMatch(match.Queuekey, match.MatchId, match.Users)

			if match.State == int32(socketapi.MatchEntry_MATCH_JOINING_PLAYERS) || match.State == int32(socketapi.MatchEntry_GAME_CREATED) {
				_, _ = LeaveGame(match.MatchId, m.gameHolder, userID, m.logger)
			}
		}else{
			err = m.redis.Do(radix.Cmd(nil, "SREM", redisKey, matchID))
			if err != nil {
				m.logger.Errorw("Redis error", "command", "SREM", "key", redisKey, "matchID", matchID, "error", err)
				return err
			}
			err = m.redis.Do(radix.Cmd(nil, "SREM", matchID, userID))
			if err != nil {
				m.logger.Errorw("Redis error", "command", "SREM", "key", matchID, "userID", userID, "error", err)
				return err
			}

			match.Users = append(match.Users[:userIndex], match.Users[userIndex+1:]...)
			m.entries[matchID] = match

			if match.State == int32(socketapi.MatchEntry_MATCH_JOINING_PLAYERS) || match.State == int32(socketapi.MatchEntry_GAME_CREATED) {
				gameData, _ := LeaveGame(match.MatchId, m.gameHolder, userID, m.logger)
				m.pipeline.broadcastGame(gameData)
			}
		}
	}
	return nil
}

//Anti pattern
func (m *LocalMatchmaker) broadcastMatch(session Session, match *socketapi.MatchEntry, selfUserID string, err error, code int32) {
	if err != nil {
		_ = session.Send(false, 0, &socketapi.Envelope{Cid: "", Message: &socketapi.Envelope_MatchError{
			MatchError: &socketapi.MatchError{
				Code: code,
				Message: err.Error(),
			},
		}})
		return
	}

	if selfUserID != "" {
		index := 0
		for i, user := range match.Users {
			if selfUserID == user.UserId {
				index = i
			}
		}
		match.Users = append(match.Users[:index], match.Users[index+1:]...)
	}

	//Need to fetch all users session by their ids from gameData and send them msg
	message := &socketapi.Envelope{Cid: "", Message: &socketapi.Envelope_MatchEntry{MatchEntry: match}}
	for _, user := range match.Users {
		session := m.sessionHolder.GetByUserID(user.UserId)
		//TODO while finding match if host user closes session throws npe. For now fixed with nil check
		if session != nil {
			_ = session.Send(false, 0, message)
		}
	}
}

func (m *LocalMatchmaker) ClearMatch(matchID string) {

	match, ok := m.entries[matchID]
	if !ok {
		return
	}

	m.clearMatch(match.Queuekey, matchID, match.Users)

}

func (m *LocalMatchmaker) clearMatch(queueKey string, matchID string, users []*socketapi.MatchEntry_MatchUser){

	err := m.redis.Do(radix.WithConn("", func(conn radix.Conn) error {

		if err := conn.Do(radix.Cmd(nil, "MULTI")); err != nil {
			return err
		}

		var err error
		defer func(){
			if err != nil {
				_ = conn.Do(radix.Cmd(nil, "DISCARD"))
			}
		}()

		if err = conn.Do(radix.Cmd(nil, "LREM", queueKey, "0", matchID)); err != nil {
			m.logger.Errorw("Redis error", "command", "LREM", "key", queueKey, "matchID", matchID, "error", err)
			return err
		}

		if err = conn.Do(radix.Cmd(nil, "DEL", matchID)); err != nil {
			m.logger.Errorw("Redis error", "command", "DEL", "key", matchID, "error", err)
			return err
		}

		if err = conn.Do(radix.Cmd(nil, "DEL", matchID+":joins")); err != nil {
			m.logger.Errorw("Redis error", "command", "DEL", "key", matchID+":joins", "error", err)
			return err
		}

		for _,user := range users {

			if err = conn.Do(radix.Cmd(nil, "SREM", "pm:"+user.UserId, matchID)); err != nil {
				m.logger.Errorw("Redis error", "command", "SREM", "key", "pm:"+user.UserId, "matchID", matchID, "error", err)
				return err
			}

			if err = conn.Do(radix.Cmd(nil, "SREM", "pam:"+user.UserId, matchID)); err != nil {
				m.logger.Errorw("Redis error", "command", "SREM", "key", "pam:"+user.UserId, "matchID", matchID, "error", err)
				return err
			}

		}

		if err = conn.Do(radix.Cmd(nil, "EXEC")); err != nil {
			m.logger.Errorw("Redis error", "command", "EXEC", "error", err)
			return err
		}

		return nil
	}))

	if err != nil {
		m.logger.Errorw("Error while executing commands in WitchConn helper", "error", err)
		return
	}

	delete(m.entries, matchID)
}

func (m *LocalMatchmaker) generateQueueKey(gameName string, queueProperties map[string]string) string {
	k := fmt.Sprintf("gq:%s", gameName)
	for _,v := range queueProperties {
		k += ":" + v
	}
	return k
}
