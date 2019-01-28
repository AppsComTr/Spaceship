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
				Username: session.Username(),
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
				UserId: session.UserID(),
				Username: session.Username(),
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

			err = m.redis.Do(radix.Cmd(nil, "SADD", "pam:" + session.UserID(), matchID))
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
					//TODO needs a ctx.cancel from caller method
					case <- time.After(timeout):
						log.Print("Match search timeout")
						//TODO inform players if any and clear queue etc.
						return
					case latestPlayerCount, ok = <- watchChan:
						if !ok {
							log.Println("SHITT !OK")
							//TODO inform players if any and clear queue etc.
							return
						}else{
							if latestPlayerCount == -1 {
								log.Print("Match watcher send -1")
								return
								//TODO Error or host user left at critical time
								//TODO inform players if any and clear queue etc.
							}else if latestPlayerCount == playerCount {
								matchEntry.State = int32(socketapi.MatchEntry_GAME_CREATED)
								m.entries[matchID] = matchEntry
								log.Println("found enough players for this match: ", matchID)
								//TODO inform players to join game or do join here for each user
								return
							}else{
								log.Println("waiting players for this match: ", matchID)
								//TODO inform players about changes with matchentry data
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

	game := "active"

	matchEntry, ok := m.entries[matchEntryID]
	if !ok {
		return nil, errors.New("MatchID not found!")
	}

	if game == "passive" {
		//TODO Call game.MatchmakerCheck() if valid ->
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

			m.entries[matchEntryID] = matchEntry

			break
		case int32(socketapi.MatchEntry_GAME_CREATED):
			//TODO we have enough player to start match, matchEntry data need to be validated via game.Matchmaked?
			for _,user := range matchEntry.Users {
				if user.UserId == session.UserID() {
					user.State = int32(socketapi.MatchEntry_MatchUser_JOINED)
				}
			}


			delete(m.entries, matchID)// only clear when last player joined


			m.entries[matchEntryID] = matchEntry
			//TODO Opponent join event
			//TODO inform players
			//game.Matchmaked validate every user join

			break
		}
	}else if game == "active" {
		//Players received game data and moved to game screen
		//TODO update player states
		//TODO inform players
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


func (m *LocalMatchmaker) generateQueueKey(gameName string, queueProperties map[string]string) string {
	k := fmt.Sprintf("gq:%s", gameName)
	for _,v := range queueProperties {
		k += ":" + v
	}
	return k
}
