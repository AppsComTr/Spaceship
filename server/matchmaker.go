package server

import (
	"errors"
	"fmt"
	"github.com/kayalardanmehmet/redsync-radix"
	"github.com/mediocregopher/radix/v3"
	"github.com/satori/go.uuid"
	"log"
	"spaceship/socketapi"
	"strconv"
	"sync"
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
		var matchEntryID string
		err = m.redis.Do(radix.Cmd(&matchEntryID, "LINDEX", queueKey, "0"))
		if err != nil {
			log.Println("Redis LRANGE: ", err)
			return nil, err
		}

		if matchEntryID == "" {//Create match
			matchEntryID := uuid.NewV4().String()

			users := make([]*socketapi.MatchEntry_MatchUser, 0, playerCount)
			users = append(users, &socketapi.MatchEntry_MatchUser{
				UserId:session.UserID(),
			})

			matchEntry := &socketapi.MatchEntry{
				MatchId: matchEntryID,
				MaxCount: int32(playerCount),
				ActiveCount: 1,
				Users: users,
				GameName: gameName,
			}

			err = m.redis.Do(radix.Cmd(nil, "LPUSH", queueKey, matchEntryID))
			if err != nil {
				log.Println("Redis LPUSH: ", err)
				return nil, err
			}


			err = m.redis.Do(radix.Cmd(nil, "SADD", matchEntryID, session.UserID()))
			if err != nil {
				log.Println("Redis SADD matchEntryID ", err)
				return nil, err
			}

			err = m.redis.Do(radix.Cmd(nil, "SADD", session.UserID() + ":matchentries", matchEntryID))
			if err != nil {
				log.Println("Redis SADD userid:matchentries ", err)
				return nil, err
			}

			m.entries[matchEntryID] = matchEntry

			return matchEntry, nil
		}else {//Join existing match
			var isMember int
			err = m.redis.Do(radix.Cmd(&isMember, "SISMEMBER", matchEntryID, session.UserID()))
			if err != nil {
				log.Println("Redis SISMEMBER: ", err)
				return nil, err
			}
			matchEntry, _ := m.entries[matchEntryID]

			if isMember != 1 && matchEntry.ActiveCount < int32(playerCount){
				matchEntry.ActiveCount++
				matchEntry.Users = append(matchEntry.Users, &socketapi.MatchEntry_MatchUser{
					UserId:session.UserID(),
				})
				m.entries[matchEntryID] = matchEntry

				if (matchEntry.ActiveCount) == int32(playerCount) {
					err = m.redis.Do(radix.Cmd(nil, "LREM", queueKey, "0", matchEntryID))
					if err != nil {
						log.Println("Redis LPOP: ", err)
						return nil, err
					}
					err = m.redis.Do(radix.Cmd(nil, "DEL", matchEntryID))
					if err != nil {
						log.Println("Redis DEL: ", err)
						return nil, err
					}
				}
			}

			return matchEntry, nil
		}
	}else {
		timeoutParameter := 30

		return nil, nil
	}
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
