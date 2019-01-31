package server

import (
	"context"
	"github.com/globalsign/mgo/bson"
	"google.golang.org/grpc/status"
	"log"
	"spaceship/api"
	"spaceship/model"
	"strconv"
)

func (as *Server) GetLeaderboard(context context.Context, request *api.LeaderboardRequest) (*api.LeaderboardResponse, error) {

	page := 0
	itemPerPage := 20

	reqPage, err := strconv.Atoi(request.Page)
	if err == nil {
		page = reqPage
	}

	response := &api.LeaderboardResponse{
		Page: int32(page),
		HasNextPage: true,
	}

	scores, err := as.leaderboard.GetScores(request.Type, request.Mode, page, itemPerPage)
	if err != nil {
		return nil, status.Error(500, err.Error())
	}

	if len(scores) == itemPerPage {
		response.HasNextPage = true
	}else{
		response.HasNextPage = false
	}

	scoresPb := make([]*api.Leaderboard, 0)
	conn := as.db.Copy()
	defer conn.Close()
	db := conn.DB("spaceship")

	for _, score := range scores {

		var user model.User
		err = db.C(model.User{}.GetCollectionName()).Find(bson.M{
			"_id": score.UserID,
		}).One(&user)
		if err != nil {
			log.Println(err)
		}
		scorePb := api.Leaderboard{}
		scorePb.User = user.MapToPB()
		scorePb.Score = score.Score

		scoresPb = append(scoresPb, &scorePb)

	}

	response.Items = scoresPb
	response.ItemCount = int32(len(scoresPb))

	return response, nil

}
