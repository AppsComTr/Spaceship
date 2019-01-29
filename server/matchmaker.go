package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/kayalardanmehmet/redsync-radix"
	"github.com/mediocregopher/radix/v3"
	"github.com/satori/go.uuid"
	"log"
	"spaceship/socketapi"
	"strconv"
	"sync"
	"time"
)

//Adds and removes users from matchmaking queue
type Matchmaker interface {
	Find(session Session,gameName string, queueProperties map[string]string) (*socketapi.MatchEntry, error)
	Join(pipeline *Pipeline, session Session, matchID string) (*socketapi.GameData, error)
	//Join(session Session, matchID string) (*socketapi.GameData, error)
	Leave(session Session, matchID string) error
}

type LocalMatchmaker struct {
	sync.RWMutex
	redis radix.Client
	gameHolder *GameHolder
	sessionHolder *SessionHolder

	entries map[string]*socketapi.MatchEntry
}

func NewLocalMatchMaker(redis *radix.Pool, gameHolder *GameHolder, sessionHolder *SessionHolder) Matchmaker {
	return &LocalMatchmaker{
		redis: redis,
		gameHolder: gameHolder,
		entries: make(map[string]*socketapi.MatchEntry),
		sessionHolder: sessionHolder,
	}
}

func (m *LocalMatchmaker) Find(session Session, gameName string, queueProperties map[string]string) (*socketapi.MatchEntry, error){
	//TODO: we can validate if game controller exists with given gameName to eliminate unnecessarry lock operations
	queueKey := m.generateQueueKey(gameName, queueProperties)

	rs := redsyncradix.New([]radix.Client{m.redis})
	mutex := rs.NewMutex("lock|gamequeue|" + queueKey)
	if err := mutex.Lock(); err != nil {
		log.Println(err.Error())
	} else {
		defer mutex.Unlock()
	}

	playerCount, err := strconv.Atoi(queueProperties["player_count"])
	if err != nil {
		log.Println("player count: ", err)
		return nil, err
	}

	gameMode := "active"

	if gameMode == "passive" {
		var matchID string
		err = m.redis.Do(radix.Cmd(&matchID, "LINDEX", queueKey, "0"))
		if err != nil {
			log.Println("Redis LRANGE: ", err)
			return nil, err
		}

		if matchID == "" {//Create match
			matchID = uuid.NewV4().String()

			users := make([]*socketapi.MatchEntry_MatchUser, 0, playerCount)
			users = append(users, &socketapi.MatchEntry_MatchUser{
				UserId:session.UserID(),
				Username: session.Username(),
			})

			matchEntry := &socketapi.MatchEntry{
				MatchId: matchID,
				MaxCount: int32(playerCount),
				ActiveCount: 1,
				Users: users,
				GameName: gameName,
				Queuekey: queueKey,
			}

			err = m.redis.Do(radix.Cmd(nil, "LPUSH", queueKey, matchID))
			if err != nil {
				log.Println("Redis LPUSH: ", err)
				return nil, err
			}

			err = m.redis.Do(radix.Cmd(nil, "SADD", matchID, session.UserID()))
			if err != nil {
				log.Println("Redis SADD matchEntryID ", err)
				return nil, err
			}

			err = m.redis.Do(radix.Cmd(nil, "SADD", "pm:" + session.UserID(), matchID))
			if err != nil {
				log.Println("Redis SADD matchEntryID ", err)
				return nil, err
			}

			m.entries[matchID] = matchEntry

			return matchEntry, nil
		}else {//Join existing match
			var isMember int
			err = m.redis.Do(radix.Cmd(&isMember, "SISMEMBER", matchID, session.UserID()))
			if err != nil {
				log.Println("Redis SISMEMBER: ", err)
				return nil, err
			}
			matchEntry, _ := m.entries[matchID]

			if isMember != 1 && matchEntry.ActiveCount < int32(playerCount){
				matchEntry.ActiveCount++
				matchEntry.Users = append(matchEntry.Users, &socketapi.MatchEntry_MatchUser{
					UserId:session.UserID(),
				})
				m.entries[matchID] = matchEntry

				err = m.redis.Do(radix.Cmd(nil, "SADD", "pm:" + session.UserID(), matchID))
				if err != nil {
					log.Println("Redis SADD matchEntryID ", err)
					return nil, err
				}

				if (matchEntry.ActiveCount) == int32(playerCount) {
					err = m.redis.Do(radix.Cmd(nil, "LREM", queueKey, "0", matchID))
					if err != nil {
						log.Println("Redis LREM: ", err)
						return nil, err
					}
					err = m.redis.Do(radix.Cmd(nil, "DEL", matchID))
					if err != nil {
						log.Println("Redis DEL: ", err)
						return nil, err
					}
				}
			}

			return matchEntry, nil
		}
	}else if gameMode == "active" {
		var matchID string
		err := m.redis.Do(radix.Cmd(&matchID, "LINDEX", queueKey, "0"))
		if err != nil {
			log.Println("Redis LINDEX: ", err)
			return nil, err
		}

		if matchID == "" {
			matchID = uuid.NewV4().String()

			users := make([]*socketapi.MatchEntry_MatchUser, 0, playerCount)
			users = append(users, &socketapi.MatchEntry_MatchUser{
				UserId: session.UserID(),
				Username: session.Username(),
			})

			matchEntry := &socketapi.MatchEntry{
				MatchId: matchID,
				MaxCount: int32(playerCount),
				ActiveCount: 1,
				Users: users,
				GameName: gameName,
				Queuekey: queueKey,
			}

			err = m.redis.Do(radix.Cmd(nil, "LPUSH", queueKey, matchID))
			if err != nil {
				log.Println("Redis LPUSH: ", err)
				return nil, err
			}

			err = m.redis.Do(radix.Cmd(nil, "SADD", matchID, session.UserID()))
			if err != nil {
				log.Println("Redis SADD: ", err)
				return nil, err
			}

			err = m.redis.Do(radix.Cmd(nil, "SADD", "pm:" + session.UserID(), matchID))
			if err != nil {
				log.Println("Redis SADD pm matchEntryID ", err)
				return nil, err
			}

			err = m.redis.Do(radix.Cmd(nil, "SADD", "pam:" + session.UserID(), matchID))
			if err != nil {
				log.Println("Redis SADD pam matchEntryID ", err)
				return nil, err
			}

			m.entries[matchID] = matchEntry

			go func(){
				// This routine must be controlled with cancel-able context which comes from main
				log.Println("find routine-start")
				ctx, cancel := context.WithCancel(context.Background())//TODO improve this, antipattern open-match/apiserv.go#CreateMatch
				defer cancel()

				watchChan := Watcher(ctx,m.redis,matchID)
				timeout := time.Duration(30 * time.Second)
				latestPlayerCount := -1
				var ok bool

				for {
					select{
					case <- ctx.Done():
						log.Println("find routine caught, closing")
						return
					case <- time.After(timeout):
						log.Println("match search timeout")
						err := errors.New("match search timeout")
						m.broadcastMatch(session, nil, "", err, int32(socketapi.MatchError_MATCH_TIMEOUT))
						m.clearMatch(queueKey,matchID, matchEntry.Users)
						return
					case latestPlayerCount, ok = <- watchChan:
						if !ok {
							log.Println("match search failed")
							err := errors.New("match search failed")
							m.broadcastMatch(session, nil, "", err, int32(socketapi.MatchError_MATCH_INTERNAL_ERROR))
							m.clearMatch(queueKey, matchID, matchEntry.Users)
							return
						}else{
							if latestPlayerCount == -1 {
								log.Println("Match watcher send -1, error happened inside watcher")
								err := errors.New("match search failed")
								m.broadcastMatch(session, nil, "", err, int32(socketapi.MatchError_MATCH_INTERNAL_ERROR))
								m.clearMatch(queueKey, matchID, matchEntry.Users)
								return
							}else if latestPlayerCount == playerCount {
								log.Println("found enough players for this match: ", matchID)

								matchEntry.State = int32(socketapi.MatchEntry_MATCH_AWAITING_PLAYERS)
								m.entries[matchID] = matchEntry

								m.broadcastMatch(session, matchEntry, "", nil, 0)

								err = m.redis.Do(radix.Cmd(nil, "LREM", queueKey, "0", matchID))
								if err != nil {
									log.Println("Redis LREM queueKey: ", err)
								}

								return
							}else{
								log.Println("waiting players for this match: ", matchID)
								m.broadcastMatch(session, matchEntry, "", nil, 0)
							}
						}
					}
				}
			}()

			return matchEntry, nil
		}else{
			var isMember int
			err = m.redis.Do(radix.Cmd(&isMember, "SISMEMBER", matchID, session.UserID()))
			if err != nil {
				log.Println("Redis SISMEMBER: ", err)
				return nil, err
			}
			matchEntry, _ := m.entries[matchID]

			if isMember != 1 && matchEntry.ActiveCount < int32(playerCount){
				matchEntry.ActiveCount++
				matchEntry.Users = append(matchEntry.Users, &socketapi.MatchEntry_MatchUser{
					UserId:session.UserID(),
				})

				err = m.redis.Do(radix.Cmd(nil, "SADD", matchID, session.UserID()))
				if err != nil {
					log.Println("Redis SADD: ", err)
					return nil, err
				}

				err = m.redis.Do(radix.Cmd(nil, "SADD", "pm:" + session.UserID(), matchID))
				if err != nil {
					log.Println("Redis SADD pm matchEntryID ", err)
					return nil, err
				}

				err = m.redis.Do(radix.Cmd(nil, "SADD", "pam:" + session.UserID(), matchID))
				if err != nil {
					log.Println("Redis SADD pam matchEntryID ", err)
					return nil, err
				}

				m.entries[matchID] = matchEntry
			}else if isMember != 1 && matchEntry.ActiveCount == int32(playerCount) {
				log.Println("Ignore user request return not found and try again")
				//TODO ?need to check matchEntry.ActiveCount > int32(playerCount)
				err = errors.New("not suitable for this match")
				return nil,err
			}

			return matchEntry, nil
		}
	}

	log.Println("if it works, problem!")
	return nil, nil//bad pattern: return error
}

