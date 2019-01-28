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

	entries map[string]*socketapi.MatchEntry
	userActiveMatches map[uuid.UUID]map[string]string //keeps users online matches by sessionID
	userMatches map[string]map[string]string //keeps users active matches by userID
}

func NewLocalMatchMaker(redis radix.Client, gameHolder *GameHolder) Matchmaker {
	return &LocalMatchmaker{
		redis: redis,
		gameHolder: gameHolder,
		entries: make(map[string]*socketapi.MatchEntry),
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
			})

			matchEntry := &socketapi.MatchEntry{
				MatchId: matchID,
				MaxCount: int32(playerCount),
				ActiveCount: 1,
				Users: users,
				GameName: gameName,
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
				UserId: session.UserID(),//TODO: also hold username etc.
			})

			matchEntry := &socketapi.MatchEntry{
				MatchId: matchID,
				MaxCount: int32(playerCount),
				ActiveCount: 1,
				Users: users,
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

			err = m.redis.Do(radix.Cmd(nil, "SADD", "pam:" + session.ID().String(), matchID))
			if err != nil {
				log.Println("Redis SADD pam matchEntryID ", err)
				return nil, err
			}

			m.entries[matchID] = matchEntry

			go func(){
				// Get a cancel-able context
				ctx, cancel := context.WithCancel(context.Background())//TODO improve this, antipattern open-match/apiserv.go#CreateMatch
				defer cancel()

				watchChan := Watcher(ctx,m.redis,matchID)
				timeout := time.Duration(30 * time.Second)
				latestPlayerCount := -1
				var ok bool

				for {
					select{
					case <- time.After(timeout):
						log.Print("Match search timeout")
						// search for this match timeout.
						// inform players if any and clear queue etc.
						return
					case latestPlayerCount, ok = <- watchChan:
						if !ok {
							log.Println("SHITT !OK")
							// inform players if any and clear queue etc.
							return
						}else{
							if latestPlayerCount == -1 {
								log.Print("Match watcher send -1")
								return
								//Error or host user left at critical time
								// inform players if any and clear queue etc.
							}else if latestPlayerCount == playerCount {
								log.Println("found enough players for this match: ", matchID)
								//Even if looks like we have enough player to start match, matchEntry data need to be validated via game.Matchmaked?
								//inform players to start game
								return
							}else{
								log.Println("waiting players for this match: ", matchID)
								//inform players about changes with matchentry data
							}
						}
					}
				}

			}()

			return matchEntry, nil
			// user have to wait for match_ready_to_join or match_find_timeout
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

				err = m.redis.Do(radix.Cmd(nil, "SADD", "pam:" + session.ID().String(), matchID))
				if err != nil {
					log.Println("Redis SADD pam matchEntryID ", err)
					return nil, err
				}

				m.entries[matchID] = matchEntry

			}else if isMember != 1 && matchEntry.ActiveCount == int32(playerCount) {
				log.Println("Ignore user request return not found and try again")
				//TODO ?need to check matchEntry.ActiveCount > int32(playerCount)
				err = errors.New("Not suitable for this match")
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
}

func (m *LocalMatchmaker) Join(session Session, matchEntryID string) (*socketapi.GameData,error){
	m.RLock() //TODO lock must be applied on matchID Key!
	defer m.RUnlock()

	game := "Passive"


	matchEntry, ok := m.entries[matchEntryID]
	if !ok {
		return nil, errors.New("MatchID not found!")
	}

	if game == "Passive" {
		//TODO Call game.MatchmakerCheck() if valid ->
		switch matchEntry.State {
		case int32(socketapi.MatchEntry_MATCH_FINDING_PLAYERS):
			//game.create
			gameObject, err := NewGame(matchID, matchEntry.GameName, m.gameHolder, pipeline, session)
			if err != nil {
				return nil, err
			}

			matchEntry.Game = gameObject.Id
			matchEntry.State = int32(socketapi.MatchEntry_GAME_CREATED)
			m.entries[matchEntryID] = matchEntry

			break
		case int32(socketapi.MatchEntry_GAME_CREATED):
			//TODO need to store users join status for more than 2 user gameplay

			delete(m.entries, matchID)// only clear when last player joined

			break
		}
	}else if game == "Active" {
		/*TODO Call game.MatchmakerCheck()
			if valid
				Notify users everyone is ready and game started
			if not valid
				Notify users with latest match state
		 */

	}else if game == "realtime" {

	}

	//We should trigger relevant game controllers methods
	gameData, err := JoinGame(matchEntry.Game, m.gameHolder, session)
	if err != nil {
		return nil, err
	}

	return gameData,nil
}

func (m *LocalMatchmaker) Leave(session Session, matchID string) error{

	rs := redsyncradix.New([]radix.Client{m.redis})
	mutex := rs.NewMutex("lock|" + matchID)
	if err := mutex.Lock(); err != nil {
		log.Println(err.Error())
	} else {
		defer mutex.Unlock()
	}

	return nil
}



func (m *LocalMatchmaker) generateQueueKey(gameName string, queueProperties map[string]string) string {
	k := fmt.Sprintf("gq:%s", gameName)
	for _,v := range queueProperties {
		k += ":" + v
	}
	return k
}
