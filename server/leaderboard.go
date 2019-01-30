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
}

func NewLeaderboard(db *mgo.Session) *Leaderboard {
	return &Leaderboard{
		db: db,
	}
}

func (l *Leaderboard) Score(userID string, mode string, score int64) error {

	year, week := time.Now().ISOWeek()

	dayID := time.Now().Format("02-01-2006")
	weekID := strconv.Itoa(week) + "-" + strconv.Itoa(year)
	monthID := time.Now().Format("01-2006")

	conn := l.db.Copy()
	defer conn.Close()
	db := conn.DB("spaceship")

	err := l.scoreDetail(db, userID, &mode, "day", &dayID, score)
	if err != nil {
		return err
	}
	err = l.scoreDetail(db, userID, &mode, "week", &weekID, score)
	if err != nil {
		return err
	}
	err = l.scoreDetail(db, userID, &mode, "month", &monthID, score)
	if err != nil {
		return err
	}
	err = l.scoreDetail(db, userID, &mode, "overall", nil, score)
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

func (l Leaderboard) scoreDetail(db *mgo.Database, userID string, mode *string, typeName string, typeID *string, score int64) error {

	//For week with mode name
	leaderboardM := &model.LeaderboardModel{}

	err := db.C(leaderboardM.GetCollectionName()).Find(bson.M{
		"userID": userID,
		"mode": mode,
		"type": typeName,
		"typeID": typeID,
	}).One(leaderboardM)

	if err != nil {
		if err == mgo.ErrNotFound {
			leaderboardM.Score = score
			leaderboardM.Mode = mode
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

func (l Leaderboard) GetScores(typeName string, mode string, page int, itemCount int) ([]model.LeaderboardModel, error) {

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

	//TODO: we can check mode name if exists from game holder
	var modeQ *string
	if mode != "all" {
		modeQ = &mode
	}

	conn := l.db.Copy()
	defer conn.Close()
	db := conn.DB("spaceship")
	err := db.C(model.LeaderboardModel{}.GetCollectionName()).Find(bson.M{
		"type": typeName,
		"mode": modeQ,
		"typeID": typeID,
	}).Sort("-score").Skip(page*itemCount).Limit(itemCount).All(&scores)
	if err != nil {
		return nil, err
	}

	return scores, nil

}