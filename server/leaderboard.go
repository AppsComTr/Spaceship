package server

import (
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"spaceship/model"
	"strconv"
	"time"
)

type Leaderboard struct {
	db *mgo.Session
	gameHolder *GameHolder
	logger *Logger
}

func NewLeaderboard(db *mgo.Session, logger *Logger) *Leaderboard {
	return &Leaderboard{
		db: db,
		logger: logger,
	}
}

func (l *Leaderboard) SetGameHolder(gameHolder *GameHolder) {
	l.gameHolder = gameHolder
}

func (l *Leaderboard) Score(userID string, gameName string, score int64) error {

	year, week := time.Now().ISOWeek()

	dayID := time.Now().Format("02-01-2006")
	weekID := strconv.Itoa(week) + "-" + strconv.Itoa(year)
	monthID := time.Now().Format("01-2006")

	conn := l.db.Copy()
	defer conn.Close()
	db := conn.DB("spaceship")

	err := l.scoreDetail(db, userID, &gameName, "day", &dayID, score)
	if err != nil {
		return err
	}
	err = l.scoreDetail(db, userID, &gameName, "week", &weekID, score)
	if err != nil {
		return err
	}
	err = l.scoreDetail(db, userID, &gameName, "month", &monthID, score)
	if err != nil {
		return err
	}
	err = l.scoreDetail(db, userID, &gameName, "overall", nil, score)
	if err != nil {
		return err
	}

	err = l.scoreDetail(db, userID, nil, "day", &dayID, score)
	if err != nil {
		return err
	}
	err = l.scoreDetail(db, userID, nil, "week", &weekID, score)
	if err != nil {
		return err
	}
	err = l.scoreDetail(db, userID, nil, "month", &monthID, score)
	if err != nil {
		return err
	}
	err = l.scoreDetail(db, userID, nil, "overall", nil, score)
	if err != nil {
		return err
	}

	return nil

}

func (l Leaderboard) scoreDetail(db *mgo.Database, userID string, gameName *string, typeName string, typeID *string, score int64) error {

	//For week with mode name
	leaderboardM := &model.LeaderboardModel{}

	err := db.C(leaderboardM.GetCollectionName()).Find(bson.M{
		"userID": userID,
		"gameName": gameName,
		"type": typeName,
		"typeID": typeID,
	}).One(leaderboardM)

	if err != nil {
		if err == mgo.ErrNotFound {
			leaderboardM.Score = score
			leaderboardM.GameName = gameName
			leaderboardM.Type = typeName
			leaderboardM.TypeID = typeID
			leaderboardM.UserID = bson.ObjectIdHex(userID)

			err = db.C(leaderboardM.GetCollectionName()).Insert(leaderboardM)
			if err != nil {
				return err
			}
		}else{
			return err
		}
	}else{
		leaderboardM.Score += score
		err = db.C(leaderboardM.GetCollectionName()).Update(bson.M{
			"_id": leaderboardM.Id,
		}, leaderboardM)
		if err != nil {
			return err
		}
	}

	return nil

}

func (l Leaderboard) GetScores(typeName string, gameName string, page int, itemCount int) ([]model.LeaderboardModel, error) {

	year, week := time.Now().ISOWeek()

	dayID := time.Now().Format("02-01-2006")
	weekID := strconv.Itoa(week) + "-" + strconv.Itoa(year)
	monthID := time.Now().Format("01-2006")

	if typeName != "day" && typeName != "week" && typeName != "month" && typeName != "overall" {
		typeName = "overall"
	}

	var typeID *string

	if typeName == "day" {
		typeID = &dayID
	}else if typeName == "week" {
		typeID = &weekID
	}else if typeName == "month" {
		typeID = &monthID
	}

	scores := make([]model.LeaderboardModel, 0)

	game := l.gameHolder.Get(gameName)

	var gameNameQ *string
	if gameName != "all" && game != nil {
		gameNameQ = &gameName
	}

	conn := l.db.Copy()
	defer conn.Close()
	db := conn.DB("spaceship")

	err := db.C(model.LeaderboardModel{}.GetCollectionName()).Find(bson.M{
		"type": typeName,
		"gameName": gameNameQ,
		"typeID": typeID,
	}).Sort("-score").Skip(page*itemCount).Limit(itemCount).All(&scores)
	if err != nil {
		return nil, err
	}

	return scores, nil

}

func (l Leaderboard) GetUserRank(typeName string, gameName string, userID string) int {

	conn := l.db.Copy()
	defer conn.Close()
	db := conn.DB("spaceship")

	year, week := time.Now().ISOWeek()

	dayID := time.Now().Format("02-01-2006")
	weekID := strconv.Itoa(week) + "-" + strconv.Itoa(year)
	monthID := time.Now().Format("01-2006")

	if typeName != "day" && typeName != "week" && typeName != "month" && typeName != "overall" {
		typeName = "overall"
	}

	var typeID *string

	if typeName == "day" {
		typeID = &dayID
	}else if typeName == "week" {
		typeID = &weekID
	}else if typeName == "month" {
		typeID = &monthID
	}

	game := l.gameHolder.Get(gameName)

	var gameNameQ *string
	if gameName != "all" && game != nil {
		gameNameQ = &gameName
	}

	//First get user score with given parameters
	score := &model.LeaderboardModel{}
	err := db.C(score.GetCollectionName()).Find(bson.M{
		"type": typeName,
		"gameName": gameNameQ,
		"typeID": typeID,
		"userID": bson.ObjectIdHex(userID),
	}).One(score)
	if err != nil {
		l.logger.Errorw("Error while fetching user score", "userID", userID, "type", typeName, "gameName", gameNameQ, "typeID", typeID, "error", err)
		return 0
	}

	//Then find count of documents that has bigger score than user's
	count, err := db.C(score.GetCollectionName()).Find(bson.M{
		"type": typeName,
		"gameName": gameNameQ,
		"typeID": typeID,
		"score": bson.M{
			"$gt": score.Score,
		},
	}).Sort("-score").Count()
	if err != nil {
		l.logger.Errorw("Error while getting count of users", "userID", userID, "type", typeName, "gameName", gameNameQ, "typeID", typeID, "error", err)
		return 0
	}

	//Then we need to find all scores that has same score with users and find index of given user
	scores := make([]model.LeaderboardModel, 0)
	err = db.C(score.GetCollectionName()).Find(bson.M{
		"type": typeName,
		"gameName": gameNameQ,
		"typeID": typeID,
		"score": score.Score,
	}).Sort("-score").All(&scores)
	if err != nil {
		l.logger.Errorw("Error while getting users that has same score with requested user", "userID", userID, "type", typeName, "gameName", gameNameQ, "typeID", typeID, "error", err)
		return 0
	}

	for _, iScore := range scores {
		count++
		if iScore.UserID.Hex() == userID {
			break
		}
	}

	return count

}