func (m *LocalMatchmaker) Join(pipeline *Pipeline, session Session, matchID string) (*socketapi.GameData,error) {
	rs := redsyncradix.New([]radix.Client{m.redis})
	mutex := rs.NewMutex("lock|" + matchID)
	if err := mutex.Lock(); err != nil {
		log.Println(err.Error())
	} else {
		defer mutex.Unlock()
	}

	m.RLock() //TODO lock must be applied on matchID Key!
	defer m.RUnlock()

	var gameData *socketapi.GameData
	var err error
	game := "active"

	matchEntry, ok := m.entries[matchID]
	if !ok {
		return nil, errors.New("MatchID not found!")
	}

	if game == "passive" {
		switch matchEntry.State {
		case int32(socketapi.MatchEntry_MATCH_FINDING_PLAYERS):

			gameObject, err := NewGame(matchID, matchEntry.GameName, m.gameHolder, pipeline, session)

			if err != nil {
				return nil, err
			}

			matchEntry.Game = gameObject.Id
			matchEntry.State = int32(socketapi.MatchEntry_GAME_CREATED)

			for _,user := range matchEntry.Users {
				if user.UserId == session.UserID() {
					user.State = int32(socketapi.MatchEntry_MatchUser_JOINED)
				}
			}

			m.entries[matchID] = matchEntry

			break
		case int32(socketapi.MatchEntry_GAME_CREATED):
			//TODO we have enough player to start match, matchEntry data need to be validated via game.Matchmaked?

			for _,user := range matchEntry.Users {
				if user.UserId == session.UserID() && user.State != int32(socketapi.MatchEntry_MatchUser_JOINED){
					user.State = int32(socketapi.MatchEntry_MatchUser_JOINED)
					m.entries[matchID] = matchEntry

					//TODO game.Matchmaked validate every user join
					m.broadcastMatch(session, matchEntry, session.UserID(), nil, 0)

					//We should trigger relevant game controllers methods
					gameData, err = JoinGame(matchEntry.Game, m.gameHolder, session)
					if err != nil {
						return nil, err
					}

					break
				}
			}


			delete(m.entries, matchID)// only clear when last player joined


			m.entries[matchEntryID] = matchEntry
			//TODO Opponent join event
			//TODO inform players
			//game.Matchmaked validate every user join

			break
		}
		return gameData,nil
	}else if game == "active" {
		if matchEntry.State == int32(socketapi.MatchEntry_MATCH_AWAITING_PLAYERS) {
			for _,user := range matchEntry.Users {
				if user.UserId == session.UserID() && user.State != int32(socketapi.MatchEntry_MatchUser_JOINED) {
					user.State = int32(socketapi.MatchEntry_MatchUser_JOINED)
				}
			}
			matchEntry.State = int32(socketapi.MatchEntry_MATCH_JOINING_PLAYERS)
			m.entries[matchID] = matchEntry


			gameObject, err := NewGame(matchEntry.GameName, m.gameHolder, session)
			if err != nil {
				m.broadcastMatch(session, nil, "", err, int32(socketapi.MatchError_MATCH_INTERNAL_ERROR))
				m.clearMatch(matchEntry.Queuekey, matchID, matchEntry.Users)
			}

			matchEntry.Game = gameObject.Id

			go func(){
				ctx, cancel := context.WithCancel(context.Background())//TODO improve this, antipattern open-match/apiserv.go#CreateMatch
				defer cancel()

				watchChan := Watcher(ctx,m.redis,matchID+":joins")
				timeout := time.Duration(10 * time.Second)
				latestPlayerCount := -1
				var ok bool

				for{
					select {
					case <- time.After(timeout):

						return
					case latestPlayerCount,ok = <- watchChan:
						if !ok {
							log.Println("shit")
						}else{
							if latestPlayerCount == -1 {
								log.Println("Match join watcher send -1, error happened inside watcher")
								err := errors.New("match join failed")
								m.broadcastMatch(session, nil, "", err, int32(socketapi.MatchError_MATCH_INTERNAL_ERROR))
								m.clearMatch(matchEntry.Queuekey, matchID, matchEntry.Users)
								return
							}else if int32(latestPlayerCount) == matchEntry.MaxCount {

							}
						}
						return
					}

				}
			}()
		}


		//Players received game data and moved to game screen
		//TODO update player states
		//TODO inform players
		return nil,nil//Maybe return gameData with GAME_WAITING_CREATE state?
	}

	log.Println("if it works, problem!")
	return nil, nil//bad pattern: return error
}

