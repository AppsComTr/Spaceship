package server

import (
	"errors"
	"fmt"
	"github.com/mediocregopher/radix/v3"
	"github.com/satori/go.uuid"
	"log"
	"spaceship/socketapi"
	"strconv"
	"sync"
)

type Matchmaker interface {
	Find(session Session,gameName string, queueProperties map[string]string) (*socketapi.MatchEntry, error)
	Join(session Session, matchID string) (string, error)
	Leave(session Session, matchID string) error
}

type LocalMatchmaker struct {
	sync.RWMutex
	redis *radix.Pool
	gameHolder *GameHolder

	entries map[string]*socketapi.MatchEntry
}

func NewLocalMatchMaker(redis *radix.Pool, gameHolder *GameHolder) Matchmaker {
	return &LocalMatchmaker{
		redis: redis,
		gameHolder: gameHolder,
		entries: make(map[string]*socketapi.MatchEntry),
	}
}

func (m *LocalMatchmaker) Find(session Session, gameName string, queueProperties map[string]string) (*socketapi.MatchEntry, error){
	m.RLock() //TODO lock must be applied on Game Queue Key!
	defer m.RUnlock()

	queueKey := m.generateQueueKey(gameName, queueProperties)
	playerCount, err := strconv.Atoi(queueProperties["player_count"])
	if err != nil {
		log.Println("player count: ", err)
		return nil, err
	}

	var matchID string
	err = m.redis.Do(radix.Cmd(&matchID, "LINDEX", queueKey, "0"))
	if err != nil {
		log.Println("Redis LRANGE: ", err)
		return nil, err
	}

	if matchID == "" {//Create match
		matchID := uuid.NewV4().String()

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
			log.Println("Redis SADD: ", err)
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

			if (matchEntry.ActiveCount) == int32(playerCount) {
				err = m.redis.Do(radix.Cmd(nil, "LREM", queueKey, "0", matchID))
				if err != nil {
					log.Println("Redis LPOP: ", err)
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
}

func (m *LocalMatchmaker) Join(session Session, matchID string) (string,error){
	m.RLock() //TODO lock must be applied on matchID Key!
	defer m.RUnlock()

	game := "Passive"
	gameID := ""

	matchEntry, ok := m.entries[matchID]
	if !ok {
		return gameID, errors.New("MatchID not found!")
	}

	if game == "Passive" {
		//TODO Call game.MatchmakerCheck() if valid ->
		switch matchEntry.State {
		case int32(socketapi.MatchEntry_MATCH_FINDING_PLAYERS):
			//game.create
			gameObject, err := NewGame(matchEntry.GameName, m.gameHolder, session)
			if err != nil {
				return "", err
			}
			gameID = gameObject.Id

			matchEntry.Game = gameObject.Id
			matchEntry.State = int32(socketapi.MatchEntry_GAME_CREATED)
			m.entries[matchID] = matchEntry

			break
		case int32(socketapi.MatchEntry_GAME_CREATED):
			//TODO need to store users join status for more than 2 user gameplay
			gameID = matchEntry.Game
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
	_, err := JoinGame(matchEntry.Game, m.gameHolder, session)
	if err != nil {
		return "", err
	}

	return gameID,nil
}

func (m *LocalMatchmaker) Leave(session Session, matchID string) error{
	return nil
}

func (m *LocalMatchmaker) generateQueueKey(gameName string, queueProperties map[string]string) string {
	k := fmt.Sprintf("gq:%s", gameName)
	for _,v := range queueProperties {
		k += ":" + v
	}
	return k
}