func (m *LocalMatchmaker) Leave(session Session, matchID string) error{


	rs := redsyncradix.New([]radix.Client{m.redis})
	mutex := rs.NewMutex("lock|" + matchID)
	if err := mutex.Lock(); err != nil {
		log.Println(err.Error())
	} else {
		defer mutex.Unlock()
	}


	m.Lock() //TODO lock must be applied on matchID Key!
	defer m.Unlock()

	var isMember int
	err := m.redis.Do(radix.Cmd(&isMember, "SISMEMBER", "pm:"+session.UserID()))
	if err != nil {
		return err
	}
	if isMember == 1 {
		matchEntry := m.entries[matchID]

		if len(matchEntry.Users) == 1 {
			err := m.redis.Do(radix.Cmd(nil, "SREM", "pm:"+session.UserID(), matchID))
			if err != nil {
				return err
			}
			err = m.redis.Do(radix.Cmd(nil, "SREM", "pam:"+session.UserID(), matchID))
			if err != nil {
				return err
			}
			err = m.redis.Do(radix.Cmd(nil, "REM", matchID))
			if err != nil {
				return err
			}

			delete(m.entries, matchID)
			//TODO notify game.leave
		}else{
			err := m.redis.Do(radix.Cmd(nil, "SREM", "pm:"+session.UserID(), matchID))
			if err != nil {
				return err
			}
			err = m.redis.Do(radix.Cmd(nil, "SREM", "pam:"+session.UserID(), matchID))
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

			//TODO notify !!opponent players if any
			//TODO notify game.leave
		}
	}else{
		return  errors.New("Match can not found")
	}

	return nil
}

func (m *LocalMatchmaker) LeaveAll(session session) error {
	m.Lock()
	defer m.Unlock()
	//Bu metodu network switch problemi cozuldugunde doldurmak daha mantikli

	//get and remove user from pam:UserID members
	//if last left player deletes matchentry
	//delete user from matchentry and update
	//inform opponents players
	//notify game.leave
	return nil
}

//Anti pattern
func (m *LocalMatchmaker) broadcastMatch(session Session, match *socketapi.MatchEntry, selfUserID string, err error, code int32) {
	if err != nil {
		log.Println(err)

		_ = session.Send(false, 0, &socketapi.Envelope{Cid: "", Message: &socketapi.Envelope_MatchError{
			MatchError: &socketapi.MatchError{
				Code: code,
				Message: err.Error(),
			},
		}})
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
		_ = session.Send(false, 0, message)
	}
}


func (m *LocalMatchmaker) generateQueueKey(gameName string, queueProperties map[string]string) string {
	k := fmt.Sprintf("gq:%s", gameName)
	for _,v := range queueProperties {
		k += ":" + v
	}
	return k
}

//TODO Muting errors inside method maybe bad idea
func (m *LocalMatchmaker) clearMatch(queueKey string, matchID string, users []*socketapi.MatchEntry_MatchUser){
	err := m.redis.Do(radix.Cmd(nil, "LREM", queueKey, "0", matchID))
	if err != nil {
		log.Println("Redis LREM: ", err)
	}

	err = m.redis.Do(radix.Cmd(nil, "DEL", matchID))
	if err != nil {
		log.Println("Redis DEL: ", err)
	}

	//TODO redis multi
	for _,user := range users {
		err = m.redis.Do(radix.Cmd(nil, "SREM", "pm:"+user.UserId))
		if err != nil {
			log.Println("Redis DEL: ", err)
		}
		err = m.redis.Do(radix.Cmd(nil, "SREM", "pam:"+user.UserId))
		if err != nil {
			log.Println("Redis DEL: ", err)
		}
	}

	delete(m.entries, matchID)
}
